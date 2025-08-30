# Análise do Sistema zemeow - WhatsApp API Gateway

## Overview

O **zemeow** é um sistema backend em Go para gerenciamento de múltiplas sessões WhatsApp, funcionando como um gateway API que permite automação de mensagens através da biblioteca whatsmeow. O sistema implementa uma arquitetura modular e robusta com autenticação por API keys, suporte a proxy por sessão, webhooks assíncronos e persistência em PostgreSQL.

## Arquitetura Atual

### Estrutura de Diretórios
```mermaid
graph TB
    subgraph "internal/ (Core)"
        API[api/]
        CONFIG[config/]
        DB[db/]
        LOGGER[logger/]
        SERVICE[service/]
    end
    
    subgraph "api/"
        HANDLERS[handlers/]
        MIDDLEWARE[middleware/]
        SERVER[server.go]
    end
    
    subgraph "service/"
        AUTH[auth/]
        MEOW[meow/]
        PROXY[proxy/]
        SESSION[session/]
        WEBHOOK[webhook/]
    end
    
    subgraph "db/"
        MODELS[models/]
        REPOS[repositories/]
        MIGRATIONS[migrations/]
    end
    
    API --> HANDLERS
    API --> MIDDLEWARE
    SERVICE --> AUTH
    SERVICE --> MEOW
    SERVICE --> SESSION
    DB --> MODELS
    DB --> REPOS
```

### Componentes Principais

#### 1. **SessionManager** (Core)
- Gerencia múltiplas sessões WhatsApp thread-safe
- Mantém mapa de clientes ativos em memória
- Sincronização entre cache e banco de dados
- Controle de lifecycle (start/stop/cleanup)

#### 2. **WhatsApp Integration** 
- Biblioteca: `go.mau.fi/whatsmeow`
- Store persistente: `sqlstore` para PostgreSQL
- Event handlers customizados por sessão
- Suporte a QR Code e pareamento por telefone

#### 3. **Authentication System**
- API Key global (admin): acesso total ao sistema
- API Keys por sessão: acesso restrito à sessão específica
- Middleware único para validação
- Headers suportados: `Authorization`, `X-API-Key`, `apikey`

#### 4. **Webhook System**
- Processamento assíncrono via workers
- Queue com buffer de 10.000 eventos
- Pool configurável de workers (padrão: 5)
- Retry automático com backoff exponencial

## Fluxos de Dados

### Criação de Sessão
```mermaid
sequenceDiagram
    participant C as Cliente
    participant API as API Server
    participant AUTH as AuthMiddleware
    participant SVC as SessionService
    participant WM as WhatsAppManager
    participant DB as Database
    
    C->>API: POST /sessions + Admin API Key
    API->>AUTH: Validate Admin Key
    AUTH->>SVC: CreateSession(config)
    SVC->>WM: Initialize WhatsApp Client
    WM->>DB: Store session + device store
    DB-->>WM: Session created
    WM-->>SVC: Session initialized
    SVC-->>API: Session info + API Key
    API-->>C: {session_id, api_key, status}
```

### Conexão WhatsApp
```mermaid
sequenceDiagram
    participant C as Cliente
    participant API as API Server
    participant WM as WhatsAppManager
    participant WA as WhatsApp Web
    participant WH as WebhookService
    
    C->>API: POST /sessions/{id}/connect
    API->>WM: ConnectSession(sessionId)
    WM->>WA: Initialize WhatsApp Connection
    WA-->>WM: QR Code Generated
    WM->>WH: Queue QR Event
    WM-->>API: QR Code Data
    API-->>C: {qr_code, status: "connecting"}
    
    Note over WA: User scans QR Code
    WA->>WM: Authentication Success
    WM->>WH: Queue Connected Event
    WM->>DB: Update session status
```

### Sistema de Webhooks
```mermaid
graph LR
    A[WhatsApp Event] --> B[Event Handler]
    B --> C[Webhook Queue]
    C --> D[Worker Pool]
    D --> E[HTTP POST]
    E --> F{Success?}
    F -->|Yes| G[Log Success]
    F -->|No| H[Retry Logic]
    H --> I{Max Retries?}
    I -->|No| D
    I -->|Yes| J[Log Failure]
```

## API Endpoints

### Session Management
| Método | Endpoint | Auth | Descrição |
|--------|----------|------|-----------|
| POST | `/sessions` | Admin | Criar nova sessão |
| GET | `/sessions` | Admin | Listar todas as sessões |
| GET | `/sessions/{id}` | Session | Detalhes da sessão |
| DELETE | `/sessions/{id}` | Session | Remover sessão |

### WhatsApp Connection
| Método | Endpoint | Auth | Descrição |
|--------|----------|------|-----------|
| POST | `/sessions/{id}/connect` | Session | Conectar WhatsApp |
| POST | `/sessions/{id}/disconnect` | Session | Desconectar |
| GET | `/sessions/{id}/status` | Session | Status da conexão |
| GET | `/sessions/{id}/qr` | Session | Obter QR Code |
| POST | `/sessions/{id}/pairphone` | Session | Pareamento por telefone |

### Message Handling
| Método | Endpoint | Auth | Descrição |
|--------|----------|------|-----------|
| POST | `/sessions/{id}/messages` | Session | Enviar mensagem |
| GET | `/sessions/{id}/messages` | Session | Listar mensagens |

## Modelos de Dados

### Session Model
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
    WebhookURL      *string        `json:"webhook_url,omitempty" db:"webhook_url"`
    WebhookEvents   []string       `json:"webhook_events" db:"webhook_events"`
    CreatedAt       time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
    Metadata        Metadata       `json:"metadata" db:"metadata"`
}
```

### Configuration Types
```go
type Config struct {
    SessionID     string         `json:"session_id"`
    Name          string         `json:"name"`
    APIKey        string         `json:"api_key,omitempty"`
    Proxy         *ProxyConfig   `json:"proxy,omitempty"`
    Webhook       *WebhookConfig `json:"webhook,omitempty"`
    AutoReconnect bool           `json:"auto_reconnect"`
}

