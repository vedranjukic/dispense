import { SandboxType, RunClaudeTaskResponseType } from '@api-client-ts';

export function formatTimestamp(timestamp: number | string): string {
  const date = new Date(typeof timestamp === 'string' ? timestamp : timestamp);

  const now = new Date();
  const diff = now.getTime() - date.getTime();

  // Less than 1 minute
  if (diff < 60000) {
    return 'just now';
  }

  // Less than 1 hour
  if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000);
    return `${minutes} minute${minutes === 1 ? '' : 's'} ago`;
  }

  // Less than 1 day
  if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000);
    return `${hours} hour${hours === 1 ? '' : 's'} ago`;
  }

  // More than 1 day
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
}

export function formatSandboxType(type: SandboxType): string {
  switch (type) {
    case SandboxType.LOCAL:
      return 'Local';
    case SandboxType.REMOTE:
      return 'Remote';
    default:
      return 'Unknown';
  }
}

export function formatLogType(type: RunClaudeTaskResponseType): string {
  switch (type) {
    case RunClaudeTaskResponseType.STDOUT:
      return 'STDOUT';
    case RunClaudeTaskResponseType.STDERR:
      return 'STDERR';
    case RunClaudeTaskResponseType.STATUS:
      return 'STATUS';
    case RunClaudeTaskResponseType.ERROR:
      return 'ERROR';
    default:
      return 'UNKNOWN';
  }
}

export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';

  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function formatSandboxStatus(state: string): { text: string; color: string } {
  switch (state.toLowerCase()) {
    case 'running':
      return { text: 'Running', color: '#059669' };
    case 'stopped':
      return { text: 'Stopped', color: '#6b7280' };
    case 'starting':
      return { text: 'Starting', color: '#d97706' };
    case 'stopping':
      return { text: 'Stopping', color: '#d97706' };
    case 'error':
      return { text: 'Error', color: '#dc2626' };
    default:
      return { text: state, color: '#6b7280' };
  }
}

export function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) {
    return text;
  }
  return text.substring(0, maxLength) + '...';
}

export function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  } else {
    return `${seconds}s`;
  }
}

export function parseLogContent(content: string): { text: string; metadata?: any } {
  try {
    // Try to parse as JSON for structured logs
    const parsed = JSON.parse(content);
    return { text: parsed.message || content, metadata: parsed };
  } catch {
    // Return as plain text
    return { text: content };
  }
}