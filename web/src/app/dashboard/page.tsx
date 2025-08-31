// ============================================================================
// DASHBOARD PAGE - Página principal do dashboard
// ============================================================================

'use client';

import { useEffect, useState } from 'react';
import { Plus, RefreshCw, Activity, Users, MessageSquare, Zap } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { DataTable } from '@/components/ui/data-table';
import { StatusBadge } from '@/components/ui/status-badge';
import { LoadingCard } from '@/components/ui/loading-overlay';
import { MainLayout } from '@/components/layout';
import { ProtectedRoute } from '@/components/auth';
import { useSessionsStore, useUIStore } from '@/store';
import { SessionResponse } from '@/types';
import { formatDistanceToNow } from 'date-fns';
import { ptBR } from 'date-fns/locale';

export default function DashboardPage() {
  const sessionStore = useSessionsStore();
  const { sessions, loading, fetchSessions } = sessionStore;
  const { openModal } = useUIStore();
  const [refreshing, setRefreshing] = useState(false);

  // Fetch sessions on mount
  useEffect(() => {
    fetchSessions();
  }, [fetchSessions]);

  // Calculate statistics
  const stats = {
    total: sessions.length,
    connected: sessions.filter(s => s.status === 'connected').length,
    disconnected: sessions.filter(s => s.status === 'disconnected').length,
    connecting: sessions.filter(s => s.status === 'connecting').length,
    error: sessions.filter(s => s.status === 'error').length,
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    await fetchSessions();
    setRefreshing(false);
  };

  // Table columns for recent sessions
  const columns = [
    {
      key: 'name' as keyof SessionResponse,
      label: 'Nome',
      render: (value: string) => (
        <div className="font-medium">{value}</div>
      ),
    },
    {
      key: 'status' as keyof SessionResponse,
      label: 'Status',
      render: (value: string) => (
        <StatusBadge status={value as any} />
      ),
    },
    {
      key: 'connected' as keyof SessionResponse,
      label: 'Conectado',
      render: (value: boolean) => (
        <Badge variant={value ? 'default' : 'secondary'}>
          {value ? 'Sim' : 'Não'}
        </Badge>
      ),
    },
    {
      key: 'created_at' as keyof SessionResponse,
      label: 'Criado',
      render: (value: string) => (
        <span className="text-sm text-muted-foreground">
          {formatDistanceToNow(new Date(value), { 
            addSuffix: true, 
            locale: ptBR 
          })}
        </span>
      ),
    },
  ];

  return (
    <ProtectedRoute requireGlobalKey>
      <MainLayout>
        <div className="space-y-6">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
              <p className="text-muted-foreground">
                Visão geral das suas sessões WhatsApp
              </p>
            </div>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleRefresh}
                disabled={refreshing}
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
                Atualizar
              </Button>
              <Button onClick={() => openModal('createSession')}>
                <Plus className="h-4 w-4 mr-2" />
                Nova Sessão
              </Button>
            </div>
          </div>

          {/* Statistics Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Total de Sessões
                </CardTitle>
                <Users className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stats.total}</div>
                <p className="text-xs text-muted-foreground">
                  Todas as sessões criadas
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Conectadas
                </CardTitle>
                <Zap className="h-4 w-4 text-green-600" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-green-600">{stats.connected}</div>
                <p className="text-xs text-muted-foreground">
                  Sessões ativas
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Desconectadas
                </CardTitle>
                <Activity className="h-4 w-4 text-gray-600" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-gray-600">{stats.disconnected}</div>
                <p className="text-xs text-muted-foreground">
                  Aguardando conexão
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Taxa de Conexão
                </CardTitle>
                <MessageSquare className="h-4 w-4 text-blue-600" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-blue-600">
                  {stats.total > 0 ? Math.round((stats.connected / stats.total) * 100) : 0}%
                </div>
                <Progress 
                  value={stats.total > 0 ? (stats.connected / stats.total) * 100 : 0} 
                  className="mt-2"
                />
              </CardContent>
            </Card>
          </div>

          {/* Recent Sessions */}
          <Card>
            <CardHeader>
              <CardTitle>Sessões Recentes</CardTitle>
              <CardDescription>
                Últimas sessões criadas e seus status
              </CardDescription>
            </CardHeader>
            <CardContent>
              {loading ? (
                <LoadingCard />
              ) : (
                <DataTable
                  data={sessions.slice(0, 10)}
                  columns={columns}
                  emptyMessage="Nenhuma sessão encontrada. Crie sua primeira sessão!"
                />
              )}
            </CardContent>
          </Card>
        </div>
      </MainLayout>
    </ProtectedRoute>
  );
}