type ProxyConfig struct {
    Enabled  bool   `json:"enabled"`
    Type     string `json:"type"`     // http, socks5
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`
}

type WebhookConfig struct {
    URL    string   `json:"url"`
    Events []string `json:"events"`
    Secret string   `json:"secret,omitempty"`
}
```

## Stack Tecnológica

### Core Dependencies
- **Web Framework**: Fiber v2 (alta performance)
- **Database**: PostgreSQL com database/sql + sqlx
- **WhatsApp**: go.mau.fi/whatsmeow + sqlstore
- **Logging**: zerolog (estruturado)
- **Config**: godotenv (variáveis de ambiente)
- **UUID**: github.com/google/uuid
- **QR Code**: github.com/skip2/go-qrcode

### Configuração de Ambiente
```bash
# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=zemeow
POSTGRES_USER=zemeow
POSTGRES_PASSWORD=zemeow123

# Server
SERVER_PORT=8080
ENVIRONMENT=development

# Authentication
ADMIN_API_KEY=sua_chave_admin_global

# Logging
LOG_LEVEL=info
LOG_PRETTY=true

# WhatsApp
QR_CODE_TIMEOUT=60s

# Webhook
WEBHOOK_TIMEOUT=10s
WEBHOOK_RETRY_COUNT=3
```

## Arquitetura de Middleware

### Stack de Middleware
```mermaid
graph TD
    A[HTTP Request] --> B[Recovery Middleware]
    B --> C[CORS Middleware]
    C --> D[Request Logging]
    D --> E[Rate Limiting]
    E --> F[Auth Middleware]
    F --> G[Session Validation]
    G --> H[Route Handler]
    H --> I[Response Logging]
    I --> J[HTTP Response]
```

### Authentication Flow
```mermaid
flowchart TD
    A[Request] --> B{API Key Header?}
    B -->|No| C[401 Unauthorized]
    B -->|Yes| D[Extract API Key]
    D --> E{Admin Key?}
    E -->|Yes| F[Admin Context]
    E -->|No| G[Check Session Key in DB]
    G --> H{Valid Session?}
    H -->|No| I[401 Invalid Key]
    H -->|Yes| J[Session Context]
    F --> K[Continue to Handler]
    J --> K
```

## Sistema de Cache e Performance

### SessionCache
- Cache em memória para sessões ativas
- TTL configurável (padrão: 1 hora)
- Cleanup automático de entradas expiradas
- Thread-safe com sync.RWMutex

### WhatsApp Client Pool
- Pool de clientes WhatsApp por sessão
- Gerenciamento de conexões ativas
- Cleanup automático de clientes inativos
- Event handlers por sessão

## Sistema de Logging

### Structured Logging
```go
// Logger principal
logger := zerolog.New(os.Stdout).With().
    Timestamp().
    Str("service", "zemeow").
    Logger()

// Logger por componente
sessionLogger := logger.With().Str("component", "session").Logger()
whatsappLogger := logger.With().Str("component", "whatsapp").Logger()
```

### Log Levels e Contextos
- **Info**: Operações normais (conexões, mensagens)
- **Warn**: Situações de atenção (timeouts, reconexões)
- **Error**: Erros recuperáveis (falhas de webhook, proxy)
- **Fatal**: Erros críticos (banco indisponível, config inválida)

## Considerações de Segurança

### API Key Management
- Admin API Key via variável de ambiente
- Session API Keys geradas automaticamente (UUID v4)
- Suporte a custom API keys fornecidas pelo usuário
- Validação em middleware único e centralizado

### Rate Limiting
- Limite por API Key quando disponível
- Fallback para IP quando não autenticado
- Configurável (padrão: 100 req/min)

### Proxy Security
- Suporte a proxy com autenticação
- Validação de configurações de proxy
- Teste de conectividade antes de aplicar

## Deployment e Infraestrutura

### Docker Compose
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: zemeow
      POSTGRES_USER: zemeow
      POSTGRES_PASSWORD: zemeow123
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  zemeow:
    build: .
    environment:
      DATABASE_URL: postgres://zemeow:zemeow123@postgres:5432/zemeow
      SERVER_PORT: 8080
      ADMIN_API_KEY: your_secure_admin_key
    ports:
      - "8080:8080"
    depends_on:
      - postgres
```

### Build e Execução
```bash
# Build
go build -o zemeow cmd/zemeow/main.go

# Execução local
./zemeow

# Docker
docker-compose up -d
```

## Métricas e Monitoramento

### Health Check
```json
GET /health
{
  "status": "ok",
  "timestamp": 1672531200,
  "version": "1.0.0",
  "uptime": 3600,
  "services": {
    "database": "ok",
    "whatsapp_manager": "ok",
    "webhook_service": "ok"
  }
}
```

### Session Statistics
```json
GET /sessions/{id}/stats
{
  "session_id": "sess_abc123",
  "status": "authenticated",
  "uptime": 3600,
  "messages_sent": 150,
  "messages_received": 75,
  "last_activity": "2024-01-01T12:00:00Z",
  "webhook_stats": {
    "sent": 225,
    "failed": 2,
    "success_rate": 99.1
  }
}
```

## Sistema de Mensagens - Engenharia Reversa e Redesign

### Análise do Sistema de Referência (`ref/`)

Com base na engenharia reversa do código em `ref/handlers.go`, identifiquei os padrões implementados no sistema original:

#### Padrões Atuais de Rotas (ref)
```
/chat/send/text      - Envio de texto
/chat/send/image     - Envio de imagem
/chat/send/audio     - Envio de áudio  
/chat/send/document  - Envio de documento
/chat/send/video     - Envio de vídeo
/chat/send/sticker   - Envio de sticker
/chat/send/location  - Envio de localização
/chat/send/contact   - Envio de contato
```

#### Estruturas de Dados Analisadas

**SendMessage (Text)**:
```go
type textStruct struct {
    Phone       string
    Body        string
    Id          string
    ContextInfo waE2E.ContextInfo
}
```

**SendImage**:
```go
type imageStruct struct {
    Phone       string
    Image       string        // Base64 data URL
    Caption     string
    Id          string
    MimeType    string
    ContextInfo waE2E.ContextInfo
}
```

**SendAudio**:
```go
type audioStruct struct {
    Phone       string
    Audio       string        // Base64 data URL (audio/ogg)
    Caption     string
    Id          string
    ContextInfo waE2E.ContextInfo
}
```

**SendDocument**:
```go
type documentStruct struct {
    Caption     string
    Phone       string
    Document    string        // Base64 data URL
    FileName    string
    Id          string
    MimeType    string
    ContextInfo waE2E.ContextInfo
}
```

**SendVideo**:
```go
type videoStruct struct {
    Phone         string
    Video         string      // Base64 data URL
    Caption       string
    Id            string
    JPEGThumbnail []byte
    MimeType      string
    ContextInfo   waE2E.ContextInfo
}
```

### Nova Arquitetura de Rotas de Mensagem

#### Rotas Propostas (Seguindo Padrão Atual)
```
POST /sessions/{sessionId}/send/text     - Envio de texto
POST /sessions/{sessionId}/send/media    - Envio unificado de mídia
POST /sessions/{sessionId}/send/location - Envio de localização
POST /sessions/{sessionId}/send/contact  - Envio de contato
POST /sessions/{sessionId}/send/poll     - Envio de enquete
```

#### Estrutura Unificada de Media

**SendTextRequest**:
```go
type SendTextRequest struct {
    To          string            `json:"to" validate:"required"`
    Text        string            `json:"text" validate:"required"`
    MessageID   string            `json:"message_id,omitempty"`
    ContextInfo *ContextInfo      `json:"context_info,omitempty"`
}
```

**SendMediaRequest** (Unificada):
```go
type SendMediaRequest struct {
    To          string       `json:"to" validate:"required"`
    Type        MediaType    `json:"type" validate:"required,oneof=image audio video document sticker"`
    Media       string       `json:"media" validate:"required"`        // Base64 data URL
    Caption     string       `json:"caption,omitempty"`
    FileName    string       `json:"filename,omitempty"`                // Para documentos
    MimeType    string       `json:"mime_type,omitempty"`
    MessageID   string       `json:"message_id,omitempty"`
    ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type MediaType string
const (
    MediaTypeImage    MediaType = "image"
    MediaTypeAudio    MediaType = "audio"
    MediaTypeVideo    MediaType = "video"
    MediaTypeDocument MediaType = "document"
    MediaTypeSticker  MediaType = "sticker"
)
```

**ContextInfo** (Para replies e menções):
```go
type ContextInfo struct {
    StanzaID     *string  `json:"stanza_id,omitempty"`     // Para reply
    Participant  *string  `json:"participant,omitempty"`   // Para reply em grupo
    MentionedJID []string `json:"mentioned_jid,omitempty"`  // Para menções
}
```

#### Implementação dos Handlers

**MessageHandler Completo**:
```go
package handlers

import (
    "context"
    "encoding/base64"
    "fmt"
    "mime"
    "net/http"
    "strings"
    "time"

    "github.com/gofiber/fiber/v2"
    "go.mau.fi/whatsmeow"
    waE2E "go.mau.fi/whatsmeow/binary/proto"
    "google.golang.org/protobuf/proto"

    "github.com/felipe/zemeow/internal/api/middleware"
    "github.com/felipe/zemeow/internal/logger"
    "github.com/felipe/zemeow/internal/service/session"
)

type MessageHandler struct {
    logger         logger.Logger
    sessionService session.Service
}

func NewMessageHandler(sessionService session.Service) *MessageHandler {
    return &MessageHandler{
        logger:         logger.GetWithSession("message_handler"),
        sessionService: sessionService,
    }
}

// SendText envia mensagem de texto
// POST /sessions/:sessionId/send/text
func (h *MessageHandler) SendText(c *fiber.Ctx) error {
    sessionID := c.Params("sessionId")
    
    if !h.hasSessionAccess(c, sessionID) {
        return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
    }

    var req SendTextRequest
    if err := c.BodyParser(&req); err != nil {
        return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
    }

    if err := h.validateTextRequest(&req); err != nil {
        return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
    }

    // Obter cliente WhatsApp da sessão
    client, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
    if err != nil {
        return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
    }

    // Preparar destinatário
    recipient, err := h.parseRecipient(req.To)
    if err != nil {
        return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
    }

    // Gerar ID da mensagem
    messageID := req.MessageID
    if messageID == "" {
        messageID = client.GenerateMessageID()
    }

    // Construir mensagem WhatsApp
    msg := &waE2E.Message{
        ExtendedTextMessage: &waE2E.ExtendedTextMessage{
            Text: proto.String(req.Text),
        },
    }

    // Adicionar context info se fornecido (reply/menções)
    if req.ContextInfo != nil {
        h.applyContextInfo(msg.ExtendedTextMessage.ContextInfo, req.ContextInfo)
    }

    // Enviar mensagem
    response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
    if err != nil {
        h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to send text message")
        return h.sendError(c, "Failed to send message", "SEND_FAILED", fiber.StatusInternalServerError)
    }

    h.logger.Info().
        Str("session_id", sessionID).
        Str("message_id", messageID).
        Str("recipient", req.To).
        Msg("Text message sent successfully")

    return c.Status(fiber.StatusOK).JSON(fiber.Map{
        "message_id": messageID,
        "status":     "sent",
        "timestamp":  response.Timestamp,
        "recipient":  req.To,
    })
}

// SendMedia envia mídia unificada (image, audio, video, document, sticker)
// POST /sessions/:sessionId/send/media
func (h *MessageHandler) SendMedia(c *fiber.Ctx) error {
    sessionID := c.Params("sessionId")
    
    if !h.hasSessionAccess(c, sessionID) {
        return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
    }

    var req SendMediaRequest
    if err := c.BodyParser(&req); err != nil {
        return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
    }

    if err := h.validateMediaRequest(&req); err != nil {
        return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
    }

    // Obter cliente WhatsApp
    client, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
    if err != nil {
        return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
    }

    // Processar upload baseado no tipo de mídia
    switch req.Type {
    case MediaTypeImage:
        return h.sendImageMessage(c, client, sessionID, &req)
    case MediaTypeAudio:
        return h.sendAudioMessage(c, client, sessionID, &req)
    case MediaTypeVideo:
        return h.sendVideoMessage(c, client, sessionID, &req)
    case MediaTypeDocument:
        return h.sendDocumentMessage(c, client, sessionID, &req)
    case MediaTypeSticker:
        return h.sendStickerMessage(c, client, sessionID, &req)
    default:
        return h.sendError(c, "Unsupported media type", "INVALID_MEDIA_TYPE", fiber.StatusBadRequest)
    }
}
```

#### Implementação de Envio por Tipo de Mídia

```go
// sendImageMessage processa e envia imagem
func (h *MessageHandler) sendImageMessage(c *fiber.Ctx, client *whatsmeow.Client, sessionID string, req *SendMediaRequest) error {
    // Decodificar data URL
    fileData, mimeType, err := h.decodeDataURL(req.Media, "image")
    if err != nil {
        return h.sendError(c, err.Error(), "INVALID_MEDIA_DATA", fiber.StatusBadRequest)
    }

    // Upload para WhatsApp
    uploaded, err := client.Upload(context.Background(), fileData, whatsmeow.MediaImage)
    if err != nil {
        return h.sendError(c, "Failed to upload image", "UPLOAD_FAILED", fiber.StatusInternalServerError)
    }

    // Gerar thumbnail se necessário
    thumbnailBytes, err := h.generateImageThumbnail(fileData)
    if err != nil {
        h.logger.Warn().Err(err).Msg("Failed to generate thumbnail")
    }

    // Preparar destinatário e message ID
    recipient, _ := h.parseRecipient(req.To)
    messageID := h.getOrGenerateMessageID(req.MessageID, client)

    // Construir mensagem de imagem
    msg := &waE2E.Message{
        ImageMessage: &waE2E.ImageMessage{
            Caption:       proto.String(req.Caption),
            URL:          proto.String(uploaded.URL),
            DirectPath:   proto.String(uploaded.DirectPath),
            MediaKey:     uploaded.MediaKey,
            Mimetype:     proto.String(h.getValidMimeType(mimeType, req.MimeType)),
            FileEncSHA256: uploaded.FileEncSHA256,
            FileSHA256:   uploaded.FileSHA256,
            FileLength:   proto.Uint64(uint64(len(fileData))),
            JPEGThumbnail: thumbnailBytes,
        },
    }

    // Aplicar context info se fornecido
    if req.ContextInfo != nil {
        h.applyContextInfo(msg.ImageMessage.ContextInfo, req.ContextInfo)
    }

    // Enviar mensagem
    response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
    if err != nil {
        return h.sendError(c, "Failed to send image", "SEND_FAILED", fiber.StatusInternalServerError)
    }

    return c.Status(fiber.StatusOK).JSON(fiber.Map{
        "message_id": messageID,
        "status":     "sent",
        "timestamp":  response.Timestamp,
        "recipient":  req.To,
        "media_type": req.Type,
        "file_size":  len(fileData),
    })
}

// sendAudioMessage processa e envia áudio
func (h *MessageHandler) sendAudioMessage(c *fiber.Ctx, client *whatsmeow.Client, sessionID string, req *SendMediaRequest) error {
    fileData, mimeType, err := h.decodeDataURL(req.Media, "audio")
    if err != nil {
        return h.sendError(c, err.Error(), "INVALID_MEDIA_DATA", fiber.StatusBadRequest)
    }

    uploaded, err := client.Upload(context.Background(), fileData, whatsmeow.MediaAudio)
    if err != nil {
        return h.sendError(c, "Failed to upload audio", "UPLOAD_FAILED", fiber.StatusInternalServerError)
    }

    recipient, _ := h.parseRecipient(req.To)
    messageID := h.getOrGenerateMessageID(req.MessageID, client)

    ptt := true // Push to talk (voice message)
    msg := &waE2E.Message{
        AudioMessage: &waE2E.AudioMessage{
            URL:          proto.String(uploaded.URL),
            DirectPath:   proto.String(uploaded.DirectPath),
            MediaKey:     uploaded.MediaKey,
            Mimetype:     proto.String(h.getValidMimeType(mimeType, "audio/ogg; codecs=opus")),
            FileEncSHA256: uploaded.FileEncSHA256,
            FileSHA256:   uploaded.FileSHA256,
            FileLength:   proto.Uint64(uint64(len(fileData))),
            PTT:          &ptt,
        },
    }

    if req.ContextInfo != nil {
        h.applyContextInfo(msg.AudioMessage.ContextInfo, req.ContextInfo)
    }

    response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
    if err != nil {
        return h.sendError(c, "Failed to send audio", "SEND_FAILED", fiber.StatusInternalServerError)
    }

    return c.Status(fiber.StatusOK).JSON(fiber.Map{
        "message_id": messageID,
        "status":     "sent",
        "timestamp":  response.Timestamp,
        "recipient":  req.To,
        "media_type": req.Type,
        "file_size":  len(fileData),
    })
}

// sendVideoMessage processa e envia vídeo
func (h *MessageHandler) sendVideoMessage(c *fiber.Ctx, client *whatsmeow.Client, sessionID string, req *SendMediaRequest) error {
    fileData, mimeType, err := h.decodeDataURL(req.Media, "video")
    if err != nil {
        return h.sendError(c, err.Error(), "INVALID_MEDIA_DATA", fiber.StatusBadRequest)
    }

    uploaded, err := client.Upload(context.Background(), fileData, whatsmeow.MediaVideo)
    if err != nil {
        return h.sendError(c, "Failed to upload video", "UPLOAD_FAILED", fiber.StatusInternalServerError)
    }

    recipient, _ := h.parseRecipient(req.To)
    messageID := h.getOrGenerateMessageID(req.MessageID, client)

    msg := &waE2E.Message{
        VideoMessage: &waE2E.VideoMessage{
            Caption:       proto.String(req.Caption),
            URL:          proto.String(uploaded.URL),
            DirectPath:   proto.String(uploaded.DirectPath),
            MediaKey:     uploaded.MediaKey,
            Mimetype:     proto.String(h.getValidMimeType(mimeType, req.MimeType)),
            FileEncSHA256: uploaded.FileEncSHA256,
            FileSHA256:   uploaded.FileSHA256,
            FileLength:   proto.Uint64(uint64(len(fileData))),
        },
    }

    if req.ContextInfo != nil {
        h.applyContextInfo(msg.VideoMessage.ContextInfo, req.ContextInfo)
    }

    response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
    if err != nil {
        return h.sendError(c, "Failed to send video", "SEND_FAILED", fiber.StatusInternalServerError)
    }

    return c.Status(fiber.StatusOK).JSON(fiber.Map{
        "message_id": messageID,
        "status":     "sent",
        "timestamp":  response.Timestamp,
        "recipient":  req.To,
        "media_type": req.Type,
        "file_size":  len(fileData),
    })
}

// sendDocumentMessage processa e envia documento
func (h *MessageHandler) sendDocumentMessage(c *fiber.Ctx, client *whatsmeow.Client, sessionID string, req *SendMediaRequest) error {
    if req.FileName == "" {
        return h.sendError(c, "filename is required for documents", "MISSING_FILENAME", fiber.StatusBadRequest)
    }

    fileData, mimeType, err := h.decodeDataURL(req.Media, "application")
    if err != nil {
        return h.sendError(c, err.Error(), "INVALID_MEDIA_DATA", fiber.StatusBadRequest)
    }

    uploaded, err := client.Upload(context.Background(), fileData, whatsmeow.MediaDocument)
    if err != nil {
        return h.sendError(c, "Failed to upload document", "UPLOAD_FAILED", fiber.StatusInternalServerError)
    }

    recipient, _ := h.parseRecipient(req.To)
    messageID := h.getOrGenerateMessageID(req.MessageID, client)

    msg := &waE2E.Message{
        DocumentMessage: &waE2E.DocumentMessage{
            URL:          proto.String(uploaded.URL),
            FileName:     proto.String(req.FileName),
            DirectPath:   proto.String(uploaded.DirectPath),
            MediaKey:     uploaded.MediaKey,
            Mimetype:     proto.String(h.getValidMimeType(mimeType, req.MimeType)),
            FileEncSHA256: uploaded.FileEncSHA256,
            FileSHA256:   uploaded.FileSHA256,
            FileLength:   proto.Uint64(uint64(len(fileData))),
            Caption:      proto.String(req.Caption),
        },
    }

    if req.ContextInfo != nil {
        h.applyContextInfo(msg.DocumentMessage.ContextInfo, req.ContextInfo)
    }

    response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
    if err != nil {
        return h.sendError(c, "Failed to send document", "SEND_FAILED", fiber.StatusInternalServerError)
    }

    return c.Status(fiber.StatusOK).JSON(fiber.Map{
        "message_id": messageID,
        "status":     "sent",
        "timestamp":  response.Timestamp,
        "recipient":  req.To,
        "media_type": req.Type,
        "filename":   req.FileName,
        "file_size":  len(fileData),
    })
}
```

#### Métodos Auxiliares

```go
// decodeDataURL decodifica data URL em bytes
func (h *MessageHandler) decodeDataURL(dataURL, expectedType string) ([]byte, string, error) {
    if !strings.HasPrefix(dataURL, "data:") {
        return nil, "", fmt.Errorf("invalid data URL format")
    }

    parts := strings.Split(dataURL, ",")
    if len(parts) != 2 {
        return nil, "", fmt.Errorf("invalid data URL structure")
    }

    header := parts[0]
    data := parts[1]

    // Extrair MIME type
    mimeType := ""
    if strings.Contains(header, ":") && strings.Contains(header, ";") {
        start := strings.Index(header, ":") + 1
        end := strings.Index(header, ";")
        mimeType = header[start:end]
    }

    // Validar tipo esperado
    if expectedType != "" && !strings.HasPrefix(mimeType, expectedType) {
        return nil, "", fmt.Errorf("expected %s media type, got %s", expectedType, mimeType)
    }

    // Decodificar base64
    decoded, err := base64.StdEncoding.DecodeString(data)
    if err != nil {
        return nil, "", fmt.Errorf("failed to decode base64 data: %v", err)
    }

    return decoded, mimeType, nil
}

// parseRecipient converte string de telefone para JID WhatsApp
func (h *MessageHandler) parseRecipient(phone string) (types.JID, error) {
    // Implementar lógica de parsing do número de telefone
    // Remover caracteres especiais, validar formato, etc.
    return types.NewJID(phone, types.DefaultUserServer), nil
}

// validateTextRequest valida requisição de texto
func (h *MessageHandler) validateTextRequest(req *SendTextRequest) error {
    if req.To == "" {
        return fmt.Errorf("recipient 'to' is required")
    }
    if req.Text == "" {
        return fmt.Errorf("text content is required")
    }
    return nil
}

// validateMediaRequest valida requisição de mídia
func (h *MessageHandler) validateMediaRequest(req *SendMediaRequest) error {
    if req.To == "" {
        return fmt.Errorf("recipient 'to' is required")
    }
    if req.Type == "" {
        return fmt.Errorf("media type is required")
    }
    if req.Media == "" {
        return fmt.Errorf("media data is required")
    }
    
    validTypes := []MediaType{MediaTypeImage, MediaTypeAudio, MediaTypeVideo, MediaTypeDocument, MediaTypeSticker}
    isValid := false
    for _, vt := range validTypes {
        if req.Type == vt {
            isValid = true
            break
        }
    }
    if !isValid {
        return fmt.Errorf("invalid media type: %s", req.Type)
    }

    if req.Type == MediaTypeDocument && req.FileName == "" {
        return fmt.Errorf("filename is required for document type")
    }

    return nil
}
```

### Exemplos de Uso da Nova API

#### Envio de Texto
```bash
curl -X POST http://localhost:8080/sessions/sess_123/send/text \
  -H "apikey: session_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "5511999999999",
    "text": "Olá! Como você está?",
    "context_info": {
      "mentioned_jid": ["5511888888888@s.whatsapp.net"]
    }
  }'
```

#### Envio de Imagem
```bash
curl -X POST http://localhost:8080/sessions/sess_123/send/media \
  -H "apikey: session_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "5511999999999",
    "type": "image",
    "media": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQ...",
    "caption": "Confira esta imagem!"
  }'
