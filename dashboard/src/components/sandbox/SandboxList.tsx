import React from 'react';
import { useSandboxes } from '../../hooks/useSandboxes';
import SandboxItem from './SandboxItem';
import { SandboxListProps } from '../../types/sandbox';

export default function SandboxList({ projectId, onSandboxSelect, selectedSandboxId }: SandboxListProps) {
  const { sandboxes, deleteSandbox, isLoading, error } = useSandboxes(projectId);

  const handleDelete = async (sandboxId: string) => {
    if (window.confirm('Are you sure you want to delete this sandbox?')) {
      await deleteSandbox(sandboxId);
    }
  };

  if (isLoading && sandboxes.length === 0) {
    return (
      <div className="p-4">
        <div className="animate-pulse space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-16 bg-gray-200 rounded"></div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4">
        <div className="bg-red-50 border border-red-200 rounded-md p-3">
          <div className="flex">
            <svg className="h-5 w-5 text-red-400" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Error loading sandboxes</h3>
              <div className="mt-1 text-sm text-red-700">{error}</div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (sandboxes.length === 0) {
    return (
      <div className="p-4 text-center">
        <div className="py-8">
          <svg className="mx-auto h-8 w-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gray-900">No sandboxes</h3>
          <p className="mt-1 text-sm text-gray-500">Get started by creating a new sandbox.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="overflow-y-auto h-full">
      <div className="p-2 space-y-2">
        {sandboxes.map((sandbox) => (
          <SandboxItem
            key={sandbox.id}
            sandbox={sandbox}
            isSelected={sandbox.id === selectedSandboxId}
            onSelect={() => onSandboxSelect(sandbox)}
            onDelete={() => handleDelete(sandbox.id)}
            onStart={() => {
              // TODO: Implement start sandbox
              console.log('Start sandbox:', sandbox.id);
            }}
            onStop={() => {
              // TODO: Implement stop sandbox
              console.log('Stop sandbox:', sandbox.id);
            }}
          />
        ))}
      </div>
    </div>
  );
}