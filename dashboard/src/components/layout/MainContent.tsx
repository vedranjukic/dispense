import React from 'react';
import { useDashboard } from '../../contexts/DashboardContext';
import TaskLogs from '../task/TaskLogs';
import TaskPrompt from '../task/TaskPrompt';

export default function MainContent() {
  const { state } = useDashboard();

  if (!state.selectedSandbox) {
    return (
      <div className="flex-1 flex items-center justify-center bg-white">
        <div className="text-center">
          <div className="mb-4">
            <svg
              className="mx-auto h-12 w-12 text-gray-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1}
                d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
              />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">Select a Sandbox</h3>
          <p className="text-gray-500">
            Choose a sandbox from the sidebar to view tasks and logs
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col bg-white overflow-hidden">
      {/* Header */}
      <div className="px-6 py-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-gray-900">
              {state.selectedSandbox.name}
            </h1>
            <p className="text-sm text-gray-500">
              {state.selectedSandbox.type === 1 ? 'Local' : 'Remote'} sandbox • {state.selectedSandbox.state}
            </p>
          </div>
          <div className="flex items-center space-x-2">
            <div className={`status-indicator status-${state.selectedSandbox.state}`}></div>
            <span className="text-sm font-medium text-gray-700">
              {state.selectedSandbox.state.charAt(0).toUpperCase() + state.selectedSandbox.state.slice(1)}
            </span>
          </div>
        </div>
      </div>

      {/* Task Logs */}
      <div className="flex-1 overflow-hidden">
        <TaskLogs
          sandboxId={state.selectedSandbox.id}
          onTaskComplete={(exitCode) => {
            console.log('Task completed with exit code:', exitCode);
          }}
        />
      </div>

      {/* Task Prompt */}
      <div className="border-t border-gray-200">
        <TaskPrompt
          sandboxId={state.selectedSandbox.id}
          onTaskStart={(description) => {
            console.log('Starting task:', description);
          }}
          isTaskRunning={state.tasks.isRunning}
        />
      </div>
    </div>
  );
}