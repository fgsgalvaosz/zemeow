// ============================================================================
// SETTINGS PAGE - Página de configurações
// ============================================================================

'use client';

import { useState } from 'react';
import { Save, TestTube, Globe, Shield, Bell, Palette, Database } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { MainLayout } from '@/components/layout';
import { ProtectedRoute } from '@/components/auth';
import { useUIStore } from '@/store';
import { notify } from '@/store/ui-store';

export default function SettingsPage() {
  const { theme, setTheme } = useUIStore();
  const [loading, setLoading] = useState(false);

  // Webhook settings
  const [webhookSettings, setWebhookSettings] = useState({
    enabled: true,
    url: 'https://api.exemplo.com/webhook',
    events: ['message', 'receipt', 'presence'],
    secret: '',
    retries: 3,
    timeout: 30,
  });

  // Proxy settings
  const [proxySettings, setProxySettings] = useState({
    enabled: false,
    type: 'http',
    host: '',
    port: 8080,
    username: '',
    password: '',
  });

  // General settings
  const [generalSettings, setGeneralSettings] = useState({
    autoReconnect: true,
    qrTimeout: 120,
    messageTimeout: 30,
    logLevel: 'info',
    maxSessions: 10,
  });

  // Notification settings
  const [notificationSettings, setNotificationSettings] = useState({
    enabled: true,
    email: true,
    browser: true,
    sound: false,
    emailAddress: 'admin@exemplo.com',
  });

  const handleSaveWebhook = async () => {
    setLoading(true);
    try {
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 1000));
      notify.success('Configurações de webhook salvas!');
    } catch (error) {
      notify.error('Erro ao salvar configurações de webhook');
    } finally {
      setLoading(false);
    }
  };

  const handleTestWebhook = async () => {
    setLoading(true);
    try {
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 2000));
      notify.success('Webhook testado com sucesso!');
    } catch (error) {
      notify.error('Falha no teste do webhook');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveProxy = async () => {
    setLoading(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      notify.success('Configurações de proxy salvas!');
    } catch (error) {
      notify.error('Erro ao salvar configurações de proxy');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveGeneral = async () => {
    setLoading(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      notify.success('Configurações gerais salvas!');
    } catch (error) {
      notify.error('Erro ao salvar configurações gerais');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveNotifications = async () => {
    setLoading(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      notify.success('Configurações de notificação salvas!');
    } catch (error) {
      notify.error('Erro ao salvar configurações de notificação');
    } finally {
      setLoading(false);
    }
  };

  const availableEvents = [
    { id: 'message', label: 'Mensagens' },
    { id: 'receipt', label: 'Confirmações de leitura' },
    { id: 'presence', label: 'Status de presença' },
    { id: 'connection', label: 'Eventos de conexão' },
    { id: 'qr', label: 'QR Code' },
  ];

  return (
    <ProtectedRoute requireGlobalKey>
      <MainLayout>
        <div className="space-y-6">
          {/* Header */}
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Configurações</h1>
            <p className="text-muted-foreground">
              Configure webhooks, proxy e outras opções do sistema
            </p>
          </div>

          <Tabs defaultValue="webhooks" className="space-y-6">
            <TabsList className="grid w-full grid-cols-5">
              <TabsTrigger value="webhooks" className="flex items-center gap-2">
                <Globe className="h-4 w-4" />
                Webhooks
              </TabsTrigger>
              <TabsTrigger value="proxy" className="flex items-center gap-2">
                <Shield className="h-4 w-4" />
                Proxy
              </TabsTrigger>
              <TabsTrigger value="general" className="flex items-center gap-2">
                <Database className="h-4 w-4" />
                Geral
              </TabsTrigger>
              <TabsTrigger value="notifications" className="flex items-center gap-2">
                <Bell className="h-4 w-4" />
                Notificações
              </TabsTrigger>
              <TabsTrigger value="appearance" className="flex items-center gap-2">
                <Palette className="h-4 w-4" />
                Aparência
              </TabsTrigger>
            </TabsList>

            {/* Webhooks Tab */}
            <TabsContent value="webhooks">
              <Card>
                <CardHeader>
                  <CardTitle>Configurações de Webhook</CardTitle>
                  <CardDescription>
                    Configure webhooks para receber eventos em tempo real
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="flex items-center space-x-2">
                    <Switch
                      id="webhook-enabled"
                      checked={webhookSettings.enabled}
                      onCheckedChange={(checked) =>
                        setWebhookSettings(prev => ({ ...prev, enabled: checked }))
                      }
                    />
                    <Label htmlFor="webhook-enabled">Habilitar Webhooks</Label>
                  </div>

                  {webhookSettings.enabled && (
                    <div className="space-y-4">
                      <div className="space-y-2">
                        <Label htmlFor="webhook-url">URL do Webhook</Label>
                        <Input
                          id="webhook-url"
                          placeholder="https://api.exemplo.com/webhook"
                          value={webhookSettings.url}
                          onChange={(e) =>
                            setWebhookSettings(prev => ({ ...prev, url: e.target.value }))
                          }
                        />
                      </div>

                      <div className="space-y-2">
                        <Label>Eventos</Label>
                        <div className="flex flex-wrap gap-2">
                          {availableEvents.map((event) => (
                            <Badge
                              key={event.id}
                              variant={webhookSettings.events.includes(event.id) ? 'default' : 'outline'}
                              className="cursor-pointer"
                              onClick={() => {
                                const events = webhookSettings.events.includes(event.id)
                                  ? webhookSettings.events.filter(e => e !== event.id)
                                  : [...webhookSettings.events, event.id];
                                setWebhookSettings(prev => ({ ...prev, events }));
                              }}
                            >
                              {event.label}
                            </Badge>
                          ))}
                        </div>
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                          <Label htmlFor="webhook-retries">Tentativas</Label>
                          <Input
                            id="webhook-retries"
                            type="number"
                            min="1"
                            max="10"
                            value={webhookSettings.retries}
                            onChange={(e) =>
                              setWebhookSettings(prev => ({ ...prev, retries: parseInt(e.target.value) }))
                            }
                          />
                        </div>

                        <div className="space-y-2">
                          <Label htmlFor="webhook-timeout">Timeout (segundos)</Label>
                          <Input
                            id="webhook-timeout"
                            type="number"
                            min="5"
                            max="120"
                            value={webhookSettings.timeout}
                            onChange={(e) =>
                              setWebhookSettings(prev => ({ ...prev, timeout: parseInt(e.target.value) }))
                            }
                          />
                        </div>
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="webhook-secret">Secret (opcional)</Label>
                        <Input
                          id="webhook-secret"
                          type="password"
                          placeholder="Chave secreta para validação"
                          value={webhookSettings.secret}
                          onChange={(e) =>
                            setWebhookSettings(prev => ({ ...prev, secret: e.target.value }))
                          }
                        />
                      </div>
                    </div>
                  )}

                  <div className="flex gap-2">
                    <Button onClick={handleSaveWebhook} disabled={loading}>
                      <Save className="h-4 w-4 mr-2" />
                      Salvar
                    </Button>
                    {webhookSettings.enabled && (
                      <Button variant="outline" onClick={handleTestWebhook} disabled={loading}>
                        <TestTube className="h-4 w-4 mr-2" />
                        Testar Webhook
                      </Button>
                    )}
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            {/* Proxy Tab */}
            <TabsContent value="proxy">
              <Card>
                <CardHeader>
                  <CardTitle>Configurações de Proxy</CardTitle>
                  <CardDescription>
                    Configure proxy para conexões WhatsApp
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="flex items-center space-x-2">
                    <Switch
                      id="proxy-enabled"
                      checked={proxySettings.enabled}
                      onCheckedChange={(checked) =>
                        setProxySettings(prev => ({ ...prev, enabled: checked }))
                      }
                    />
                    <Label htmlFor="proxy-enabled">Usar Proxy</Label>
                  </div>

                  {proxySettings.enabled && (
                    <div className="space-y-4">
                      <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                          <Label htmlFor="proxy-type">Tipo</Label>
                          <Select
                            value={proxySettings.type}
                            onValueChange={(value) =>
                              setProxySettings(prev => ({ ...prev, type: value }))
                            }
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="http">HTTP</SelectItem>
                              <SelectItem value="https">HTTPS</SelectItem>
                              <SelectItem value="socks5">SOCKS5</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>

                        <div className="space-y-2">
                          <Label htmlFor="proxy-port">Porta</Label>
                          <Input
                            id="proxy-port"
                            type="number"
                            value={proxySettings.port}
                            onChange={(e) =>
                              setProxySettings(prev => ({ ...prev, port: parseInt(e.target.value) }))
                            }
                          />
                        </div>
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="proxy-host">Host</Label>
                        <Input
                          id="proxy-host"
                          placeholder="proxy.exemplo.com"
                          value={proxySettings.host}
                          onChange={(e) =>
                            setProxySettings(prev => ({ ...prev, host: e.target.value }))
                          }
                        />
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                          <Label htmlFor="proxy-username">Usuário (opcional)</Label>
                          <Input
                            id="proxy-username"
                            value={proxySettings.username}
                            onChange={(e) =>
                              setProxySettings(prev => ({ ...prev, username: e.target.value }))
                            }
                          />
                        </div>

                        <div className="space-y-2">
                          <Label htmlFor="proxy-password">Senha (opcional)</Label>
                          <Input
                            id="proxy-password"
                            type="password"
                            value={proxySettings.password}
                            onChange={(e) =>
                              setProxySettings(prev => ({ ...prev, password: e.target.value }))
                            }
                          />
                        </div>
                      </div>
                    </div>
                  )}

                  <Button onClick={handleSaveProxy} disabled={loading}>
                    <Save className="h-4 w-4 mr-2" />
                    Salvar Configurações
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>

            {/* General Tab */}
            <TabsContent value="general">
              <Card>
                <CardHeader>
                  <CardTitle>Configurações Gerais</CardTitle>
                  <CardDescription>
                    Configurações gerais do sistema
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-4">
                    <div className="flex items-center space-x-2">
                      <Switch
                        id="auto-reconnect"
                        checked={generalSettings.autoReconnect}
                        onCheckedChange={(checked) =>
                          setGeneralSettings(prev => ({ ...prev, autoReconnect: checked }))
                        }
                      />
                      <Label htmlFor="auto-reconnect">Reconexão automática</Label>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label htmlFor="qr-timeout">Timeout QR Code (segundos)</Label>
                        <Input
                          id="qr-timeout"
                          type="number"
                          min="60"
                          max="300"
                          value={generalSettings.qrTimeout}
                          onChange={(e) =>
                            setGeneralSettings(prev => ({ ...prev, qrTimeout: parseInt(e.target.value) }))
                          }
                        />
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="message-timeout">Timeout Mensagem (segundos)</Label>
                        <Input
                          id="message-timeout"
                          type="number"
                          min="10"
                          max="120"
                          value={generalSettings.messageTimeout}
                          onChange={(e) =>
                            setGeneralSettings(prev => ({ ...prev, messageTimeout: parseInt(e.target.value) }))
                          }
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label htmlFor="log-level">Nível de Log</Label>
                        <Select
                          value={generalSettings.logLevel}
                          onValueChange={(value) =>
                            setGeneralSettings(prev => ({ ...prev, logLevel: value }))
                          }
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="debug">Debug</SelectItem>
                            <SelectItem value="info">Info</SelectItem>
                            <SelectItem value="warning">Warning</SelectItem>
                            <SelectItem value="error">Error</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="max-sessions">Máximo de Sessões</Label>
                        <Input
                          id="max-sessions"
                          type="number"
                          min="1"
                          max="100"
                          value={generalSettings.maxSessions}
                          onChange={(e) =>
                            setGeneralSettings(prev => ({ ...prev, maxSessions: parseInt(e.target.value) }))
                          }
                        />
                      </div>
                    </div>
                  </div>

                  <Button onClick={handleSaveGeneral} disabled={loading}>
                    <Save className="h-4 w-4 mr-2" />
                    Salvar Configurações
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>

            {/* Notifications Tab */}
            <TabsContent value="notifications">
              <Card>
                <CardHeader>
                  <CardTitle>Configurações de Notificação</CardTitle>
                  <CardDescription>
                    Configure como receber notificações do sistema
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-4">
                    <div className="flex items-center space-x-2">
                      <Switch
                        id="notifications-enabled"
                        checked={notificationSettings.enabled}
                        onCheckedChange={(checked) =>
                          setNotificationSettings(prev => ({ ...prev, enabled: checked }))
                        }
                      />
                      <Label htmlFor="notifications-enabled">Habilitar notificações</Label>
                    </div>

                    {notificationSettings.enabled && (
                      <div className="space-y-4 pl-6 border-l-2 border-muted">
                        <div className="flex items-center space-x-2">
                          <Switch
                            id="email-notifications"
                            checked={notificationSettings.email}
                            onCheckedChange={(checked) =>
                              setNotificationSettings(prev => ({ ...prev, email: checked }))
                            }
                          />
                          <Label htmlFor="email-notifications">Notificações por email</Label>
                        </div>

                        {notificationSettings.email && (
                          <div className="space-y-2">
                            <Label htmlFor="email-address">Endereço de email</Label>
                            <Input
                              id="email-address"
                              type="email"
                              value={notificationSettings.emailAddress}
                              onChange={(e) =>
                                setNotificationSettings(prev => ({ ...prev, emailAddress: e.target.value }))
                              }
                            />
                          </div>
                        )}

                        <div className="flex items-center space-x-2">
                          <Switch
                            id="browser-notifications"
                            checked={notificationSettings.browser}
                            onCheckedChange={(checked) =>
                              setNotificationSettings(prev => ({ ...prev, browser: checked }))
                            }
                          />
                          <Label htmlFor="browser-notifications">Notificações do navegador</Label>
                        </div>

                        <div className="flex items-center space-x-2">
                          <Switch
                            id="sound-notifications"
                            checked={notificationSettings.sound}
                            onCheckedChange={(checked) =>
                              setNotificationSettings(prev => ({ ...prev, sound: checked }))
                            }
                          />
                          <Label htmlFor="sound-notifications">Sons de notificação</Label>
                        </div>
                      </div>
                    )}
                  </div>

                  <Button onClick={handleSaveNotifications} disabled={loading}>
                    <Save className="h-4 w-4 mr-2" />
                    Salvar Configurações
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>

            {/* Appearance Tab */}
            <TabsContent value="appearance">
              <Card>
                <CardHeader>
                  <CardTitle>Aparência</CardTitle>
                  <CardDescription>
                    Personalize a aparência da interface
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-2">
                    <Label>Tema</Label>
                    <Select value={theme} onValueChange={(value: any) => setTheme(value)}>
                      <SelectTrigger className="w-[200px]">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="light">Claro</SelectItem>
                        <SelectItem value="dark">Escuro</SelectItem>
                        <SelectItem value="system">Sistema</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <Alert>
                    <Palette className="h-4 w-4" />
                    <AlertDescription>
                      O tema será aplicado imediatamente e salvo automaticamente.
                    </AlertDescription>
                  </Alert>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>
      </MainLayout>
    </ProtectedRoute>
  );
}
