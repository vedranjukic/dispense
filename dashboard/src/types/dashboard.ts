import { SandboxInfo } from './sandbox';
import { TaskState } from './task';
import { FileItem } from './file';
import { TerminalLine } from './terminal';

export interface DashboardState {
  // Current selection
  selectedProject: string;
  selectedSandbox: SandboxInfo | null;
  selectedTask: string | null;

  // Data
  sandboxes: SandboxInfo[];
  tasks: TaskState;
  modifiedFiles: FileItem[];
  terminalOutput: TerminalLine[];

  // UI state
  isLoading: boolean;
  error: string | null;
  sidebarCollapsed: boolean;
  terminalHeight: number;
}

export interface DashboardAction {
  type: 'SET_SELECTED_PROJECT' | 'SET_SELECTED_SANDBOX' | 'SET_SELECTED_TASK' |
        'SET_SANDBOXES' | 'SET_TASKS' | 'SET_MODIFIED_FILES' | 'SET_TERMINAL_OUTPUT' |
        'SET_LOADING' | 'SET_ERROR' | 'TOGGLE_SIDEBAR' | 'SET_TERMINAL_HEIGHT' |
        'ADD_LOG_ENTRY' | 'ADD_TERMINAL_LINE' | 'UPDATE_SANDBOX';
  payload?: any;
}

export interface LayoutProps {
  children: React.ReactNode;
}