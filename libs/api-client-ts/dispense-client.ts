/**
 * TypeScript API client for Dispense REST endpoints
 * Generated based on the gRPC gateway configuration
 */

// Base types from proto definitions
export enum SandboxType {
  UNSPECIFIED = 0,
  LOCAL = 1,
  REMOTE = 2
}

export interface GitHubIssue {
  url: string;
  number: number;
  owner: string;
  repo: string;
  title: string;
  body: string;
}

export interface GitHubPR {
  url: string;
  number: number;
  owner: string;
  repo: string;
  title: string;
  body: string;
}

export interface TaskData {
  description: string;
  github_issue?: GitHubIssue;
  github_pr?: GitHubPR;
}

export interface ResourceAllocation {
  snapshot?: string;
  target?: string;
  cpu?: number;
  memory?: number;
  disk?: number;
  auto_stop?: number;
}

export interface SandboxInfo {
  id: string;
  name: string;
  type: SandboxType;
  state: string;
  shell_command: string;
  created_at: string; // ISO timestamp
  group?: string;
  metadata: Record<string, string>;
}

export interface ErrorResponse {
  code: string;
  message: string;
  details: Record<string, string>;
}

// Request/Response types
export interface CreateSandboxRequest {
  name: string;
  branch_name?: string;
  is_remote?: boolean;
  force?: boolean;
  skip_copy?: boolean;
  skip_daemon?: boolean;
  group?: string;
  model?: string;
  task?: string;
  resources?: ResourceAllocation;
  source_directory?: string;
  task_data?: TaskData;
}

export interface CreateSandboxResponse {
  sandbox?: SandboxInfo;
  error?: ErrorResponse;
}

export interface ListSandboxesRequest {
  show_local?: boolean;
  show_remote?: boolean;
  verbose?: boolean;
  group?: string;
}

export interface ListSandboxesResponse {
  sandboxes: SandboxInfo[];
  error?: ErrorResponse;
}

export interface DeleteSandboxRequest {
  identifier: string;
  delete_all?: boolean;
  force?: boolean;
}

export interface DeleteSandboxResponse {
  success: boolean;
  message: string;
  error?: ErrorResponse;
}

export interface GetSandboxRequest {
  identifier: string;
}

export interface GetSandboxResponse {
  sandbox?: SandboxInfo;
  error?: ErrorResponse;
}

export interface WaitForSandboxRequest {
  identifier: string;
  timeout_seconds?: number;
  group?: string;
}

export interface WaitForSandboxResponse {
  success: boolean;
  message: string;
  error?: ErrorResponse;
}

export interface CreateClaudeTaskRequest {
  sandbox_identifier: string;
  task_description: string;
  model?: string;
  working_directory?: string;
  environment_vars?: { [key: string]: string };
  anthropic_api_key?: string;
}

export interface CreateClaudeTaskResponse {
  success: boolean;
  task_id?: string;
  message?: string;
  error?: ErrorResponse;
}

export interface RunClaudeTaskRequest {
  sandbox_identifier: string;
  task_description: string;
  model?: string;
}

export enum RunClaudeTaskResponseType {
  STDOUT = 0,
  STDERR = 1,
  STATUS = 2,
  ERROR = 3
}

export interface RunClaudeTaskResponse {
  type: RunClaudeTaskResponseType;
  content: string;
  timestamp: number;
  exit_code?: number;
  is_finished?: boolean;
}

export interface GetClaudeStatusRequest {
  sandbox_identifier: string;
}

export interface GetClaudeStatusResponse {
  connected: boolean;
  daemon_info: string;
  work_dir: string;
  error?: ErrorResponse;
}

export interface GetClaudeLogsRequest {
  sandbox_identifier: string;
  task_id?: string;
}

export interface GetClaudeLogsResponse {
  success: boolean;
  logs: string[];
  error?: ErrorResponse;
}

export interface TaskInfo {
  taskId: string;
  prompt: string;
  state: string; // PENDING, RUNNING, COMPLETED, FAILED
  startedAt: string;
  finishedAt: string;
  exitCode: number;
  error: string;
  workingDirectory: string;
}

