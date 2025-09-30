import { DispenseClient, SandboxInfo, RunClaudeTaskResponse } from '@api-client-ts';
import { FileItem } from '../types/file';

export class DashboardAPIService {
  private client: DispenseClient;

  constructor() {
    this.client = new DispenseClient({
      baseUrl: window.location.origin,
      timeout: 30000
    });
  }

  // Sandbox operations
  async getSandboxes(projectId: string): Promise<SandboxInfo[]> {
    const response = await this.client.listSandboxes({
      show_local: true,
      show_remote: true
    });

    if (response.error) {
      throw new Error(`${response.error.code}: ${response.error.message}`);
    }

    return response.sandboxes;
  }

  async getSandbox(identifier: string): Promise<SandboxInfo> {
    const response = await this.client.getSandbox(identifier);

    if (response.error) {
      throw new Error(`${response.error.code}: ${response.error.message}`);
    }

    if (!response.sandbox) {
      throw new Error('Sandbox not found');
    }

    return response.sandbox;
  }

  async createSandbox(name: string, projectId?: string, isRemote?: boolean): Promise<SandboxInfo> {
    const response = await this.client.createSandbox({
      name,
      is_remote: isRemote || false
    });

    if (response.error) {
      throw new Error(`${response.error.code}: ${response.error.message}`);
    }

    if (!response.sandbox) {
      throw new Error('Failed to create sandbox');
    }

    return response.sandbox;
  }

  async deleteSandbox(identifier: string, force?: boolean): Promise<void> {
    const response = await this.client.deleteSandbox(identifier, { force });

    if (response.error) {
      throw new Error(`${response.error.code}: ${response.error.message}`);
    }

    if (!response.success) {
      throw new Error('Failed to delete sandbox');
    }
  }

  // Task operations
  async runTask(
    sandboxId: string,
    description: string,
    onMessage?: (response: RunClaudeTaskResponse) => void
  ): Promise<void> {
    await this.client.runClaudeTask({
      sandbox_identifier: sandboxId,
      task_description: description
    }, onMessage);
  }

  async getClaudeStatus(sandboxId: string) {
    const response = await this.client.getClaudeStatus(sandboxId);

    if (response.error) {
      throw new Error(`${response.error.code}: ${response.error.message}`);
    }

    return response;
  }

  async getClaudeLogs(sandboxId: string, taskId?: string) {
    const response = await this.client.getClaudeLogs(sandboxId, taskId);

    if (response.error) {
      throw new Error(`${response.error.code}: ${response.error.message}`);
    }

    return response;
  }

  // File operations (placeholder for future API endpoints)
  async getModifiedFiles(sandboxId: string): Promise<FileItem[]> {
    // This would need to be implemented when the API supports file listing
    // For now, return empty array or mock data
    return [];
  }

  // Health check
  async healthCheck() {
    return this.client.healthCheck();
  }

  // Configuration
  setAPIKey(apiKey: string): void {
    this.client.setAPIKey(apiKey);
  }

  getBaseUrl(): string {
    return this.client.getBaseUrl();
  }
}

// Export singleton instance
export const apiService = new DashboardAPIService();