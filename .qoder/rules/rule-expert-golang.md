---
trigger: always_on
alwaysApply: true
---

# Regra de Especialista em Go para ZeMeow

Você é um especialista em Go trabalhando no projeto ZeMeow, uma API REST para integração com WhatsApp baseada na biblioteca `whatsmeow`. Siga rigorosamente estas diretrizes:

## 🏗️ Estrutura e Organização

### Estrutura de Pacotes (REAL DO PROJETO)
- **Handlers**: `internal/handlers/` - Lógica HTTP (Fiber)
- **Services**: `internal/services/` - Lógica de negócio (session, webhook, media, meow)
- **Models**: `internal/models/` - Estruturas de dados
- **Repositories**: `internal/repositories/` - Acesso a dados
- **DTOs**: `internal/dto/` - Transfer Objects
- **Utils**: `internal/handlers/utils/` - Utilitários HTTP
- **Middleware**: `internal/middleware/` - Autenticação, validação, logging
- **Routers**: `internal/routers/` - Configuração de rotas
- **Config**: `internal/config/` - Configurações da aplicação
- **Database**: `internal/database/` - Conexão e migrações
- **Logger**: `internal/logger/` - Sistema de logging avançado

### Imports (PADRÃO REAL)
```go
// Padrão de imports observado no projeto:
// 1. Stdlib primeiro
// 2. Dependências externas (whatsmeow, fiber, etc.)
// 3. Pacotes internos do projeto

import (
    "context"
    "fmt"
    "time"
    
    "github.com/gofiber/fiber/v2"
    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/types"
    
    "github.com/felipe/zemeow/internal/dto"
    "github.com/felipe/zemeow/internal/handlers/utils"
    "github.com/felipe/zemeow/internal/logger"
    "github.com/felipe/zemeow/internal/repositories"
    "github.com/felipe/zemeow/internal/services/session"
)
```

## 📋 Convenções de Código

### Nomenclatura
- **Structs**: PascalCase (`SessionHandler`, `CreateSessionRequest`)
- **Interfaces**: PascalCase com sufixo apropriado (`Service`, `Repository`)
- **Métodos**: PascalCase (`CreateSession`, `GetSession`)
- **Variáveis**: camelCase (`sessionID`, `apiKey`)
- **Constantes**: PascalCase com prefixo (`SessionStatusConnected`)
- **Packages**: lowercase simples (`session`, `handlers`, `dto`)

### Estruturas de Handler (PADRÃO REAL)
```go
// Template padrão para handlers (IMPLEMENTAÇÃO REAL)
type SessionHandler struct {
    sessionService session.Service
    sessionRepo    repositories.SessionRepository
    logger         logger.Logger
}

func NewSessionHandler(service session.Service, repo repositories.SessionRepository) *SessionHandler {
    return &SessionHandler{
        sessionService: service,
        sessionRepo:    repo,
        logger:         logger.GetWithSession("session_handler"),
    }
}

// Outros handlers seguem o mesmo padrão:
// WebhookHandler, GroupHandler, MessageHandler, MediaHandler
```

### Tratamento de Erros
```go
// SEMPRE usar o padrão estabelecido:
if err != nil {
    h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get session")
    return utils.SendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
}
```

## 🔧 Padrões Específicos do Projeto

### Sistema de Logger (IMPLEMENTAÇÃO REAL)
```go
// TIPOS DE LOGGER DISPONÍVEIS:
// 1. logger.Logger (interface básica)
// 2. logger.ComponentLogger (com contexto de componente)
// 3. logger.OperationLogger (rastreamento de operações com duração)
// 4. logger.RequestLogger (contexto completo de requisição)

// Inicialização nos handlers:
logger := logger.GetWithSession("component_name")

// Para services:
logger := logger.ForComponent("session")

// Padrão de logging estruturado (OBRIGATÓRIO):
logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get session")
logger.Info().Str("operation", "create_session").Msg("Session created successfully")

// Para operações com duração:
opLogger := logger.ForComponent("session").ForOperation("create_session")
opLogger.Starting().Msg("Creating session")
// ... operação ...
opLogger.Success().Msg("Session created successfully")
```

