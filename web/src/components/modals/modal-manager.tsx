// ============================================================================
// MODAL MANAGER - Gerenciador central de modais
// ============================================================================

'use client';

import { useUIStore, useSessionsStore } from '@/store';
import { CreateSessionModal } from './create-session-modal';
import { QRCodeModal } from './qr-code-modal';

export function ModalManager() {
  const { modals, closeModal } = useUIStore();
  const { sessions } = useSessionsStore();

  // Get the first session as current session for now
  const currentSession = sessions[0] || null;

  return (
    <>
      {/* Create Session Modal */}
      <CreateSessionModal
        open={modals.createSession}
        onOpenChange={(open) => !open && closeModal('createSession')}
      />

      {/* QR Code Modal */}
      <QRCodeModal
        open={modals.qrCode}
        onOpenChange={(open) => !open && closeModal('qrCode')}
        sessionId={currentSession?.id || null}
        sessionName={currentSession?.name}
      />

      {/* Edit Session Modal */}
      {/* TODO: Implement EditSessionModal */}

      {/* Delete Session Modal */}
      {/* TODO: Implement DeleteSessionModal */}

      {/* Settings Modal */}
      {/* TODO: Implement SettingsModal */}
    </>
  );
}
