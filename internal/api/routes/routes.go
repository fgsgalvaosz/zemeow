package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/felipe/zemeow/internal/api/dto"
)

// setupGlobalMiddleware configura middleware global
func (r *Router) setupGlobalMiddleware() {
	r.app.Use(r.authMiddleware.CORS())
	r.app.Use(r.authMiddleware.RequestLogger())
}

// setupHealthRoutes configura rotas de health check
func (r *Router) setupHealthRoutes() {
	r.app.Get("/health", r.healthCheck)
}

// healthCheck endpoint de health check
func (r *Router) healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"service":   "zemeow-api",
		"version":   "1.0.0",
		"timestamp": "1640995200",
	})
}

// setupSessionRoutes configura todas as rotas relacionadas a sessões
func (r *Router) setupSessionRoutes() {
	// Grupo principal de sessões
	sessions := r.app.Group("/sessions")
	
	// Rotas que requerem Global API Key (criação e listagem global)
	r.setupGlobalSessionRoutes(sessions)
	
	// Rotas que requerem API Key (global ou session)
	r.setupSessionOperationRoutes(sessions)
	
	// Rotas de mensagens
	r.setupMessageRoutes()
}

// setupGlobalSessionRoutes configura rotas que requerem Global API Key
func (r *Router) setupGlobalSessionRoutes(sessions fiber.Router) {
	globalRoutes := sessions.Group("/", r.authMiddleware.RequireGlobalAPIKey())
	
	// POST /sessions - Criar nova sessão
	globalRoutes.Post("/", 
		r.validationMiddleware.ValidateJSON(&dto.CreateSessionRequest{}),
		r.sessionHandler.CreateSession,
	)
	
	// GET /sessions - Listar todas as sessões
	globalRoutes.Get("/",
		r.validationMiddleware.ValidatePaginationParams(),
		r.sessionHandler.GetAllSessions,
	)
	
	// GET /sessions/active - Listar conexões ativas
	globalRoutes.Get("/active", r.sessionHandler.GetActiveConnections)
}

