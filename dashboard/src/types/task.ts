import {
  RunClaudeTaskResponse,
  RunClaudeTaskResponseType,
  StreamTaskLogsResponse,
  StreamTaskLogsResponseType,
  TaskInfo
} from '@api-client-ts';

export { RunClaudeTaskResponse, RunClaudeTaskResponseType, StreamTaskLogsResponse, StreamTaskLogsResponseType, TaskInfo };

export interface LogEntry {
  type: RunClaudeTaskResponseType | StreamTaskLogsResponseType;
  content: string;
  timestamp: number;
  exitCode?: number;
  isFinished?: boolean;
  taskCompleted?: boolean;
  taskStatus?: string;
}

export interface TaskLogsProps {
  sandboxId: string;
  onTaskComplete: (exitCode: number) => void;
}

export interface TaskPromptProps {
  sandboxId: string;
  onTaskStart: (taskDescription: string) => void;
  isTaskRunning: boolean;
  // Task functions passed from parent (to avoid duplicate useTasks calls)
  runStreamingTask: (taskDescription: string) => Promise<void>;
  taskHistory: TaskHistory[];
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
  tasks: TaskInfo[];
}