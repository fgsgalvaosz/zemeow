// ============================================================================
// API HOOKS - React hooks for API communication
// ============================================================================

import { useState, useEffect, useCallback } from 'react';
import { apiClient } from '@/lib/api-client';
import {
  SessionResponse,
  CreateSessionRequest,
  UpdateSessionRequest,
  QRCodeResponse,
  PaginationResponse,
  UseApiOptions,
  UseApiResult,
} from '@/types';

// Generic API hook
export function useApi<T>(
  apiCall: () => Promise<T>,
  options: UseApiOptions = {}
): UseApiResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const { enabled = true, refetchInterval, onSuccess, onError } = options;

  const fetchData = useCallback(async () => {
    if (!enabled) return;

    setLoading(true);
    setError(null);

    try {
      const result = await apiCall();
      setData(result);
      onSuccess?.(result);
    } catch (err: any) {
      const errorMessage = err.message || 'An error occurred';
      setError(errorMessage);
      onError?.(err);
    } finally {
      setLoading(false);
    }
  }, [apiCall, enabled, onSuccess, onError]);

  const mutate = useCallback((newData: T) => {
    setData(newData);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  useEffect(() => {
    if (refetchInterval && enabled) {
      const interval = setInterval(fetchData, refetchInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refetchInterval, enabled]);

  return {
    data,
    loading,
    error,
    refetch: fetchData,
    mutate,
  };
}

// Sessions hooks
export function useSessions(params?: {
  page?: number;
  limit?: number;
  status?: string;
  search?: string;
}, options?: UseApiOptions) {
  return useApi(
    () => apiClient.getSessions(params),
    options
  );
}

export function useSession(sessionId: string, options?: UseApiOptions) {
  return useApi(
    () => apiClient.getSession(sessionId),
    { ...options, enabled: !!sessionId && (options?.enabled ?? true) }
  );
}

export function useSessionQR(sessionId: string, options?: UseApiOptions) {
  return useApi(
    () => apiClient.getSessionQR(sessionId),
    { ...options, enabled: !!sessionId && (options?.enabled ?? true) }
  );
}

export function useSessionStatus(sessionId: string, options?: UseApiOptions) {
  return useApi(
    () => apiClient.getSessionStatus(sessionId),
    { 
      ...options, 
      enabled: !!sessionId && (options?.enabled ?? true),
      refetchInterval: options?.refetchInterval ?? 5000 // Default 5 seconds
    }
  );
}

// Session mutations
export function useCreateSession() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createSession = useCallback(async (data: CreateSessionRequest): Promise<SessionResponse> => {
    setLoading(true);
    setError(null);

    try {
      const result = await apiClient.createSession(data);
      return result;
    } catch (err: any) {
      const errorMessage = err.message || 'Failed to create session';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    createSession,
    loading,
    error,
  };
}

export function useUpdateSession() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateSession = useCallback(async (
    sessionId: string, 
    data: UpdateSessionRequest
  ): Promise<SessionResponse> => {
    setLoading(true);
    setError(null);

    try {
      const result = await apiClient.updateSession(sessionId, data);
      return result;
    } catch (err: any) {
      const errorMessage = err.message || 'Failed to update session';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    updateSession,
    loading,
    error,
  };
}

export function useDeleteSession() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const deleteSession = useCallback(async (sessionId: string): Promise<void> => {
    setLoading(true);
    setError(null);

    try {
      await apiClient.deleteSession(sessionId);
    } catch (err: any) {
      const errorMessage = err.message || 'Failed to delete session';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    deleteSession,
    loading,
    error,
  };
}

export function useConnectSession() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const connectSession = useCallback(async (sessionId: string) => {
    setLoading(true);
    setError(null);

    try {
      const result = await apiClient.connectSession(sessionId);
      return result;
    } catch (err: any) {
      const errorMessage = err.message || 'Failed to connect session';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    connectSession,
    loading,
    error,
  };
}

export function useDisconnectSession() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const disconnectSession = useCallback(async (sessionId: string) => {
    setLoading(true);
    setError(null);

    try {
      const result = await apiClient.disconnectSession(sessionId);
      return result;
    } catch (err: any) {
      const errorMessage = err.message || 'Failed to disconnect session';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    disconnectSession,
    loading,
    error,
  };
}

// Authentication hooks
export function useAuth() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const validateApiKey = useCallback(async (apiKey: string) => {
    setLoading(true);
    setError(null);

    try {
      const result = await apiClient.validateApiKey(apiKey);
      return result;
    } catch (err: any) {
      const errorMessage = err.message || 'Invalid API key';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    validateApiKey,
    loading,
    error,
  };
}

// Health check hook
export function useHealth(options?: UseApiOptions) {
  return useApi(
    () => apiClient.health(),
    { ...options, refetchInterval: options?.refetchInterval ?? 30000 } // Default 30 seconds
  );
}

// Message hooks
export function useSendMessage() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sendTextMessage = useCallback(async (
    sessionId: string,
    data: { to: string; text: string; message_id?: string }
  ) => {
    setLoading(true);
    setError(null);

    try {
      const result = await apiClient.sendTextMessage(sessionId, data);
      return result;
    } catch (err: any) {
      const errorMessage = err.message || 'Failed to send message';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  const sendMediaMessage = useCallback(async (
    sessionId: string,
    data: { to: string; media: any; caption?: string; message_id?: string }
  ) => {
    setLoading(true);
    setError(null);

    try {
      const result = await apiClient.sendMediaMessage(sessionId, data);
      return result;
    } catch (err: any) {
      const errorMessage = err.message || 'Failed to send media';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    sendTextMessage,
    sendMediaMessage,
    loading,
    error,
  };
}