// setupSessionOperationRoutes configura rotas de operação de sessão
func (r *Router) setupSessionOperationRoutes(sessions fiber.Router) {
	// Middleware para validar API Key (global ou session)
	sessionRoutes := sessions.Group("/",
		r.authMiddleware.RequireAPIKey(),
		r.validationMiddleware.ValidateParams(),
	)
	
	// === OPERAÇÕES BÁSICAS DE SESSÃO ===
	
	// GET /sessions/:sessionId - Obter detalhes da sessão
	sessionRoutes.Get("/:sessionId",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetSession,
	)
	
	// PUT /sessions/:sessionId - Atualizar sessão
	sessionRoutes.Put("/:sessionId",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.UpdateSessionRequest{}),
		r.sessionHandler.UpdateSession,
	)
	
	// DELETE /sessions/:sessionId - Deletar sessão
	sessionRoutes.Delete("/:sessionId",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.DeleteSession,
	)
	
	// === OPERAÇÕES DE CONEXÃO WHATSAPP ===
	
	// POST /sessions/:sessionId/connect - Conectar sessão
	sessionRoutes.Post("/:sessionId/connect",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.ConnectSession,
	)
	
	// POST /sessions/:sessionId/disconnect - Desconectar sessão
	sessionRoutes.Post("/:sessionId/disconnect",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.DisconnectSession,
	)
	
	// POST /sessions/:sessionId/logout - Logout da sessão
	sessionRoutes.Post("/:sessionId/logout",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.LogoutSession,
	)
	
	// GET /sessions/:sessionId/status - Status da conexão
	sessionRoutes.Get("/:sessionId/status",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetSessionStatus,
	)
	
	// GET /sessions/:sessionId/qr - Obter QR Code
	sessionRoutes.Get("/:sessionId/qr",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetSessionQRCode,
	)
	
	// GET /sessions/:sessionId/stats - Estatísticas da sessão
	sessionRoutes.Get("/:sessionId/stats",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetSessionStats,
	)
	
	// === OPERAÇÕES DE PAREAMENTO ===
	
	// POST /sessions/:sessionId/pairphone - Pareamento por telefone
	sessionRoutes.Post("/:sessionId/pairphone",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.PairPhoneRequest{}),
		r.sessionHandler.PairPhone,
	)
	
	// === OPERAÇÕES DE PROXY ===
	
	// POST /sessions/:sessionId/proxy - Configurar proxy
	sessionRoutes.Post("/:sessionId/proxy",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ProxyRequest{}),
		r.sessionHandler.SetProxy,
	)
	
	// GET /sessions/:sessionId/proxy - Obter configuração de proxy
	sessionRoutes.Get("/:sessionId/proxy",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetProxy,
	)
	
	// POST /sessions/:sessionId/proxy/test - Testar proxy
	sessionRoutes.Post("/:sessionId/proxy/test",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.TestProxy,
	)
	
	// === OPERAÇÕES DE MENSAGENS LEGADAS (MANTIDAS PARA COMPATIBILIDADE) ===
	
	// POST /sessions/:sessionId/messages - Enviar mensagem (compatibilidade)
	sessionRoutes.Post("/:sessionId/messages",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendMessageRequest{}),
		r.messageHandler.SendMessage,
	)
	
	// GET /sessions/:sessionId/messages - Listar mensagens
	sessionRoutes.Get("/:sessionId/messages",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidatePaginationParams(),
		r.validationMiddleware.ValidateQuery(&dto.MessageListRequest{}),
		r.messageHandler.GetMessages,
	)
	
	// POST /sessions/:sessionId/messages/bulk - Envio em lote
	sessionRoutes.Post("/:sessionId/messages/bulk",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.BulkMessageRequest{}),
		r.messageHandler.SendBulkMessages,
	)
	
	// GET /sessions/:sessionId/messages/:messageId/status - Status da mensagem
	sessionRoutes.Get("/:sessionId/messages/:messageId/status",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateParams(),
		r.messageHandler.GetMessageStatus,
	)
}

