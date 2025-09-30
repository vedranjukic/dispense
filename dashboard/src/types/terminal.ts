export interface TerminalLine {
  id: string;
  content: string;
  timestamp: number;
  type: 'input' | 'output' | 'error';
}

export interface TerminalProps {
  sandboxId: string;
  onCommandExecute: (command: string) => void;
  isConnected: boolean;
}

export interface TerminalTab {
  id: string;
  name: string;
  isActive: boolean;
  sandboxId: string;
}

export interface TerminalSession {
  sandboxId: string;
  lines: TerminalLine[];
  commandHistory: string[];
  currentCommand: string;
  isConnected: boolean;
}