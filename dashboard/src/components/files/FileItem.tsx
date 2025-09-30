import React from 'react';
import { FileItem as FileItemType } from '../../types/file';
import { formatFileSize, formatTimestamp } from '../../utils/formatters';

interface FileItemProps {
  file: FileItemType;
  onClick?: () => void;
}

export default function FileItem({ file, onClick }: FileItemProps) {
  const getFileIcon = (path: string) => {
    const ext = path.split('.').pop()?.toLowerCase();

    switch (ext) {
      case 'js':
      case 'jsx':
      case 'ts':
      case 'tsx':
        return (
          <svg className="w-4 h-4 text-yellow-600" fill="currentColor" viewBox="0 0 20 20">
            <path d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z" />
          </svg>
        );
      case 'css':
      case 'scss':
      case 'less':
        return (
          <svg className="w-4 h-4 text-blue-600" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M4 4a2 2 0 00-2 2v8a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2H4zm0 2h12v8H4V6z" clipRule="evenodd" />
          </svg>
        );
      case 'html':
      case 'htm':
        return (
          <svg className="w-4 h-4 text-orange-600" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V4zM3 10a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H4a1 1 0 01-1-1v-6zM14 9a1 1 0 00-1 1v6a1 1 0 001 1h2a1 1 0 001-1v-6a1 1 0 00-1-1h-2z" clipRule="evenodd" />
          </svg>
        );
      case 'json':
        return (
          <svg className="w-4 h-4 text-green-600" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M4 4a2 2 0 00-2 2v8a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2H4zm2 2h8a1 1 0 011 1v6a1 1 0 01-1 1H6a1 1 0 01-1-1V7a1 1 0 011-1z" clipRule="evenodd" />
          </svg>
        );
      case 'md':
      case 'markdown':
        return (
          <svg className="w-4 h-4 text-gray-600" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M4 4a2 2 0 00-2 2v8a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2H4zm4.707 4.293a1 1 0 00-1.414 1.414L9.586 12l-2.293 2.293a1 1 0 101.414 1.414L12 12.414l3.293 3.293a1 1 0 001.414-1.414L14.414 12l2.293-2.293a1 1 0 00-1.414-1.414L12 11.586 8.707 8.293z" clipRule="evenodd" />
          </svg>
        );
      default:
        return (
          <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
        );
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'added':
        return 'text-green-600 bg-green-50';
      case 'modified':
        return 'text-blue-600 bg-blue-50';
      case 'deleted':
        return 'text-red-600 bg-red-50';
      default:
        return 'text-gray-600 bg-gray-50';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'added':
        return (
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z" clipRule="evenodd" />
          </svg>
        );
      case 'modified':
        return (
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
          </svg>
        );
      case 'deleted':
        return (
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
          </svg>
        );
      default:
        return null;
    }
  };

  const fileName = file.path.split('/').pop() || file.path;
  const filePath = file.path.split('/').slice(0, -1).join('/');

  return (
    <div
      onClick={onClick}
      className="flex items-center p-2 rounded hover:bg-gray-50 cursor-pointer group transition-colors duration-150"
    >
      {/* File Icon */}
      <div className="flex-shrink-0 mr-2">
        {getFileIcon(file.path)}
      </div>

      {/* File Info */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center">
          <span className="text-sm font-medium text-gray-900 truncate">
            {fileName}
          </span>
          {/* Status Badge */}
          <div className={`ml-2 flex-shrink-0 flex items-center px-1.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(file.status)}`}>
            {getStatusIcon(file.status)}
            <span className="ml-1 capitalize">{file.status}</span>
          </div>
        </div>

        {filePath && (
          <div className="text-xs text-gray-500 truncate mt-0.5">
            {filePath}
          </div>
        )}

        <div className="flex items-center space-x-2 text-xs text-gray-400 mt-0.5">
          {file.size && (
            <span>{formatFileSize(file.size)}</span>
          )}
          {file.lastModified && (
            <>
              {file.size && <span>•</span>}
              <span>{formatTimestamp(file.lastModified)}</span>
            </>
          )}
        </div>
      </div>

      {/* Action Icons (shown on hover) */}
      <div className="flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity duration-150">
        <button
          onClick={(e) => {
            e.stopPropagation();
            // TODO: Add view diff functionality
            console.log('View diff for:', file.path);
          }}
          className="p-1 text-gray-400 hover:text-gray-600 rounded"
          title="View diff"
        >
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
          </svg>
        </button>
      </div>
    </div>
  );
}