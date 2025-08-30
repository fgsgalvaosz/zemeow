package routes

import (
	"github.com/felipe/zemeow/internal/api/dto"
)

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