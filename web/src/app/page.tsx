// ============================================================================
// HOME PAGE - Página inicial com redirecionamento
// ============================================================================

'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store';
import { LoadingOverlay } from '@/components/ui/loading-overlay';

export default function HomePage() {
  const router = useRouter();
  const { isAuthenticated, checkAuth } = useAuthStore();

  useEffect(() => {
    const handleRedirect = async () => {
      if (isAuthenticated) {
        router.push('/dashboard');
      } else {
        // Try to check if there's a stored auth
        const isValid = await checkAuth();
        if (isValid) {
          router.push('/dashboard');
        } else {
          router.push('/login');
        }
      }
    };

    handleRedirect();
  }, [isAuthenticated, checkAuth, router]);

  return <LoadingOverlay message="Carregando..." />;
}
