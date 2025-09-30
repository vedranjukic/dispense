import { useState, useEffect, useCallback } from 'react';
import { FileItem } from '../types/file';
import { apiService } from '../services/api';
import { useDashboard } from '../contexts/DashboardContext';
import { POLLING_INTERVALS } from '../utils/constants';

export function useFiles(sandboxId?: string) {
  const { state, dispatch } = useDashboard();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchFiles = useCallback(async () => {
    if (!sandboxId) return;

    try {
      setIsLoading(true);
      setError(null);
      const files = await apiService.getModifiedFiles(sandboxId);
      dispatch({ type: 'SET_MODIFIED_FILES', payload: files });
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch modified files';
      setError(errorMessage);
    } finally {
      setIsLoading(false);
    }
  }, [sandboxId, dispatch]);

  const refreshFiles = useCallback(() => {
    fetchFiles();
  }, [fetchFiles]);

  // Auto-refresh files
  useEffect(() => {
    if (sandboxId) {
      fetchFiles();

      const interval = setInterval(fetchFiles, POLLING_INTERVALS.FILE_CHANGES);
      return () => clearInterval(interval);
    }
  }, [sandboxId, fetchFiles]);

  return {
    files: state.modifiedFiles,
    isLoading,
    error,
    fetchFiles,
    refreshFiles
  };
}