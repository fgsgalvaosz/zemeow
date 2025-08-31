// ============================================================================
// MAIN LAYOUT - Layout principal da aplicação
// ============================================================================

'use client';

import { useEffect } from 'react';
import { useAuthStore, useUIStore } from '@/store';
import { AppSidebar } from '@/components/app-sidebar';
import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar';
import { Header } from './header';
import { Toaster } from '@/components/ui/sonner';
import { LoadingOverlay } from '@/components/ui/loading-overlay';
import { ModalManager } from '@/components/modals/modal-manager';

interface MainLayoutProps {
  children: React.ReactNode;
}

export function MainLayout({ children }: MainLayoutProps) {
  const { isAuthenticated, checkAuth } = useAuthStore();
  const { theme, setTheme, loading } = useUIStore();

  // Initialize theme on mount
  useEffect(() => {
    setTheme(theme);
  }, [theme, setTheme]);

  // Check authentication on mount
  useEffect(() => {
    if (isAuthenticated) {
      checkAuth();
    }
  }, [isAuthenticated, checkAuth]);

  // Apply theme class to document
  useEffect(() => {
    const root = window.document.documentElement;
    
    if (theme === 'system') {
      const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
      root.classList.toggle('dark', systemTheme === 'dark');
    } else {
      root.classList.toggle('dark', theme === 'dark');
    }
  }, [theme]);

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen bg-background">
        {children}
        <Toaster />
      </div>
    );
  }

  return (
    <SidebarProvider>
      <div className="min-h-screen flex w-full bg-background">
        <AppSidebar />
        
        <SidebarInset className="flex-1">
          <Header />
          
          <main className="flex-1 p-6">
            {children}
          </main>
        </SidebarInset>
      </div>

      {/* Global Loading Overlay */}
      {loading.global && <LoadingOverlay />}

      {/* Toast Notifications */}
      <Toaster />

      {/* Modal Manager */}
      <ModalManager />
    </SidebarProvider>
  );
}
