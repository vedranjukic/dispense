import { useState, useCallback, useEffect, useRef } from 'react';
import { RunClaudeTaskResponse, StreamTaskLogsResponse, StreamTaskLogsResponseType } from '@api-client-ts';
import { apiService } from '../services/api';
import { useDashboard } from '../contexts/DashboardContext';
import { LogEntry, TaskHistory } from '../types/task';

export function useTasks(sandboxId?: string) {
  const { state, dispatch } = useDashboard();
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  // Guard against React 18 StrictMode double-effect invocation
  const hasLoadedForSandboxRef = useRef<Record<string, boolean>>({});

  // Load existing logs when sandbox changes
  const loadExistingLogs = useCallback(async () => {
    if (!sandboxId) return;

    // Prevent duplicate fetches when effects run twice in StrictMode
    if (hasLoadedForSandboxRef.current[sandboxId]) {
      return;
    }
    hasLoadedForSandboxRef.current[sandboxId] = true;

    try {
      setIsLoading(true);
      setError(null);

      // Get task list for this sandbox
      const taskListResponse = await apiService.listClaudeTasks(sandboxId);

      if (taskListResponse.success && taskListResponse.tasks) {
        const taskInfos = taskListResponse.tasks;

        // Build task history from the task info
        const history: TaskHistory[] = taskInfos.map(task => ({
          id: task.taskId,
          description: task.prompt,
          timestamp: parseInt(task.startedAt) * 1000 || Date.now() // Convert seconds to milliseconds
        })).slice(0, 10); // Keep last 10 tasks

        // Update the tasks in state
        dispatch({
          type: 'SET_TASKS',
          payload: {
            logs: [], // We'll phase out logs in favor of tasks
            isRunning: false,
            currentTask: undefined,
            history,
            tasks: taskInfos
          }
        });
      } else {
        // No tasks found
        dispatch({
          type: 'SET_TASKS',
          payload: {
            logs: [],
            isRunning: false,
            currentTask: undefined,
            history: [],
            tasks: []
          }
        });
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load existing logs';
      setError(errorMessage);
      console.warn('Failed to load existing logs:', err);

      // Clear tasks on error
      dispatch({
        type: 'SET_TASKS',
        payload: {
          logs: [],
          isRunning: false,
          currentTask: undefined,
          history: [],
          tasks: []
        }
      });
    } finally {
      setIsLoading(false);
    }
  }, [sandboxId, dispatch]);

  // Load existing logs when sandbox changes
  useEffect(() => {
    if (sandboxId) {
      loadExistingLogs();
    }
  }, [sandboxId]); // Only depend on sandboxId, not loadExistingLogs function

  const runTask = useCallback(async (taskDescription: string) => {
    if (!sandboxId || isRunning) return;

    try {
      setIsRunning(true);
      setError(null);

      // Clear previous logs
      dispatch({ type: 'SET_TASKS', payload: { ...state.tasks, logs: [], isRunning: true } });

      // Add task to history
      const taskId = Date.now().toString();
      const historyEntry: TaskHistory = {
        id: taskId,
        description: taskDescription,
        timestamp: Date.now()
      };

      const updatedTasks = {
        ...state.tasks,
        isRunning: true,
        currentTask: taskId,
        history: [historyEntry, ...state.tasks.history.slice(0, 9)] // Keep last 10 tasks
      };
      dispatch({ type: 'SET_TASKS', payload: updatedTasks });

      // Start async task - this will return immediately with task ID
      await apiService.runTask(sandboxId, taskDescription, (response: RunClaudeTaskResponse) => {
        const logEntry: LogEntry = {
          type: response.type,
          content: response.content,
          timestamp: response.timestamp,
          exitCode: response.exit_code,
          isFinished: response.is_finished
        };

        dispatch({ type: 'ADD_LOG_ENTRY', payload: logEntry });
      });

      // Since the API now returns immediately after creating the task,
      // we should reset the running state right after the API call
      setIsRunning(false);

      // Add a success message to indicate the task was started
      const successLogEntry: LogEntry = {
        type: 'STATUS',
        content: `✅ Task started successfully: "${taskDescription}"`,
        timestamp: Date.now(),
        exitCode: 0,
        isFinished: true
      };
      dispatch({ type: 'ADD_LOG_ENTRY', payload: successLogEntry });

      const resetTasks = {
        ...state.tasks,
        isRunning: false,
        currentTask: undefined
      };
      dispatch({ type: 'SET_TASKS', payload: resetTasks });

    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to run task';
      setError(errorMessage);
      setIsRunning(false);

      dispatch({ type: 'SET_TASKS', payload: { ...state.tasks, isRunning: false } });
    }
  }, [sandboxId, isRunning, dispatch, state.tasks]);

  // New streaming task method using the new API
  const runStreamingTask = useCallback(async (taskDescription: string) => {
    if (!sandboxId || isRunning) return;

    let retryCount = 0;
    const maxRetries = 3;
    const retryDelay = 2000; // 2 seconds

    const attemptStreamConnection = async (taskId: string): Promise<void> => {
      return new Promise((resolve, reject) => {
        console.log(`Starting log stream attempt ${retryCount + 1} for task:`, taskId);

        // Add connecting status message
        const connectingLogEntry: LogEntry = {
          type: StreamTaskLogsResponseType.STATUS,
          content: `🔗 Connecting to task log stream${retryCount > 0 ? ` (attempt ${retryCount + 1}/${maxRetries + 1})` : ''}...`,
          timestamp: Date.now(),
          exitCode: 0,
          isFinished: false
        };
        dispatch({ type: 'ADD_LOG_ENTRY', payload: connectingLogEntry });

        apiService.streamTaskLogs(
          {
            task_id: taskId,
            sandbox_identifier: sandboxId,
            follow: true,
            include_history: true
          },
          // onMessage callback
          (response: StreamTaskLogsResponse) => {
            console.log('Received log:', response);
            const logEntry: LogEntry = {
              type: response.type as StreamTaskLogsResponseType,
              content: response.content,
              timestamp: response.timestamp * 1000, // Convert to milliseconds
              taskCompleted: response.task_completed,
              taskStatus: response.task_status,
              isFinished: response.task_completed || false
            };

            dispatch({ type: 'ADD_LOG_ENTRY', payload: logEntry });
          },
          // onComplete callback
          (taskStatus: string) => {
            console.log('Task completed with status:', taskStatus);
            setIsRunning(false);

            const completionLogEntry: LogEntry = {
              type: StreamTaskLogsResponseType.STATUS,
              content: `✅ Task completed with status: ${taskStatus}`,
              timestamp: Date.now(),
              exitCode: taskStatus === 'COMPLETED' ? 0 : 1,
              isFinished: true,
              taskCompleted: true,
              taskStatus
            };
            dispatch({ type: 'ADD_LOG_ENTRY', payload: completionLogEntry });

            const resetTasks = {
              ...state.tasks,
              isRunning: false,
              currentTask: undefined
            };
            dispatch({ type: 'SET_TASKS', payload: resetTasks });
            resolve();
          },
          // onError callback
          (error: Error) => {
            console.error('Streaming error:', error);
            reject(error);
          }
        ).catch((error) => {
          console.error('Stream connection error:', error);
          reject(error);
        });
      });
    };

    try {
      setIsRunning(true);
      setError(null);

      // Clear previous logs
      dispatch({ type: 'SET_TASKS', payload: { ...state.tasks, logs: [], isRunning: true } });

      // Create task and get task ID
      console.log('Creating task...');
      const startLogEntry: LogEntry = {
        type: StreamTaskLogsResponseType.STATUS,
        content: `🚀 Creating task: "${taskDescription}"`,
        timestamp: Date.now(),
        exitCode: 0,
        isFinished: false
      };
      dispatch({ type: 'ADD_LOG_ENTRY', payload: startLogEntry });

      const taskId = await apiService.createTask(sandboxId, taskDescription);

      console.log('Task created with ID:', taskId);

      // Add task to history
      const historyEntry: TaskHistory = {
        id: taskId,
        description: taskDescription,
        timestamp: Date.now()
      };

      const updatedTasks = {
        ...state.tasks,
        isRunning: true,
        currentTask: taskId,
        history: [historyEntry, ...state.tasks.history.slice(0, 9)] // Keep last 10 tasks
      };
      dispatch({ type: 'SET_TASKS', payload: updatedTasks });

      // Add task created status message
      const createdLogEntry: LogEntry = {
        type: StreamTaskLogsResponseType.STATUS,
        content: `✅ Task created successfully (ID: ${taskId})`,
        timestamp: Date.now(),
        exitCode: 0,
        isFinished: false
      };
      dispatch({ type: 'ADD_LOG_ENTRY', payload: createdLogEntry });

      // Wait for daemon to initialize the task
      const initLogEntry: LogEntry = {
        type: StreamTaskLogsResponseType.STATUS,
        content: `⏳ Waiting for task to initialize...`,
        timestamp: Date.now(),
        exitCode: 0,
        isFinished: false
      };
      dispatch({ type: 'ADD_LOG_ENTRY', payload: initLogEntry });

      await new Promise(resolve => setTimeout(resolve, 1000));

      // Start streaming logs with retry logic
      while (retryCount <= maxRetries) {
        try {
          await attemptStreamConnection(taskId);
          break; // Success, exit retry loop
        } catch (streamError) {
          retryCount++;

          const errorMessage = streamError instanceof Error ? streamError.message : 'Unknown streaming error';
          console.error(`Stream attempt ${retryCount} failed:`, errorMessage);

          if (retryCount <= maxRetries) {
            // Add retry message
            const retryLogEntry: LogEntry = {
              type: StreamTaskLogsResponseType.STATUS,
              content: `⚠️ Connection failed: ${errorMessage}. Retrying in ${retryDelay/1000}s... (${retryCount}/${maxRetries})`,
              timestamp: Date.now(),
              exitCode: 0,
              isFinished: false
            };
            dispatch({ type: 'ADD_LOG_ENTRY', payload: retryLogEntry });

            // Wait before retry
            await new Promise(resolve => setTimeout(resolve, retryDelay));
          } else {
            // Max retries exceeded
            throw new Error(`Failed to connect to log stream after ${maxRetries + 1} attempts. Last error: ${errorMessage}`);
          }
        }
      }

    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to run streaming task';
      console.error('Task creation or streaming error:', err);
      setError(errorMessage);
      setIsRunning(false);

      const errorLogEntry: LogEntry = {
        type: StreamTaskLogsResponseType.ERROR,
        content: `❌ Failed to start task: ${errorMessage}`,
        timestamp: Date.now(),
        exitCode: 1,
        isFinished: true
      };
      dispatch({ type: 'ADD_LOG_ENTRY', payload: errorLogEntry });

      dispatch({ type: 'SET_TASKS', payload: { ...state.tasks, isRunning: false } });
    }
  }, [sandboxId, isRunning, dispatch, state.tasks]);

  const clearLogs = useCallback(() => {
    dispatch({ type: 'SET_TASKS', payload: { ...state.tasks, logs: [] } });
  }, [dispatch, state.tasks]);

  const getTaskHistory = useCallback((): TaskHistory[] => {
    return state.tasks.history;
  }, [state.tasks.history]);

  return {
    logs: state.tasks.logs,
    isRunning,
    isLoading,
    error,
    taskHistory: state.tasks.history,
    currentTask: state.tasks.currentTask,
    tasks: state.tasks.tasks || [],
    runTask,
    runStreamingTask,
    clearLogs,
    getTaskHistory,
    loadExistingLogs
  };
}