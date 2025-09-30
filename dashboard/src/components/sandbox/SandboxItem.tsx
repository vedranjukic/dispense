import React, { useState } from 'react';
import { SandboxItemProps } from '../../types/sandbox';
import { formatTimestamp, formatSandboxType, formatSandboxStatus } from '../../utils/formatters';

export default function SandboxItem({
  sandbox,
  isSelected,
  onSelect,
  onDelete,
  onStart,
  onStop
}: SandboxItemProps) {
  const [showActions, setShowActions] = useState(false);

  const statusInfo = formatSandboxStatus(sandbox.state);
  const isRunning = sandbox.state.toLowerCase() === 'running';
  const canStart = ['stopped', 'error'].includes(sandbox.state.toLowerCase());
  const canStop = ['running', 'starting'].includes(sandbox.state.toLowerCase());

  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    setShowActions(!showActions);
  };

  return (
    <div
      className={`relative p-3 rounded-lg border cursor-pointer transition-all duration-150 ${
        isSelected
          ? 'bg-blue-50 border-blue-200 shadow-md'
          : 'bg-white border-gray-200 hover:border-gray-300 hover:shadow-sm'
      }`}
      onClick={onSelect}
      onContextMenu={handleContextMenu}
    >
      {/* Main Content */}
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <div className="flex items-center mb-1">
            <div
              className="status-indicator mr-2"
              style={{ backgroundColor: statusInfo.color }}
            />
            <h4 className="text-sm font-medium text-gray-900 truncate">
              {sandbox.name}
            </h4>
          </div>

          <div className="flex items-center text-xs text-gray-500 space-x-2 mb-1">
            <span className="flex items-center">
              {sandbox.type === 1 ? (
                <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V4zm0 4a1 1 0 011-1h12a1 1 0 011 1v6a1 1 0 01-1 1H4a1 1 0 01-1-1V8z" clipRule="evenodd" />
                </svg>
              ) : (
                <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M2 5a2 2 0 012-2h8a2 2 0 012 2v10a2 2 0 002 2H4a2 2 0 01-2-2V5zm3 1h6v4H5V6zm6 6H5v2h6v-2z" clipRule="evenodd" />
                  <path d="M15 7h1a2 2 0 012 2v5.5a1.5 1.5 0 01-3 0V9a1 1 0 00-1-1h-1v1z" />
                </svg>
              )}
              {formatSandboxType(sandbox.type)}
            </span>
            <span>•</span>
            <span>{statusInfo.text}</span>
          </div>

          <div className="text-xs text-gray-400">
            Created {formatTimestamp(sandbox.created_at)}
          </div>
        </div>

        {/* Quick Actions */}
        <div className="flex items-center space-x-1 ml-2">
          {canStart && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onStart();
              }}
              className="p-1 text-gray-400 hover:text-green-600 rounded"
              title="Start sandbox"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clipRule="evenodd" />
              </svg>
            </button>
          )}

          {canStop && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onStop();
              }}
              className="p-1 text-gray-400 hover:text-yellow-600 rounded"
              title="Stop sandbox"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clipRule="evenodd" />
              </svg>
            </button>
          )}

          <button
            onClick={(e) => {
              e.stopPropagation();
              onDelete();
            }}
            className="p-1 text-gray-400 hover:text-red-600 rounded"
            title="Delete sandbox"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1-1H8a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>

      {/* Context Menu */}
      {showActions && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-white border border-gray-200 rounded-md shadow-lg z-10">
          <div className="py-1">
            {canStart && (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onStart();
                  setShowActions(false);
                }}
                className="flex items-center w-full px-3 py-2 text-sm text-gray-700 hover:bg-gray-100"
              >
                <svg className="w-4 h-4 mr-2" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clipRule="evenodd" />
                </svg>
                Start
              </button>
            )}
            {canStop && (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onStop();
                  setShowActions(false);
                }}
                className="flex items-center w-full px-3 py-2 text-sm text-gray-700 hover:bg-gray-100"
              >
                <svg className="w-4 h-4 mr-2" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clipRule="evenodd" />
                </svg>
                Stop
              </button>
            )}
            <button
              onClick={(e) => {
                e.stopPropagation();
                onDelete();
                setShowActions(false);
              }}
              className="flex items-center w-full px-3 py-2 text-sm text-red-700 hover:bg-red-50"
            >
              <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1-1H8a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              Delete
            </button>
          </div>
        </div>
      )}
    </div>
  );
}