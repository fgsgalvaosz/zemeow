// ============================================================================
// INSTALL PROMPT - Componente para prompt de instalação PWA
// ============================================================================

'use client';

import { useState } from 'react';
import { Download, X, Smartphone, Monitor } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { usePWA } from '@/hooks/use-pwa';

interface InstallPromptProps {
  onInstall?: () => void;
  onDismiss?: () => void;
}

export function InstallPrompt({ onInstall, onDismiss }: InstallPromptProps) {
  const { isInstallable, isInstalled, installApp } = usePWA();
  const [isInstalling, setIsInstalling] = useState(false);
  const [isDismissed, setIsDismissed] = useState(false);

  // Don't show if already installed or dismissed
  if (isInstalled || !isInstallable || isDismissed) {
    return null;
  }

  const handleInstall = async () => {
    setIsInstalling(true);
    
    try {
      const success = await installApp();
      
      if (success) {
        onInstall?.();
      }
    } catch (error) {
      console.error('Installation failed:', error);
    } finally {
      setIsInstalling(false);
    }
  };

  const handleDismiss = () => {
    setIsDismissed(true);
    onDismiss?.();
  };

  return (
    <Card className="fixed bottom-4 right-4 w-80 z-50 shadow-lg border-primary/20">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="p-2 bg-primary/10 rounded-lg">
              <Smartphone className="h-4 w-4 text-primary" />
            </div>
            <div>
              <CardTitle className="text-sm">Instalar ZeMeow</CardTitle>
              <Badge variant="secondary" className="text-xs">
                PWA
              </Badge>
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleDismiss}
            className="h-6 w-6 p-0"
          >
            <X className="h-3 w-3" />
          </Button>
        </div>
      </CardHeader>
      
      <CardContent className="pt-0">
        <CardDescription className="text-xs mb-4">
          Instale o ZeMeow no seu dispositivo para acesso rápido e funcionalidades offline.
        </CardDescription>
        
        <div className="space-y-3">
          {/* Features */}
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="flex items-center gap-1">
              <div className="w-1 h-1 bg-green-500 rounded-full" />
              <span>Acesso offline</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-1 h-1 bg-green-500 rounded-full" />
              <span>Notificações</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-1 h-1 bg-green-500 rounded-full" />
              <span>Tela cheia</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-1 h-1 bg-green-500 rounded-full" />
              <span>Ícone na tela</span>
            </div>
          </div>

          {/* Install Button */}
          <Button
            onClick={handleInstall}
            disabled={isInstalling}
            className="w-full"
            size="sm"
          >
            <Download className="h-3 w-3 mr-2" />
            {isInstalling ? 'Instalando...' : 'Instalar App'}
          </Button>

          {/* Manual Instructions */}
          <details className="text-xs">
            <summary className="cursor-pointer text-muted-foreground hover:text-foreground">
              Instalação manual
            </summary>
            <div className="mt-2 space-y-2 text-muted-foreground">
              <div>
                <strong>Chrome/Edge:</strong>
                <br />
                Menu → Instalar ZeMeow
              </div>
              <div>
                <strong>Safari (iOS):</strong>
                <br />
                Compartilhar → Adicionar à Tela de Início
              </div>
              <div>
                <strong>Firefox:</strong>
                <br />
                Menu → Instalar
              </div>
            </div>
          </details>
        </div>
      </CardContent>
    </Card>
  );
}

// Compact version for header
export function InstallButton() {
  const { isInstallable, isInstalled, installApp } = usePWA();
  const [isInstalling, setIsInstalling] = useState(false);

  if (isInstalled || !isInstallable) {
    return null;
  }

  const handleInstall = async () => {
    setIsInstalling(true);
    
    try {
      await installApp();
    } catch (error) {
      console.error('Installation failed:', error);
    } finally {
      setIsInstalling(false);
    }
  };

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={handleInstall}
      disabled={isInstalling}
      className="hidden md:flex"
    >
      <Download className="h-4 w-4 mr-2" />
      {isInstalling ? 'Instalando...' : 'Instalar'}
    </Button>
  );
}

// Status indicator
export function PWAStatus() {
  const { isInstalled, isOnline, isStandalone } = usePWA();

  if (!isInstalled && !isStandalone) {
    return null;
  }

  return (
    <div className="flex items-center gap-2">
      {isStandalone && (
        <Badge variant="secondary" className="text-xs">
          <Monitor className="h-3 w-3 mr-1" />
          PWA
        </Badge>
      )}
      
      <div className={`w-2 h-2 rounded-full ${isOnline ? 'bg-green-500' : 'bg-red-500'}`} />
      <span className="text-xs text-muted-foreground">
        {isOnline ? 'Online' : 'Offline'}
      </span>
    </div>
  );
}
