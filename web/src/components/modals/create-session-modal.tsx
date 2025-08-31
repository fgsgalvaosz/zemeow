// ============================================================================
// CREATE SESSION MODAL - Modal para criar nova sessão
// ============================================================================

'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { LoadingSpinner } from '@/components/ui/loading-overlay';
import { useSessionsStore, useUIStore } from '@/store';
import { notify } from '@/store/ui-store';

// Form validation schema
const createSessionSchema = z.object({
  name: z
    .string()
    .min(1, 'Nome é obrigatório')
    .max(100, 'Nome deve ter no máximo 100 caracteres'),
  sessionId: z
    .string()
    .optional()
    .refine((val) => !val || /^[a-zA-Z0-9_-]+$/.test(val), 'ID deve conter apenas letras, números, _ e -'),
  webhook: z.object({
    enabled: z.boolean(),
    url: z.string().url('URL inválida').optional().or(z.literal('')),
    events: z.array(z.string()).optional(),
  }),
  proxy: z.object({
    enabled: z.boolean(),
    type: z.enum(['http', 'https', 'socks5']).optional(),
    host: z.string().optional(),
    port: z.number().min(1).max(65535).optional(),
    username: z.string().optional(),
    password: z.string().optional(),
  }),
});

type CreateSessionForm = z.infer<typeof createSessionSchema>;

interface CreateSessionModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateSessionModal({ open, onOpenChange }: CreateSessionModalProps) {
  const { createSession } = useSessionsStore();
  const [loading, setLoading] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
    setValue,
    watch,
    reset,
  } = useForm<CreateSessionForm>({
    resolver: zodResolver(createSessionSchema),
    defaultValues: {
      name: '',
      sessionId: '',
      webhook: {
        enabled: false,
        url: '',
        events: ['message', 'receipt'],
      },
      proxy: {
        enabled: false,
        type: 'http',
        host: '',
        port: 8080,
        username: '',
        password: '',
      },
    },
  });

  const webhookEnabled = watch('webhook.enabled');
  const proxyEnabled = watch('proxy.enabled');

  const onSubmit = async (data: CreateSessionForm) => {
    setLoading(true);

    try {
      const payload = {
        name: data.name,
        session_id: data.sessionId || undefined,
        webhook: data.webhook.enabled ? {
          url: data.webhook.url!,
          events: data.webhook.events || ['message', 'receipt'],
          active: true,
        } : undefined,
        proxy: data.proxy.enabled ? {
          enabled: true,
          type: data.proxy.type!,
          host: data.proxy.host!,
          port: data.proxy.port!,
          username: data.proxy.username || undefined,
          password: data.proxy.password || undefined,
        } : undefined,
      };

      await createSession(payload);
      
      notify.success('Sessão criada com sucesso!');
      reset();
      onOpenChange(false);
    } catch (error: any) {
      notify.error('Erro ao criar sessão', error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    reset();
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Nova Sessão WhatsApp</DialogTitle>
          <DialogDescription>
            Crie uma nova sessão para conectar ao WhatsApp
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          {/* Basic Info */}
          <div className="space-y-4">
            <h3 className="text-lg font-medium">Informações Básicas</h3>
            
            <div className="space-y-2">
              <Label htmlFor="name">Nome da Sessão *</Label>
              <Input
                id="name"
                placeholder="Ex: Minha Sessão WhatsApp"
                {...register('name')}
              />
              {errors.name && (
                <p className="text-sm text-red-600">{errors.name.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="sessionId">ID da Sessão (opcional)</Label>
              <Input
                id="sessionId"
                placeholder="Ex: sessao123 (será gerado automaticamente se vazio)"
                {...register('sessionId')}
              />
              {errors.sessionId && (
                <p className="text-sm text-red-600">{errors.sessionId.message}</p>
              )}
            </div>
          </div>

          {/* Webhook Configuration */}
          <div className="space-y-4">
            <div className="flex items-center space-x-2">
              <Switch
                id="webhook-enabled"
                checked={webhookEnabled}
                onCheckedChange={(checked) => setValue('webhook.enabled', checked)}
              />
              <Label htmlFor="webhook-enabled">Configurar Webhook</Label>
            </div>

            {webhookEnabled && (
              <div className="space-y-4 pl-6 border-l-2 border-muted">
                <div className="space-y-2">
                  <Label htmlFor="webhook-url">URL do Webhook</Label>
                  <Input
                    id="webhook-url"
                    placeholder="https://seu-site.com/webhook"
                    {...register('webhook.url')}
                  />
                  {errors.webhook?.url && (
                    <p className="text-sm text-red-600">{errors.webhook.url.message}</p>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Proxy Configuration */}
          <div className="space-y-4">
            <div className="flex items-center space-x-2">
              <Switch
                id="proxy-enabled"
                checked={proxyEnabled}
                onCheckedChange={(checked) => setValue('proxy.enabled', checked)}
              />
              <Label htmlFor="proxy-enabled">Usar Proxy</Label>
            </div>

            {proxyEnabled && (
              <div className="space-y-4 pl-6 border-l-2 border-muted">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="proxy-type">Tipo</Label>
                    <Select onValueChange={(value) => setValue('proxy.type', value as any)}>
                      <SelectTrigger>
                        <SelectValue placeholder="Selecione o tipo" />
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
                      placeholder="8080"
                      {...register('proxy.port', { valueAsNumber: true })}
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="proxy-host">Host</Label>
                  <Input
                    id="proxy-host"
                    placeholder="proxy.exemplo.com"
                    {...register('proxy.host')}
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="proxy-username">Usuário (opcional)</Label>
                    <Input
                      id="proxy-username"
                      {...register('proxy.username')}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="proxy-password">Senha (opcional)</Label>
                    <Input
                      id="proxy-password"
                      type="password"
                      {...register('proxy.password')}
                    />
                  </div>
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancelar
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? (
                <>
                  <LoadingSpinner size="sm" className="mr-2" />
                  Criando...
                </>
              ) : (
                'Criar Sessão'
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
