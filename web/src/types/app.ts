// ============================================================================
// APPLICATION TYPES - Frontend State Management
// ============================================================================

import { SessionResponse, SessionStatus, QRCodeResponse } from './api';

// Application State Types
export interface AppState {
  auth: AuthState;
  sessions: SessionsState;
  ui: UIState;
}

// Authentication State
export interface AuthState {
  isAuthenticated: boolean;
  apiKey: string | null;
  isGlobalKey: boolean;
  sessionId?: string;
  loading: boolean;
  error: string | null;
}

// Sessions State
export interface SessionsState {
  sessions: SessionResponse[];
  currentSession: SessionResponse | null;
  loading: boolean;
  error: string | null;
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
    hasPrev: boolean;
  };
  filters: SessionFilters;
}

export interface SessionFilters {
  status?: SessionStatus;
  search?: string;
  sortBy?: 'name' | 'created_at' | 'updated_at' | 'status';
  sortOrder?: 'asc' | 'desc';
}

// UI State
export interface UIState {
  sidebarOpen: boolean;
  theme: 'light' | 'dark' | 'system';
  notifications: Notification[];
  modals: {
    createSession: boolean;
    editSession: boolean;
    deleteSession: boolean;
    qrCode: boolean;
    settings: boolean;
  };
  loading: {
    global: boolean;
    sessions: boolean;
    qrCode: boolean;
  };
}

// Notification Types
export interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  duration?: number;
  timestamp: number;
  actions?: NotificationAction[];
}

export interface NotificationAction {
  label: string;
  action: () => void;
  variant?: 'default' | 'destructive';
}

// Form Types
export interface CreateSessionForm {
  name: string;
  sessionId?: string;
  webhook?: {
    enabled: boolean;
    url?: string;
    events?: string[];
    secret?: string;
  };
  proxy?: {
    enabled: boolean;
    type?: 'http' | 'https' | 'socks5';
    host?: string;
    port?: number;
    username?: string;
    password?: string;
  };
}

export interface EditSessionForm extends Partial<CreateSessionForm> {
  id: string;
}

export interface LoginForm {
  apiKey: string;
  rememberMe: boolean;
}

// QR Code State
export interface QRCodeState {
  sessionId: string | null;
  qrCode: string | null;
  status: 'idle' | 'generating' | 'ready' | 'expired' | 'error';
  expiresAt: number | null;
  timeLeft: number;
  error: string | null;
  autoRefresh: boolean;
}

// Dashboard Types
export interface DashboardStats {
  totalSessions: number;
  connectedSessions: number;
  disconnectedSessions: number;
  connectingSessions: number;
  errorSessions: number;
  totalMessages: number;
  messagesLast24h: number;
  uptime: number;
}

export interface SessionActivity {
  sessionId: string;
  sessionName: string;
  type: 'connected' | 'disconnected' | 'message_sent' | 'message_received' | 'error';
  message?: string;
  timestamp: string;
}

// Settings Types
export interface AppSettings {
  theme: 'light' | 'dark' | 'system';
  language: 'pt' | 'en' | 'es';
  notifications: {
    enabled: boolean;
    sound: boolean;
    desktop: boolean;
    email: boolean;
  };
  dashboard: {
    autoRefresh: boolean;
    refreshInterval: number;
    showStats: boolean;
    showActivity: boolean;
  };
  sessions: {
    autoConnect: boolean;
    qrCodeTimeout: number;
    defaultWebhookEvents: string[];
  };
}

// Table Types
export interface TableColumn<T = any> {
  key: keyof T;
  label: string;
  sortable?: boolean;
  width?: string;
  align?: 'left' | 'center' | 'right';
  render?: (value: any, row: T) => React.ReactNode;
}

export interface TableProps<T = any> {
  data: T[];
  columns: TableColumn<T>[];
  loading?: boolean;
  pagination?: {
    page: number;
    limit: number;
    total: number;
    onPageChange: (page: number) => void;
    onLimitChange: (limit: number) => void;
  };
  sorting?: {
    sortBy: keyof T;
    sortOrder: 'asc' | 'desc';
    onSort: (column: keyof T) => void;
  };
  selection?: {
    selectedRows: string[];
    onSelectionChange: (selectedRows: string[]) => void;
  };
  actions?: {
    label: string;
    icon?: React.ReactNode;
    onClick: (row: T) => void;
    variant?: 'default' | 'destructive' | 'outline' | 'secondary';
    disabled?: (row: T) => boolean;
  }[];
}

// Modal Types
export interface ModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  children: React.ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
}

// Form Field Types
export interface FormField {
  name: string;
  label: string;
  type: 'text' | 'email' | 'password' | 'number' | 'select' | 'textarea' | 'checkbox' | 'switch' | 'file';
  placeholder?: string;
  description?: string;
  required?: boolean;
  disabled?: boolean;
  options?: { label: string; value: string }[];
  validation?: {
    min?: number;
    max?: number;
    pattern?: RegExp;
    custom?: (value: any) => string | undefined;
  };
}

// Navigation Types
export interface NavItem {
  title: string;
  href?: string;
  icon?: React.ReactNode;
  badge?: string | number;
  children?: NavItem[];
  disabled?: boolean;
  external?: boolean;
}

export interface BreadcrumbItem {
  label: string;
  href?: string;
  current?: boolean;
}

// API Hook Types
export interface UseApiOptions {
  enabled?: boolean;
  refetchInterval?: number;
  onSuccess?: (data: any) => void;
  onError?: (error: any) => void;
}

export interface UseApiResult<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
  mutate: (data: T) => void;
}

// WebSocket Types
export interface WebSocketMessage {
  type: string;
  sessionId?: string;
  data: any;
  timestamp: number;
}

export interface WebSocketState {
  connected: boolean;
  connecting: boolean;
  error: string | null;
  lastMessage: WebSocketMessage | null;
  reconnectAttempts: number;
}

// Export all types
export type * from './api';