```

#### Envio de Documento
```bash
curl -X POST http://localhost:8080/sessions/sess_123/send/media \
  -H "apikey: session_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "5511999999999",
    "type": "document",
    "media": "data:application/pdf;base64,JVBERi0xLjQKMSAwIG9iai...",
    "filename": "documento.pdf",
    "mime_type": "application/pdf",
    "caption": "Segue o documento solicitado"
  }'
```

#### Envio de Áudio
```bash
curl -X POST http://localhost:8080/sessions/sess_123/send/media \
  -H "apikey: session_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "5511999999999",
    "type": "audio",
    "media": "data:audio/ogg;base64,T2dnUwACAAAAAAAAAABDYW1...",
    "caption": "Mensagem de voz"
  }'
```

### Configuração de Rotas

```go
// No arquivo internal/api/server.go
func (s *Server) setupMessageRoutes(sessions *fiber.App) {
    messageHandler := handlers.NewMessageHandler(s.sessionService)
    
    // Aplicar middleware de autenticação de sessão
    sessions.Use(s.authMiddleware.RequireAPIKey())
    
    // Rotas de envio de mensagens
    sessions.Post("/:sessionId/send/text", messageHandler.SendText)
    sessions.Post("/:sessionId/send/media", messageHandler.SendMedia)
    sessions.Post("/:sessionId/send/location", messageHandler.SendLocation)
    sessions.Post("/:sessionId/send/contact", messageHandler.SendContact)
    sessions.Post("/:sessionId/send/poll", messageHandler.SendPoll)
    
    // Rotas de controle de mensagens
    sessions.Get("/:sessionId/messages", messageHandler.GetMessages)
    sessions.Get("/:sessionId/messages/:messageId/status", messageHandler.GetMessageStatus)
    sessions.Post("/:sessionId/messages/bulk", messageHandler.SendBulkMessages)
}
```

### Vantagens da Nova Arquitetura

1. **Rota Unificada para Mídia**: Uma única rota `/send/media` para todos os tipos de mídia
2. **Tipagem Forte**: Enum para tipos de mídia com validação
3. **Validação Robusta**: Validação de data URLs, tipos MIME e campos obrigatórios
4. **Context Info**: Suporte completo a replies e menções
5. **Error Handling**: Tratamento de erros padronizado e informativos
6. **Logging Estruturado**: Logs detalhados para troubleshooting
7. **Compatibilidade**: Mantém compatibilidade com whatsmeow
8. **Flexibilidade**: Fácil extensão para novos tipos de mídia

## Análise Aprofundada da Lógica de Referência (`ref/`)

### Arquitetura de DTOs e Schemas do Sistema de Referência

Aprofundando a análise do código em `ref/`, identifiquei padrões sofisticados de estruturação de dados e lógica de negócio que devem ser incorporados na nova arquitetura:

#### **1. Sistema de Context e Autenticação**

**Values Struct (Context Carrier)**:
```go
type Values struct {
    m map[string]string
}

