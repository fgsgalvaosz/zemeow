// ============================================================================
// UI STORE - UI state management with Zustand
// ============================================================================

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { UIState, Notification } from '@/types';

export interface UIStore extends UIState {
  // Sidebar actions
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;

  // Theme actions
  setTheme: (theme: 'light' | 'dark' | 'system') => void;

  // Notification actions
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp'>) => void;
  removeNotification: (id: string) => void;
  clearNotifications: () => void;

  // Modal actions
  openModal: (modal: keyof UIState['modals']) => void;
  closeModal: (modal: keyof UIState['modals']) => void;
  closeAllModals: () => void;

  // Loading actions
  setGlobalLoading: (loading: boolean) => void;
  setSessionsLoading: (loading: boolean) => void;
  setQRCodeLoading: (loading: boolean) => void;
}

export const useUIStore = create<UIStore>()(
  persist(
    (set, get) => ({
      // Initial state
      sidebarOpen: true,
      theme: 'system',
      notifications: [],
      modals: {
        createSession: false,
        editSession: false,
        deleteSession: false,
        qrCode: false,
        settings: false,
      },
      loading: {
        global: false,
        sessions: false,
        qrCode: false,
      },

      // Sidebar actions
      toggleSidebar: () => {
        set((state) => ({ sidebarOpen: !state.sidebarOpen }));
      },

      setSidebarOpen: (open: boolean) => {
        set({ sidebarOpen: open });
      },

      // Theme actions
      setTheme: (theme: 'light' | 'dark' | 'system') => {
        set({ theme });
        
        // Apply theme to document
        if (typeof window !== 'undefined') {
          const root = window.document.documentElement;
          
          if (theme === 'system') {
            const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
            root.classList.toggle('dark', systemTheme === 'dark');
          } else {
            root.classList.toggle('dark', theme === 'dark');
          }
        }
      },

      // Notification actions
      addNotification: (notification) => {
        const id = Math.random().toString(36).substr(2, 9);
        const timestamp = Date.now();
        
        const newNotification: Notification = {
          ...notification,
          id,
          timestamp,
        };

        set((state) => ({
          notifications: [newNotification, ...state.notifications],
        }));

        // Auto-remove notification after duration
        if (notification.duration !== 0) {
          const duration = notification.duration || 5000;
          setTimeout(() => {
            get().removeNotification(id);
          }, duration);
        }
      },

      removeNotification: (id: string) => {
        set((state) => ({
          notifications: state.notifications.filter(n => n.id !== id),
        }));
      },

      clearNotifications: () => {
        set({ notifications: [] });
      },

      // Modal actions
      openModal: (modal) => {
        set((state) => ({
          modals: {
            ...state.modals,
            [modal]: true,
          },
        }));
      },

      closeModal: (modal) => {
        set((state) => ({
          modals: {
            ...state.modals,
            [modal]: false,
          },
        }));
      },

      closeAllModals: () => {
        set({
          modals: {
            createSession: false,
            editSession: false,
            deleteSession: false,
            qrCode: false,
            settings: false,
          },
        });
      },

      // Loading actions
      setGlobalLoading: (loading: boolean) => {
        set((state) => ({
          loading: {
            ...state.loading,
            global: loading,
          },
        }));
      },

      setSessionsLoading: (loading: boolean) => {
        set((state) => ({
          loading: {
            ...state.loading,
            sessions: loading,
          },
        }));
      },

      setQRCodeLoading: (loading: boolean) => {
        set((state) => ({
          loading: {
            ...state.loading,
            qrCode: loading,
          },
        }));
      },
    }),
    {
      name: 'zemeow-ui',
      partialize: (state) => ({
        sidebarOpen: state.sidebarOpen,
        theme: state.theme,
      }),
    }
  )
);

// Notification helper functions
export const notify = {
  success: (title: string, message?: string, duration?: number) => {
    useUIStore.getState().addNotification({
      type: 'success',
      title,
      message,
      duration,
    });
  },

  error: (title: string, message?: string, duration?: number) => {
    useUIStore.getState().addNotification({
      type: 'error',
      title,
      message,
      duration: duration || 0, // Errors don't auto-dismiss by default
    });
  },

  warning: (title: string, message?: string, duration?: number) => {
    useUIStore.getState().addNotification({
      type: 'warning',
      title,
      message,
      duration,
    });
  },

  info: (title: string, message?: string, duration?: number) => {
    useUIStore.getState().addNotification({
      type: 'info',
      title,
      message,
      duration,
    });
  },
};
