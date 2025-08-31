// ============================================================================
// TYPES INDEX - Export all types
// ============================================================================

// API Types
export * from './api';

// Application Types  
export * from './app';

// Re-export commonly used types for convenience
export type {
  // API
  BaseResponse,
  ErrorResponse,
  SessionResponse,
  SessionStatus,
  CreateSessionRequest,
  UpdateSessionRequest,
  QRCodeResponse,
  AuthContext,
  
  // App State
  AppState,
  AuthState,
  SessionsState,
  UIState,
  
  // Forms
  CreateSessionForm,
  EditSessionForm,
  LoginForm,
  
  // UI Components
  Notification,
  TableProps,
  ModalProps,
  NavItem,
  
  // Dashboard
  DashboardStats,
  SessionActivity,
  
  // Settings
  AppSettings,
} from './api';

export type {
  AppState,
  AuthState,
  SessionsState,
  UIState,
  Notification,
  CreateSessionForm,
  EditSessionForm,
  LoginForm,
  QRCodeState,
  DashboardStats,
  SessionActivity,
  AppSettings,
  TableProps,
  ModalProps,
  NavItem,
  BreadcrumbItem,
  UseApiOptions,
  UseApiResult,
  WebSocketMessage,
  WebSocketState,
} from './app';