func (v Values) Get(key string) string {
    return v.m[key]
}
```

**User Context Structure**:
```go
// Extraido do middleware authalice
v := Values{map[string]string{
    "Id":      txtid,        // User/Session ID
    "Name":    name,         // Session Name
    "Jid":     jid,          // WhatsApp JID
    "Webhook": webhook,      // Webhook URL
    "Token":   token,        // API Token
    "Proxy":   proxy_url,    // Proxy Configuration
    "Events":  events,       // Subscribed Events
    "Qrcode":  qrcode,       // QR Code Data
}}
```

#### **2. Padrões de DTOs para Mensagens**

**Text Message DTO**:
```go
type textStruct struct {
    Phone       string               `json:"phone"`
    Body        string               `json:"body"`
    Id          string               `json:"id,omitempty"`
    ContextInfo waE2E.ContextInfo    `json:"context_info,omitempty"`
}
```

**Media Message DTOs (Padrão Unificado)**:
```go
// Image DTO
type imageStruct struct {
    Phone       string               `json:"phone"`
    Image       string               `json:"image"`         // Base64 data URL
    Caption     string               `json:"caption,omitempty"`
    Id          string               `json:"id,omitempty"`
    MimeType    string               `json:"mime_type,omitempty"`
    ContextInfo waE2E.ContextInfo    `json:"context_info,omitempty"`
}

