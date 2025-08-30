# Documentação Swagger - Zemeow WhatsApp API

## 📋 Visão Geral

A documentação Swagger da API Zemeow é **auto-gerada** através de comentários no código Go usando a biblioteca `swaggo/swag`. Isso garante que a documentação esteja sempre atualizada com o código atual.

## 🚀 Acessando a Documentação

### Interface Web (Swagger UI)
- **URL Principal**: http://localhost:8080/swagger/index.html
- **URL Alternativa**: http://localhost:8080/docs/index.html

### Formatos da Documentação
- **JSON**: http://localhost:8080/swagger/doc.json
- **YAML**: Arquivo `docs/swagger.yaml`

## 🔧 Como Funciona

### 1. Comentários Swagger no Código
A documentação é gerada automaticamente através de comentários especiais nos handlers Go:

```go
// SendText envia uma mensagem de texto
// @Summary Enviar mensagem de texto
// @Description Envia uma mensagem de texto via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendTextRequest true "Dados da mensagem"
// @Success 200 {object} map[string]interface{} "Mensagem enviada com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Router /sessions/{sessionId}/send/text [post]
func (h *MessageHandler) SendText(c *fiber.Ctx) error {
    // implementação...
}
```

### 2. Configuração Principal
O arquivo `cmd/zemeow/main.go` contém as configurações globais da API:

```go
// @title Zemeow WhatsApp API
// @version 1.0
// @description API para integração com WhatsApp usando whatsmeow
// @host localhost:8080
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
```

### 3. Geração Automática
Para regenerar a documentação após mudanças no código:

```bash
swag init -g cmd/zemeow/main.go
```

## 📚 Endpoints Documentados

### Health Check
- `GET /health` - Verificar status da API

### Sessões WhatsApp
- `POST /sessions` - Criar nova sessão
- `GET /sessions` - Listar todas as sessões
- `GET /sessions/active` - Listar conexões ativas
- `GET /sessions/{sessionId}` - Obter detalhes da sessão
- `POST /sessions/{sessionId}/connect` - Conectar sessão
- `POST /sessions/{sessionId}/disconnect` - Desconectar sessão
- `GET /sessions/{sessionId}/qr` - Obter QR Code

### Mensagens
- `POST /sessions/{sessionId}/send/text` - Enviar mensagem de texto
- `POST /sessions/{sessionId}/send/image` - Enviar imagem
- `POST /sessions/{sessionId}/send/document` - Enviar documento
- `POST /sessions/{sessionId}/send/audio` - Enviar áudio
- `POST /sessions/{sessionId}/send/video` - Enviar vídeo

### Grupos
- `POST /sessions/{sessionId}/group/create` - Criar grupo
- `POST /sessions/{sessionId}/group/add` - Adicionar participantes
- `POST /sessions/{sessionId}/group/remove` - Remover participantes
- `POST /sessions/{sessionId}/group/leave` - Sair do grupo

### Webhooks
- `POST /webhooks/send` - Enviar webhook manualmente
- `GET /webhooks/stats` - Estatísticas de webhooks
- `GET /webhooks/status` - Status do serviço

## 🔐 Autenticação

A API usa autenticação via API Key no header:
```
X-API-Key: sua_api_key_aqui
```

### Tipos de API Key:
- **Global**: Acesso completo a todas as funcionalidades
- **Session**: Acesso limitado a uma sessão específica

## 🛠️ Desenvolvimento

### Adicionando Novos Endpoints

1. **Adicione comentários Swagger** no handler:
```go
// @Summary Descrição curta
// @Description Descrição detalhada
// @Tags categoria
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param nome path/query/body tipo true/false "Descrição"
// @Success 200 {object} TipoResposta "Descrição sucesso"
// @Failure 400 {object} TipoErro "Descrição erro"
// @Router /caminho [método]
```

2. **Regenere a documentação**:
```bash
swag init -g cmd/zemeow/main.go
```

3. **Reinicie o servidor** para aplicar as mudanças

### Estruturas de Dados (DTOs)

As estruturas de dados são automaticamente documentadas quando referenciadas nos comentários:

```go
type SendTextRequest struct {
    To        string      `json:"to" validate:"required"`
    Text      string      `json:"text" validate:"required,max=4096"`
    MessageID string      `json:"message_id,omitempty"`
    ContextInfo *ContextInfo `json:"context_info,omitempty"`
}
```

## 📝 Tags Organizacionais

- **health**: Endpoints de health check
- **sessions**: Gerenciamento de sessões WhatsApp
- **messages**: Envio e gerenciamento de mensagens
- **groups**: Operações com grupos WhatsApp
- **webhooks**: Configuração e gerenciamento de webhooks

## 🔄 Atualização Automática

A documentação é regenerada automaticamente sempre que:
1. Novos comentários Swagger são adicionados
2. O comando `swag init` é executado
3. O servidor é reiniciado

## 📖 Recursos Adicionais

- **Swagger Editor**: https://editor.swagger.io/
- **Documentação swaggo**: https://github.com/swaggo/swag
- **Especificação OpenAPI**: https://swagger.io/specification/

## 🐛 Troubleshooting

### Documentação não aparece
1. Verifique se o servidor está rodando
2. Confirme que a importação `_ "github.com/felipe/zemeow/docs"` está presente
3. Regenere a documentação com `swag init`

### Endpoints não aparecem
1. Verifique se os comentários Swagger estão corretos
2. Confirme que o handler está registrado nas rotas
3. Regenere a documentação

### Estruturas não documentadas
1. Verifique se as structs estão sendo referenciadas nos comentários
2. Confirme que as tags JSON estão corretas
3. Regenere a documentação