### Sistema de Middleware de Autenticação (NOVO)
```go
// TIPOS DE MIDDLEWARE DISPONÍVEIS:
r.authMiddleware.RequireAPIKey()           // Para API keys de sessão
r.authMiddleware.RequireGlobalAPIKey()     // Para admin API key  
r.validationMiddleware.ValidateSessionAccess()  // Validação de acesso

// Extração do contexto de autenticação:
auth := middleware.GetAuthContext(c)
if auth.IsGlobalKey {
    // Acesso global (admin)
} else {
    // Acesso específico da sessão
    sessionID := auth.SessionID
}
```

### Sistema de Validação (IMPLEMENTAÇÃO REAL)
```go
// Middleware de validação nas rotas:
r.validationMiddleware.ValidateJSON(&dto.CreateSessionRequest{})
r.validationMiddleware.ValidateParams()
r.validationMiddleware.ValidatePaginationParams()

// Extração no handler (PADRÃO OBRIGATÓRIO):
validatedBody := c.Locals("validated_body")
if validatedBody == nil {
    return utils.SendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
}

req, ok := validatedBody.(*dto.CreateSessionRequest)
if !ok {
    return utils.SendError(c, "Invalid request format", "INVALID_REQUEST", fiber.StatusBadRequest)
}
```

### Respostas HTTP Padronizadas (IMPLEMENTAÇÃO REAL)
```go
// FUNÇÕES UTILITÁRIAS OBRIGATÓRIAS:
utils.SendError(c, message, code, status)
utils.SendSuccess(c, data, message)
utils.SendValidationError(c, message)
utils.SendAuthenticationError(c, message)
utils.SendAuthorizationError(c, message)
utils.SendNotFoundError(c, message)
utils.SendInternalError(c, message)
utils.SendAccessDeniedError(c)
utils.SendInvalidJSONError(c)

// Estrutura de resposta padronizada:
type ErrorResponse struct {
    Success   bool                   `json:"success"`
    Error     ErrorDetails           `json:"error"`
    Timestamp int64                  `json:"timestamp"`
    Meta      map[string]interface{} `json:"meta,omitempty"`
}

type SuccessResponse struct {
    Success   bool                   `json:"success"`
    Data      interface{}            `json:"data,omitempty"`
    Message   string                 `json:"message,omitempty"`
    Timestamp int64                  `json:"timestamp"`
    Meta      map[string]interface{} `json:"meta,omitempty"`
}

// Códigos de erro padronizados:
const (
    ErrCodeValidation      = "VALIDATION_ERROR"
    ErrCodeAuthentication  = "AUTHENTICATION_ERROR" 
    ErrCodeAuthorization   = "AUTHORIZATION_ERROR"
    ErrCodeNotFound        = "NOT_FOUND"
    ErrCodeInternalError   = "INTERNAL_ERROR"
    ErrCodeBadRequest      = "BAD_REQUEST"
    ErrCodeSessionNotReady = "SESSION_NOT_READY"
    ErrCodeAccessDenied    = "ACCESS_DENIED"
    ErrCodeInvalidJSON     = "INVALID_JSON"
    ErrCodeSendFailed      = "SEND_FAILED"
)
```

### Configuração de Rotas (PADRÃO REAL)
```go
// Estrutura do Router:
type Router struct {
    app                  *fiber.App
    authMiddleware       *middleware.AuthMiddleware
    validationMiddleware *middleware.ValidationMiddleware
    sessionHandler       *handlers.SessionHandler
    messageHandler       *handlers.MessageHandler
    webhookHandler       *handlers.WebhookHandler
    groupHandler         *handlers.GroupHandler
    mediaHandler         *handlers.MediaHandler
}

// Agrupamento de rotas com middleware:
sessions := r.app.Group("/sessions")

// Rotas globais (admin):
globalRoutes := sessions.Group("/", r.authMiddleware.RequireGlobalAPIKey())
globalRoutes.Post("/add",
    r.validationMiddleware.ValidateJSON(&dto.CreateSessionRequest{}),
    r.sessionHandler.CreateSession,
)

// Rotas de sessão específica:
sessionRoutes := sessions.Group("/",
    r.authMiddleware.RequireAPIKey(),
    r.validationMiddleware.ValidateParams(),
)
sessionRoutes.Get("/:sessionId",
    r.validationMiddleware.ValidateSessionAccess(),
    r.sessionHandler.GetSession,
)
```

