// ============================================================================
// LOGIN PAGE - Página de autenticação com API Key
// ============================================================================

'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Eye, EyeOff, Key, Shield, CheckCircle, AlertCircle } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { LoadingSpinner } from '@/components/ui/loading-overlay';
import { useAuthStore } from '@/store';
import { notify } from '@/store/ui-store';

// Form validation schema
const loginSchema = z.object({
  apiKey: z
    .string()
    .min(1, 'API Key é obrigatória')
    .regex(/^[a-zA-Z0-9_-]+$/, 'API Key contém caracteres inválidos'),
  rememberMe: z.boolean().default(false),
});

type LoginForm = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const router = useRouter();
  const { login, isAuthenticated, loading, error, clearError } = useAuthStore();
  const [showApiKey, setShowApiKey] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setValue,
    watch,
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      apiKey: '',
      rememberMe: false,
    },
  });

  // Redirect if already authenticated
  useEffect(() => {
    if (isAuthenticated) {
      router.push('/dashboard');
    }
  }, [isAuthenticated, router]);

  // Clear errors when component mounts
  useEffect(() => {
    clearError();
  }, [clearError]);

  const onSubmit = async (data: LoginForm) => {
    try {
      const success = await login(data.apiKey);
      
      if (success) {
        notify.success('Login realizado com sucesso!');
        router.push('/dashboard');
      }
    } catch (error: any) {
      notify.error('Erro no login', error.message);
    }
  };

  const handleTestApiKey = () => {
    setValue('apiKey', 'test123');
    setValue('rememberMe', true);
  };

  if (isAuthenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-green-50 to-blue-50 dark:from-gray-900 dark:to-gray-800 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-1 text-center">
          <div className="flex items-center justify-center mb-4">
            <div className="p-3 bg-green-100 dark:bg-green-900 rounded-full">
              <Shield className="h-8 w-8 text-green-600 dark:text-green-400" />
            </div>
          </div>
          <CardTitle className="text-2xl font-bold">ZeMeow API</CardTitle>
          <CardDescription>
            Entre com sua API Key para acessar o painel de controle
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* Error Alert */}
          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {/* Login Form */}
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {/* API Key Field */}
            <div className="space-y-2">
              <Label htmlFor="apiKey">API Key</Label>
              <div className="relative">
                <Key className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                <Input
                  id="apiKey"
                  type={showApiKey ? 'text' : 'password'}
                  placeholder="Digite sua API Key..."
                  className="pl-10 pr-10"
                  {...register('apiKey')}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                  onClick={() => setShowApiKey(!showApiKey)}
                >
                  {showApiKey ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </Button>
              </div>
              {errors.apiKey && (
                <p className="text-sm text-red-600">{errors.apiKey.message}</p>
              )}
            </div>

            {/* Remember Me */}
            <div className="flex items-center space-x-2">
              <Switch
                id="rememberMe"
                checked={watch('rememberMe')}
                onCheckedChange={(checked) => setValue('rememberMe', checked)}
              />
              <Label htmlFor="rememberMe" className="text-sm">
                Lembrar API Key
              </Label>
            </div>

            {/* Submit Button */}
            <Button
              type="submit"
              className="w-full"
              disabled={isSubmitting || loading}
            >
              {isSubmitting || loading ? (
                <>
                  <LoadingSpinner size="sm" className="mr-2" />
                  Verificando...
                </>
              ) : (
                'Entrar'
              )}
            </Button>
          </form>

          {/* Test API Key Button */}
          <div className="pt-4 border-t">
            <Button
              type="button"
              variant="outline"
              className="w-full"
              onClick={handleTestApiKey}
            >
              <CheckCircle className="mr-2 h-4 w-4" />
              Usar API Key de Teste
            </Button>
          </div>

          {/* Help Text */}
          <div className="text-center space-y-2">
            <p className="text-sm text-muted-foreground">
              Não tem uma API Key?
            </p>
            <div className="text-xs text-muted-foreground space-y-1">
              <p>• <strong>Global API Key:</strong> Acesso completo ao sistema</p>
              <p>• <strong>Session API Key:</strong> Acesso a uma sessão específica</p>
              <p>• <strong>Teste:</strong> Use "test123" para demonstração</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
