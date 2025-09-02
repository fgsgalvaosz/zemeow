---
trigger: always_on
alwaysApply: true
---

# Regra de Especialista em Go para ZeMeow

Você é um especialista em Go trabalhando no projeto ZeMeow, uma API REST para WhatsApp. Siga rigorosamente estas diretrizes:

## 🏗️ Estrutura e Organização

### Estrutura de Pacotes
- **Handlers**: `internal/api/handlers/` - Lógica HTTP (Fiber)
- **Services**: `internal/service/` - Lógica de negócio
- **Models**: `internal/db/models/` - Estruturas de dados
- **Repositories**: `internal/db/repositories/` - Acesso a dados
- **DTOs**: `internal/api/dto/` - Transfer Objects
- **Utils**: `internal/api/utils/` - Utilitários HTTP

### Imports
```go
// Padrão de imports observado no projeto:
// 1. Stdlib primeiro
// 2. Dependências externas
// 3. Pacotes internos do projeto

import (
    "context"
    "fmt"
    "time"
    
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    
    "github.com/felipe/zemeow/internal/api/dto"
    "github.com/felipe/zemeow/internal/logger"
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

### Estruturas de Handler
```go
// Template padrão para handlers
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

### Logger
- SEMPRE usar structured logging com zerolog
- Incluir contexto relevante (session_id, user_id, etc.)
- Mensagens em inglês, descritivas

### Validação
```go
// Usar o middleware de validação estabelecido
validatedBody := c.Locals("validated_body")
if validatedBody == nil {
    return utils.SendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
}
```

### Respostas HTTP
```go
// Padrão de resposta estabelecido
response := fiber.Map{
    "success": true,
    "session": fiber.Map{
        "id":         sessionInfo.ID,
        "session_id": sessionInfo.ID,
        "name":       sessionInfo.Name,
        "status":     sessionInfo.Status,
    },
}
return c.Status(fiber.StatusCreated).JSON(response)
```

### Paginação
```go
// Usar o padrão estabelecido para listagem
type SessionFilter struct {
    Status   *SessionStatus `json:"status,omitempty"`
    Name     *string        `json:"name,omitempty"`
    Page     int            `json:"page"`
    PerPage  int            `json:"per_page"`
}
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

### Tags de Validação
```go
type CreateSessionRequest struct {
    Name      string `json:"name" validate:"required,min=1,max=255"`
    SessionID string `json:"session_id" validate:"omitempty,min=3,max=255"`
}
```

### Context Usage
- SEMPRE usar `context.Context` em services
- Propagar contexto através de chamadas
- Usar `context.Background()` quando necessário

## 🗃️ Banco de Dados

### Models
```go
// Usar tags apropriadas para JSON e DB
type Session struct {
    ID        uuid.UUID     `json:"id" db:"id"`
    Name      string        `json:"name" db:"name"`
    Status    SessionStatus `json:"status" db:"status"`
    CreatedAt time.Time     `json:"created_at" db:"created_at"`
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

## 📝 Exemplos de Código Correto

### Service Method
```go
func (s *SessionService) CreateSession(ctx context.Context, config *Config) (*SessionInfo, error) {
    s.logger.Info().Str("name", config.Name).Msg("Creating new session")
    
    if err := s.validateConfig(config); err != nil {
        s.logger.Error().Err(err).Msg("Invalid session configuration")
        return nil, err
    }
    
    // Implementation...
    
    s.logger.Info().Str("session_id", session.ID.String()).Msg("Session created successfully")
    return sessionInfo, nil
}
```

### Handler Method
```go
func (h *SessionHandler) CreateSession(c *fiber.Ctx) error {
    validatedBody := c.Locals("validated_body")
    if validatedBody == nil {
        return utils.SendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
    }
    
    req, ok := validatedBody.(*dto.CreateSessionRequest)
    if !ok {
        return utils.SendError(c, "Invalid request format", "INVALID_REQUEST", fiber.StatusBadRequest)
    }
    
    // Implementation...
    
    return c.Status(fiber.StatusCreated).JSON(response)
}
```

Siga estas diretrizes religiosamente para manter a consistência e qualidade do código no projeto ZeMeow.