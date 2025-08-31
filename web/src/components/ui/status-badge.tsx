// ============================================================================
// STATUS BADGE - Badge para exibir status de sessões
// ============================================================================

'use client';

import { Badge } from '@/components/ui/badge';
import { SessionStatus } from '@/types';
import { CheckCircle, XCircle, Clock, AlertTriangle } from 'lucide-react';

interface StatusBadgeProps {
  status: SessionStatus;
  showIcon?: boolean;
  size?: 'sm' | 'default' | 'lg';
}

export function StatusBadge({ status, showIcon = true, size = 'default' }: StatusBadgeProps) {
  const getStatusConfig = (status: SessionStatus) => {
    switch (status) {
      case 'connected':
        return {
          label: 'Conectado',
          variant: 'default' as const,
          className: 'bg-green-100 text-green-800 hover:bg-green-100 dark:bg-green-900 dark:text-green-300',
          icon: CheckCircle,
        };
      case 'disconnected':
        return {
          label: 'Desconectado',
          variant: 'secondary' as const,
          className: 'bg-gray-100 text-gray-800 hover:bg-gray-100 dark:bg-gray-800 dark:text-gray-300',
          icon: XCircle,
        };
      case 'connecting':
        return {
          label: 'Conectando',
          variant: 'outline' as const,
          className: 'bg-blue-100 text-blue-800 hover:bg-blue-100 dark:bg-blue-900 dark:text-blue-300',
          icon: Clock,
        };
      case 'error':
        return {
          label: 'Erro',
          variant: 'destructive' as const,
          className: 'bg-red-100 text-red-800 hover:bg-red-100 dark:bg-red-900 dark:text-red-300',
          icon: AlertTriangle,
        };
      default:
        return {
          label: 'Desconhecido',
          variant: 'secondary' as const,
          className: '',
          icon: XCircle,
        };
    }
  };

  const config = getStatusConfig(status);
  const Icon = config.icon;

  const sizeClasses = {
    sm: 'text-xs px-2 py-0.5',
    default: 'text-sm px-2.5 py-0.5',
    lg: 'text-base px-3 py-1',
  };

  const iconSizes = {
    sm: 'h-3 w-3',
    default: 'h-4 w-4',
    lg: 'h-5 w-5',
  };

  return (
    <Badge 
      variant={config.variant}
      className={`${config.className} ${sizeClasses[size]} flex items-center gap-1.5`}
    >
      {showIcon && <Icon className={iconSizes[size]} />}
      {config.label}
    </Badge>
  );
}

// Connection Status Indicator (just the icon)
export function ConnectionIndicator({ status, size = 'default' }: {
  status: SessionStatus;
  size?: 'sm' | 'default' | 'lg';
}) {
  const getStatusConfig = (status: SessionStatus) => {
    switch (status) {
      case 'connected':
        return {
          color: 'text-green-500',
          icon: CheckCircle,
        };
      case 'disconnected':
        return {
          color: 'text-gray-400',
          icon: XCircle,
        };
      case 'connecting':
        return {
          color: 'text-blue-500',
          icon: Clock,
        };
      case 'error':
        return {
          color: 'text-red-500',
          icon: AlertTriangle,
        };
      default:
        return {
          color: 'text-gray-400',
          icon: XCircle,
        };
    }
  };

  const config = getStatusConfig(status);
  const Icon = config.icon;

  const iconSizes = {
    sm: 'h-3 w-3',
    default: 'h-4 w-4',
    lg: 'h-5 w-5',
  };

  return (
    <Icon className={`${config.color} ${iconSizes[size]}`} />
  );
}