export interface ListClaudeTasksRequest {
  sandbox_identifier: string;
}

export interface ListClaudeTasksResponse {
  success: boolean;
  tasks: TaskInfo[];
  error?: ErrorResponse;
}

// Streaming task logs types
export enum StreamTaskLogsResponseType {
  STDOUT = 0,
  STDERR = 1,
  STATUS = 2,
  ERROR = 3
}

export interface StreamTaskLogsRequest {
  task_id: string;
  sandbox_identifier: string;
  follow?: boolean;
  include_history?: boolean;
  from_timestamp?: number;
}

export interface StreamTaskLogsResponse {
  type: StreamTaskLogsResponseType;
  content: string;
  timestamp: number;
  task_completed?: boolean;
  task_status?: string;
}

export interface GetAPIKeyRequest {
  interactive?: boolean;
}

export interface GetAPIKeyResponse {
  api_key: string;
  error?: ErrorResponse;
}

export interface SetAPIKeyRequest {
  api_key: string;
}

export interface SetAPIKeyResponse {
  success: boolean;
  message: string;
  error?: ErrorResponse;
}

export interface ValidateAPIKeyRequest {
  api_key: string;
}

export interface ValidateAPIKeyResponse {
  valid: boolean;
  message: string;
  error?: ErrorResponse;
}

export interface HealthResponse {
  status: string;
  service: string;
}

// Client configuration
export interface DispenseClientConfig {
  baseUrl?: string;
  apiKey?: string;
  timeout?: number;
}

// HTTP client class
export class DispenseClient {
  private baseUrl: string;
  private apiKey?: string;
  private timeout: number;

  constructor(config: DispenseClientConfig = {}) {
    this.baseUrl = (config.baseUrl || window.location.origin) + '/api';
    this.apiKey = config.apiKey;
    this.timeout = config.timeout || 30000;
  }

  // Helper method to make HTTP requests
  private async request<T>(
    method: string,
    path: string,
    body?: any,
    options: { stream?: boolean; timeout?: number } = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    // Add API key if available
    if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }

