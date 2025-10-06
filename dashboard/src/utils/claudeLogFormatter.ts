/**
 * Claude log formatting utilities for human-readable display
 * Ported from daemon/internal/server/log_formatter.go
 */

// Claude message types from Claude Code
interface ClaudeMessage {
  type: string;
  message?: any;
  event?: any;
  parent_tool_use_id?: string;
  session_id: string;
  uuid: string;
}

interface AssistantMessage {
  id: string;
  type: string;
  role: string;
  model: string;
  content: ContentBlock[];
}

interface ContentBlock {
  type: string;
  text?: string;
  id?: string;
  name?: string;
  input?: Record<string, any>;
}

interface StreamEvent {
  type: string;
  index?: number;
  delta?: {
    type: string;
    text?: string;
  };
}

interface UserMessage {
  id: string;
  type: string;
  role: string;
  content: string;
}

interface SystemMessage {
  type: string;
  current_working_dir: string;
  tools: string[];
  permission_mode: string;
  api_key_source: string;
  model: string;
  additional_metadata: Record<string, any>;
}

interface ResultMessage {
  type: string;
  duration_ms: number;
  input_tokens: number;
  output_tokens: number;
  total_cost: number;
  permission_denial?: string;
}

/**
 * Formats Claude JSON output line into human-readable format
 */
export function formatClaudeOutput(line: string, taskPrompt?: string): string {
  // Skip empty lines
  if (!line || line.trim() === '') {
    return '';
  }

  // Try to parse as Claude message
  let claudeMsg: ClaudeMessage;
  try {
    claudeMsg = JSON.parse(line);
  } catch (error) {
    // If it's not JSON, return as plain text with emoji
    return `💬 ${line}`;
  }

  switch (claudeMsg.type) {
    case 'user':
      return formatUserMessage(claudeMsg, taskPrompt);
    case 'assistant':
      return formatAssistantMessage(claudeMsg);
    case 'stream_event':
      return formatStreamEvent(claudeMsg);
    case 'system':
      return formatSystemMessage(claudeMsg);
    case 'result':
      return formatResultMessage(claudeMsg);
    default:
      // Fallback for unknown types
      return `❓ Unknown message type: ${claudeMsg.type}`;
  }
}

/**
 * Formats user input messages
 */
function formatUserMessage(msg: ClaudeMessage, taskPrompt?: string): string {
  const parts: string[] = [];

  parts.push('👤 **Task Started**');

  if (taskPrompt) {
    parts.push(`**Prompt**: ${taskPrompt}`);
  }

  // Try to parse the message content
  if (msg.message) {
    try {
      const userMsg = msg.message as UserMessage;
      if (userMsg.content) {
        parts.push(`**Request**: ${userMsg.content}`);
      }
    } catch (error) {
      // Ignore parsing errors
    }
  }

  return parts.join(' - ');
}

/**
 * Formats assistant messages
 */
function formatAssistantMessage(msg: ClaudeMessage): string {
  const parts: string[] = [];

  // Try to parse the message content
  if (msg.message) {
    try {
      const assistantMsg = msg.message as AssistantMessage;

      for (const content of assistantMsg.content || []) {
        switch (content.type) {
          case 'text':
            if (content.text) {
              parts.push(`🤖 ${content.text}`);
            }
            break;
          case 'tool_use':
            const toolDesc = `🛠️ **Using ${content.name}**`;
            const inputStr = formatToolInput(content.input || {});
            if (inputStr) {
              parts.push(`${toolDesc} - ${inputStr}`);
            } else {
              parts.push(toolDesc);
            }
            break;
        }
      }
    } catch (error) {
      return '❓ Could not parse assistant message';
    }
  }

  return parts.join('\n');
}

/**
 * Formats streaming events
 */
