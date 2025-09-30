import { useState, useEffect, useCallback, useRef } from 'react';
import { TerminalWebSocketService } from '../services/websocket';
import { TerminalLine } from '../types/terminal';
import { useDashboard } from '../contexts/DashboardContext';

export function useTerminal(sandboxId?: string) {
  const { state, dispatch } = useDashboard();
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [commandHistory, setCommandHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const wsRef = useRef<TerminalWebSocketService | null>(null);

  const connect = useCallback(async () => {
    if (!sandboxId || wsRef.current) return;

    try {
      const ws = new TerminalWebSocketService(sandboxId);
      wsRef.current = ws;

      ws.onOutput((output: string) => {
        const terminalLine: TerminalLine = {
          id: Date.now().toString() + Math.random(),
          content: output,
          timestamp: Date.now(),
          type: 'output'
        };
        dispatch({ type: 'ADD_TERMINAL_LINE', payload: terminalLine });
      });

      ws.onError((error: string) => {
        const terminalLine: TerminalLine = {
          id: Date.now().toString() + Math.random(),
          content: error,
          timestamp: Date.now(),
          type: 'error'
        };
        dispatch({ type: 'ADD_TERMINAL_LINE', payload: terminalLine });
      });

      await ws.connect();
      setIsConnected(true);
      setError(null);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to connect to terminal';
      setError(errorMessage);
      setIsConnected(false);
    }
  }, [sandboxId, dispatch]);

  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.disconnect();
      wsRef.current = null;
      setIsConnected(false);
    }
  }, []);

  const sendCommand = useCallback((command: string) => {
    if (!wsRef.current || !isConnected) return;

    // Add to terminal output as input
    const terminalLine: TerminalLine = {
      id: Date.now().toString() + Math.random(),
      content: command,
      timestamp: Date.now(),
      type: 'input'
    };
    dispatch({ type: 'ADD_TERMINAL_LINE', payload: terminalLine });

    // Add to command history
    if (command.trim() && command !== commandHistory[0]) {
      setCommandHistory(prev => [command, ...prev.slice(0, 99)]); // Keep last 100 commands
    }
    setHistoryIndex(-1);

    // Send to terminal
    wsRef.current.sendCommand(command);
  }, [isConnected, commandHistory, dispatch]);

  const clearTerminal = useCallback(() => {
    dispatch({ type: 'SET_TERMINAL_OUTPUT', payload: [] });
  }, [dispatch]);

  const getPreviousCommand = useCallback(() => {
    if (commandHistory.length === 0) return '';
    const newIndex = Math.min(historyIndex + 1, commandHistory.length - 1);
    setHistoryIndex(newIndex);
    return commandHistory[newIndex] || '';
  }, [commandHistory, historyIndex]);

  const getNextCommand = useCallback(() => {
    if (commandHistory.length === 0) return '';
    const newIndex = Math.max(historyIndex - 1, -1);
    setHistoryIndex(newIndex);
    return newIndex === -1 ? '' : commandHistory[newIndex];
  }, [commandHistory, historyIndex]);

  // Auto-connect when sandbox changes
  useEffect(() => {
    if (sandboxId) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [sandboxId, connect, disconnect]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  return {
    terminalOutput: state.terminalOutput,
    isConnected,
    error,
    commandHistory,
    connect,
    disconnect,
    sendCommand,
    clearTerminal,
    getPreviousCommand,
    getNextCommand
  };
}