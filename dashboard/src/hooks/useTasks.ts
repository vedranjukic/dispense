import { useState, useCallback } from 'react';
import { RunClaudeTaskResponse } from '@api-client-ts';
import { apiService } from '../services/api';
import { useDashboard } from '../contexts/DashboardContext';
import { LogEntry, TaskHistory } from '../types/task';

export function useTasks(sandboxId?: string) {
  const { state, dispatch } = useDashboard();
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

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
    error,
    taskHistory: state.tasks.history,
    currentTask: state.tasks.currentTask,
    runTask,
    clearLogs,
    getTaskHistory
  };
}