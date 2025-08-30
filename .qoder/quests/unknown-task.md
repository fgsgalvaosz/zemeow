# Design: Sistema de API Multisessão WhatsApp - Análise Completa

## Overview

O **zemeow** é um sistema backend avançado em Go para gerenciamento de múltiplas sessões do WhatsApp, utilizando a biblioteca `go.mau.fi/whatsmeow`. O sistema oferece uma arquitetura completa com API REST, gerenciamento de sessões, autenticação robusta, configurações de proxy, webhooks e persistência em PostgreSQL.

### Objetivos Principais
- Gerenciar múltiplas sessões WhatsApp independentes de forma thread-safe
- Fornecer API REST completa com autenticação baseada em tokens
- Implementar sistema de proxy por sessão (HTTP/SOCKS5)
- Suporte completo a webhooks com processamento assíncrono
- Persistir dados de sessão com alta disponibilidade
- Configuração flexível através de variáveis de ambiente

### Stack Tecnológica Completa
- **Framework Web**: Fiber v2 (HTTP server com alta performance)
- **Banco de Dados**: PostgreSQL com database/sql
- **WhatsApp Integration**: go.mau.fi/whatsmeow com sqlstore
- **Logging**: Zerolog estruturado
- **Configuration**: godotenv para variáveis de ambiente
- **UUID**: Geração de identificadores únicos
- **QR Code**: go-qrcode para geração de QR codes

## Architecture

### Estrutura de Projetos Real

```mermaid
graph TB
    subgraph "API Layer (/internal/api)"
        A[server.go] --> B[handlers/]
        B --> C[middleware/]
        B --> D[session.go]
        B --> E[auth.go]
        B --> F[message.go]
        B --> G[webhook.go]
        C --> H[auth.go]
        C --> I[logging.go]
    end
    
    subgraph "Service Layer (/internal/service)"
        D --> J[session/manager.go]
        J --> K[meow/manager.go]
        K --> L[meow/client.go]
        J --> M[session/cache.go]
        J --> N[session/lifecycle.go]
        E --> O[auth/service.go]
        G --> P[webhook/service.go]
    end
    
    subgraph "Data Layer (/internal/db)"
        J --> Q[repositories/session_repository.go]
        Q --> R[models/session.go]
        Q --> S[db.go]
        S --> T[migrations/migrations.go]
    end
    
    subgraph "Configuration (/internal)"
        A --> U[config/config.go]
        A --> V[logger/logger.go]
    end
    
    subgraph "External Integrations"
        K --> W[WhatsApp Servers]
        P --> X[Webhook Endpoints]
        S --> Y[PostgreSQL]
    end
```

### Componentes Principais

#### 1. SessionManager (Core Component)
- **Responsabilidade**: Gerenciar múltiplos clientes WhatsApp de forma thread-safe
- **Funcionalidades**:
  - Manter mapa de sessões ativas em memória
  - Controlar lifecycle de sessões (start/stop/cleanup)
  - Sincronizar estado entre memória e banco de dados

#### 2. WhatsApp Client Integration
- **Biblioteca**: `go.mau.fi/whatsmeow`
- **Store**: `go.mau.fi/whatsmeow/store/sqlstore` para PostgreSQL
- **Funcionalidades**:
  - Cliente WhatsApp por sessão
  - Event handlers customizados
  - Gerenciamento de autenticação (QR Code, Phone Pairing)

#### 3. Data Persistence Layer
- **Banco Principal**: PostgreSQL com migrations
- **Cache**: In-memory para performance
- **Repository Pattern**: Abstração de acesso aos dados

## API Endpoints Reference

### Session Management Endpoints

| Método | Endpoint | Descrição | Autenticação |
|--------|----------|-----------|--------------|
| POST | `/sessions` | Criar nova sessão | Admin Token |
| GET | `/sessions` | Listar sessões com filtros | Admin Token |
| GET | `/sessions/{sessionID}` | Detalhes da sessão | Session Token |
| DELETE | `/sessions/{sessionID}` | Remover sessão | Session Token |

