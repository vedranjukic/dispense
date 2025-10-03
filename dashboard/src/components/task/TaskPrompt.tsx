import React, { useState, useRef, useEffect } from 'react';
import { TaskPromptProps } from '../../types/task';
import { useTasks } from '../../hooks/useTasks';

export default function TaskPrompt({ sandboxId, onTaskStart, isTaskRunning }: TaskPromptProps) {
  const { runTask, runStreamingTask, taskHistory } = useTasks(sandboxId);
  const [description, setDescription] = useState('');
  const [showHistory, setShowHistory] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const maxLength = 2000;
  const characterCount = description.length;

  useEffect(() => {
    // Auto-resize textarea
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = textareaRef.current.scrollHeight + 'px';
    }
  }, [description]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!description.trim() || isTaskRunning) return;

    try {
      onTaskStart(description);
      // Use the new streaming method instead of the legacy runTask
      await runStreamingTask(description);
      setDescription('');
    } catch (error) {
      console.error('Failed to start task:', error);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSubmit(e);
    }

    if (e.key === 'Escape') {
      setShowHistory(false);
    }
  };

  const selectHistoryItem = (historyDescription: string) => {
    setDescription(historyDescription);
    setShowHistory(false);
    textareaRef.current?.focus();
  };

  const clearInput = () => {
    setDescription('');
    textareaRef.current?.focus();
  };

  const quickTasks = [
    'List all files in the current directory',
    'Show the current git status',
    'Run the test suite',
    'Build the project',
    'Show recent commit history'
  ];

  return (
    <div className="p-4 bg-white">
      <form onSubmit={handleSubmit}>
        {/* Quick Tasks */}
        <div className="mb-3">
          <div className="flex flex-wrap gap-2">
            {quickTasks.map((task, index) => (
              <button
                key={index}
                type="button"
                onClick={() => setDescription(task)}
                disabled={isTaskRunning}
                className="text-xs px-2 py-1 bg-gray-100 text-gray-700 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {task}
              </button>
            ))}
          </div>
        </div>

        {/* Main Input Area */}
        <div className="relative">
          <textarea
            ref={textareaRef}
            value={description}
            onChange={(e) => setDescription(e.target.value.slice(0, maxLength))}
            onKeyDown={handleKeyDown}
            placeholder="Describe what you want Claude to do in this sandbox..."
            disabled={isTaskRunning}
            rows={3}
            className="w-full p-3 border border-gray-300 rounded-lg resize-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
            style={{ minHeight: '80px', maxHeight: '200px' }}
          />

          {/* Character Counter */}
          <div className="absolute bottom-2 right-2 text-xs text-gray-500">
            {characterCount}/{maxLength}
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center justify-between mt-3">
          <div className="flex items-center space-x-2">
            {/* History Button */}
            <div className="relative">
              <button
                type="button"
                onClick={() => setShowHistory(!showHistory)}
                disabled={isTaskRunning || taskHistory.length === 0}
                className="btn btn-secondary btn-sm disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                History ({taskHistory.length})
              </button>

              {/* History Dropdown */}
              {showHistory && taskHistory.length > 0 && (
                <div className="absolute bottom-full left-0 mb-2 w-96 max-h-48 overflow-y-auto bg-white border border-gray-200 rounded-lg shadow-lg z-10">
                  <div className="p-2">
                    <h4 className="text-xs font-medium text-gray-700 mb-2">Recent Tasks</h4>
                    <div className="space-y-1">
                      {taskHistory.slice(0, 10).map((task) => (
                        <button
                          key={task.id}
                          type="button"
                          onClick={() => selectHistoryItem(task.description)}
                          className="w-full text-left p-2 text-xs bg-gray-50 hover:bg-gray-100 rounded border"
                        >
                          <div className="truncate font-medium">{task.description}</div>
                          <div className="text-gray-500 mt-1">
                            {new Date(task.timestamp).toLocaleString()}
                            {task.exitCode !== undefined && (
                              <span className={`ml-2 px-1 rounded text-xs ${
                                task.exitCode === 0 ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                              }`}>
                                Exit: {task.exitCode}
                              </span>
                            )}
                          </div>
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Clear Button */}
            <button
              type="button"
              onClick={clearInput}
              disabled={isTaskRunning || !description.trim()}
              className="btn btn-secondary btn-sm disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Clear
            </button>
          </div>

          {/* Submit Button */}
          <button
            type="submit"
            disabled={!description.trim() || isTaskRunning}
            className="btn btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isTaskRunning ? (
              <>
                <div className="animate-spin w-4 h-4 border border-white border-t-transparent rounded-full mr-2"></div>
                Running...
              </>
            ) : (
              <>
                <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                </svg>
                Send Task
              </>
            )}
          </button>
        </div>

        {/* Keyboard Shortcut Hint */}
        <div className="mt-2 text-xs text-gray-500">
          Press <kbd className="px-1 py-0.5 bg-gray-100 rounded text-xs">Cmd</kbd> + <kbd className="px-1 py-0.5 bg-gray-100 rounded text-xs">Enter</kbd> to send
        </div>
      </form>
    </div>
  );
}