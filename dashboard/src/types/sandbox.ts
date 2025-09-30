import { SandboxInfo, SandboxType } from '@api-client-ts';

export { SandboxInfo, SandboxType };

export interface SandboxItemProps {
  sandbox: SandboxInfo;
  isSelected: boolean;
  onSelect: () => void;
  onDelete: () => void;
  onStart: () => void;
  onStop: () => void;
}

export interface SandboxListProps {
  projectId: string;
  onSandboxSelect: (sandbox: SandboxInfo) => void;
  selectedSandboxId?: string;
}

export interface SandboxOperation {
  type: 'start' | 'stop' | 'delete' | 'create';
  sandboxId: string;
  payload?: any;
}