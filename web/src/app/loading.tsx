// ============================================================================
// GLOBAL LOADING - Loading component para App Router
// ============================================================================

import { LoadingOverlay } from '@/components/ui/loading-overlay';

export default function Loading() {
  return <LoadingOverlay message="Carregando..." />;
}
