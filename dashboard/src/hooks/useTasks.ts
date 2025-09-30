import { useState, useCallback, useEffect } from 'react';
import { RunClaudeTaskResponse } from '@api-client-ts';
import { apiService } from '../services/api';
import { useDashboard } from '../contexts/DashboardContext';
import { LogEntry, TaskHistory } from '../types/task';

export function useTasks(sandboxId?: string) {
  const { state, dispatch } = useDashboard();
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Load existing logs when sandbox changes
  const loadExistingLogs = useCallback(async () => {
    if (!sandboxId) return;

    try {
      setIsLoading(true);
      setError(null);

      // Get recent logs for this sandbox (this returns task list)
      const taskListResponse = await apiService.listClaudeTasks(sandboxId);

      if (taskListResponse.success && taskListResponse.logs && taskListResponse.logs.length > 0) {
        const logEntries: LogEntry[] = [];

        // Parse the task list to extract task IDs
        const taskIds: string[] = [];
        for (const logLine of taskListResponse.logs) {
          const trimmedLine = logLine.trim();
          if (trimmedLine.startsWith('🔹 ')) {
            const taskId = trimmedLine.replace('🔹 ', '').trim();
            taskIds.push(taskId);
          }
        }

        // Add header
        logEntries.push({
          type: 'STATUS',
          content: `📋 Found ${taskIds.length} task(s) for sandbox '${sandboxId}'`,
          timestamp: Date.now(),
          exitCode: 0,
          isFinished: false
        });

        if (taskIds.length === 0) {
          logEntries.push({
            type: 'STATUS',
            content: '💡 No Claude tasks found in this sandbox',
            timestamp: Date.now(),
            exitCode: 0,
            isFinished: false
          });
        } else {
          // Fetch logs for each task (limit to most recent 5 to avoid too many requests)
          const recentTaskIds = taskIds.slice(0, 5);

          for (const taskId of recentTaskIds) {
            try {
              // Extract timestamp from task ID
              const timestampMatch = taskId.match(/claude_(\d+)/);
              let taskTimestamp = Date.now();
              if (timestampMatch) {
                const nsTimestamp = parseInt(timestampMatch[1]);
                // Convert nanosecond timestamp to milliseconds
                // The timestamp appears to be nanoseconds since epoch
                taskTimestamp = Math.floor(nsTimestamp / 1000000);

                // If the timestamp is too large (future date), it might be microseconds
                if (taskTimestamp > Date.now() + 86400000) { // More than 1 day in future
                  taskTimestamp = Math.floor(nsTimestamp / 1000);
                }

                // If still too large or too small, use current time
                if (taskTimestamp < 1000000000000 || taskTimestamp > Date.now() + 86400000) {
                  taskTimestamp = Date.now() - (Math.random() * 3600000); // Random time within last hour
                }
              }

              // Add task header
              logEntries.push({
                type: 'STATUS',
                content: `\n🔄 Task: ${taskId}`,
                timestamp: taskTimestamp,
                exitCode: 0,
                isFinished: false
              });

              // Fetch actual logs for this task
              const taskLogsResponse = await apiService.getClaudeLogs(sandboxId, taskId);

              if (taskLogsResponse.success && taskLogsResponse.logs) {
                for (const logLine of taskLogsResponse.logs) {
                  const trimmedLine = logLine.trim();
                  if (!trimmedLine ||
                      trimmedLine.startsWith('📄 ') ||
                      trimmedLine === '─────────────────────────────────────' ||
                      trimmedLine === '(log file is empty)') {
                    continue; // Skip headers and separators
                  }

                  logEntries.push({
                    type: 'STDOUT',
                    content: trimmedLine,
                    timestamp: taskTimestamp + logEntries.length,
                    exitCode: 0,
                    isFinished: false
                  });
                }
              } else {
                logEntries.push({
                  type: 'STDERR',
                  content: `⚠️ Could not load logs for task ${taskId}`,
                  timestamp: taskTimestamp,
                  exitCode: 1,
                  isFinished: false
                });
              }

            } catch (taskErr) {
              console.warn(`Failed to load logs for task ${taskId}:`, taskErr);
              logEntries.push({
                type: 'STDERR',
                content: `❌ Error loading logs for task ${taskId}: ${taskErr instanceof Error ? taskErr.message : 'Unknown error'}`,
                timestamp: Date.now(),
                exitCode: 1,
                isFinished: false
              });
            }
          }

          if (taskIds.length > 5) {
            logEntries.push({
              type: 'STATUS',
              content: `\n... and ${taskIds.length - 5} more task(s). Use individual task log commands for older tasks.`,
              timestamp: Date.now(),
              exitCode: 0,
              isFinished: false
            });
          }
        }

        // Update the logs in state
        dispatch({
          type: 'SET_TASKS',
          payload: {
            logs: logEntries,
            isRunning: false,
            currentTask: undefined,
            history: []
          }
        });
      } else {
        // No tasks found
        dispatch({
          type: 'SET_TASKS',
          payload: {
            logs: [{
              type: 'STATUS',
              content: '📋 No task logs found for this sandbox',
              timestamp: Date.now(),
              exitCode: 0,
              isFinished: false
            }],
            isRunning: false,
            currentTask: undefined,
            history: []
          }
        });
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load existing logs';
      setError(errorMessage);
      console.warn('Failed to load existing logs:', err);

      // Clear logs on error
      dispatch({
        type: 'SET_TASKS',
        payload: {
          logs: [{
            type: 'ERROR',
            content: `❌ Failed to load task logs: ${errorMessage}`,
            timestamp: Date.now(),
            exitCode: 1,
            isFinished: false
          }],
          isRunning: false,
          currentTask: undefined,
          history: []
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
  }, [sandboxId, loadExistingLogs]);

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
    runTask,
    clearLogs,
    getTaskHistory,
    loadExistingLogs
  };
}