// Audio DTO
type audioStruct struct {
    Phone       string               `json:"phone"`
    Audio       string               `json:"audio"`         // Base64 data URL
    Caption     string               `json:"caption,omitempty"`
    Id          string               `json:"id,omitempty"`
    ContextInfo waE2E.ContextInfo    `json:"context_info,omitempty"`
}

// Document DTO
type documentStruct struct {
    Caption     string               `json:"caption,omitempty"`
    Phone       string               `json:"phone"`
    Document    string               `json:"document"`      // Base64 data URL
    FileName    string               `json:"filename"`
    Id          string               `json:"id,omitempty"`
    MimeType    string               `json:"mime_type,omitempty"`
    ContextInfo waE2E.ContextInfo    `json:"context_info,omitempty"`
}

// Video DTO
type videoStruct struct {
    Phone         string             `json:"phone"`
    Video         string             `json:"video"`         // Base64 data URL
    Caption       string             `json:"caption,omitempty"`
    Id            string             `json:"id,omitempty"`
    JPEGThumbnail []byte             `json:"jpeg_thumbnail,omitempty"`
    MimeType      string             `json:"mime_type,omitempty"`
    ContextInfo   waE2E.ContextInfo  `json:"context_info,omitempty"`
}
```

#### **3. Webhook e Connection DTOs**

**Connection DTO**:
```go
type connectStruct struct {
    Subscribe []string `json:"subscribe"`          // Event types to subscribe
    Immediate bool     `json:"immediate"`          // Immediate connection flag
}
```

**Webhook Configuration DTOs**:
```go
type webhookStruct struct {
    WebhookURL string   `json:"webhookurl"`
    Events     []string `json:"events,omitempty"`
}

