import { useState, useEffect, useCallback } from 'react';
import { SandboxInfo } from '@api-client-ts';
import { apiService } from '../services/api';
import { useDashboard } from '../contexts/DashboardContext';
import { POLLING_INTERVALS } from '../utils/constants';

export function useSandboxes(projectId?: string) {
  const { state, dispatch } = useDashboard();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSandboxes = useCallback(async () => {
    if (!projectId) return;

    try {
      setIsLoading(true);
      setError(null);
      const sandboxes = await apiService.getSandboxes(projectId);
      dispatch({ type: 'SET_SANDBOXES', payload: sandboxes });
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch sandboxes';
      setError(errorMessage);
      dispatch({ type: 'SET_ERROR', payload: errorMessage });
    } finally {
      setIsLoading(false);
    }
  }, [projectId, dispatch]);

  const createSandbox = useCallback(async (name: string, isRemote?: boolean): Promise<SandboxInfo | null> => {
    if (!projectId) return null;

    try {
      setError(null);
      const sandbox = await apiService.createSandbox(name, projectId, isRemote);
      await fetchSandboxes(); // Refresh the list
      return sandbox;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to create sandbox';
      setError(errorMessage);
      return null;
    }
  }, [projectId, fetchSandboxes]);

  const deleteSandbox = useCallback(async (identifier: string, force?: boolean): Promise<boolean> => {
    try {
      setError(null);
      await apiService.deleteSandbox(identifier, force);
      await fetchSandboxes(); // Refresh the list

      // Clear selection if the deleted sandbox was selected
      if (state.selectedSandbox?.id === identifier) {
        dispatch({ type: 'SET_SELECTED_SANDBOX', payload: null });
      }

      return true;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to delete sandbox';
      setError(errorMessage);
      return false;
    }
  }, [fetchSandboxes, state.selectedSandbox?.id, dispatch]);

  const selectSandbox = useCallback((sandbox: SandboxInfo) => {
    dispatch({ type: 'SET_SELECTED_SANDBOX', payload: sandbox });
  }, [dispatch]);

  // Auto-refresh sandboxes
  useEffect(() => {
    if (projectId) {
      fetchSandboxes();

      const interval = setInterval(fetchSandboxes, POLLING_INTERVALS.SANDBOX_STATUS);
      return () => clearInterval(interval);
    }
  }, [projectId, fetchSandboxes]);

  return {
    sandboxes: state.sandboxes,
    selectedSandbox: state.selectedSandbox,
    isLoading,
    error,
    fetchSandboxes,
    createSandbox,
    deleteSandbox,
    selectSandbox
  };
}