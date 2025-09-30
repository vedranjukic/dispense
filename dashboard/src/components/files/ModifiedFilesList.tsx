import React from 'react';
import { useFiles } from '../../hooks/useFiles';
import { ModifiedFilesListProps } from '../../types/file';
import FileItem from './FileItem';

export default function ModifiedFilesList({ sandboxId, onFileSelect }: ModifiedFilesListProps) {
  const { files, isLoading, error, refreshFiles } = useFiles(sandboxId);

  if (isLoading && files.length === 0) {
    return (
      <div className="p-4">
        <div className="animate-pulse space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-8 bg-gray-200 rounded"></div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4">
        <div className="bg-red-50 border border-red-200 rounded p-3">
          <div className="flex items-center">
            <svg className="w-4 h-4 text-red-400 mr-2" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
            <div>
              <h4 className="text-sm font-medium text-red-800">Error</h4>
              <p className="text-sm text-red-700">{error}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (files.length === 0) {
    return (
      <div className="p-4 text-center">
        <div className="py-4">
          <svg className="mx-auto h-8 w-8 text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <h4 className="text-sm font-medium text-gray-900 mb-1">No modified files</h4>
          <p className="text-xs text-gray-500">Files changed in this sandbox will appear here</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-2 space-y-1">
        {files.map((file, index) => (
          <FileItem
            key={`${file.path}-${index}`}
            file={file}
            onClick={() => onFileSelect?.(file.path)}
          />
        ))}
      </div>

      {/* Footer */}
      <div className="p-2 border-t border-gray-100 bg-gray-50">
        <div className="flex items-center justify-between text-xs text-gray-500">
          <span>{files.length} file{files.length !== 1 ? 's' : ''} modified</span>
          <button
            onClick={refreshFiles}
            className="text-blue-600 hover:text-blue-800"
          >
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}