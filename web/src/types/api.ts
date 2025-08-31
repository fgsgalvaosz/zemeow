// ============================================================================
// API TYPES - ZeMeow WhatsApp API
// ============================================================================

// Base Response Types
export interface BaseResponse<T = any> {
  success: boolean;
  message?: string;
  error?: string;
  code?: string;
  timestamp: number;
  data?: T;
}

export interface ErrorResponse {
  success: false;
  error: string;
  message: string;
  code?: string;
  status: number;
  timestamp: number;
}

export interface PaginationInfo {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

export interface PaginationResponse<T> extends BaseResponse {
  data: {
    pagination: PaginationInfo;
    [key: string]: T[] | PaginationInfo;
  };
}

// Session Types
export type SessionStatus = 'connected' | 'disconnected' | 'connecting' | 'error';

export interface ProxyConfig {
  enabled: boolean;
  type?: 'http' | 'https' | 'socks5';
  host?: string;
  port?: number;
  username?: string;
  password?: string;
}

export interface WebhookConfig {
  url: string;
  events: string[];
  secret?: string;
  active?: boolean;
}

export interface CreateSessionRequest {
  name: string;
  session_id?: string;
  api_key?: string;
  webhook?: WebhookConfig;
  proxy?: ProxyConfig;
}

export interface UpdateSessionRequest {
  name?: string;
  webhook?: WebhookConfig;
  proxy?: ProxyConfig;
}

export interface SessionResponse {
  id: string;
  session_id: string;
  name: string;
  api_key: string;
  status: SessionStatus;
  jid?: string;
  webhook?: string;
  proxy?: string;
  events?: string;
  connected: boolean;
  last_seen?: string;
  created_at: string;
  updated_at: string;
}

export interface SessionListResponse {
  sessions: SessionResponse[];
  total: number;
}

export interface SessionStatusResponse {
  session_id: string;
  status: SessionStatus;
  connected: boolean;
  jid?: string;
  last_seen?: string;
  connection_at?: string;
  battery_level?: number;
  is_charging?: boolean;
}

export interface SessionStatsResponse {
  session_id: string;
  messages_sent: number;
  messages_received: number;
  messages_failed: number;
  uptime_seconds: number;
  last_activity?: string;
}

// QR Code Types
export interface QRCodeResponse {
  session_id: string;
  qr_code: string;
  qr_data?: string;
  expires_at?: number;
  timeout?: number;
  timestamp?: string;
  status: string;
}

// Authentication Types
export interface AuthContext {
  api_key: string;
  is_global_key: boolean;
  session_id?: string;
  has_global_access: boolean;
}

export interface AuthValidationRequest {
  api_key: string;
}

export interface AuthValidationResponse {
  valid: boolean;
  type: 'global' | 'session';
  session_id?: string;
  expires_at?: string;
}

// Message Types
export interface MessageMedia {
  url?: string;
  data?: string;
  mimetype?: string;
  filename?: string;
  caption?: string;
}

export interface MessageLocation {
  latitude: number;
  longitude: number;
  name?: string;
  address?: string;
}

export interface MessageContact {
  name: string;
  phone: string;
  organization?: string;
}

export interface SendTextRequest {
  to: string;
  text: string;
  message_id?: string;
}

export interface SendMediaRequest {
  to: string;
  media: MessageMedia;
  caption?: string;
  message_id?: string;
}

export interface SendLocationRequest {
  to: string;
  location: MessageLocation;
  message_id?: string;
}

export interface SendMessageResponse {
  message_id: string;
  session_id: string;
  to: string;
  type: string;
  status: string;
  sent_at: string;
  delivered_at?: string;
  read_at?: string;
}

// Webhook Types
export interface WebhookRequest {
  url: string;
  method: 'POST' | 'PUT' | 'PATCH';
  headers?: Record<string, string>;
  payload: Record<string, any>;
}

export interface WebhookResponse {
  id: string;
  url: string;
  method: string;
  status_code: number;
  response?: string;
  headers?: Record<string, string>;
  duration_ms: number;
  success: boolean;
  error?: string;
  attempts: number;
  sent_at: string;
  completed_at: string;
}

// API Client Types
export interface ApiClientConfig {
  baseURL: string;
  apiKey: string;
  timeout?: number;
}

export interface ApiError extends Error {
  status?: number;
  code?: string;
  response?: ErrorResponse;
}

// Health Check
export interface HealthResponse {
  status: string;
  service: string;
  version: string;
  timestamp: string;
}

// Contact Types
export interface ContactInfo {
  jid: string;
  name?: string;
  notify?: string;
  business_name?: string;
  is_business?: boolean;
  is_enterprise?: boolean;
  verified_name?: string;
}

export interface CheckContactRequest {
  phones: string[];
}

export interface ContactInfoRequest {
  phone: string;
}

export interface ContactAvatarRequest {
  phone: string;
}

// Group Types
export interface CreateGroupRequest {
  name: string;
  participants: string[];
  description?: string;
}

export interface GroupInfoRequest {
  group_id: string;
}

// Configuration Types
export interface ProxyConfigRequest {
  enabled: boolean;
  type?: 'http' | 'https' | 'socks5';
  host?: string;
  port?: number;
  username?: string;
  password?: string;
}

export interface PairPhoneRequest {
  phone: string;
}

export interface PairPhoneResponse {
  success: boolean;
  phone: string;
  code?: string;
  message: string;
  expires_at?: string;
}
