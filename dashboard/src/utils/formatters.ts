import { SandboxType, RunClaudeTaskResponseType, StreamTaskLogsResponseType } from '@api-client-ts';
import { formatClaudeOutput, isClaudeJsonOutput } from './claudeLogFormatter';

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

export function formatLogType(type: RunClaudeTaskResponseType | StreamTaskLogsResponseType): string {
  // Handle both old and new log types
  switch (type) {
    case RunClaudeTaskResponseType.STDOUT:
    case StreamTaskLogsResponseType.STDOUT:
      return 'STDOUT';
    case RunClaudeTaskResponseType.STDERR:
    case StreamTaskLogsResponseType.STDERR:
      return 'STDERR';
    case RunClaudeTaskResponseType.STATUS:
    case StreamTaskLogsResponseType.STATUS:
      return 'STATUS';
    case RunClaudeTaskResponseType.ERROR:
    case StreamTaskLogsResponseType.ERROR:
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

export function parseLogContent(content: string, taskPrompt?: string): { text: string; metadata?: any } {
  // First check if it's a Claude JSON output that can be formatted
  if (isClaudeJsonOutput(content)) {
    const formattedText = formatClaudeOutput(content, taskPrompt);
    if (formattedText) {
      try {
        const parsed = JSON.parse(content);
        return { text: formattedText, metadata: parsed };
      } catch {
        return { text: formattedText };
      }
    }
  }

  try {
    // Try to parse as JSON for other structured logs
    const parsed = JSON.parse(content);
    return { text: parsed.message || content, metadata: parsed };
  } catch {
    // Return as plain text
    return { text: content };
  }
}

export function formatLogEntry(content: string, logType: string, timestamp: number, taskDescription?: string): { formattedText: string; isFormatted: boolean } {
  const timeStr = new Date(timestamp).toTimeString().substring(0, 8); // HH:MM:SS format

  // Handle different log types
  switch (logType) {
    case 'STDOUT':
      // Check if it's Claude JSON output
      if (isClaudeJsonOutput(content)) {
        const formattedContent = formatClaudeOutput(content, taskDescription);
        if (formattedContent) {
          return {
            formattedText: `[${timeStr}] ${formattedContent}`,
            isFormatted: true
          };
        }
      }
      // Plain stdout content - only show if it's not empty or meaningless
      if (content.trim() && !content.includes('thinking')) {
        return {
          formattedText: `[${timeStr}] 💬 ${content}`,
          isFormatted: true
        };
      }
      break;

    case 'STDERR':
      return {
        formattedText: `[${timeStr}] ⚠️ ${content}`,
        isFormatted: true
      };

    case 'ERROR':
      return {
        formattedText: `[${timeStr}] ❌ ${content}`,
        isFormatted: true
      };

    case 'STATUS':
      return {
        formattedText: `[${timeStr}] ℹ️ ${content}`,
        isFormatted: true
      };

    default:
      return {
        formattedText: `[${timeStr}] ${content}`,
        isFormatted: false
      };
  }

  return { formattedText: '', isFormatted: false };
}