// setupMessageRoutes configura todas as rotas relacionadas a mensagens
func (r *Router) setupMessageRoutes() {
	// Grupo principal de mensagens
	messages := r.app.Group("/sessions/:sessionId")
	
	// Middleware para validar API Key (global ou session)
	messageRoutes := messages.Group("/",
		r.authMiddleware.RequireAPIKey(),
		r.validationMiddleware.ValidateParams(),
	)
	
	// === ROTAS DE ENVIO DE MENSAGENS ===
	
	// POST /sessions/:sessionId/send/text - Envio de texto
	messageRoutes.Post("/send/text",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendTextRequest{}),
		r.messageHandler.SendText,
	)
	
	// POST /sessions/:sessionId/send/media - Envio unificado de mídia
	messageRoutes.Post("/send/media",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendMediaRequest{}),
		r.messageHandler.SendMedia,
	)
	
	// POST /sessions/:sessionId/send/location - Envio de localização
	messageRoutes.Post("/send/location",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendLocationRequest{}),
		r.messageHandler.SendLocation,
	)
	
	// POST /sessions/:sessionId/send/contact - Envio de contato
	messageRoutes.Post("/send/contact",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendContactRequest{}),
		r.messageHandler.SendContact,
	)

	// === NOVOS ENDPOINTS DE ENVIO ===

	// POST /sessions/:sessionId/send/sticker - Envio de sticker
	messageRoutes.Post("/send/sticker",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendStickerRequest{}),
		r.messageHandler.SendSticker,
	)

	// === OPERAÇÕES DE MENSAGEM ===

	// POST /sessions/:sessionId/react - Reagir a mensagem
	messageRoutes.Post("/react",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ReactRequest{}),
		r.messageHandler.ReactToMessage,
	)

	// POST /sessions/:sessionId/delete - Deletar mensagem
	messageRoutes.Post("/delete",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.DeleteMessageRequest{}),
		r.messageHandler.DeleteMessage,
	)

	// === OPERAÇÕES DE CHAT ===

	// POST /sessions/:sessionId/chat/presence - Definir presença no chat
	messageRoutes.Post("/chat/presence",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ChatPresenceRequest{}),
		r.messageHandler.SetChatPresence,
	)

	// POST /sessions/:sessionId/chat/markread - Marcar como lido
	messageRoutes.Post("/chat/markread",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.MarkReadRequest{}),
		r.messageHandler.MarkAsRead,
	)

	// POST /sessions/:sessionId/download/image - Download de imagem
	messageRoutes.Post("/download/image",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadImage,
	)

	// POST /sessions/:sessionId/download/video - Download de vídeo
	messageRoutes.Post("/download/video",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadVideo,
	)

	// POST /sessions/:sessionId/download/audio - Download de áudio
	messageRoutes.Post("/download/audio",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadAudio,
	)

	// POST /sessions/:sessionId/download/document - Download de documento
	messageRoutes.Post("/download/document",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadDocument,
	)

	// === OPERAÇÕES DE SESSÃO (WHATSAPP) ===

	// POST /sessions/:sessionId/presence - Definir presença da sessão
	messageRoutes.Post("/presence",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SessionPresenceRequest{}),
		r.sessionHandler.SetPresence,
	)

	// POST /sessions/:sessionId/check - Verificar contatos
	messageRoutes.Post("/check",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.CheckContactRequest{}),
		r.sessionHandler.CheckContacts,
	)

	// POST /sessions/:sessionId/info - Obter informações de contato
	messageRoutes.Post("/info",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ContactInfoRequest{}),
		r.sessionHandler.GetContactInfo,
	)

	// POST /sessions/:sessionId/avatar - Obter avatar de contato
	messageRoutes.Post("/avatar",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ContactAvatarRequest{}),
		r.sessionHandler.GetContactAvatar,
	)

	// GET /sessions/:sessionId/contacts - Listar contatos da sessão
	messageRoutes.Get("/contacts",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetContacts,
	)

	// === OPERAÇÕES DE GRUPO ===
	// TODO: Implementar handlers de grupo quando necessário
}

// setupWebhookRoutes configura todas as rotas relacionadas a webhooks
func (r *Router) setupWebhookRoutes() {
	// Grupo principal de webhooks (requer Global API Key)
	webhooks := r.app.Group("/webhooks", r.authMiddleware.RequireGlobalAPIKey())
	
	// === OPERAÇÕES DE WEBHOOK ===
	
	// POST /webhooks/send - Enviar webhook manual
	webhooks.Post("/send",
		r.validationMiddleware.ValidateJSON(&dto.WebhookRequest{}),
		r.webhookHandler.SendWebhook,
	)
	
	// GET /webhooks/stats - Estatísticas globais de webhooks
	webhooks.Get("/stats", r.webhookHandler.GetWebhookStats)
	
	// === CONTROLE DO SERVIÇO ===
	
	// POST /webhooks/start - Iniciar serviço de webhooks
	webhooks.Post("/start", r.webhookHandler.StartWebhookService)
	
	// POST /webhooks/stop - Parar serviço de webhooks
	webhooks.Post("/stop", r.webhookHandler.StopWebhookService)
	
	// GET /webhooks/status - Status do serviço de webhooks
	webhooks.Get("/status", r.webhookHandler.GetWebhookServiceStatus)
	
	// === ESTATÍSTICAS POR SESSÃO ===
	
	// GET /webhooks/sessions/:sessionId/stats - Estatísticas por sessão
	webhooks.Get("/sessions/:sessionId/stats",
		r.validationMiddleware.ValidateParams(),
		r.validationMiddleware.ValidateSessionAccess(),
		r.webhookHandler.GetSessionWebhookStats,
	)
}