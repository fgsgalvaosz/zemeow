// ============================================================================
// APP SIDEBAR - Sidebar principal da aplicação ZeMeow
// ============================================================================

'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import {
  LayoutDashboard,
  MessageSquare,
  FileText,
  Settings,
  Shield,
  Plus,
  Activity,
} from 'lucide-react';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from '@/components/ui/sidebar';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useAuthStore, useSessionsStore, useUIStore } from '@/store';

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const pathname = usePathname();
  const { isGlobalKey } = useAuthStore();
  const { sessions } = useSessionsStore();
  const { openModal } = useUIStore();

  // Navigation items
  const navigationItems = [
    {
      title: 'Dashboard',
      url: '/dashboard',
      icon: LayoutDashboard,
      isActive: pathname === '/dashboard',
    },
    {
      title: 'Sessões',
      url: '/sessions',
      icon: MessageSquare,
      isActive: pathname.startsWith('/sessions'),
      badge: sessions.length,
    },
    {
      title: 'Logs',
      url: '/logs',
      icon: FileText,
      isActive: pathname.startsWith('/logs'),
      requireGlobal: true,
    },
    {
      title: 'Configurações',
      url: '/settings',
      icon: Settings,
      isActive: pathname.startsWith('/settings'),
      requireGlobal: true,
    },
  ];

  // Filter items based on permissions
  const filteredItems = navigationItems.filter(item =>
    !item.requireGlobal || isGlobalKey
  );

  // Get connected sessions count
  const connectedSessions = sessions.filter(s => s.status === 'connected').length;

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <div className="flex items-center gap-2 px-4 py-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Shield className="h-4 w-4" />
          </div>
          <div className="grid flex-1 text-left text-sm leading-tight">
            <span className="truncate font-semibold">ZeMeow</span>
            <span className="truncate text-xs text-muted-foreground">
              WhatsApp API
            </span>
          </div>
        </div>
      </SidebarHeader>

      <SidebarContent>
        {/* Quick Actions */}
        <div className="px-4 py-2">
          <Button
            onClick={() => openModal('createSession')}
            className="w-full justify-start"
            size="sm"
          >
            <Plus className="mr-2 h-4 w-4" />
            Nova Sessão
          </Button>
        </div>

        {/* Navigation */}
        <SidebarMenu>
          {filteredItems.map((item) => (
            <SidebarMenuItem key={item.title}>
              <SidebarMenuButton asChild isActive={item.isActive}>
                <Link href={item.url}>
                  <item.icon className="h-4 w-4" />
                  <span>{item.title}</span>
                  {item.badge !== undefined && item.badge > 0 && (
                    <Badge variant="secondary" className="ml-auto">
                      {item.badge}
                    </Badge>
                  )}
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>

        {/* Status Section */}
        <div className="mt-auto px-4 py-2">
          <div className="rounded-lg border p-3">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Status</span>
              <div className="flex items-center gap-1">
                <Activity className="h-3 w-3 text-green-500" />
                <span className="text-green-600">Online</span>
              </div>
            </div>
            <div className="mt-2 text-xs text-muted-foreground">
              {connectedSessions} de {sessions.length} sessões conectadas
            </div>
          </div>
        </div>
      </SidebarContent>

      <SidebarFooter>
        <div className="px-4 py-2">
          <div className="text-xs text-muted-foreground">
            {isGlobalKey ? 'Acesso Global' : 'Acesso de Sessão'}
          </div>
        </div>
      </SidebarFooter>

      <SidebarRail />
    </Sidebar>
  );
}
