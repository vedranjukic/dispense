import React, { useEffect, useRef, useState } from 'react';
import { TaskLogsProps, TaskInfo, LogEntry } from '../../types/task';
import { useTasks } from '../../hooks/useTasks';
import FormattedText from '../common/FormattedText';
import { apiService } from '../../services/api';
import { StreamTaskLogsResponseType } from '@api-client-ts';

interface TaskDisplayProps {
  task: TaskInfo;
  isExpanded: boolean;
  onToggleExpanded: () => void;
  sandboxId: string;
}

function TaskDisplay({ task, isExpanded, onToggleExpanded, sandboxId }: TaskDisplayProps) {
  const [taskLogs, setTaskLogs] = useState<LogEntry[]>([]);
  const [isLoadingLogs, setIsLoadingLogs] = useState(false);
  const [showDetailedLogs, setShowDetailedLogs] = useState(false);
  const [logError, setLogError] = useState<string | null>(null);

  // Auto-fetch logs when component mounts (only for completed tasks)
  useEffect(() => {
    if (taskLogs.length === 0 && !isLoadingLogs && task.state === 'COMPLETED') {
      fetchTaskLogs();
    }
  }, [task.taskId, task.state]);

  // Fetch logs for this specific task
  const fetchTaskLogs = async () => {
    if (isLoadingLogs) return;
    
    setIsLoadingLogs(true);
    setLogError(null);
    
    try {
      // Get historical logs first
      const logsResponse = await apiService.getClaudeLogs(sandboxId, task.taskId);
      
      if (logsResponse.success && logsResponse.logs) {
        // Parse logs and extract meaningful content
        const logEntries: LogEntry[] = [];
        
        // Process each log entry (each log is a single string with multiple lines)
        let globalFoundAnswer = false;
        
        // Only process the first log entry that contains meaningful content
        for (const log of logsResponse.logs) {
          // Skip file references and separators
          if (log.includes('📄 Log file:') || log.includes('─────────────────────────────────────')) {
            continue;
          }
          
          // Split the log into individual lines
          const lines = log.split('\n');
          
          for (const line of lines) {
            if (!line.trim() || globalFoundAnswer) break;
            
            try {
              // Try to parse JSON content from STDOUT logs
              const jsonMatch = line.match(/\[([^\]]+)\] \[STDOUT\] (.+)/);
              if (jsonMatch) {
                const timestamp = jsonMatch[1];
                const jsonContent = jsonMatch[2];
                
                try {
                  const parsed = JSON.parse(jsonContent);
                  
                  // Extract meaningful content based on JSON structure
                  let content = '';
                  
                  // Priority 1: Look for result content (final answer) - most reliable
                  if (parsed.type === 'result' && parsed.result) {
                    content = parsed.result;
                    globalFoundAnswer = true; // Stop after finding the result
                  }
                  // Priority 2: Look for assistant message with text content - complete response
                  else if (parsed.type === 'assistant' && parsed.message && parsed.message.content) {
                    const textContent = parsed.message.content
                      .filter((c: any) => c.type === 'text')
                      .map((c: any) => c.text)
                      .join('');
                    if (textContent.trim()) {
                      content = textContent.trim();
                      globalFoundAnswer = true; // Stop after finding the assistant message
                    }
                  }
                  // Skip stream events - they're just deltas, not the final answer
                  
                  if (content) {
                    logEntries.push({
                      type: StreamTaskLogsResponseType.STDOUT,
                      content: content,
                      timestamp: new Date(timestamp).getTime(),
                      isFinished: false,
                      taskCompleted: false
                    });
                  }
                } catch (jsonError) {
                  // Skip JSON parsing errors
                }
              }
            } catch (error) {
              // Skip problematic log entries
            }
          }
          
          // If we found an answer, stop processing other log entries
          if (globalFoundAnswer) break;
        }
        
        setTaskLogs(logEntries);
      }
      
      // If task is still running, start streaming for real-time updates
      if (task.state === 'RUNNING') {
        await apiService.streamTaskLogs(
          {
            task_id: task.taskId,
            sandbox_identifier: sandboxId,
            follow: true,
            include_history: false // We already have history
          },
          (response) => {
            const logEntry: LogEntry = {
              type: response.type,
              content: response.content,
              timestamp: response.timestamp * 1000,
              isFinished: response.task_completed || false,
              taskCompleted: response.task_completed || false,
              taskStatus: response.task_status
            };
            
            setTaskLogs(prev => [...prev, logEntry]);
          },
          (taskStatus) => {
            console.log('Task completed with status:', taskStatus);
          },
          (error) => {
            console.error('Streaming error:', error);
            setLogError(error.message);
          }
        );
      }
    } catch (error) {
      console.error('Failed to fetch task logs:', error);
      setLogError(error instanceof Error ? error.message : 'Failed to fetch logs');
    } finally {
      setIsLoadingLogs(false);
    }
  };

  // Determine task status and styling
  const getTaskStatus = (state: string) => {
    switch (state) {
      case 'COMPLETED':
        return { icon: '✅', color: 'text-green-400', bg: 'bg-green-900/20' };
      case 'FAILED':
        return { icon: '❌', color: 'text-red-400', bg: 'bg-red-900/20' };
      case 'RUNNING':
        return { icon: '🔄', color: 'text-blue-400', bg: 'bg-blue-900/20' };
      case 'PENDING':
        return { icon: '⏳', color: 'text-yellow-400', bg: 'bg-yellow-900/20' };
      default:
        return { icon: '❓', color: 'text-gray-400', bg: 'bg-gray-900/20' };
    }
  };

  const status = getTaskStatus(task.state);
  const startTime = new Date(parseInt(task.startedAt) * 1000);
  const finishTime = task.finishedAt ? new Date(parseInt(task.finishedAt) * 1000) : null;
  const duration = finishTime ? finishTime.getTime() - startTime.getTime() : null;

  return (
    <div className={`rounded-lg border ${status.bg} border-gray-700 mb-3`}>
      {/* Task Header */}
      <div
        className="flex items-start justify-between p-4 cursor-pointer hover:bg-gray-800/50 transition-colors"
        onClick={onToggleExpanded}
      >
        <div className="flex items-start space-x-3 flex-1 min-w-0">
          <div className="flex-1 min-w-0">
            <p className="text-gray-300 text-sm mb-2 break-words">
            <button className="border-none bg-transparent mr-2" style={{ float: 'left' }}>
            💬
            </button> <FormattedText text={task.prompt} />
            </p>
            
            {/* Show recent logs even when not expanded (only for completed tasks) */}
            {taskLogs.length > 0 && task.state === 'COMPLETED' && (
              <div className="mt-2 bg-gray-800 rounded p-2 max-h-32 overflow-y-auto">
                <div className="space-y-1 text-xs">
                  {taskLogs.map((log, index) => (
                    <div key={index} className="flex items-start space-x-2">
                      <span className={`text-xs ${
                        log.type === StreamTaskLogsResponseType.STDERR ? 'text-red-400' :
                        log.type === StreamTaskLogsResponseType.STATUS ? 'text-yellow-400' :
                        'text-gray-200'
                      }`}>
                        {log.content}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Expanded Details */}
      {isExpanded && (
        <div className="px-4 pb-4 border-t border-gray-700">
          <div className="mt-3 space-y-2 text-sm">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <span className="text-gray-400">Task ID:</span>
                <p className="text-gray-200 font-mono text-xs mt-1 break-all">{task.taskId}</p>
              </div>
              <div>
                <span className="text-gray-400">Working Directory:</span>
                <p className="text-gray-200 font-mono text-xs mt-1">{task.workingDirectory}</p>
              </div>
            </div>

            <div>
              <span className="text-gray-400">Full Prompt:</span>
              <div className="bg-gray-800 rounded p-3 mt-1 text-gray-200 text-sm whitespace-pre-wrap">
                <FormattedText text={task.prompt} />
              </div>
            </div>

            {task.error && task.error.trim() !== '' && (
              <div>
                <span className="text-red-400">Error Details:</span>
                <div className="bg-red-900/20 border border-red-700 rounded p-3 mt-1 text-red-200 text-sm">
                  {task.error}
                </div>
              </div>
            )}

            {/* Task Logs Section */}
            {showDetailedLogs && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-gray-400">Task Logs:</span>
                  {isLoadingLogs && (
                    <div className="flex items-center text-xs text-blue-400">
                      <div className="animate-spin w-3 h-3 border border-blue-400 border-t-transparent rounded-full mr-2"></div>
                      Loading logs...
                    </div>
                  )}
                </div>
                
                {logError && (
                  <div className="bg-red-900/20 border border-red-700 rounded p-2 mb-2 text-red-200 text-sm">
                    Error loading logs: {logError}
                  </div>
                )}
                
                {taskLogs.length > 0 ? (
                  <div className="bg-gray-800 rounded p-3 max-h-64 overflow-y-auto">
                    <div className="space-y-1 text-sm">
                      {taskLogs.map((log, index) => (
                        <div key={index} className="flex items-start space-x-2">
                          <span className="text-gray-500 text-xs mt-0.5 flex-shrink-0">
                            {new Date(log.timestamp).toLocaleTimeString()}
                          </span>
                          <span className={`text-xs ${
                            log.type === StreamTaskLogsResponseType.STDERR ? 'text-red-400' :
                            log.type === StreamTaskLogsResponseType.STATUS ? 'text-yellow-400' :
                            'text-gray-200'
                          }`}>
                            {log.content}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : !isLoadingLogs && !logError && (
                  <div className="bg-gray-800 rounded p-3 text-gray-400 text-sm">
                    No logs available for this task.
                  </div>
                )}
              </div>
            )}

            <div className="flex justify-between items-center pt-2 border-t border-gray-700">
              <div className="text-xs text-gray-500">
                {finishTime ? (
                  `Completed in ${Math.round((duration || 0) / 1000)} seconds`
                ) : task.state === 'RUNNING' ? (
                  'Currently running...'
                ) : (
                  'Pending execution'
                )}
              </div>
              <button
                className="text-xs px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
                onClick={(e) => {
                  e.stopPropagation();
                  if (!showDetailedLogs && taskLogs.length === 0) {
                    fetchTaskLogs();
                  }
                  setShowDetailedLogs(!showDetailedLogs);
                }}
                disabled={isLoadingLogs}
              >
                {isLoadingLogs ? 'Loading...' : showDetailedLogs ? 'Hide Logs' : 'View Logs'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function TaskLogs({ sandboxId, onTaskComplete }: TaskLogsProps) {
  const { tasks, isRunning, isLoading, error } = useTasks(sandboxId);
  const [expandedTasks, setExpandedTasks] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState<'all' | 'COMPLETED' | 'FAILED' | 'RUNNING' | 'PENDING'>('all');
  const [taskLogsHeight, setTaskLogsHeight] = useState<number>(0);

  // Calculate height for task-logs element
  useEffect(() => {
    const calculateHeight = () => {
      const taskPromptElement = document.getElementById('task-prompt');
      if (taskPromptElement) {
        const viewportHeight = window.innerHeight;
        const taskPromptHeight = taskPromptElement.offsetHeight;
        // Add some padding to account for headers and borders
        const padding = 200; // Approximate height for headers and margins
        const calculatedHeight = Math.max(300, viewportHeight - taskPromptHeight - padding);
        setTaskLogsHeight(calculatedHeight);
      }
    };

    // Calculate initial height with a small delay to ensure DOM is ready
    const timeoutId = setTimeout(calculateHeight, 100);

    // Recalculate on window resize
    const handleResize = () => {
      calculateHeight();
    };

    window.addEventListener('resize', handleResize);
    return () => {
      clearTimeout(timeoutId);
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  // Handle task completion
  useEffect(() => {
    const completedTasks = tasks.filter(task => task.state === 'COMPLETED' || task.state === 'FAILED');
    if (completedTasks.length > 0) {
      const lastCompletedTask = completedTasks[completedTasks.length - 1];
      onTaskComplete(lastCompletedTask.exitCode || 0);
    }
  }, [tasks, onTaskComplete]);

  // Filter tasks based on selected filter
  const filteredTasks = tasks.filter(task => {
    if (filter === 'all') return true;
    return task.state === filter;
  });

  // Sort tasks by start time (most recent first)
  const sortedTasks = [...filteredTasks].sort((a, b) => {
    const timeA = parseInt(a.startedAt) || 0;
    const timeB = parseInt(b.startedAt) || 0;
    return timeB - timeA;
  });

  const toggleTaskExpansion = (taskId: string) => {
    setExpandedTasks(prev => {
      const newSet = new Set(prev);
      if (newSet.has(taskId)) {
        newSet.delete(taskId);
      } else {
        newSet.add(taskId);
      }
      return newSet;
    });
  };

  return (
    <div className="h-full flex flex-col max-w-full overflow-hidden">
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

        <div className="flex items-center space-x-2">
          {/* Task Filter */}
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value as typeof filter)}
            className="text-xs border border-gray-600 rounded px-2 py-1 bg-gray-800 text-gray-200"
          >
            <option value="all">All Tasks</option>
            <option value="COMPLETED">Completed</option>
            <option value="FAILED">Failed</option>
            <option value="RUNNING">Running</option>
            <option value="PENDING">Pending</option>
          </select>

          {/* Expand All Button */}
          <button
            onClick={() => {
              if (expandedTasks.size === sortedTasks.length) {
                setExpandedTasks(new Set());
              } else {
                setExpandedTasks(new Set(sortedTasks.map(t => t.taskId)));
              }
            }}
            className="text-xs px-2 py-1 rounded border border-gray-600 text-gray-300 hover:bg-gray-700 bg-gray-800"
          >
            {expandedTasks.size === sortedTasks.length ? 'Collapse All' : 'Expand All'}
          </button>
        </div>
      </div>

      {/* Tasks Content */}
      <div 
        id="task-logs" 
        className="flex-1 overflow-y-auto p-4 bg-gray-900 text-gray-100"
        style={{ maxHeight: taskLogsHeight > 0 ? `${taskLogsHeight}px` : 'auto' }}
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
            <p>Loading tasks...</p>
          </div>
        ) : sortedTasks.length === 0 ? (
          <div className="text-center text-gray-500 py-8">
            <div>
              <svg className="mx-auto h-12 w-12 text-gray-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <p className="text-lg mb-2">No tasks found</p>
              <p className="text-sm">
                {filter === 'all'
                  ? 'No Claude tasks have been executed in this sandbox yet'
                  : `No ${filter.toLowerCase()} tasks found`
                }
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            {sortedTasks.map((task) => (
              <TaskDisplay
                key={task.taskId}
                task={task}
                isExpanded={expandedTasks.has(task.taskId)}
                onToggleExpanded={() => toggleTaskExpansion(task.taskId)}
                sandboxId={sandboxId}
              />
            ))}
          </div>
        )}
      </div>

      {/* Status Bar */}
      <div className="px-4 py-2 bg-gray-50 border-t border-gray-200 text-xs text-gray-500" style={{ display: 'none' }}>
        {sortedTasks.length} task{sortedTasks.length !== 1 ? 's' : ''}
        {filter !== 'all' && `(${filter.toLowerCase()})`}
        {tasks.filter(t => t.state === 'RUNNING').length > 0 && ' • Tasks running'}
      </div>
    </div>
  );
}