function formatStreamEvent(msg: ClaudeMessage): string {
  if (!msg.event) {
    return '';
  }

  try {
    const streamEvent = msg.event as StreamEvent;

    switch (streamEvent.type) {
      case 'content_block_start':
        return '⏳ Claude is thinking...';
      case 'content_block_delta':
        if (streamEvent.delta?.text) {
          return streamEvent.delta.text;
        }
        break;
      case 'content_block_stop':
        return '✅ Response complete';
      case 'message_start':
        return '🚀 **Claude Started Working**';
      case 'message_stop':
        return '🏁 **Claude Finished**';
    }
  } catch (error) {
    // Ignore parsing errors
  }

  return '';
}

/**
 * Formats system messages
 */
function formatSystemMessage(msg: ClaudeMessage): string {
  const parts: string[] = ['⚙️ **System Initialized**'];

  // Handle both nested message format and direct properties format
  const sysData = (msg as any).message || msg;

  try {
    if (sysData.cwd || sysData.current_working_dir) {
      parts.push(`Working Directory: ${sysData.cwd || sysData.current_working_dir}`);
    }
    if (sysData.model) {
      parts.push(`Model: ${sysData.model}`);
    }
    if (sysData.tools && Array.isArray(sysData.tools) && sysData.tools.length > 0) {
      parts.push(`Tools: ${sysData.tools.slice(0, 5).join(', ')}${sysData.tools.length > 5 ? '...' : ''}`);
    }

    return parts.join(' - ');
  } catch (error) {
    return '⚙️ System message received';
  }
}

/**
 * Formats result messages
 */
function formatResultMessage(msg: ClaudeMessage): string {
  const parts: string[] = ['📊 **Task Summary**'];

  // Handle both nested message format and direct properties format
  const resultData = (msg as any).message || msg;

  try {
    if (resultData.duration_ms) {
      const duration = formatDuration(resultData.duration_ms);
      parts.push(`Duration: ${duration}`);
    }

    // Handle usage data (could be in usage object or direct properties)
    const usage = resultData.usage || resultData;
    if (usage.input_tokens > 0 || usage.output_tokens > 0) {
      parts.push(`Tokens: ${usage.input_tokens || 0} input, ${usage.output_tokens || 0} output`);
    }

    // Handle cost (could be total_cost_usd or total_cost)
    const cost = resultData.total_cost_usd || resultData.total_cost;
    if (cost > 0) {
      parts.push(`Cost: $${cost.toFixed(4)}`);
    }

    if (resultData.permission_denials && Array.isArray(resultData.permission_denials) && resultData.permission_denials.length > 0) {
      parts.push(`Permission Denied: ${resultData.permission_denials.join(', ')}`);
    }

    return parts.join(' - ');
  } catch (error) {
    return '📊 Task completed';
  }
}

/**
 * Formats tool input parameters in a readable way
 */
function formatToolInput(input: Record<string, any>): string {
  if (!input || Object.keys(input).length === 0) {
    return '';
  }

  const parts: string[] = [];

  for (const [key, value] of Object.entries(input)) {
    if (typeof value === 'string') {
      if (value.length > 100) {
        parts.push(`${key}: ${value.substring(0, 100)}...`);
      } else {
        parts.push(`${key}: ${value}`);
      }
    } else if (Array.isArray(value)) {
      parts.push(`${key}: [${value.length} items]`);
    } else if (typeof value === 'object' && value !== null) {
      parts.push(`${key}: {...}`);
    } else {
      parts.push(`${key}: ${value}`);
    }
  }

  return parts.join(', ');
}

/**
 * Formats duration in milliseconds to human-readable format
 */
function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  } else if (seconds > 0) {
    return `${seconds}s`;
  } else {
    return `${ms}ms`;
  }
}

/**
 * Checks if a log line contains Claude JSON output that should be formatted
 */
export function isClaudeJsonOutput(line: string): boolean {
  if (!line || line.trim() === '') {
    return false;
  }

  try {
    const parsed = JSON.parse(line);
    return typeof parsed === 'object' &&
           parsed !== null &&
           typeof parsed.type === 'string' &&
           ['user', 'assistant', 'stream_event', 'system', 'result'].includes(parsed.type);
  } catch (error) {
    return false;
  }
}