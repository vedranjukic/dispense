import React, { useState } from 'react';
import { useDashboard } from '../../contexts/DashboardContext';
import ModifiedFilesList from '../files/ModifiedFilesList';
import Terminal from '../terminal/Terminal';

export default function RightPanel() {
  const { state, dispatch } = useDashboard();
  const [isResizing, setIsResizing] = useState(false);

  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);

    const startY = e.clientY;
    const startHeight = state.terminalHeight;

    const handleMouseMove = (e: MouseEvent) => {
      const deltaY = startY - e.clientY;
      const newHeight = Math.max(100, Math.min(600, startHeight + deltaY));
      dispatch({ type: 'SET_TERMINAL_HEIGHT', payload: newHeight });
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  };

  return (
    <div className="h-full flex flex-col">
      {/* Modified Files Section */}
      <div className="flex-1 overflow-hidden">
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium text-gray-900">Modified Files</h3>
            <button
              className="text-xs text-blue-600 hover:text-blue-800"
              onClick={() => {
                // Refresh files logic would go here
              }}
            >
              Refresh
            </button>
          </div>
        </div>
        <div className="overflow-y-auto" style={{ height: `calc(100% - 60px)` }}>
          {state.selectedSandbox ? (
            <ModifiedFilesList
              sandboxId={state.selectedSandbox.id}
              onFileSelect={(filePath) => {
                console.log('File selected:', filePath);
              }}
            />
          ) : (
            <div className="p-4 text-center text-gray-500 text-sm">
              No sandbox selected
            </div>
          )}
        </div>
      </div>

      {/* Resize Handle */}
      <div
        className={`h-1 bg-gray-200 cursor-row-resize hover:bg-gray-300 ${
          isResizing ? 'bg-blue-300' : ''
        } transition-colors duration-150`}
        onMouseDown={handleMouseDown}
      />

      {/* Terminal Section */}
      <div
        className="bg-gray-900 flex flex-col"
        style={{ height: `${state.terminalHeight}px` }}
      >
        <div className="flex items-center justify-between px-3 py-2 bg-gray-800 border-b border-gray-700">
          <div className="flex items-center space-x-2">
            <div className="w-3 h-3 bg-red-500 rounded-full"></div>
            <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
            <div className="w-3 h-3 bg-green-500 rounded-full"></div>
            <span className="text-xs text-gray-300 ml-2">Terminal</span>
          </div>
          <button
            className="text-xs text-gray-400 hover:text-gray-200"
            onClick={() => {
              // Clear terminal logic would go here
            }}
          >
            Clear
          </button>
        </div>
        <div className="flex-1 overflow-hidden">
          {state.selectedSandbox ? (
            <Terminal
              sandboxId={state.selectedSandbox.id}
              onCommandExecute={(command) => {
                console.log('Executing command:', command);
              }}
              isConnected={true} // This would come from WebSocket connection status
            />
          ) : (
            <div className="p-4 text-center text-gray-500 text-sm">
              No sandbox selected
            </div>
          )}
        </div>
      </div>
    </div>
  );
}