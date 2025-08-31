// ============================================================================
// SESSIONS STORE - Sessions state management with Zustand
// ============================================================================

import { create } from 'zustand';
import { apiClient } from '@/lib/api-client';
import { SessionsState, SessionResponse, SessionFilters, CreateSessionRequest, UpdateSessionRequest } from '@/types';

export interface SessionsStore extends SessionsState {
  // Actions
  fetchSessions: (params?: {
    page?: number;
    limit?: number;
    status?: string;
    search?: string;
  }) => Promise<void>;
  
  fetchSession: (sessionId: string) => Promise<SessionResponse | null>;
  
  createSession: (data: CreateSessionRequest) => Promise<SessionResponse>;
  
  updateSession: (sessionId: string, data: UpdateSessionRequest) => Promise<SessionResponse>;
  
  deleteSession: (sessionId: string) => Promise<void>;
  
  connectSession: (sessionId: string) => Promise<void>;
  
  disconnectSession: (sessionId: string) => Promise<void>;
  
  setCurrentSession: (session: SessionResponse | null) => void;
  
  setFilters: (filters: Partial<SessionFilters>) => void;
  
  clearError: () => void;
  
  refreshSessions: () => Promise<void>;
}

export const useSessionsStore = create<SessionsStore>((set, get) => ({
  // Initial state
  sessions: [],
  currentSession: null,
  loading: false,
  error: null,
  pagination: {
    page: 1,
    limit: 20,
    total: 0,
    totalPages: 0,
    hasNext: false,
    hasPrev: false,
  },
  filters: {
    sortBy: 'created_at',
    sortOrder: 'desc',
  },

  // Actions
  fetchSessions: async (params) => {
    set({ loading: true, error: null });

    try {
      const { filters, pagination } = get();
      
      const queryParams = {
        page: params?.page ?? pagination.page,
        limit: params?.limit ?? pagination.limit,
        status: params?.status ?? filters.status,
        search: params?.search ?? filters.search,
      };

      const response = await apiClient.getSessions(queryParams);

      if (response.data && response.data.sessions) {
        set({
          sessions: response.data.sessions,
          pagination: {
            page: response.data.pagination.page,
            limit: response.data.pagination.limit || 20,
            total: response.data.pagination.total,
            totalPages: response.data.pagination.total_pages,
            hasNext: response.data.pagination.has_next,
            hasPrev: response.data.pagination.has_prev,
          },
          loading: false,
          error: null,
        });
      } else {
        throw new Error('Invalid response format');
      }
    } catch (error: any) {
      const errorMessage = error.message || 'Failed to fetch sessions';
      set({
        loading: false,
        error: errorMessage,
      });
    }
  },

  fetchSession: async (sessionId: string) => {
    try {
      const session = await apiClient.getSession(sessionId);
      
      // Update session in the list if it exists
      const { sessions } = get();
      const updatedSessions = sessions.map(s => 
        s.id === sessionId ? session : s
      );
      
      set({ sessions: updatedSessions });
      
      return session;
    } catch (error: any) {
      console.error('Failed to fetch session:', error);
      return null;
    }
  },

  createSession: async (data: CreateSessionRequest) => {
    set({ loading: true, error: null });

    try {
      const newSession = await apiClient.createSession(data);
      
      const { sessions } = get();
      set({
        sessions: [newSession, ...sessions],
        loading: false,
        error: null,
      });

      return newSession;
    } catch (error: any) {
      const errorMessage = error.message || 'Failed to create session';
      set({
        loading: false,
        error: errorMessage,
      });
      throw error;
    }
  },

  updateSession: async (sessionId: string, data: UpdateSessionRequest) => {
    set({ loading: true, error: null });

    try {
      const updatedSession = await apiClient.updateSession(sessionId, data);
      
      const { sessions } = get();
      const updatedSessions = sessions.map(session =>
        session.id === sessionId ? updatedSession : session
      );

      set({
        sessions: updatedSessions,
        currentSession: get().currentSession?.id === sessionId ? updatedSession : get().currentSession,
        loading: false,
        error: null,
      });

      return updatedSession;
    } catch (error: any) {
      const errorMessage = error.message || 'Failed to update session';
      set({
        loading: false,
        error: errorMessage,
      });
      throw error;
    }
  },

  deleteSession: async (sessionId: string) => {
    set({ loading: true, error: null });

    try {
      await apiClient.deleteSession(sessionId);
      
      const { sessions } = get();
      const updatedSessions = sessions.filter(session => session.id !== sessionId);

      set({
        sessions: updatedSessions,
        currentSession: get().currentSession?.id === sessionId ? null : get().currentSession,
        loading: false,
        error: null,
      });
    } catch (error: any) {
      const errorMessage = error.message || 'Failed to delete session';
      set({
        loading: false,
        error: errorMessage,
      });
      throw error;
    }
  },

  connectSession: async (sessionId: string) => {
    try {
      await apiClient.connectSession(sessionId);
      
      // Update session status optimistically
      const { sessions } = get();
      const updatedSessions = sessions.map(session =>
        session.id === sessionId 
          ? { ...session, status: 'connecting' as const }
          : session
      );

      set({ sessions: updatedSessions });
    } catch (error: any) {
      const errorMessage = error.message || 'Failed to connect session';
      set({ error: errorMessage });
      throw error;
    }
  },

  disconnectSession: async (sessionId: string) => {
    try {
      await apiClient.disconnectSession(sessionId);
      
      // Update session status optimistically
      const { sessions } = get();
      const updatedSessions = sessions.map(session =>
        session.id === sessionId 
          ? { ...session, status: 'disconnected' as const, connected: false }
          : session
      );

      set({ sessions: updatedSessions });
    } catch (error: any) {
      const errorMessage = error.message || 'Failed to disconnect session';
      set({ error: errorMessage });
      throw error;
    }
  },

  setCurrentSession: (session: SessionResponse | null) => {
    set({ currentSession: session });
  },

  setFilters: (filters: Partial<SessionFilters>) => {
    const currentFilters = get().filters;
    set({
      filters: { ...currentFilters, ...filters },
    });
  },

  clearError: () => {
    set({ error: null });
  },

  refreshSessions: async () => {
    const { pagination, filters } = get();
    await get().fetchSessions({
      page: pagination.page,
      limit: pagination.limit,
      status: filters.status,
      search: filters.search,
    });
  },
}));
