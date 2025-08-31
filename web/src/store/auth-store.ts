// ============================================================================
// AUTH STORE - Authentication state management with Zustand
// ============================================================================

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { apiClient } from '@/lib/api-client';
import { AuthState } from '@/types';

export interface AuthStore extends AuthState {
  // Actions
  login: (apiKey: string) => Promise<boolean>;
  logout: () => void;
  setApiKey: (apiKey: string) => void;
  clearError: () => void;
  checkAuth: () => Promise<boolean>;
}

export const useAuthStore = create<AuthStore>()(
  persist(
    (set, get) => ({
      // Initial state
      isAuthenticated: false,
      apiKey: null,
      isGlobalKey: false,
      sessionId: undefined,
      loading: false,
      error: null,

      // Actions
      login: async (apiKey: string) => {
        set({ loading: true, error: null });

        try {
          // Set API key in client
          apiClient.setApiKey(apiKey);

          // Validate API key with server
          const response = await apiClient.validateApiKey(apiKey);
          console.log('🔍 Auth validation response:', response);

          if (response.success && response.data.valid) {
            set({
              isAuthenticated: true,
              apiKey,
              isGlobalKey: response.data.type === 'global',
              sessionId: undefined, // Will be set later if needed
              loading: false,
              error: null,
            });
            return true;
          } else {
            throw new Error('Invalid API key');
          }
        } catch (error: any) {
          const errorMessage = error.message || 'Authentication failed';
          
          set({
            isAuthenticated: false,
            apiKey: null,
            isGlobalKey: false,
            sessionId: undefined,
            loading: false,
            error: errorMessage,
          });

          // Clear API key from client
          apiClient.clearApiKey();
          
          return false;
        }
      },

      logout: () => {
        // Clear API key from client
        apiClient.clearApiKey();

        set({
          isAuthenticated: false,
          apiKey: null,
          isGlobalKey: false,
          sessionId: undefined,
          loading: false,
          error: null,
        });
      },

      setApiKey: (apiKey: string) => {
        apiClient.setApiKey(apiKey);
        set({ apiKey });
      },

      clearError: () => {
        set({ error: null });
      },

      checkAuth: async () => {
        const { apiKey } = get();
        
        if (!apiKey) {
          return false;
        }

        set({ loading: true });

        try {
          // Set API key in client
          apiClient.setApiKey(apiKey);

          // Validate with server
          const response = await apiClient.validateApiKey(apiKey);

          if (response.success && response.data.valid) {
            set({
              isAuthenticated: true,
              isGlobalKey: response.data.type === 'global',
              sessionId: undefined, // Will be set later if needed
              loading: false,
              error: null,
            });
            return true;
          } else {
            throw new Error('API key is no longer valid');
          }
        } catch (error: any) {
          const errorMessage = error.message || 'Authentication check failed';
          
          set({
            isAuthenticated: false,
            apiKey: null,
            isGlobalKey: false,
            sessionId: undefined,
            loading: false,
            error: errorMessage,
          });

          // Clear API key from client
          apiClient.clearApiKey();
          
          return false;
        }
      },
    }),
    {
      name: 'zemeow-auth',
      partialize: (state) => ({
        apiKey: state.apiKey,
        isAuthenticated: state.isAuthenticated,
        isGlobalKey: state.isGlobalKey,
        sessionId: state.sessionId,
      }),
    }
  )
);