### WhatsApp Connection Endpoints

| Método | Endpoint | Descrição | Autenticação |
|--------|----------|-----------|--------------|
| POST | `/sessions/{sessionID}/connect` | Conectar sessão WhatsApp | Session Token |
| POST | `/sessions/{sessionID}/disconnect` | Desconectar sessão WhatsApp | Session Token |
| GET | `/sessions/{sessionID}/status` | Status da sessão | Session Token |
| GET | `/sessions/{sessionID}/stats` | Estatísticas da sessão | Session Token |
| GET | `/sessions/{sessionID}/qr` | Obter QR Code para login | Session Token |
| POST | `/sessions/{sessionID}/pairphone` | Autenticação por telefone | Session Token |

### Request/Response Schema

#### POST /sessions
```json
{
  "name": "string",
  "api_key": "string (optional - se não informado será gerado automaticamente)",
  "webhook_url": "string (optional)",
  "proxy": {
    "enabled": "boolean",
    "type": "http|socks5",
    "host": "string",
    "port": "integer", 
    "username": "string (optional)",
    "password": "string (optional)"
  }
}
```

#### Response
```json
{
  "sessionID": "string",
  "api_key": "string (gerada ou informada)",
  "status": "created",
  "createdAt": "datetime"
}
```

#### GET /sessions/{sessionID}
```json
{
  "sessionID": "string",
  "name": "string",
  "status": "connected|disconnected|authenticating",
  "whatsappJID": "string",
  "connectedAt": "datetime",
  "lastActivity": "datetime",
  "proxy": {
    "enabled": "boolean",
    "status": "active|inactive"
  },
  "webhook": {
    "url": "string",
    "last_sent": "datetime",
    "status": "active|failed"
  }
}
```

### Authentication Requirements

#### Sistema de API Key Simplificado
- **Header**: `Authorization: Bearer {API_KEY}` ou `X-API-Key: {API_KEY}`
- **Tipos de API Key**:
  - **Global Admin Key**: Definida em variável de ambiente, acesso total
  - **Session API Key**: Gerada automaticamente ou informada pelo usuário na criação da sessão
- **Funcionamento**:
  - Admin pode criar sessões e gerenciar o sistema
  - Cada sessão tem sua própria API key para acesso restrito
  - API key da sessão é gerada automaticamente se não informada
- **Validação**: Middleware único que verifica admin key ou session key

## Data Models & ORM Mapping

### Session Model (Real Implementation)
```go
type Session struct {
    ID              uuid.UUID      `json:"id" db:"id"`
    SessionID       string         `json:"session_id" db:"session_id"`
    Name            string         `json:"name" db:"name"`
    APIKey          string         `json:"api_key" db:"api_key"`
    JID             *string        `json:"jid,omitempty" db:"jid"`
    Status          SessionStatus  `json:"status" db:"status"`
    ProxyEnabled    bool           `json:"proxy_enabled" db:"proxy_enabled"`
    ProxyHost       *string        `json:"proxy_host,omitempty" db:"proxy_host"`
    ProxyPort       *int           `json:"proxy_port,omitempty" db:"proxy_port"`
    ProxyUsername   *string        `json:"proxy_username,omitempty" db:"proxy_username"`
    ProxyPassword   *string        `json:"proxy_password,omitempty" db:"proxy_password"`
    WebhookURL      *string        `json:"webhook_url,omitempty" db:"webhook_url"`
    WebhookEvents   pq.StringArray `json:"webhook_events" db:"webhook_events"`
    CreatedAt       time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
    LastConnectedAt *time.Time     `json:"last_connected_at,omitempty" db:"last_connected_at"`
    Metadata        Metadata       `json:"metadata" db:"metadata"`
}
```

### Session Status Enum
```go
type SessionStatus string

const (
    SessionStatusDisconnected SessionStatus = "disconnected"
    SessionStatusConnecting   SessionStatus = "connecting"
    SessionStatusConnected    SessionStatus = "connected"
    SessionStatusAuthenticated SessionStatus = "authenticated"
    SessionStatusError        SessionStatus = "error"
)
```

