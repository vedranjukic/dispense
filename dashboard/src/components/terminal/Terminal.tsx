import React, { useState, useRef, useEffect } from 'react';
import { useTerminal } from '../../hooks/useTerminal';
import { TerminalProps } from '../../types/terminal';
import TerminalOutput from './TerminalOutput';

export default function Terminal({ sandboxId, onCommandExecute, isConnected }: TerminalProps) {
  const {
    terminalOutput,
    isConnected: wsConnected,
    sendCommand,
    clearTerminal,
    getPreviousCommand,
    getNextCommand
  } = useTerminal(sandboxId);

  const [currentCommand, setCurrentCommand] = useState('');
  const [isInputFocused, setIsInputFocused] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const actuallyConnected = wsConnected && isConnected;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentCommand.trim() || !actuallyConnected) return;

    onCommandExecute(currentCommand);
    sendCommand(currentCommand);
    setCurrentCommand('');
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowUp':
        e.preventDefault();
        setCurrentCommand(getPreviousCommand());
        break;
      case 'ArrowDown':
        e.preventDefault();
        setCurrentCommand(getNextCommand());
        break;
      case 'Tab':
        e.preventDefault();
        // TODO: Add tab completion
        break;
      case 'l':
        if (e.ctrlKey) {
          e.preventDefault();
          clearTerminal();
        }
        break;
    }
  };

  // Auto-focus input when terminal is clicked
  const handleTerminalClick = () => {
    inputRef.current?.focus();
  };

  useEffect(() => {
    // Auto-focus on mount
    inputRef.current?.focus();
  }, []);

  return (
    <div
      className="h-full flex flex-col bg-gray-900 text-gray-100 font-mono text-sm"
      onClick={handleTerminalClick}
    >
      {/* Terminal Output */}
      <div className="flex-1 overflow-y-auto">
        <TerminalOutput
          lines={terminalOutput}
          isConnected={actuallyConnected}
        />
      </div>

      {/* Command Input */}
      <div className="border-t border-gray-700 p-2">
        <form onSubmit={handleSubmit} className="flex items-center">
          {/* Prompt */}
          <div className="flex-shrink-0 text-green-400 mr-2">
            <span className="opacity-80">$</span>
          </div>

          {/* Input */}
          <input
            ref={inputRef}
            type="text"
            value={currentCommand}
            onChange={(e) => setCurrentCommand(e.target.value)}
            onKeyDown={handleKeyDown}
            onFocus={() => setIsInputFocused(true)}
            onBlur={() => setIsInputFocused(false)}
            disabled={!actuallyConnected}
            placeholder={actuallyConnected ? "Type a command..." : "Terminal not connected"}
            className="flex-1 bg-transparent border-none outline-none text-gray-100 placeholder-gray-500 disabled:cursor-not-allowed"
            autoComplete="off"
            spellCheck={false}
          />

          {/* Connection Status Indicator */}
          <div className="flex-shrink-0 ml-2">
            <div className={`w-2 h-2 rounded-full ${
              actuallyConnected ? 'bg-green-400' : 'bg-red-400'
            }`} title={actuallyConnected ? 'Connected' : 'Disconnected'} />
          </div>
        </form>

        {/* Status Bar */}
        <div className="flex items-center justify-between text-xs text-gray-500 mt-1">
          <div className="flex items-center space-x-4">
            {actuallyConnected ? (
              <span className="text-green-400">Connected to {sandboxId}</span>
            ) : (
              <span className="text-red-400">Disconnected</span>
            )}
          </div>

          <div className="flex items-center space-x-2 text-gray-600">
            <span>Ctrl+L to clear</span>
            <span>•</span>
            <span>↑/↓ for history</span>
          </div>
        </div>
      </div>

      {/* Cursor */}
      {isInputFocused && actuallyConnected && (
        <style jsx>{`
          @keyframes blink {
            0%, 50% { opacity: 1; }
            51%, 100% { opacity: 0; }
          }

          .terminal-cursor::after {
            content: '';
            display: inline-block;
            width: 8px;
            height: 16px;
            background-color: #4ade80;
            animation: blink 1s infinite;
            margin-left: 1px;
            vertical-align: baseline;
          }
        `}</style>
      )}
    </div>
  );
}