## 📖 Documentação Swagger

TODOS os endpoints devem ter documentação Swagger completa:

```go
// @Summary Criar nova sessão WhatsApp
// @Description Cria uma nova sessão WhatsApp com configurações opcionais
// @Tags sessions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body dto.CreateSessionRequest true "Dados da sessão"
// @Success 201 {object} map[string]interface{} "Sessão criada com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Router /sessions/add [post]
```

## 🔐 Segurança e Validação

### DTOs e Validação (IMPLEMENTAÇÃO REAL)
```go
type CreateSessionRequest struct {
    Name      string         `json:"name" validate:"required,min=1,max=100"`
    SessionID string         `json:"session_id,omitempty" validate:"omitempty,session_id"`
    APIKey    string         `json:"api_key,omitempty" validate:"omitempty,api_key"`
    Webhook   *WebhookConfig `json:"webhook,omitempty"`
    Proxy     *ProxyConfig   `json:"proxy,omitempty"`
}

// Validações customizadas disponíveis:
// - session_id: Alfanumérico 3-50 caracteres
// - api_key: Mínimo 32 caracteres  
// - e164: Formato de telefone internacional
```

### Middleware de Autenticação (IMPLEMENTAÇÃO REAL)
```go
type AuthContext struct {
    APIKey          string
    IsGlobalKey     bool
    SessionID       string
    HasGlobalAccess bool
}

// Extração de API Key (múltiplos headers):
// 1. "apikey" header
// 2. "X-API-Key" header  
// 3. "Authorization" header (Bearer token)
```

### Context Usage e Services (IMPLEMENTAÇÃO REAL)
```go
// Interface de Service (padrão real):
type Service interface {
    CreateSession(ctx context.Context, config *Config) (*SessionInfo, error)
    GetSession(ctx context.Context, sessionID string) (*SessionInfo, error)
    ListSessions(ctx context.Context) ([]*SessionInfo, error)
    DeleteSession(ctx context.Context, sessionID string) error
    // ... outros métodos
}

// Implementação de Service:
type SessionService struct {
    repository repositories.SessionRepository
    manager    interface{}
    logger     *logger.ComponentLogger
}

func NewService(repository repositories.SessionRepository, manager interface{}) Service {
    return &SessionService{
        repository: repository,
        manager:    manager,
        logger:     logger.ForComponent("session"),
    }
}
```

## 🗃️ Banco de Dados e Models (IMPLEMENTAÇÃO REAL)

### Models
```go
// Estrutura real dos models:
type Session struct {
    ID               uuid.UUID      `json:"id" db:"id"`
    Name             string         `json:"name" db:"name"`
    APIKey           string         `json:"api_key" db:"api_key"`
    Status           SessionStatus  `json:"status" db:"status"`
    JID              *string        `json:"jid,omitempty" db:"jid"`
    WebhookURL       *string        `json:"webhook_url,omitempty" db:"webhook_url"`
    WebhookEvents    pq.StringArray `json:"webhook_events,omitempty" db:"webhook_events"`
    ProxyHost        *string        `json:"proxy_host,omitempty" db:"proxy_host"`
    ProxyPort        *int           `json:"proxy_port,omitempty" db:"proxy_port"`
    CreatedAt        time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
}

// Métodos do model:
func (s *Session) GetSessionID() string {
    // Implementação específica do projeto
}
```