type updateWebhookStruct struct {
    WebhookURL string   `json:"webhook"`
    Events     []string `json:"events,omitempty"`
    Active     bool     `json:"active"`
}
```

#### **4. Sistema de Validação de Dados**

**Validação de Campos de Mensagem**:
```go
func validateMessageFields(phone string, stanzaid *string, participant *string) (types.JID, error) {
    // Parse e valida JID do destinatário
    recipient, ok := parseJID(phone)
    if !ok {
        return types.NewJID("", types.DefaultUserServer), errors.New("could not parse Phone")
    }

    // Validação de Context Info para replies
    if stanzaid != nil {
        if participant == nil {
            return types.NewJID("", types.DefaultUserServer), errors.New("missing Participant in ContextInfo")
        }
    }

    if participant != nil {
        if stanzaid == nil {
            return types.NewJID("", types.DefaultUserServer), errors.New("missing StanzaID in ContextInfo")
        }
    }

    return recipient, nil
}
```

**Parser de JID WhatsApp**:
```go
func parseJID(arg string) (types.JID, bool) {
    // Remove '+' do início se presente
    if arg[0] == '+' {
        arg = arg[1:]
    }
    
    // Se não contém '@', é um número de telefone
    if !strings.ContainsRune(arg, '@') {
        return types.NewJID(arg, types.DefaultUserServer), true
    } else {
        // Parse JID completo
        recipient, err := types.ParseJID(arg)
        if err != nil {
            log.Error().Err(err).Msg("Invalid JID")
            return recipient, false
        } else if recipient.User == "" {
            log.Error().Err(err).Msg("Invalid JID no server specified")
            return recipient, false
        }
        return recipient, true
    }
}
```

#### **5. Sistema de Response Padronizado**

**Response Envelope**:
```go
func (s *server) Respond(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    dataenvelope := map[string]interface{}{"code": status}
    
    if err, ok := data.(error); ok {
        // Error response
        dataenvelope["error"] = err.Error()
        dataenvelope["success"] = false
    } else {
        // Success response - tenta deserializar JSON
        var mydata map[string]interface{}
        if err := json.Unmarshal([]byte(data.(string)), &mydata); err == nil {
            dataenvelope["data"] = mydata
        } else {
            // Se falhar como map, tenta como array
            var mySlice []interface{}
            if err := json.Unmarshal([]byte(data.(string)), &mySlice); err == nil {
                dataenvelope["data"] = mySlice
            } else {
                log.Error().Str("error", fmt.Sprintf("%v", err)).Msg("error unmarshalling JSON")
            }
        }
        dataenvelope["success"] = true
    }

    if err := json.NewEncoder(w).Encode(dataenvelope); err != nil {
        panic("respond: " + err.Error())
    }
}
```

#### **6. Sistema de Eventos e Constantes**

**Event Types Suportados** (de `constants.go`):
```go
var supportedEventTypes = []string{
    // Mensagens e Comunicação
    "Message", "UndecryptableMessage", "Receipt", "MediaRetry", "ReadReceipt",
    
    // Grupos e Contatos
    "GroupInfo", "JoinedGroup", "Picture", "BlocklistChange", "Blocklist",
    
    // Conexão e Sessão
    "Connected", "Disconnected", "ConnectFailure", "KeepAliveRestored", 
    "KeepAliveTimeout", "LoggedOut", "ClientOutdated", "TemporaryBan", 
    "StreamError", "StreamReplaced", "PairSuccess", "PairError", "QR", 
    "QRScannedWithoutMultidevice",
    
    // Privacidade e Configurações
    "PrivacySettings", "PushNameSetting", "UserAbout",
    
    // Sincronização e Estado
    "AppState", "AppStateSyncComplete", "HistorySync", 
    "OfflineSyncCompleted", "OfflineSyncPreview",
    
    // Chamadas
    "CallOffer", "CallAccept", "CallTerminate", "CallOfferNotice", "CallRelayLatency",
    
    // Presença e Atividade
    "Presence", "ChatPresence",
    
    // Identidade
    "IdentityChange",
    
    // Newsletter (WhatsApp Channels)
    "NewsletterJoin", "NewsletterLeave", "NewsletterMuteChange", "NewsletterLiveUpdate",
    
    // Especial - recebe todos os eventos
    "All",
}
```

#### **7. Sistema de Middleware e Autenticação**

**Admin Authentication Middleware**:
```go
func (s *server) authadmin(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token != *adminToken {
            s.Respond(w, r, http.StatusUnauthorized, errors.New("unauthorized"))
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

**User/Session Authentication Middleware**:
```go
func (s *server) authalice(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var ctx context.Context
        txtid := ""
        
        // Obter token de headers ou parâmetros de URL
        token := r.Header.Get("token")
        if token == "" {
            token = strings.Join(r.URL.Query()["token"], "")
        }

        // Verificar cache primeiro
        myuserinfo, found := userinfocache.Get(token)
        if !found {
            // Buscar no banco de dados
            rows, err := s.db.Query("SELECT id,name,webhook,jid,events,proxy_url,qrcode FROM users WHERE token=$1 LIMIT 1", token)
            if err != nil {
                s.Respond(w, r, http.StatusInternalServerError, err)
                return
            }
            defer rows.Close()
            
            // Processar resultado e armazenar no cache
            for rows.Next() {
                // ... scan e construção do Values struct
                userinfocache.Set(token, v, cache.NoExpiration)
                ctx = context.WithValue(r.Context(), "userinfo", v)
            }
        } else {
            ctx = context.WithValue(r.Context(), "userinfo", myuserinfo)
            txtid = myuserinfo.(Values).Get("Id")
        }

        if txtid == "" {
            s.Respond(w, r, http.StatusUnauthorized, errors.New("unauthorized"))
            return
        }
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

#### **8. Sistema de Helpers e Utilitários**

**Webhook Helpers**:
```go
// Webhook para mensagens regulares
func callHook(myurl string, payload map[string]string, id string) {
    client := clientManager.GetHTTPClient(id)
    
    format := os.Getenv("WEBHOOK_FORMAT")
    if format == "json" {
        // Envio como JSON puro
        var body interface{} = payload
        if jsonStr, ok := payload["jsonData"]; ok {
            var postmap map[string]interface{}
            err := json.Unmarshal([]byte(jsonStr), &postmap)
            if err == nil {
                postmap["token"] = payload["token"]
                body = postmap
            }
        }
        _, err := client.R().
            SetHeader("Content-Type", "application/json").
            SetBody(body).
            Post(myurl)
    } else {
        // Padrão: envio como form-urlencoded
        _, err := client.R().SetFormData(payload).Post(myurl)
    }
}

// Webhook para mensagens com anexos
func callHookFile(myurl string, payload map[string]string, id string, file string) error {
    client := clientManager.GetHTTPClient(id)
    
    resp, err := client.R().
        SetFiles(map[string]string{
            "file": file,
        }).
        SetFormData(payload).
        Post(myurl)
        
    return err
}
```

**Database Configuration Helper**:
```go
type DatabaseConfig struct {
    Type     string
    Host     string
    Port     string
    User     string
    Password string
    Name     string
    Path     string
}

func getDatabaseConfig(exPath string) DatabaseConfig {
    // Verificar configuração PostgreSQL
    dbUser := os.Getenv("DB_USER")
    dbPassword := os.Getenv("DB_PASSWORD")
    dbName := os.Getenv("DB_NAME")
    dbHost := os.Getenv("DB_HOST")
    dbPort := os.Getenv("DB_PORT")

    // Se todas as configs PostgreSQL estão presentes, usar PostgreSQL
    if dbUser != "" && dbPassword != "" && dbName != "" && dbHost != "" && dbPort != "" {
        return DatabaseConfig{
            Type:     "postgres",
            Host:     dbHost,
            Port:     dbPort,
            User:     dbUser,
            Password: dbPassword,
            Name:     dbName,
        }
    }

    // Padrão: SQLite
    return DatabaseConfig{
        Type: "sqlite",
        Path: filepath.Join(exPath, "dbdata"),
    }
}
```

### **Padrões Identificados para Nova Implementação**

#### **1. Context Pattern Melhorado**
```go
// Nova implementação baseada no padrão ref
type SessionContext struct {
    SessionID     string            `json:"session_id"`
    Name          string            `json:"name"`
    JID           string            `json:"jid,omitempty"`
    APIKey        string            `json:"-"` // Não serializar por segurança
    WebhookURL    string            `json:"webhook_url,omitempty"`
    ProxyURL      string            `json:"proxy_url,omitempty"`
    Events        []string          `json:"events"`
    QRCode        string            `json:"qr_code,omitempty"`
    IsGlobalKey   bool              `json:"is_global_key"`
    Metadata      map[string]string `json:"metadata,omitempty"`
}

func (sc *SessionContext) Get(key string) string {
    // Implementar getter genérico similar ao Values
    switch key {
    case "SessionID", "Id":
        return sc.SessionID
    case "Name":
        return sc.Name
    case "JID", "Jid":
        return sc.JID
    case "APIKey", "Token":
        return sc.APIKey
    case "WebhookURL", "Webhook":
        return sc.WebhookURL
    case "ProxyURL", "Proxy":
        return sc.ProxyURL
    case "QRCode", "Qrcode":
        return sc.QRCode
    default:
        if sc.Metadata != nil {
            return sc.Metadata[key]
        }
        return ""
    }
}
```

#### **2. DTO Unificado para Mensagens**
```go
// Baseado nos padrões do ref, mas unificado
type UnifiedMessageRequest struct {
    // Campos comuns
    To          string            `json:"to" validate:"required"`
    Type        MessageType       `json:"type" validate:"required"`
    MessageID   string            `json:"message_id,omitempty"`
    ContextInfo *ContextInfo      `json:"context_info,omitempty"`
    
    // Conteúdo da mensagem
    Text        string            `json:"text,omitempty"`
    Caption     string            `json:"caption,omitempty"`
    
    // Mídia (Base64 Data URL)
    Media       string            `json:"media,omitempty"`
    FileName    string            `json:"filename,omitempty"`
    MimeType    string            `json:"mime_type,omitempty"`
    
    // Específicos
    Thumbnail   []byte            `json:"thumbnail,omitempty"`
}

type MessageType string
const (
    MessageTypeText     MessageType = "text"
    MessageTypeImage    MessageType = "image"
    MessageTypeAudio    MessageType = "audio"
    MessageTypeVideo    MessageType = "video"
    MessageTypeDocument MessageType = "document"
    MessageTypeSticker  MessageType = "sticker"
    MessageTypeLocation MessageType = "location"
    MessageTypeContact  MessageType = "contact"
)

type ContextInfo struct {
    StanzaID     *string  `json:"stanza_id,omitempty"`     // Para reply
    Participant  *string  `json:"participant,omitempty"`   // Para reply em grupo
    MentionedJID []string `json:"mentioned_jid,omitempty"`  // Para menções
}
```

#### **3. Response Envelope Padronizado**
```go
type APIResponse struct {
    Success   bool        `json:"success"`
    Code      int         `json:"code"`
    Data      interface{} `json:"data,omitempty"`
    Error     string      `json:"error,omitempty"`
    Timestamp int64       `json:"timestamp"`
}

func (h *BaseHandler) SendResponse(c *fiber.Ctx, status int, data interface{}, err error) error {
    response := APIResponse{
        Success:   err == nil,
        Code:      status,
        Timestamp: time.Now().Unix(),
    }
    
    if err != nil {
        response.Error = err.Error()
    } else {
        response.Data = data
    }
    
    return c.Status(status).JSON(response)
}
```

### **Implementação Baseada nos Padrões do Ref**

Com base nesta análise aprofundada, a nova implementação deve incorporar:

1. **Sistema de Context robusto** com cache em memória
2. **DTOs tipados e validados** seguindo os padrões existentes
3. **Sistema de eventos configuravel** baseado na lista de constantes
4. **Response envelope consistente** para todas as APIs
5. **Validação de JID e campos** seguindo as regras do WhatsApp
6. **Sistema de webhook flexível** com suporte a JSON e form-data
7. **Cache de usuário/sessão** para performance
8. **Configuração de banco flexivel** (PostgreSQL/SQLite)