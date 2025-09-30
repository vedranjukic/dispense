import { RunClaudeTaskResponse, RunClaudeTaskResponseType } from '@api-client-ts';

export { RunClaudeTaskResponse, RunClaudeTaskResponseType };

export interface LogEntry {
  type: RunClaudeTaskResponseType;
  content: string;
  timestamp: number;
  exitCode?: number;
  isFinished?: boolean;
}

export interface TaskLogsProps {
  sandboxId: string;
  taskId?: string;
  onTaskComplete: (exitCode: number) => void;
}

export interface TaskPromptProps {
  sandboxId: string;
  onTaskStart: (taskDescription: string) => void;
  isTaskRunning: boolean;
}

export interface TaskHistory {
  id: string;
  description: string;
  timestamp: number;
  exitCode?: number;
}

export interface TaskState {
  isRunning: boolean;
  currentTask?: string;
  logs: LogEntry[];
  history: TaskHistory[];
}