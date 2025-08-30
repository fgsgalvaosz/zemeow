# Documentação da API com Swagger Go

## 1. Visão Geral

Este documento descreve a implementação de documentação automática da API usando Swagger (OpenAPI) para o projeto zemeow. A API é baseada em Go com o framework Fiber e segue uma arquitetura RESTful para gerenciamento de sessões do WhatsApp, envio de mensagens e integração com webhooks.

A documentação automática será gerada usando anotações Swagger nos handlers existentes, proporcionando uma interface interativa para testar e explorar a API.

## 2. Estrutura da API

A API está organizada nas seguintes categorias principais:

- **Sessões**: Gerenciamento de sessões do WhatsApp
- **Mensagens**: Envio e gerenciamento de mensagens
- **Webhooks**: Configuração e monitoramento de webhooks
- **Autenticação**: Validação de chaves de API

## 3. Arquitetura da Documentação

### 3.1 Estrutura de Pastas

```
internal/api/
├── dto/              # Data Transfer Objects
├── handlers/         # Manipuladores de requisições
├── middleware/       # Middleware de autenticação e logging
├── routes/           # Configuração de rotas
├── validators/       # Validação de dados
└── server.go         # Configuração do servidor
```

### 3.2 Integração com Swagger

Para implementar a documentação automática com Swagger, será necessário:

1. Adicionar anotações Swagger aos handlers existentes
2. Configurar o middleware Swagger no servidor
3. Gerar a especificação OpenAPI a partir das anotações
4. Servir a interface Swagger UI

A documentação será gerada automaticamente a partir das anotações nos handlers, garantindo que esteja sempre sincronizada com o código.

## 4. Endpoints da API

### 4.1 Gerenciamento de Sessões

#### Criar Sessão
- **Endpoint**: `POST /sessions`
- **Descrição**: Cria uma nova sessão do WhatsApp
- **Autenticação**: Requer API Key Global
- **Request Body**: `CreateSessionRequest`
- **Response**: `SessionResponse`

#### Listar Sessões
- **Endpoint**: `GET /sessions`
- **Descrição**: Lista todas as sessões
- **Autenticação**: Requer API Key Global
- **Query Parameters**: Paginação
- **Response**: `SessionListResponse`

#### Obter Detalhes da Sessão
- **Endpoint**: `GET /sessions/{sessionId}`
- **Descrição**: Obtém detalhes de uma sessão específica
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: `SessionResponse`

#### Atualizar Sessão
- **Endpoint**: `PUT /sessions/{sessionId}`
- **Descrição**: Atualiza configurações da sessão
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Request Body**: `UpdateSessionRequest`
- **Response**: `SessionResponse`

#### Deletar Sessão
- **Endpoint**: `DELETE /sessions/{sessionId}`
- **Descrição**: Remove uma sessão
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: Confirmação de exclusão

#### Conectar Sessão
- **Endpoint**: `POST /sessions/{sessionId}/connect`
- **Descrição**: Inicia a conexão com o WhatsApp
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: Confirmação de conexão

#### Desconectar Sessão
- **Endpoint**: `POST /sessions/{sessionId}/disconnect`
- **Descrição**: Encerra a conexão com o WhatsApp
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: Confirmação de desconexão

#### Status da Sessão
- **Endpoint**: `GET /sessions/{sessionId}/status`
- **Descrição**: Obtém o status atual da conexão
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: `SessionStatusResponse`

#### QR Code
- **Endpoint**: `GET /sessions/{sessionId}/qr`
- **Descrição**: Obtém o QR Code para pareamento
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: `QRCodeResponse`

#### Estatísticas da Sessão
- **Endpoint**: `GET /sessions/{sessionId}/stats`
- **Descrição**: Obtém estatísticas de uso da sessão
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: `SessionStatsResponse`

#### Pareamento por Telefone
- **Endpoint**: `POST /sessions/{sessionId}/pairphone`
- **Descrição**: Inicia pareamento via número de telefone
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Request Body**: `PairPhoneRequest`
- **Response**: `PairPhoneResponse`

### 4.2 Gerenciamento de Proxy

#### Configurar Proxy
- **Endpoint**: `POST /sessions/{sessionId}/proxy`
- **Descrição**: Define configurações de proxy para a sessão
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Request Body**: `ProxyRequest`
- **Response**: `ProxyResponse`

#### Obter Configuração de Proxy
- **Endpoint**: `GET /sessions/{sessionId}/proxy`
- **Descrição**: Obtém as configurações de proxy atuais
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: `ProxyResponse`

