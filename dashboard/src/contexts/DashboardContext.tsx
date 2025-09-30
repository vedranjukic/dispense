import React, { createContext, useContext, useReducer, ReactNode } from 'react';
import { DashboardState, DashboardAction } from '../types/dashboard';

const initialState: DashboardState = {
  selectedProject: '',
  selectedSandbox: null,
  selectedTask: null,
  sandboxes: [],
  tasks: {
    isRunning: false,
    logs: [],
    history: []
  },
  modifiedFiles: [],
  terminalOutput: [],
  isLoading: false,
  error: null,
  sidebarCollapsed: false,
  terminalHeight: 200
};

function dashboardReducer(state: DashboardState, action: DashboardAction): DashboardState {
  switch (action.type) {
    case 'SET_SELECTED_PROJECT':
      return { ...state, selectedProject: action.payload };

    case 'SET_SELECTED_SANDBOX':
      return { ...state, selectedSandbox: action.payload };

    case 'SET_SELECTED_TASK':
      return { ...state, selectedTask: action.payload };

    case 'SET_SANDBOXES':
      return { ...state, sandboxes: action.payload };

    case 'SET_TASKS':
      return { ...state, tasks: action.payload };

    case 'SET_MODIFIED_FILES':
      return { ...state, modifiedFiles: action.payload };

    case 'SET_TERMINAL_OUTPUT':
      return { ...state, terminalOutput: action.payload };

    case 'SET_LOADING':
      return { ...state, isLoading: action.payload };

    case 'SET_ERROR':
      return { ...state, error: action.payload };

    case 'TOGGLE_SIDEBAR':
      return { ...state, sidebarCollapsed: !state.sidebarCollapsed };

    case 'SET_TERMINAL_HEIGHT':
      return { ...state, terminalHeight: action.payload };

    case 'ADD_LOG_ENTRY':
      return {
        ...state,
        tasks: {
          ...state.tasks,
          logs: [...state.tasks.logs, action.payload]
        }
      };

    case 'ADD_TERMINAL_LINE':
      return {
        ...state,
        terminalOutput: [...state.terminalOutput, action.payload]
      };

    case 'UPDATE_SANDBOX':
      return {
        ...state,
        sandboxes: state.sandboxes.map(sandbox =>
          sandbox.id === action.payload.id ? action.payload : sandbox
        )
      };

    default:
      return state;
  }
}

interface DashboardContextType {
  state: DashboardState;
  dispatch: React.Dispatch<DashboardAction>;
}

const DashboardContext = createContext<DashboardContextType | undefined>(undefined);

export function DashboardProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(dashboardReducer, initialState);

  return (
    <DashboardContext.Provider value={{ state, dispatch }}>
      {children}
    </DashboardContext.Provider>
  );
}

export function useDashboard() {
  const context = useContext(DashboardContext);
  if (context === undefined) {
    throw new Error('useDashboard must be used within a DashboardProvider');
  }
  return context;
}