    // Handle streaming requests
    if (options.stream) {
      headers['Accept'] = 'text/event-stream';
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), options.timeout || this.timeout);

    try {
      const response = await fetch(url, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
        try {
          const errorData = await response.json();
          if (errorData.error) {
            errorMessage = `${errorData.error.code}: ${errorData.error.message}`;
          }
        } catch {
          // Ignore JSON parsing errors for error responses
        }
        throw new Error(errorMessage);
      }

      // Handle streaming responses
      if (options.stream) {
        return response as T;
      }

      return await response.json();
    } catch (error) {
      clearTimeout(timeoutId);
      if (error instanceof Error && error.name === 'AbortError') {
        throw new Error('Request timeout');
      }
      throw error;
    }
  }

  // Sandbox management methods
  async createSandbox(request: CreateSandboxRequest): Promise<CreateSandboxResponse> {
    return this.request<CreateSandboxResponse>('POST', '/v1/sandboxes', request);
  }

  async listSandboxes(params?: {
    show_local?: boolean;
    show_remote?: boolean;
    verbose?: boolean;
  }): Promise<ListSandboxesResponse> {
    const searchParams = new URLSearchParams();
    if (params?.show_local !== undefined) searchParams.set('show_local', params.show_local.toString());
    if (params?.show_remote !== undefined) searchParams.set('show_remote', params.show_remote.toString());
    if (params?.verbose !== undefined) searchParams.set('verbose', params.verbose.toString());

    const query = searchParams.toString();
    return this.request<ListSandboxesResponse>('GET', `/v1/sandboxes${query ? '?' + query : ''}`);
  }

  async getSandbox(identifier: string): Promise<GetSandboxResponse> {
    return this.request<GetSandboxResponse>('GET', `/v1/sandboxes/${encodeURIComponent(identifier)}`);
  }

  async deleteSandbox(
    identifier: string,
    params?: { delete_all?: boolean; force?: boolean }
  ): Promise<DeleteSandboxResponse> {
    const searchParams = new URLSearchParams();
    if (params?.delete_all !== undefined) searchParams.set('delete_all', params.delete_all.toString());
    if (params?.force !== undefined) searchParams.set('force', params.force.toString());

    const query = searchParams.toString();
    return this.request<DeleteSandboxResponse>(
      'DELETE',
      `/v1/sandboxes/${encodeURIComponent(identifier)}${query ? '?' + query : ''}`
    );
  }

  async waitForSandbox(identifier: string, request: Omit<WaitForSandboxRequest, 'identifier'>): Promise<WaitForSandboxResponse> {
    return this.request<WaitForSandboxResponse>(
      'POST',
      `/v1/sandboxes/${encodeURIComponent(identifier)}/wait`,
      request
    );
  }

  // Claude operations
  async createClaudeTask(request: CreateClaudeTaskRequest): Promise<CreateClaudeTaskResponse> {
    return this.request<CreateClaudeTaskResponse>(
      'POST',
      `/v1/sandboxes/${encodeURIComponent(request.sandbox_identifier)}/tasks`,
      request
    );
  }

  async runClaudeTask(
    request: RunClaudeTaskRequest,
    onMessage?: (response: RunClaudeTaskResponse) => void
  ): Promise<void> {
    const response = await this.request<Response>('POST', '/v1/claude/tasks', request, { stream: true });

    if (!response.body) {
      throw new Error('No response body for streaming request');
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();

    try {
      while (true) {
        const { value, done } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value);
        const lines = chunk.split('\n');

        for (const line of lines) {
          if (line.startsWith('data: ') && line.length > 6) {
            try {
              const data = JSON.parse(line.slice(6));
              if (onMessage) {
                onMessage(data as RunClaudeTaskResponse);
              }

              // Break if task is finished
              if (data.is_finished) {
                return;
              }
            } catch (error) {
              console.warn('Failed to parse SSE data:', error);
            }
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }

  async getClaudeStatus(sandbox_identifier: string): Promise<GetClaudeStatusResponse> {
    return this.request<GetClaudeStatusResponse>('GET', `/v1/claude/${encodeURIComponent(sandbox_identifier)}/status`);
  }

  async getClaudeLogs(sandbox_identifier: string, task_id?: string): Promise<GetClaudeLogsResponse> {
    const searchParams = new URLSearchParams();
    if (task_id) searchParams.set('task_id', task_id);

    const query = searchParams.toString();
    return this.request<GetClaudeLogsResponse>(
      'GET',
      `/v1/claude/${encodeURIComponent(sandbox_identifier)}/logs${query ? '?' + query : ''}`
    );
  }

  async listClaudeTasks(sandbox_identifier: string): Promise<ListClaudeTasksResponse> {
    return this.request<ListClaudeTasksResponse>(
      'GET',
      `/v1/claude/${encodeURIComponent(sandbox_identifier)}/tasks`
    );
  }

  // Stream task logs with real-time updates
  async streamTaskLogs(
    request: StreamTaskLogsRequest,
    onMessage?: (response: StreamTaskLogsResponse) => void,
    onComplete?: (taskStatus: string) => void,
    onError?: (error: Error) => void
  ): Promise<void> {
    try {
      const searchParams = new URLSearchParams();
      if (request.follow !== undefined) searchParams.set('follow', request.follow.toString());
      if (request.include_history !== undefined) searchParams.set('include_history', request.include_history.toString());
      if (request.from_timestamp !== undefined) searchParams.set('from_timestamp', request.from_timestamp.toString());

      const query = searchParams.toString();
      const response = await this.request<Response>(
        'GET',
        `/v1/sandboxes/${encodeURIComponent(request.sandbox_identifier)}/tasks/${encodeURIComponent(request.task_id)}/logs/stream${query ? '?' + query : ''}`,
        undefined,
        { stream: true }
      );

      if (!response.body) {
        throw new Error('No response body for streaming request');
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();

      try {
        while (true) {
          const { value, done } = await reader.read();
          if (done) break;

          const chunk = decoder.decode(value);
          const lines = chunk.split('\n');

          for (const line of lines) {
            if (line.startsWith('data: ') && line.length > 6) {
              try {
                const data = JSON.parse(line.slice(6));
                const logResponse = data as StreamTaskLogsResponse;

                if (onMessage) {
                  onMessage(logResponse);
                }

                // Check if task is completed
                if (logResponse.task_completed) {
                  if (onComplete && logResponse.task_status) {
                    onComplete(logResponse.task_status);
                  }
                  return;
                }
              } catch (error) {
                console.warn('Failed to parse SSE data:', error);
                if (onError) {
                  onError(new Error(`Failed to parse streaming data: ${error}`));
                }
              }
            }
          }
        }
      } finally {
        reader.releaseLock();
      }
    } catch (error) {
      if (onError) {
        onError(error instanceof Error ? error : new Error(String(error)));
      } else {
        throw error;
      }
    }
  }

  // Configuration management
  async getAPIKey(interactive?: boolean): Promise<GetAPIKeyResponse> {
    const searchParams = new URLSearchParams();
    if (interactive !== undefined) searchParams.set('interactive', interactive.toString());

    const query = searchParams.toString();
    return this.request<GetAPIKeyResponse>('GET', `/v1/config/api-key${query ? '?' + query : ''}`);
  }

  async validateAPIKey(request: ValidateAPIKeyRequest): Promise<ValidateAPIKeyResponse> {
    return this.request<ValidateAPIKeyResponse>('POST', '/v1/config/api-key/validate', request);
  }

  // Health checks
  async healthCheck(): Promise<HealthResponse> {
    return this.request<HealthResponse>('GET', '/health');
  }

  async readinessCheck(): Promise<HealthResponse> {
    return this.request<HealthResponse>('GET', '/ready');
  }

  // Utility methods
  setAPIKey(apiKey: string): void {
    this.apiKey = apiKey;
  }

  getBaseUrl(): string {
    return this.baseUrl;
  }

  setTimeout(timeout: number): void {
    this.timeout = timeout;
  }
}

// Export a default instance
export const dispenseClient = new DispenseClient();

// Usage examples (commented out to avoid execution)
/*
// Basic usage
const client = new DispenseClient({
  baseUrl: 'http://localhost:8081',
  apiKey: 'your-api-key'
});

// Create a sandbox
const sandbox = await client.createSandbox({
  name: 'my-sandbox',
  is_remote: false
});

// List sandboxes
const sandboxes = await client.listSandboxes({
  show_local: true,
  show_remote: true
});

// Run a Claude task with streaming
await client.runClaudeTask({
  sandbox_identifier: 'my-sandbox',
  task_description: 'List all files in the current directory'
}, (response) => {
  console.log('Task output:', response.content);
  if (response.is_finished) {
    console.log('Task completed with exit code:', response.exit_code);
  }
});

// Stream task logs in real-time (new streaming functionality)
await client.streamTaskLogs({
  task_id: 'claude_1234567890',
  sandbox_identifier: 'my-sandbox',
  follow: true,
  include_history: true
},
// onMessage callback
(logResponse) => {
  console.log(`[${logResponse.type}] ${logResponse.content}`);
},
// onComplete callback
(taskStatus) => {
  console.log(`Task completed with status: ${taskStatus}`);
},
// onError callback
(error) => {
  console.error('Streaming error:', error.message);
});

// Stream task logs without following (get historical logs only)
await client.streamTaskLogs({
  task_id: 'claude_1234567890',
  sandbox_identifier: 'my-sandbox',
  follow: false,
  include_history: true,
  from_timestamp: Date.now() - 3600000 // Last hour
}, (logResponse) => {
  console.log('Historical log:', logResponse.content);
});

// Get Claude status
const status = await client.getClaudeStatus('my-sandbox');
console.log('Claude connected:', status.connected);

// List Claude tasks for a sandbox
const tasksResponse = await client.listClaudeTasks('my-sandbox');
if (tasksResponse.success) {
  console.log('Tasks:', tasksResponse.tasks);
  tasksResponse.tasks.forEach(task => {
    console.log(`Task ${task.task_id}: ${task.state} - ${task.prompt}`);
  });
}

// Clean up
await client.deleteSandbox('my-sandbox');
*/