#### Testar Proxy
- **Endpoint**: `POST /sessions/{sessionId}/proxy/test`
- **Descrição**: Testa a conexão com o proxy configurado
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: `ProxyTestResponse`

### 4.3 Envio de Mensagens

#### Envio de Texto
- **Endpoint**: `POST /sessions/{sessionId}/send/text`
- **Descrição**: Envia mensagem de texto
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Request Body**: `SendTextRequest`
- **Response**: `MessageSentResponse`

#### Envio Unificado de Mídia
- **Endpoint**: `POST /sessions/{sessionId}/send/media`
- **Descrição**: Envia mídia (imagem, áudio, vídeo, documento, sticker)
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Request Body**: `SendMediaRequest`
- **Response**: `MessageSentResponse`

#### Envio de Localização
- **Endpoint**: `POST /sessions/{sessionId}/send/location`
- **Descrição**: Envia localização geográfica
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Request Body**: `SendLocationRequest`
- **Response**: `MessageSentResponse`

#### Envio de Contato
- **Endpoint**: `POST /sessions/{sessionId}/send/contact`
- **Descrição**: Envia contato
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Request Body**: `SendContactRequest`
- **Response**: `MessageSentResponse`

### 4.4 Mensagens Legadas (Compatibilidade)

#### Envio de Mensagem
- **Endpoint**: `POST /sessions/{sessionId}/messages`
- **Descrição**: Envio genérico de mensagem (mantido para compatibilidade)
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Request Body**: `SendMessageRequest`
- **Response**: `SendMessageResponse`

#### Listar Mensagens
- **Endpoint**: `GET /sessions/{sessionId}/messages`
- **Descrição**: Lista mensagens da sessão
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Query Parameters**: Filtros e paginação
- **Response**: `MessageListResponse`

#### Envio em Lote
- **Endpoint**: `POST /sessions/{sessionId}/messages/bulk`
- **Descrição**: Envio de múltiplas mensagens
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Request Body**: `BulkMessageRequest`
- **Response**: `BulkMessageResponse`

#### Status da Mensagem
- **Endpoint**: `GET /sessions/{sessionId}/messages/{messageId}/status`
- **Descrição**: Obtém status de uma mensagem específica
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: `MessageStatusResponse`

### 4.5 Gerenciamento de Webhooks

#### Enviar Webhook Manual
- **Endpoint**: `POST /webhooks/send`
- **Descrição**: Envia um webhook manualmente
- **Autenticação**: Requer API Key Global
- **Request Body**: `WebhookRequest`
- **Response**: `WebhookResponse`

#### Estatísticas Globais de Webhooks
- **Endpoint**: `GET /webhooks/stats`
- **Descrição**: Obtém estatísticas globais de webhooks
- **Autenticação**: Requer API Key Global
- **Response**: `WebhookStatsResponse`

#### Iniciar Serviço de Webhooks
- **Endpoint**: `POST /webhooks/start`
- **Descrição**: Inicia o serviço de processamento de webhooks
- **Autenticação**: Requer API Key Global
- **Response**: `WebhookServiceControlResponse`

#### Parar Serviço de Webhooks
- **Endpoint**: `POST /webhooks/stop`
- **Descrição**: Para o serviço de processamento de webhooks
- **Autenticação**: Requer API Key Global
- **Response**: `WebhookServiceControlResponse`

#### Status do Serviço de Webhooks
- **Endpoint**: `GET /webhooks/status`
- **Descrição**: Obtém o status do serviço de webhooks
- **Autenticação**: Requer API Key Global
- **Response**: `WebhookServiceStatusResponse`

#### Estatísticas de Webhooks por Sessão
- **Endpoint**: `GET /webhooks/sessions/{sessionId}/stats`
- **Descrição**: Obtém estatísticas de webhooks para uma sessão específica
- **Autenticação**: Requer API Key (Global ou da Sessão)
- **Response**: `SessionWebhookStatsResponse`

## 5. DTOs (Data Transfer Objects)

### 5.1 Sessão

#### CreateSessionRequest
```go
type CreateSessionRequest struct {
    Name      string `json:"name" validate:"required,min=1,max=100"`
    SessionID string `json:"session_id,omitempty" validate:"omitempty,alphanum,min=3,max=50"`
    APIKey    string `json:"api_key,omitempty" validate:"omitempty,min=32"`
    Webhook   string `json:"webhook,omitempty" validate:"omitempty,url"`
    Proxy     string `json:"proxy,omitempty" validate:"omitempty,url"`
    Events    string `json:"events,omitempty"`
}
```