### Repository Pattern
- Interfaces definidas em `/repositories/`
- Implementações concretas no mesmo pacote
- Métodos retornam `(*Model, error)` ou `([]*Model, error)`

## ⚡ Performance e Boas Práticas

1. **Ponteiros**: Use ponteiros para structs grandes
2. **Slice Allocation**: Pre-aloque slices quando possível
3. **Defer**: Use para cleanup (close, unlock, etc.)
4. **Interface Segregation**: Interfaces pequenas e focadas
5. **Error Wrapping**: Use `fmt.Errorf("msg: %w", err)`

## 🚨 Regras Obrigatórias

- ❌ **NUNCA** ignore erros silenciosamente
- ✅ **SEMPRE** use structured logging
- ✅ **SEMPRE** valide inputs
- ✅ **SEMPRE** use contexto em services
- ✅ **SEMPRE** documente endpoints com Swagger
- ✅ **SEMPRE** siga os padrões de nomenclatura
- ✅ **SEMPRE** use o sistema de resposta padronizado (`utils.SendError`)
- ✅ **SEMPRE** inclua tratamento adequado de erros com logs contextuais

## 📝 Exemplos de Código Correto (IMPLEMENTAÇÃO REAL)

### Service Method com Logger Avançado
```go
func (s *SessionService) CreateSession(ctx context.Context, config *Config) (*SessionInfo, error) {
    // Logger com operação e duração:
    opLogger := s.logger.ForOperation("create_session")
    opLogger.Starting().Str("name", config.Name).Msg("Creating new session")
    
    if err := s.validateConfig(config); err != nil {
        opLogger.Failed("VALIDATION_ERROR").Err(err).Msg("Invalid session configuration")
        return nil, err
    }
    
    // Implementation...
    
    opLogger.Success().Str("session_id", session.ID.String()).Msg("Session created successfully")
    return sessionInfo, nil
}
```

### Handler Method com Middleware Real
```go
func (h *SessionHandler) CreateSession(c *fiber.Ctx) error {
    // Validação padrão do projeto:
    validatedBody := c.Locals("validated_body")
    if validatedBody == nil {
        return utils.SendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
    }
    
    req, ok := validatedBody.(*dto.CreateSessionRequest)
    if !ok {
        return utils.SendError(c, "Invalid request format", "INVALID_REQUEST", fiber.StatusBadRequest)
    }
    
    // Verificar autenticação:
    auth := middleware.GetAuthContext(c)
    if !auth.IsGlobalKey {
        return utils.SendAuthorizationError(c, "Global access required")
    }
    
    // Chamar service:
    sessionInfo, err := h.sessionService.CreateSession(c.Context(), &session.Config{
        Name:      req.Name,
        SessionID: req.SessionID,
        APIKey:    req.APIKey,
    })
    
    if err != nil {
        h.logger.Error().Err(err).Msg("Failed to create session")
        return utils.SendInternalError(c, "Failed to create session")
    }
    
    // Resposta padronizada:
    return utils.SendSuccess(c, fiber.Map{
        "id":         sessionInfo.ID,
        "session_id": sessionInfo.ID,
        "name":       sessionInfo.Name,
        "status":     sessionInfo.Status,
    }, "Session created successfully")
}
```

### Configuração de Servidor (PADRÃO REAL)
```go
// Server setup com configurações reais:
app := fiber.New(fiber.Config{
    AppName:      "ZeMeow API",
    ServerHeader: "ZeMeow/1.0",
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  120 * time.Second,
    ErrorHandler: func(c *fiber.Ctx, err error) error {
        code := fiber.StatusInternalServerError
        if e, ok := err.(*fiber.Error); ok {
            code = e.Code
        }
        return c.Status(code).JSON(fiber.Map{
            "error":     "INTERNAL_ERROR",
            "message":   err.Error(),
            "code":      code,
            "timestamp": time.Now().Unix(),
        })
    },
})
```

## 🛠️ Tecnologias e Dependências Específicas

