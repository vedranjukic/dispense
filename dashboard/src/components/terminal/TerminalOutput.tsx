import React, { useEffect, useRef } from 'react';
import { TerminalLine } from '../../types/terminal';
import { formatTimestamp } from '../../utils/formatters';

interface TerminalOutputProps {
  lines: TerminalLine[];
  isConnected: boolean;
}

export default function TerminalOutput({ lines, isConnected }: TerminalOutputProps) {
  const endRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new lines arrive
  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [lines]);

  const getLineColor = (type: string) => {
    switch (type) {
      case 'input':
        return 'text-green-400';
      case 'output':
        return 'text-gray-100';
      case 'error':
        return 'text-red-400';
      default:
        return 'text-gray-300';
    }
  };

  const getLinePrefix = (type: string) => {
    switch (type) {
      case 'input':
        return '$ ';
      case 'error':
        return '! ';
      default:
        return '';
    }
  };

  if (!isConnected) {
    return (
      <div className="h-full flex items-center justify-center text-center p-4">
        <div>
          <svg className="mx-auto h-8 w-8 text-gray-600 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
          <h3 className="text-sm font-medium text-gray-300 mb-1">Terminal Disconnected</h3>
          <p className="text-xs text-gray-500">
            Waiting for connection to sandbox terminal...
          </p>
        </div>
      </div>
    );
  }

  if (lines.length === 0) {
    return (
      <div className="h-full flex items-center justify-center text-center p-4">
        <div>
          <svg className="mx-auto h-8 w-8 text-gray-600 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <h3 className="text-sm font-medium text-gray-300 mb-1">Terminal Ready</h3>
          <p className="text-xs text-gray-500">
            Connected to sandbox. Type a command to get started.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto p-3 space-y-1">
      {/* Welcome Message */}
      <div className="text-gray-500 text-xs mb-3 border-b border-gray-700 pb-2">
        Terminal connected to sandbox: {isConnected ? 'active' : 'inactive'}
      </div>

      {/* Terminal Lines */}
      {lines.map((line) => (
        <div
          key={line.id}
          className={`flex items-start space-x-2 ${getLineColor(line.type)} leading-relaxed`}
        >
          {/* Timestamp (hidden on small screens) */}
          <div className="hidden sm:block flex-shrink-0 text-xs text-gray-600 w-16">
            {formatTimestamp(line.timestamp).split(' ')[1] || ''}
          </div>

          {/* Line Content */}
          <div className="flex-1 min-w-0">
            <span className="text-gray-500">{getLinePrefix(line.type)}</span>
            <span className="whitespace-pre-wrap break-words">
              {line.content}
            </span>
          </div>
        </div>
      ))}

      {/* Auto-scroll anchor */}
      <div ref={endRef} />
    </div>
  );
}