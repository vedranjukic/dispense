import React, { useEffect, useRef } from 'react';
import { TaskLogsProps } from '../../types/task';
import { useTasks } from '../../hooks/useTasks';
import { formatLogType, formatTimestamp, parseLogContent } from '../../utils/formatters';
import { RunClaudeTaskResponseType, StreamTaskLogsResponseType } from '@api-client-ts';
import FormattedText from '../common/FormattedText';

export default function TaskLogs({ sandboxId, taskId, onTaskComplete }: TaskLogsProps) {
  const { logs, isRunning, isLoading, error, taskHistory, currentTask } = useTasks(sandboxId);
  const logsEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs]);

  // Handle task completion
  useEffect(() => {
    const lastLog = logs[logs.length - 1];
    if (lastLog && (lastLog.isFinished || lastLog.taskCompleted) && lastLog.exitCode !== undefined) {
      onTaskComplete(lastLog.exitCode);
    }
  }, [logs, onTaskComplete]);


  const getLogTypeColor = (type: RunClaudeTaskResponseType | StreamTaskLogsResponseType) => {
    // Handle both old and new log types
    switch (type) {
      case RunClaudeTaskResponseType.STDOUT:
      case StreamTaskLogsResponseType.STDOUT:
        return 'text-gray-800';
      case RunClaudeTaskResponseType.STDERR:
      case StreamTaskLogsResponseType.STDERR:
        return 'text-red-600';
      case RunClaudeTaskResponseType.STATUS:
      case StreamTaskLogsResponseType.STATUS:
        return 'text-blue-600';
      case RunClaudeTaskResponseType.ERROR:
      case StreamTaskLogsResponseType.ERROR:
        return 'text-red-700';
      default:
        return 'text-gray-600';
    }
  };

  const getLogTypeBackground = (type: RunClaudeTaskResponseType | StreamTaskLogsResponseType) => {
    // Handle both old and new log types
    switch (type) {
      case RunClaudeTaskResponseType.STDOUT:
      case StreamTaskLogsResponseType.STDOUT:
        return 'bg-white';
      case RunClaudeTaskResponseType.STDERR:
      case StreamTaskLogsResponseType.STDERR:
        return 'bg-red-50';
      case RunClaudeTaskResponseType.STATUS:
      case StreamTaskLogsResponseType.STATUS:
        return 'bg-blue-50';
      case RunClaudeTaskResponseType.ERROR:
      case StreamTaskLogsResponseType.ERROR:
        return 'bg-red-50';
      default:
        return 'bg-gray-50';
    }
  };


  return (
    <div className="h-full w-full flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 bg-gray-50">
        <div className="flex items-center space-x-4">
          <h3 className="text-sm font-medium text-gray-900">Task Logs</h3>
          {isRunning && (
            <div className="flex items-center text-xs text-blue-600">
              <div className="animate-spin w-3 h-3 border border-blue-600 border-t-transparent rounded-full mr-2"></div>
              Running...
            </div>
          )}
          {isLoading && (
            <div className="flex items-center text-xs text-gray-600">
              <div className="animate-spin w-3 h-3 border border-gray-600 border-t-transparent rounded-full mr-2"></div>
              Loading logs...
            </div>
          )}
        </div>

      </div>

      {/* Logs Content */}
      <div
        className="flex-1 min-h-0 overflow-y-auto overflow-x-auto p-4 font-mono text-sm bg-gray-900 text-gray-100"
      >
        {error && (
          <div className="mb-4 p-3 bg-red-900 border border-red-700 rounded">
            <div className="flex items-center">
              <svg className="w-4 h-4 text-red-400 mr-2" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
              <span className="text-red-100">Error: {error}</span>
            </div>
          </div>
        )}

        {isLoading ? (
          <div className="text-center text-gray-500 py-8">
            <div className="animate-spin w-8 h-8 border-2 border-gray-300 border-t-blue-500 rounded-full mx-auto mb-4"></div>
            <p>Loading existing task logs...</p>
          </div>
        ) : logs.length === 0 ? (
          <div className="text-center text-gray-500 py-8">
            <div>
              <svg className="mx-auto h-8 w-8 text-gray-600 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <p>No task logs yet</p>
              <p className="text-sm">Start a task to see logs here</p>
            </div>
          </div>
        ) : (
          <div className="space-y-1 max-h-600">
            {logs.map((log, index) => {
              // Get the current task description for Claude log formatting
              const currentTaskInfo = taskHistory.find(task => task.id === currentTask);
              const taskDescription = currentTaskInfo?.description;

              const { text, metadata } = parseLogContent(log.content, taskDescription);
              return (
                <div
                  key={index}
                  className={`flex items-start space-x-3 py-1 px-2 rounded w-full min-w-0 ${
                    log.type === RunClaudeTaskResponseType.ERROR ||
                    log.type === RunClaudeTaskResponseType.STDERR ||
                    log.type === StreamTaskLogsResponseType.ERROR ||
                    log.type === StreamTaskLogsResponseType.STDERR
                      ? 'bg-red-900/20'
                      : log.type === RunClaudeTaskResponseType.STATUS ||
                        log.type === StreamTaskLogsResponseType.STATUS
                      ? 'bg-blue-900/20'
                      : ''
                  }`}
                >
                  <div className="flex-shrink-0 text-xs text-gray-500 w-16">
                    {formatTimestamp(log.timestamp).split(' ')[1] || 'now'}
                  </div>
                  <div className={`flex-shrink-0 text-xs font-medium w-16 ${
                    log.type === RunClaudeTaskResponseType.ERROR ||
                    log.type === RunClaudeTaskResponseType.STDERR ||
                    log.type === StreamTaskLogsResponseType.ERROR ||
                    log.type === StreamTaskLogsResponseType.STDERR
                      ? 'text-red-400'
                      : log.type === RunClaudeTaskResponseType.STATUS ||
                        log.type === StreamTaskLogsResponseType.STATUS
                      ? 'text-blue-400'
                      : 'text-gray-400'
                  }`}>
                    {formatLogType(log.type)}
                  </div>
                  <div className="flex-1 min-w-0 break-all overflow-hidden flex-shrink">
                    <FormattedText text={text} />
                    {(log.isFinished || log.taskCompleted) && (
                      <div className="mt-1 text-xs text-yellow-400">
                        Task completed{log.exitCode !== undefined && ` with exit code: ${log.exitCode}`}
                        {log.taskStatus && ` (Status: ${log.taskStatus})`}
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}

        <div ref={logsEndRef} />
      </div>

      {/* Status Bar */}
      <div className="px-4 py-2 bg-gray-50 border-t border-gray-200 text-xs text-gray-500">
        {logs.length} log{logs.length !== 1 ? 's' : ''}
        {isRunning && ' • Task is running'}
      </div>
    </div>
  );
}