### Configuration Models
```go
type SessionConfig struct {
    SessionID     string        `json:"session_id"`
    Name          string        `json:"name"`
    Proxy         *ProxyConfig  `json:"proxy,omitempty"`
    Webhook       *WebhookConfig `json:"webhook,omitempty"`
    AutoReconnect bool          `json:"auto_reconnect"`
    Timeout       time.Duration `json:"timeout"`
}

type ProxyConfig struct {
    Enabled  bool   `json:"enabled"`
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`
    Type     string `json:"type"` // http, socks5
}

type WebhookConfig struct {
    URL    string   `json:"url"`
    Events []string `json:"events"`
    Secret string   `json:"secret,omitempty"`
}
```

### Database Schema

```mermaid
erDiagram
    sessions {
        uuid id PK
        string session_id UK
        string name
        string api_key UK
        string jid
        string status
        boolean proxy_enabled
        string proxy_host
        integer proxy_port
        string proxy_username
        string proxy_password
        string webhook_url
        text_array webhook_events
        timestamp created_at
        timestamp updated_at
        timestamp last_connected_at
        jsonb metadata
    }
    
    whatsmeow_device_store {
        string jid PK
        bytea registration_id
        bytea identity_key
        bytea signed_pre_key
        bytea pre_key_bundle
    }
    
    whatsmeow_identity_keys {
        string our_jid
        string their_id
        bytea identity
    }
```

## Business Logic Layer

### Arquitetura de Serviços Detalhada

#### 1. AuthService Simplificado (internal/service/auth/service.go)
```go
type AuthService struct {
    adminAPIKey string
    repository  repositories.SessionRepository
    logger      zerolog.Logger
}
```

**Responsabilidades**:
- Validar admin API key global
- Validar session API keys contra banco de dados
- Gerar API keys automaticamente para novas sessões
- Controle de acesso simples (admin ou session)

**Métodos Principais**:
- `ValidateAPIKey()`: Validação de admin ou session key
- `GenerateSessionAPIKey()`: Geração automática de API key
- `ValidateSessionAccess()`: Verificação de acesso por sessão

#### 2. WebhookService (internal/service/webhook/service.go)
```go
type WebhookService struct {
    mu         sync.RWMutex
    client     *http.Client
    repository repositories.SessionRepository
    config     *config.Config
    logger     logger.Logger
    ctx        context.Context
    cancel     context.CancelFunc
    queue      chan WebhookPayload
    workers    int
}
```

**Responsabilidades**:
- Processamento assíncrono de webhooks via workers
- Queue com buffer de 10.000 eventos
- Pool de workers configuráveis (padrão: 5)
- Retry automático com backoff
- Filtragem de eventos por configuração de sessão

**Estrutura de Payload**:
```go
type WebhookPayload struct {
    SessionID string                 `json:"session_id"`
    Event     string                 `json:"event"`
    Data      interface{}            `json:"data,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
    Retries   int                    `json:"-"`
    URL       string                 `json:"-"`
}
```

#### 3. ProxyService (internal/service/proxy/proxy.go)
```go
type ProxyService struct {
    logger logger.Logger
}

type ProxyConfig struct {
    Enabled  bool   `json:"enabled"`
    Type     string `json:"type"`     // http, socks5
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`
}
```

**Responsabilidades**:
- Validação de configurações de proxy (HTTP/SOCKS5)
- Teste de conectividade de proxy
- Geração de http.Transport configurado
- Suporte a autenticação de proxy

#### 4. Session Service Interface (internal/service/session/service.go)
```go
type Service interface {
    // Gerenciamento de sessões
    CreateSession(ctx context.Context, config *Config) (*SessionInfo, error)
    GetSession(ctx context.Context, sessionID string) (*SessionInfo, error)
    ListSessions(ctx context.Context) ([]*SessionInfo, error)
    DeleteSession(ctx context.Context, sessionID string) error

    // Controle de conexão
    ConnectSession(ctx context.Context, sessionID string) error
    DisconnectSession(ctx context.Context, sessionID string) error
    GetSessionStatus(ctx context.Context, sessionID string) (Status, error)

    // Autenticação WhatsApp
    GetQRCode(ctx context.Context, sessionID string) (*QRCodeInfo, error)
    PairPhone(ctx context.Context, sessionID string, request *PairPhoneRequest) (*PairPhoneResponse, error)

    // Configurações
    SetProxy(ctx context.Context, sessionID string, proxy *ProxyConfig) error
    SetWebhook(ctx context.Context, sessionID string, webhook *WebhookConfig) error
}
```

### Configuração Completa do Sistema (internal/config/config.go)

```go
type Config struct {
    Database DatabaseConfig
    Server   ServerConfig
    Admin    AdminConfig
    WhatsApp WhatsAppConfig
    Logging  LoggingConfig
    Webhook  WebhookConfig
}

