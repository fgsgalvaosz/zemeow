// ============================================================================
// API CLIENT - ZeMeow WhatsApp API
// ============================================================================

import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import {
  BaseResponse,
  ErrorResponse,
  SessionResponse,
  CreateSessionRequest,
  UpdateSessionRequest,
  QRCodeResponse,
  PaginationResponse,
  SessionListResponse,
  AuthValidationResponse,
  ApiError,
  HealthResponse,
} from '@/types';

// API Client Configuration
export interface ApiClientConfig {
  baseURL?: string;
  apiKey?: string;
  timeout?: number;
}

// Default configuration
const DEFAULT_CONFIG: ApiClientConfig = {
  baseURL: typeof window !== 'undefined' 
    ? `${window.location.protocol}//${window.location.host}`
    : 'http://localhost:8080',
  timeout: 30000,
};

class ZeMeowApiClient {
  private client: AxiosInstance;
  private apiKey: string | null = null;

  constructor(config: ApiClientConfig = {}) {
    const finalConfig = { ...DEFAULT_CONFIG, ...config };
    
    this.client = axios.create({
      baseURL: finalConfig.baseURL,
      timeout: finalConfig.timeout,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Set API key if provided
    if (finalConfig.apiKey) {
      this.setApiKey(finalConfig.apiKey);
    }

    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        // Add API key to headers
        if (this.apiKey) {
          config.headers['X-API-Key'] = this.apiKey;
        }

        // Log request in development
        if (process.env.NODE_ENV === 'development') {
          console.log(`🚀 API Request: ${config.method?.toUpperCase()} ${config.url}`);
        }

        return config;
      },
      (error) => {
        console.error('❌ Request Error:', error);
        return Promise.reject(error);
      }
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response: AxiosResponse) => {
        // Log response in development
        if (process.env.NODE_ENV === 'development') {
          console.log(`✅ API Response: ${response.status} ${response.config.url}`);
        }

        return response;
      },
      (error) => {
        const apiError = this.handleError(error);
        console.error('❌ API Error:', apiError);
        return Promise.reject(apiError);
      }
    );
  }

  // Set API Key
  setApiKey(apiKey: string): void {
    this.apiKey = apiKey;
  }

  // Clear API Key
  clearApiKey(): void {
    this.apiKey = null;
  }

  // Error handler
  private handleError(error: any): ApiError {
    const apiError = new Error() as ApiError;
    
    if (error.response) {
      // Server responded with error status
      const response = error.response.data as ErrorResponse;
      apiError.message = response.error || response.message || 'API Error';
      apiError.status = error.response.status;
      apiError.code = response.code;
      apiError.response = response;
    } else if (error.request) {
      // Request was made but no response received
      apiError.message = 'Network Error - No response from server';
      apiError.status = 0;
    } else {
      // Something else happened
      apiError.message = error.message || 'Unknown Error';
    }

    return apiError;
  }

  // Generic request method
  private async request<T>(config: AxiosRequestConfig): Promise<T> {
    try {
      const response = await this.client.request<BaseResponse<T>>(config);
      
      if (response.data.success) {
        return response.data.data as T;
      } else {
        throw new Error(response.data.error || response.data.message || 'API Error');
      }
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // Health check
  async health(): Promise<HealthResponse> {
    return this.request<HealthResponse>({
      method: 'GET',
      url: '/health',
    });
  }

  // Authentication
  async validateApiKey(apiKey: string): Promise<AuthValidationResponse> {
    try {
      console.log('🔑 Validating API key:', apiKey);

      // Use sessions endpoint to validate API key - direct axios call to avoid BaseResponse wrapper
      const response = await this.client.get('/sessions', {
        headers: {
          'Authorization': `Bearer ${apiKey}`,
        },
      });

      console.log('✅ API key validation successful:', response.status);

      // If we get a successful response (status 200), the API key is valid
      if (response.status === 200) {
        return {
          success: true,
          data: { valid: true, type: 'global' }
        };
      } else {
        return {
          success: false,
          data: { valid: false, type: 'global' }
        };
      }
    } catch (error) {
      console.log('❌ API key validation failed:', error);

      // If we get an error, the API key is invalid
      return {
        success: false,
        data: { valid: false, type: 'global' }
      };
    }
  }

  // Sessions API
  async getSessions(params?: {
    page?: number;
    limit?: number;
    status?: string;
    search?: string;
  }): Promise<PaginationResponse<SessionResponse>> {
    return this.request<PaginationResponse<SessionResponse>>({
      method: 'GET',
      url: '/sessions',
      params,
    });
  }

  async getSession(sessionId: string): Promise<SessionResponse> {
    return this.request<SessionResponse>({
      method: 'GET',
      url: `/sessions/${sessionId}`,
    });
  }

  async createSession(data: CreateSessionRequest): Promise<SessionResponse> {
    return this.request<SessionResponse>({
      method: 'POST',
      url: '/sessions/add',
      data,
    });
  }

  async updateSession(sessionId: string, data: UpdateSessionRequest): Promise<SessionResponse> {
    return this.request<SessionResponse>({
      method: 'PUT',
      url: `/sessions/${sessionId}`,
      data,
    });
  }

  async deleteSession(sessionId: string): Promise<void> {
    return this.request<void>({
      method: 'DELETE',
      url: `/sessions/${sessionId}`,
    });
  }

  async connectSession(sessionId: string): Promise<BaseResponse> {
    return this.request<BaseResponse>({
      method: 'POST',
      url: `/sessions/${sessionId}/connect`,
    });
  }

  async disconnectSession(sessionId: string): Promise<BaseResponse> {
    return this.request<BaseResponse>({
      method: 'POST',
      url: `/sessions/${sessionId}/disconnect`,
    });
  }

  async getSessionQR(sessionId: string): Promise<QRCodeResponse> {
    return this.request<QRCodeResponse>({
      method: 'GET',
      url: `/sessions/${sessionId}/qr`,
    });
  }

  async getSessionStatus(sessionId: string): Promise<any> {
    return this.request<any>({
      method: 'GET',
      url: `/sessions/${sessionId}/status`,
    });
  }

  // Messages API
  async sendTextMessage(sessionId: string, data: {
    to: string;
    text: string;
    message_id?: string;
  }): Promise<any> {
    return this.request<any>({
      method: 'POST',
      url: `/sessions/${sessionId}/send/text`,
      data,
    });
  }

  async sendMediaMessage(sessionId: string, data: {
    to: string;
    media: any;
    caption?: string;
    message_id?: string;
  }): Promise<any> {
    return this.request<any>({
      method: 'POST',
      url: `/sessions/${sessionId}/send/media`,
      data,
    });
  }

  // Webhooks API
  async getWebhook(sessionId: string): Promise<any> {
    return this.request<any>({
      method: 'GET',
      url: `/webhooks/sessions/${sessionId}/find`,
    });
  }

  async setWebhook(sessionId: string, data: {
    url: string;
    events: string[];
    secret?: string;
  }): Promise<any> {
    return this.request<any>({
      method: 'POST',
      url: `/webhooks/sessions/${sessionId}/set`,
      data,
    });
  }

  // Contacts API
  async checkContacts(sessionId: string, phones: string[]): Promise<any> {
    return this.request<any>({
      method: 'POST',
      url: `/sessions/${sessionId}/contacts/check`,
      data: { phones },
    });
  }

  async getContactInfo(sessionId: string, phone: string): Promise<any> {
    return this.request<any>({
      method: 'GET',
      url: `/sessions/${sessionId}/contacts/info`,
      params: { phone },
    });
  }

  // Groups API
  async createGroup(sessionId: string, data: {
    name: string;
    participants: string[];
    description?: string;
  }): Promise<any> {
    return this.request<any>({
      method: 'POST',
      url: `/sessions/${sessionId}/groups/create`,
      data,
    });
  }

  async getGroupInfo(sessionId: string, groupId: string): Promise<any> {
    return this.request<any>({
      method: 'GET',
      url: `/sessions/${sessionId}/groups/info`,
      params: { group_id: groupId },
    });
  }
}

// Create singleton instance
export const apiClient = new ZeMeowApiClient();

// Export class for custom instances
export { ZeMeowApiClient };

// Export default instance
export default apiClient;
