import React, { useState } from 'react';
import { useDashboard } from '../../contexts/DashboardContext';
import { useSandboxes } from '../../hooks/useSandboxes';
import SandboxList from '../sandbox/SandboxList';

export default function Sidebar() {
  const { state, dispatch } = useDashboard();
  const [projectId, setProjectId] = useState('default');
  const { sandboxes, createSandbox, isLoading } = useSandboxes(projectId);

  const handleToggleSidebar = () => {
    dispatch({ type: 'TOGGLE_SIDEBAR' });
  };

  const handleProjectChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newProjectId = e.target.value;
    setProjectId(newProjectId);
    dispatch({ type: 'SET_SELECTED_PROJECT', payload: newProjectId });
  };

  const handleSandboxSelect = (sandbox: any) => {
    dispatch({ type: 'SET_SELECTED_SANDBOX', payload: sandbox });
  };

  const handleCreateSandbox = async () => {
    const name = prompt('Enter sandbox name:');
    if (name) {
      await createSandbox(name, false);
    }
  };

  if (state.sidebarCollapsed) {
    return (
      <div className="h-full flex flex-col p-2">
        <button
          onClick={handleToggleSidebar}
          className="p-2 rounded hover:bg-gray-100"
          title="Expand sidebar"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </button>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold text-gray-900">Sandboxes</h2>
          <button
            onClick={handleToggleSidebar}
            className="p-1 rounded hover:bg-gray-100"
            title="Collapse sidebar"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
        </div>

        {/* Project Selector */}
        <div className="mb-3">
          <label htmlFor="project-select" className="block text-sm font-medium text-gray-700 mb-1">
            Project
          </label>
          <select
            id="project-select"
            value={projectId}
            onChange={handleProjectChange}
            className="w-full p-2 text-sm border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="default">Default</option>
            <option value="project-1">Project 1</option>
            <option value="project-2">Project 2</option>
          </select>
        </div>

        {/* Create Sandbox Button */}
        <button
          onClick={handleCreateSandbox}
          disabled={isLoading}
          className="w-full btn btn-primary btn-sm"
        >
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          New Sandbox
        </button>
      </div>

      {/* Sandbox List */}
      <div className="flex-1 overflow-hidden">
        <SandboxList
          projectId={projectId}
          onSandboxSelect={handleSandboxSelect}
          selectedSandboxId={state.selectedSandbox?.id}
        />
      </div>

      {/* Footer */}
      <div className="p-4 border-t border-gray-200 text-xs text-gray-500">
        {sandboxes.length} sandbox{sandboxes.length !== 1 ? 'es' : ''}
      </div>
    </div>
  );
}