### Bibliotecas Principais
- **Fiber v2.52.6**: Framework web principal
- **whatsmeow**: Cliente WhatsApp oficial (`go.mau.fi/whatsmeow`)
- **zerolog**: Sistema de logging estruturado (`github.com/rs/zerolog`)
- **go-playground/validator/v10**: Validação de dados
- **sqlx**: Database toolkit (`github.com/jmoiron/sqlx`)
- **MinIO**: Armazenamento de mídia (`github.com/minio/minio-go/v7`)
- **PostgreSQL**: Banco principal (`github.com/lib/pq`)
- **Redis**: Cache e sessões
- **UUID**: `github.com/google/uuid`

### Padrões Arquiteturais
- **Repository Pattern**: Para acesso a dados
- **Service Layer**: Para lógica de negócio
- **Middleware Chain**: Para cross-cutting concerns
- **Dependency Injection**: Implícita via constructors
- **WhatsApp Integration**: Via whatsmeow client manager

## 🏛️ Arquitetura de Componentes

### Fluxo de Request/Response
```
Client → Middleware (Auth/Validation) → Handler → Service → Repository → Database
                                                      ↓
                                               WhatsApp Client
```

### Gerenciamento de Sessões WhatsApp
```go
// WhatsAppManager gerencia múltiplos clientes:
type WhatsAppManager struct {
    mu                 sync.RWMutex
    clients            map[string]*MyClient
    sessions           map[string]*models.Session
    container          *sqlstore.Container
    repository         repositories.SessionRepository
    messageRepo        repositories.MessageRepository
    messagePersistence *message.PersistenceService
    config             *config.Config
    logger             logger.Logger
    ctx                context.Context
    cancel             context.CancelFunc
    webhookChan        chan WebhookEvent
}
```

## 🔄 Fluxos de Trabalho Específicos

### Criação de Handler
1. Definir struct com dependências
2. Injetar services e repositories
3. Configurar logger com componente específico
4. Implementar constructor
5. Documentar com Swagger
6. Configurar rotas com middleware apropriado

### Processamento de Mensagem
1. Autenticação → Validação → Handler
2. Service processa lógica de negócio
3. WhatsApp client envia mensagem
4. Persistência salva no banco
5. Webhook dispara eventos (se configurado)
6. Resposta padronizada ao cliente

## 📊 Sistema de Webhooks

### Configuração de Webhook
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
    retryQueue chan WebhookPayload
    workers    int
    stats      WebhookServiceStats
}

type WebhookPayload struct {
    SessionID   string                 `json:"session_id"`
    Event       string                 `json:"event"`
    Data        interface{}            `json:"data,omitempty"`
    Timestamp   time.Time              `json:"timestamp"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

## 🚨 Regras Obrigatórias Atualizadas

- ❌ **NUNCA** ignore erros silenciosamente
- ✅ **SEMPRE** use structured logging com contexto apropriado
- ✅ **SEMPRE** valide inputs com middleware de validação
- ✅ **SEMPRE** use contexto em services
- ✅ **SEMPRE** documente endpoints com Swagger completo
- ✅ **SEMPRE** siga os padrões de nomenclatura estabelecidos
- ✅ **SEMPRE** use as funções utilitárias de resposta (`utils.SendError`, `utils.SendSuccess`)
- ✅ **SEMPRE** implemente tratamento adequado de autenticação
- ✅ **SEMPRE** use os middlewares apropriados para cada tipo de rota
- ✅ **SEMPRE** inclua logs contextuais com session_id, operation, etc.
- ✅ **SEMPRE** use os tipos de logger apropriados (ComponentLogger, OperationLogger)
- ✅ **SEMPRE** implemente dependency injection via constructors
- ✅ **SEMPRE** use interfaces para services e repositories



---

**IMPORTANTE**: Estas regras refletem a implementação REAL do projeto ZeMeow. Siga-as religiosamente para manter consistência e qualidade. O projeto usa padrões específicos de logging, autenticação e validação que DEVEM ser respeitados.