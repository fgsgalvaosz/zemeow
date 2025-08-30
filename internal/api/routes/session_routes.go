package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/felipe/zemeow/internal/api/dto"
)

// setupSessionRoutes configura todas as rotas relacionadas a sessões
func (r *Router) setupSessionRoutes() {
	// Grupo principal de sessões
	sessions := r.app.Group("/sessions")
	
	// Rotas que requerem Global API Key (criação e listagem global)
	r.setupGlobalSessionRoutes(sessions)
	
	// Rotas que requerem API Key (global ou session)
	r.setupSessionOperationRoutes(sessions)
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
	
	// === OPERAÇÕES DE MENSAGENS ===
	
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
	
	// === ROTAS DE MENSAGEM SEGUINDO PADRÃO DO SISTEMA DE REFERÊNCIA ===
	
	// POST /sessions/:sessionId/send/text - Envio de texto
	sessionRoutes.Post("/:sessionId/send/text",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendTextRequest{}),
		r.messageHandler.SendText,
	)
	
	// POST /sessions/:sessionId/send/media - Envio unificado de mídia
	sessionRoutes.Post("/:sessionId/send/media",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendMediaRequest{}),
		r.messageHandler.SendMedia,
	)
	
	// POST /sessions/:sessionId/send/location - Envio de localização
	sessionRoutes.Post("/:sessionId/send/location",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendLocationRequest{}),
		r.messageHandler.SendLocation,
	)
	
	// POST /sessions/:sessionId/send/contact - Envio de contato
	sessionRoutes.Post("/:sessionId/send/contact",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendContactRequest{}),
		r.messageHandler.SendContact,
	)
}