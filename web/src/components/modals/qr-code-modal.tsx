// ============================================================================
// QR CODE MODAL - Modal avançado para exibição de QR Code
// ============================================================================

'use client';

import { useState, useEffect, useCallback } from 'react';
import { QrCode, RefreshCw, CheckCircle, XCircle, Clock, Smartphone } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { LoadingSpinner } from '@/components/ui/loading-overlay';
import { useSessionQR } from '@/hooks/use-api';
import { notify } from '@/store/ui-store';

interface QRCodeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sessionId: string | null;
  sessionName?: string;
}

export function QRCodeModal({ open, onOpenChange, sessionId, sessionName }: QRCodeModalProps) {
  const [timeLeft, setTimeLeft] = useState(0);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [refreshCount, setRefreshCount] = useState(0);

  const {
    data: qrData,
    loading,
    error,
    refetch,
  } = useSessionQR(sessionId || '', {
    enabled: open && !!sessionId,
    refetchInterval: autoRefresh ? 5000 : undefined,
  });

  // Timer countdown
  useEffect(() => {
    if (!qrData?.expires_at) return;

    const expiresAt = qrData.expires_at * 1000; // Convert to milliseconds
    const interval = setInterval(() => {
      const now = Date.now();
      const remaining = Math.max(0, expiresAt - now);
      setTimeLeft(remaining);

      if (remaining === 0) {
        setAutoRefresh(false);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [qrData?.expires_at]);

  // Manual refresh
  const handleRefresh = useCallback(async () => {
    setRefreshCount(prev => prev + 1);
    setAutoRefresh(true);
    await refetch();
  }, [refetch]);

  // Copy QR code data
  const handleCopyQRData = useCallback(async () => {
    if (!qrData?.qr_data) return;

    try {
      await navigator.clipboard.writeText(qrData.qr_data);
      notify.success('Dados do QR Code copiados!');
    } catch (error) {
      notify.error('Erro ao copiar dados do QR Code');
    }
  }, [qrData?.qr_data]);

  // Download QR code as image
  const handleDownloadQR = useCallback(() => {
    if (!qrData?.qr_code) return;

    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    const img = new Image();
    
    img.onload = () => {
      canvas.width = img.width;
      canvas.height = img.height;
      ctx?.drawImage(img, 0, 0);
      
      const link = document.createElement('a');
      link.download = `qr-code-${sessionName || sessionId}.png`;
      link.href = canvas.toDataURL();
      link.click();
    };
    
    img.src = `data:image/svg+xml;base64,${btoa(qrData.qr_code)}`;
  }, [qrData?.qr_code, sessionName, sessionId]);

  // Format time remaining
  const formatTimeLeft = (ms: number) => {
    const minutes = Math.floor(ms / 60000);
    const seconds = Math.floor((ms % 60000) / 1000);
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  };

  // Get status info
  const getStatusInfo = () => {
    if (loading) {
      return {
        icon: <LoadingSpinner size="sm" />,
        text: 'Gerando QR Code...',
        color: 'text-blue-600',
        bgColor: 'bg-blue-100',
      };
    }

    if (error) {
      return {
        icon: <AlertCircle className="h-4 w-4" />,
        text: 'Erro ao gerar QR Code',
        color: 'text-red-600',
        bgColor: 'bg-red-100',
      };
    }

    if (timeLeft === 0) {
      return {
        icon: <Clock className="h-4 w-4" />,
        text: 'QR Code expirado',
        color: 'text-orange-600',
        bgColor: 'bg-orange-100',
      };
    }

    if (qrData?.status === 'connected') {
      return {
        icon: <CheckCircle className="h-4 w-4" />,
        text: 'Conectado com sucesso!',
        color: 'text-green-600',
        bgColor: 'bg-green-100',
      };
    }

    return {
      icon: <Smartphone className="h-4 w-4" />,
      text: 'Aguardando leitura do QR Code',
      color: 'text-blue-600',
      bgColor: 'bg-blue-100',
    };
  };

  const statusInfo = getStatusInfo();
  const progressValue = qrData?.expires_at 
    ? Math.max(0, (timeLeft / (qrData.timeout * 1000)) * 100)
    : 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Conectar WhatsApp</DialogTitle>
          <DialogDescription>
            {sessionName ? `Sessão: ${sessionName}` : `ID: ${sessionId}`}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Status Badge */}
          <div className="flex items-center justify-center">
            <Badge 
              variant="outline" 
              className={`${statusInfo.bgColor} ${statusInfo.color} border-current`}
            >
              {statusInfo.icon}
              <span className="ml-2">{statusInfo.text}</span>
            </Badge>
          </div>

          {/* QR Code Display */}
          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-center">
                {loading ? (
                  <div className="flex flex-col items-center space-y-4">
                    <LoadingSpinner size="lg" />
                    <p className="text-sm text-muted-foreground">
                      Gerando QR Code...
                    </p>
                  </div>
                ) : error ? (
                  <div className="flex flex-col items-center space-y-4 text-center">
                    <AlertCircle className="h-12 w-12 text-red-500" />
                    <div>
                      <p className="font-medium text-red-600">Erro ao gerar QR Code</p>
                      <p className="text-sm text-muted-foreground mt-1">
                        {error}
                      </p>
                    </div>
                    <Button onClick={handleRefresh} size="sm">
                      <RefreshCw className="h-4 w-4 mr-2" />
                      Tentar Novamente
                    </Button>
                  </div>
                ) : qrData?.qr_code ? (
                  <div className="space-y-4">
                    <div className="bg-white p-4 rounded-lg">
                      <QRCode
                        value={qrData.qr_data || qrData.qr_code}
                        size={200}
                        style={{ height: "auto", maxWidth: "100%", width: "100%" }}
                      />
                    </div>
                    
                    {/* Timer and Progress */}
                    {timeLeft > 0 && (
                      <div className="space-y-2">
                        <div className="flex items-center justify-between text-sm">
                          <span className="text-muted-foreground">Expira em:</span>
                          <span className="font-mono font-medium">
                            {formatTimeLeft(timeLeft)}
                          </span>
                        </div>
                        <Progress value={progressValue} className="h-2" />
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="text-center">
                    <p className="text-muted-foreground">
                      Nenhum QR Code disponível
                    </p>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Instructions */}
          <div className="space-y-3 text-sm text-muted-foreground">
            <h4 className="font-medium text-foreground">Como conectar:</h4>
            <ol className="space-y-1 list-decimal list-inside">
              <li>Abra o WhatsApp no seu celular</li>
              <li>Toque em "Mais opções" (⋮) e depois em "Aparelhos conectados"</li>
              <li>Toque em "Conectar um aparelho"</li>
              <li>Aponte a câmera para este QR Code</li>
            </ol>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleRefresh}
                disabled={loading}
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
                Atualizar
              </Button>
              
              {qrData?.qr_data && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCopyQRData}
                >
                  <Copy className="h-4 w-4 mr-2" />
                  Copiar
                </Button>
              )}
            </div>

            {qrData?.qr_code && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleDownloadQR}
              >
                <Download className="h-4 w-4 mr-2" />
                Baixar
              </Button>
            )}
          </div>

          {/* Refresh Counter */}
          {refreshCount > 0 && (
            <p className="text-xs text-center text-muted-foreground">
              Atualizado {refreshCount} vez{refreshCount > 1 ? 'es' : ''}
            </p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