#### SessionResponse
```go
type SessionResponse struct {
    ID          string     `json:"id"`
    SessionID   string     `json:"session_id"`
    Name        string     `json:"name"`
    APIKey      string     `json:"api_key"`
    Status      string     `json:"status"`
    JID         string     `json:"jid,omitempty"`
    Webhook     string     `json:"webhook,omitempty"`
    Proxy       string     `json:"proxy,omitempty"`
    Events      string     `json:"events,omitempty"`
    QRCode      string     `json:"qr_code,omitempty"`
    Connected   bool       `json:"connected"`
    LastSeen    *time.Time `json:"last_seen,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}
```

### 5.2 Mensagens

#### SendTextRequest
```go
type SendTextRequest struct {
    To          string       `json:"to" validate:"required"`
    Text        string       `json:"text" validate:"required,max=4096"`
    MessageID   string       `json:"message_id,omitempty"`
    ContextInfo *ContextInfo `json:"context_info,omitempty"`
}
```

#### SendMediaRequest
```go
type SendMediaRequest struct {
    To          string       `json:"to" validate:"required"`
    Type        MediaType    `json:"type" validate:"required,oneof=image audio video document sticker"`
    Media       string       `json:"media" validate:"required"` // Base64 data URL
    Caption     string       `json:"caption,omitempty"`
    Filename    string       `json:"filename,omitempty"`
    MimeType    string       `json:"mime_type,omitempty"`
    MessageID   string       `json:"message_id,omitempty"`
    ContextInfo *ContextInfo `json:"context_info,omitempty"`
}
```

### 5.3 Webhooks

#### WebhookRequest
```go
type WebhookRequest struct {
    URL     string                 `json:"url" validate:"required,url"`
    Method  string                 `json:"method" validate:"oneof=POST PUT PATCH"`
    Headers map[string]string      `json:"headers,omitempty"`
    Payload map[string]interface{} `json:"payload" validate:"required"`
    Retry   *WebhookRetryConfig    `json:"retry,omitempty"`
}
```

## 6. Autenticação

A API utiliza autenticação baseada em chaves API:

- **API Key Global**: Chave administrativa definida em variáveis de ambiente
- **API Key por Sessão**: Chave única gerada para cada sessão
- **Headers Aceitos**: `apikey`, `X-API-Key`, `Authorization`

A autenticação é tratada pelo middleware de autenticação que valida as chaves antes do processamento das requisições.

## 7. Implementação do Swagger

### 7.1 Bibliotecas Necessárias

Para implementar a documentação com Swagger, serão utilizadas as seguintes bibliotecas:

```go
import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/swagger"
    "github.com/swaggo/swag"
)
```

### 7.2 Anotações Necessárias

Será necessário adicionar anotações Swagger aos handlers existentes:

```go
// @Summary Criar Sessão
// @Description Cria uma nova sessão do WhatsApp
// @Tags Sessões
// @Accept json
// @Produce json
// @Param request body dto.CreateSessionRequest true "Dados da Sessão"
// @Success 200 {object} dto.SessionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /sessions [post]
func (h *SessionHandler) CreateSession(c *fiber.Ctx) error {
    // ...
}
```

As anotações devem ser adicionadas a todos os handlers da API para garantir uma documentação completa.

### 7.3 Configuração do Servidor

Adicionar configuração do Swagger no `server.go`:

```go
import (
    "github.com/gofiber/swagger"
    _ "github.com/felipe/zemeow/docs" // docs gerados pelo swag
)

func (s *Server) SetupRoutes() {
    // ... rotas existentes ...
    
    // Swagger
    s.app.Get("/swagger/*", swagger.HandlerDefault)
}
```

### 7.4 Geração da Documentação

Comandos para gerar a documentação:

```bash
# Instalar swag
go install github.com/swaggo/swag/cmd/swag@latest

# Gerar documentação
swag init -g internal/api/server.go -o docs/

# Atualizar documentação
swag fmt
```

## 8. Benefícios da Documentação Automática

1. **Consistência**: A documentação é gerada diretamente do código, garantindo que esteja sempre atualizada
2. **Interface Interativa**: Swagger UI permite testar endpoints diretamente no navegador
3. **Validação de Dados**: Documenta claramente os formatos de entrada e saída esperados
4. **Facilidade de Integração**: Desenvolvedores podem entender e integrar com a API mais facilmente
5. **Manutenção Reduzida**: Menos esforço para manter a documentação sincronizada com o código

Além disso, a documentação automática reduz erros de comunicação entre equipes e fornece uma referência confiável para integrações de terceiros.