type DatabaseConfig struct {
    Host     string
    Port     int
    Name     string
    User     string
    Password string
    URL      string
}

type ServerConfig struct {
    Port        int
    Environment string
}

type AdminConfig struct {
    APIKey string
}

type WhatsAppConfig struct {
    QRCodeTimeout time.Duration
}

type LoggingConfig struct {
    Level  string
    Pretty bool
}

type WebhookConfig struct {
    Timeout    time.Duration
    RetryCount int
}
```

**Carregamento de Configurações Simplificado**:
- Variáveis de ambiente essenciais apenas
- Valores padrão para desenvolvimento
- Admin API Key global única
- Construção automática de URL do banco

### Repository Layer Detalhado (internal/db/repositories/session_repository.go)

```go
type SessionRepository interface {
    Create(session *models.Session) error
    GetByID(id uuid.UUID) (*models.Session, error)
    GetBySessionID(sessionID string) (*models.Session, error)
    GetByAPIKey(apiKey string) (*models.Session, error)
    GetAll(filter *models.SessionFilter) (*models.SessionListResponse, error)
    Update(session *models.Session) error
    UpdateStatus(sessionID string, status models.SessionStatus) error
    Delete(id uuid.UUID) error
    DeleteBySessionID(sessionID string) error
    Exists(sessionID string) (bool, error)
    Count() (int, error)
    GetActiveConnections() ([]*models.Session, error)
}
```

**Funcionalidades Implementadas**:
- CRUD completo com validação
- Busca por múltiplos índices (ID, SessionID, APIKey)
- Filtragem avançada com paginação
- Gerenciamento de conexões ativas
- Logs estruturados de operações

### API Server Architecture (internal/api/server.go)

```go
type Server struct {
    app            *fiber.App
    config         *config.Config
    logger         logger.Logger
    sessionHandler *handlers.SessionHandler
    webhookHandler *handlers.WebhookHandler
    authMiddleware *middleware.AuthMiddleware
}
```

**Rotas Implementadas**:
- `/sessions/*` - Gerenciamento de sessões
- `/webhooks/*` - Configuração e stats de webhooks
- `/health` - Health check

**Middleware Stack**:
1. Recovery middleware (panic recovery)
2. CORS middleware
3. Request logging
4. API Key authentication
5. Rate limiting
6. Session access validation

### Session Lifecycle Management Detalhado

```mermaid
stateDiagram-v2
    [*] --> disconnected: Session Created
    disconnected --> connecting: POST /connect
    connecting --> connected: WhatsApp Connected
    connecting --> authenticated: QR Success / Phone Pair
    connecting --> error: Auth Timeout / Error
    connected --> authenticated: Login Success
    authenticated --> disconnected: POST /logout / Network Error
    error --> connecting: Retry Connection
    authenticated --> [*]: DELETE /sessions/{id}
    disconnected --> [*]: DELETE /sessions/{id}
```

### Fluxo de Eventos WhatsApp e Webhook

```mermaid
sequenceDiagram
    participant WA as WhatsApp Server
    participant MC as MyClient
    participant WM as WhatsAppManager
    participant WS as WebhookService
    participant W as Worker Pool
    participant EXT as External Webhook
    participant DB as Repository
    
    WA->>MC: WhatsApp Event (Message, Status, etc.)
    MC->>WM: Process Event via Handler
    WM->>DB: Update Session State
    WM->>WS: Queue Webhook Event (async)
    
    WS->>W: Distribute to Worker
    W->>EXT: HTTP POST Webhook
    alt Webhook Success
        EXT-->>W: 200 OK
        W->>DB: Log Success
    else Webhook Failed
        EXT-->>W: Error
        W->>W: Retry with Backoff
        W->>DB: Log Failure
    end
    
    Note over WS: Buffer: 10,000 events
    Note over W: 5 workers by default
    Note over W: Configurable retry count
```

### Fluxo de API Key Simplificado

```mermaid
sequenceDiagram
    participant U as User/Admin
    participant API as API Server
    participant AUTH as AuthMiddleware
    participant DB as Database
    
    Note over U, DB: Criar Nova Sessão
    U->>API: POST /sessions + Admin API Key
    API->>AUTH: Validate Admin Key
    AUTH->>API: Admin Validated
    
    alt API Key fornecida pelo usuário
        API->>DB: Save Session + Custom API Key
    else API Key não fornecida
        API->>API: Generate UUID API Key
        API->>DB: Save Session + Generated API Key
    end
    
    API-->>U: Return {sessionID, api_key}
    
    Note over U, DB: Usar Sessão
    U->>API: GET /sessions/{id} + Session API Key
    API->>AUTH: Validate Session Key
    AUTH->>DB: Check Session API Key
    DB-->>AUTH: Session Found
    AUTH->>API: Session Validated
    API-->>U: Session Data
```

## Middleware & Interceptors

### Authentication Architecture (Real Implementation)

```mermaid
graph LR
    A[HTTP Request] --> B[CORS Middleware]
    B --> C[Rate Limiting]
    C --> D[Request Logging]
    D --> E[Auth Middleware]
    E --> F[Session Context]
    F --> G[Route Handler]
    G --> H[Response Logging]
    H --> I[Response]
```

### Authentication Context Structure Simplificada
```go
type AuthContext struct {
    IsAdmin   bool
    SessionID string
    APIKey    string
}

type SessionInfo struct {
    ID        string `json:"id"`
    SessionID string `json:"session_id"`
    Name      string `json:"name"`
    APIKey    string `json:"api_key"`
    JID       string `json:"jid"`
    Status    string `json:"status"`
}
```

### Middleware Components Simplificados

#### 1. API Key Middleware Único
- **Função**: Validar admin API key ou session API key
- **Suporte**: Header Authorization ou X-API-Key
- **Context**: Injeta AuthContext no contexto Fiber
- **Lógica**: 
  - Se API key = admin key → acesso total
  - Se API key = session key → acesso restrito à sessão
  - Caso contrário → 401 Unauthorized

#### 2. Request Logging Middleware
- **Função**: Log estruturado de requisições/respostas
- **Dados**: Request ID, API key type, duration, status codes
- **Logger**: Integrado com zerolog

## Implementação Detalhada por Módulos

### 1. Módulo de Configuração (internal/config)

#### Variáveis de Ambiente Simplificadas
```bash
# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=zemeow
POSTGRES_USER=zemeow
POSTGRES_PASSWORD=zemeow123
DATABASE_URL=postgres://...

# Server Configuration
SERVER_PORT=8080
ENVIRONMENT=development

# Global Admin API Key
ADMIN_API_KEY=admin_secret_key_here

# WhatsApp Configuration
QR_CODE_TIMEOUT=60s

# Logging Configuration
LOG_LEVEL=info
LOG_PRETTY=true

# Webhook Configuration (optional)
WEBHOOK_TIMEOUT=10s
WEBHOOK_RETRY_COUNT=3
```

### 2. Módulo de Migração de Banco (internal/db/migrations)

#### Schema de Tabela Sessions
```sql
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) UNIQUE NOT NULL,
    jid VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    proxy_enabled BOOLEAN DEFAULT FALSE,
    proxy_host VARCHAR(255),
    proxy_port INTEGER,
    proxy_username VARCHAR(255),
    proxy_password VARCHAR(255),
    webhook_url TEXT,
    webhook_events TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_connected_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'
);

-- Índices para performance
CREATE INDEX idx_sessions_session_id ON sessions(session_id);
CREATE INDEX idx_sessions_api_key ON sessions(api_key);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_created_at ON sessions(created_at);
CREATE INDEX idx_sessions_jid ON sessions(jid);
```

### 3. Estrutura de Handlers Completa

#### SessionHandler Endpoints Detalhados
```go
// Admin Only Endpoints
// POST /sessions - Criar nova sessão
// GET  /sessions - Listar todas as sessões
// DELETE /sessions/:sessionId - Deletar sessão

// Session Authenticated Endpoints
// GET  /sessions/:sessionId - Detalhes da sessão
// PUT  /sessions/:sessionId - Atualizar sessão
// POST /sessions/:sessionId/connect - Conectar WhatsApp
// POST /sessions/:sessionId/disconnect - Desconectar
// GET  /sessions/:sessionId/status - Status da sessão
// GET  /sessions/:sessionId/qr - QR Code
// POST /sessions/:sessionId/pairphone - Pareamento por telefone
// GET  /sessions/:sessionId/stats - Estatísticas

// Proxy Endpoints
// POST /sessions/:sessionId/proxy - Configurar proxy
// GET  /sessions/:sessionId/proxy - Obter config proxy
// POST /sessions/:sessionId/proxy/test - Testar proxy
```

#### WebhookHandler Endpoints
```go
// Admin Only Endpoints
// POST /webhooks/send - Enviar webhook manual
// GET  /webhooks/stats - Estatísticas globais

// Session Authenticated Endpoints
// GET /webhooks/sessions/:sessionId/stats - Stats da sessão
```

### 4. Integração WhatsApp (internal/service/meow)

#### WhatsAppManager Funcionalidades
```go
type WhatsAppManager struct {
    mu          sync.RWMutex
    clients     map[string]*MyClient     // sessionID -> client
    sessions    map[string]*models.Session
    container   *sqlstore.Container     // whatsmeow store
    repository  repositories.SessionRepository
    config      *config.Config
    logger      logger.Logger
    ctx         context.Context
    cancel      context.CancelFunc
    webhookChan chan WebhookEvent
}

// Métodos principais:
// - CreateSession() - Criar nova sessão WhatsApp
// - ConnectSession() - Conectar e gerar QR Code
// - DisconnectSession() - Desconectar sessão
// - DeleteSession() - Remover sessão completamente
// - GetConnectionInfo() - Info da conexão ativa
// - ListSessions() - Listar sessões gerenciadas
// - processEvents() - Processar eventos WhatsApp
```

#### MyClient Wrapper
```go
type MyClient struct {
    SessionID    string
    Client       *whatsmeow.Client
    EventHandler func(evt interface{})
    Log          waLog.Logger
    QRChan       chan string
    Connected    bool
    JID          string
    Store        *sqlstore.Container
}

// Eventos WhatsApp suportados:
// - events.Message - Mensagens recebidas
// - events.Connected - Conexão estabelecida
// - events.Disconnected - Desconexão
// - events.QR - QR Code gerado
// - events.PairSuccess - Pareamento bem-sucedido
// - events.LoggedOut - Logout do WhatsApp
```

### 5. Sistema de Cache (internal/service/session/cache.go)

#### SessionCache Implementation
```go
type SessionCache struct {
    mu      sync.RWMutex
    cache   map[string]*CacheEntry
    ttl     time.Duration
    ticker  *time.Ticker
    stop    chan struct{}
}

type CacheEntry struct {
    Session   *models.Session
    ExpiresAt time.Time
    Status    models.SessionStatus
}

// Funcionalidades:
// - Set() - Adicionar/atualizar sessão no cache
// - Get() - Buscar sessão do cache
// - Delete() - Remover sessão do cache
// - List() - Listar todas as sessões em cache
// - Cleanup() - Limpeza automática de entradas expiradas
// - Stats() - Estatísticas do cache
```

### 6. Lifecycle Management (internal/service/session/lifecycle.go)

#### Gerenciamento de Ciclo de Vida
```go
type LifecycleManager struct {
    mu        sync.RWMutex
    events    chan LifecycleEvent
    handlers  map[EventType][]LifecycleHandler
    ctx       context.Context
    cancel    context.CancelFunc
}

type LifecycleEvent struct {
    SessionID string
    Type      EventType
    Data      interface{}
    Timestamp time.Time
}

const (
    EventSessionCreated    EventType = "session.created"
    EventSessionConnected  EventType = "session.connected"
    EventSessionDisconnected EventType = "session.disconnected"
    EventSessionDeleted    EventType = "session.deleted"
    EventSessionError      EventType = "session.error"
)

// Funcionalidades:
// - RegisterHandler() - Registrar handler de evento
// - EmitEvent() - Emitir evento de lifecycle
// - Start() - Iniciar processamento de eventos
// - Stop() - Parar processamento
```

## Fluxos de Implementação Prioritários

### Fase 1: Infraestrutura Base

#### 1.1 Configuração e Banco de Dados
- **Prioridade**: Crítica
- **Implementação**: 
  - Finalizar `internal/config/config.go` com todas as variáveis
  - Implementar migrations em `internal/db/migrations/migrations.go`
  - Configurar pool de conexões PostgreSQL
  - Implementar health checks de banco

#### 1.2 Repository Layer Completo
- **Prioridade**: Crítica
- **Implementação**:
  - Finalizar todos os métodos do SessionRepository
  - Implementar filtragem avançada com paginação
  - Adicionar validação de dados e constraints
  - Implementar transações para operações complexas

#### 1.3 Sistema de Logging
- **Prioridade**: Alta
- **Implementação**:
  - Configurar zerolog com níveis apropriados
  - Implementar log rotation
  - Adicionar correlation IDs para rastreamento
  - Integrar logs estruturados em todos os componentes

### Fase 2: Core Services

#### 2.1 AuthService Simplificado
- **Prioridade**: Crítica
- **Implementação**:
  - Validar admin API key das variáveis de ambiente
  - Validar session API keys do banco de dados
  - Gerar API keys automaticamente (UUID v4)
  - Middleware único para autenticação

#### 2.2 Session Manager Simplificado
- **Prioridade**: Crítica
- **Implementação**:
  - CRUD básico de sessões com PostgreSQL
  - Geração automática de session ID e API key
  - Recuperação de sessões após restart
  - Thread-safety em operações críticas

#### 2.3 WebhookService
- **Prioridade**: Alta
- **Implementação**:
  - Implementar worker pool configurável
  - Adicionar retry com backoff exponencial
  - Implementar circuit breaker para endpoints falhos
  - Adicionar métricas de webhook (success/failure rates)

### Fase 3: Integração WhatsApp

#### 3.1 WhatsAppManager Base
- **Prioridade**: Crítica
- **Implementação**:
  - Configurar sqlstore para persistência
  - Implementar MyClient wrapper
  - Adicionar event handlers para todos os tipos de evento
  - Implementar reconexão automática

#### 3.2 Autenticação WhatsApp
- **Prioridade**: Alta
- **Implementação**:
  - Geração e exposição de QR Code
  - Implementar pair by phone
  - Adicionar timeout para processos de autenticação
  - Implementar validação de device store

#### 3.3 Gerenciamento de Eventos
- **Prioridade**: Média
- **Implementação**:
  - Processar eventos de mensagem
  - Implementar eventos de status de conexão
  - Adicionar eventos de grupo e contatos
  - Implementar filtragem de eventos por sessão

### Fase 4: API Endpoints

#### 4.1 Session Management Endpoints
- **Prioridade**: Crítica
- **Implementação**:
  - `POST /sessions` - Criar nova sessão
  - `GET /sessions` - Listar sessões
  - `GET /sessions/{id}` - Detalhes da sessão
  - `DELETE /sessions/{id}` - Remover sessão

#### 4.2 WhatsApp Connection Endpoints
- **Prioridade**: Alta
- **Implementação**:
  - Endpoints de controle de conexão WhatsApp
  - Endpoints de autenticação (QR, phone pairing)
  - Endpoints de status e estatísticas

#### 4.3 Webhook Management Endpoints
- **Prioridade**: Média
- **Implementação**:
  - Endpoints admin para gerenciamento do serviço
  - Endpoints de estatísticas e monitoramento
  - Endpoints para configuração por sessão

### Fase 5: Funcionalidades Avançadas

#### 5.1 Sistema de Proxy
- **Prioridade**: Média
- **Implementação**:
  - Suporte completo a HTTP e SOCKS5 proxy
  - Validação e teste de conectividade
  - Configuração de proxy por sessão
  - Monitoramento de saúde do proxy

#### 5.2 Monitoramento e Métricas
- **Prioridade**: Média
- **Implementação**:
  - Métricas de performance (Prometheus)
  - Health checks avançados
  - Alertas de sistema
  - Dashboard de monitoramento

#### 5.3 Segurança Avançada
- **Prioridade**: Alta
- **Implementação**:
  - Rate limiting avançado
  - Audit logging completo
  - Encryptação de dados sensíveis
  - Validação de entrada robusta

## Considerações de Deploy e Operação

### Docker Setup

#### Dockerfile Multi-stage
```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o zemeow cmd/zemeow/main.go

# Production stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/zemeow .
EXPOSE 8080
CMD ["./zemeow"]
```

#### Docker Compose Completo
```yaml
version: '3.8'
services:
  zemeow:
    build: .
    ports:
      - "8080:8080"
    environment:
      - POSTGRES_HOST=postgres
      - POSTGRES_DB=zemeow
      - POSTGRES_USER=zemeow
      - POSTGRES_PASSWORD=zemeow123
      - ADMIN_TOKEN=your_admin_token_here
    depends_on:
      - postgres
    restart: unless-stopped

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=zemeow
      - POSTGRES_USER=zemeow
      - POSTGRES_PASSWORD=zemeow123
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./scripts/init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "5432:5432"
    restart: unless-stopped

volumes:
  postgres_data:
```

### Variáveis de Ambiente para Produção

```bash
# Database
DATABASE_URL=postgres://user:pass@host:5432/zemeow?sslmode=require

# Server
ENVIRONMENT=production
SERVER_PORT=8080

# Security
ADMIN_API_KEY=your_super_secure_admin_key_here

# Logging
LOG_LEVEL=info
LOG_PRETTY=false

# Performance
WEBHOOK_TIMEOUT=15s
WEBHOOK_RETRY_COUNT=5
QR_CODE_TIMEOUT=120s
```

### Monitoramento de Saúde

#### Health Check Endpoint
```go
// GET /health
{
  "status": "ok",
  "timestamp": 1672531200,
  "version": "1.0.0",
  "checks": {
    "database": "ok",
    "whatsapp_manager": "ok", 
    "webhook_service": "ok",
    "active_sessions": 15
  }
}
```

#### Readiness Check
```go
// GET /ready
{
  "ready": true,
  "services": {
    "database": true,
    "migrations": true,
    "whatsapp_store": true
  }
}
```

### Backup e Recuperação

#### Backup de Sessões
- Backup automático do PostgreSQL
- Export/import de configurações de sessão
- Backup de device stores do WhatsApp
- Procedimentos de disaster recovery

#### Recuperação de Sessões
- Reconexão automática após restart
- Restauração de estado de sessões
- Reinitialization de webhooks
- Validação de integridade de dados

