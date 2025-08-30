package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/logger"
)

// WebhookHandler gerencia endpoints de webhooks
type WebhookHandler struct {
	logger logger.Logger
}

// NewWebhookHandler cria uma nova instância do handler de webhooks
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		logger: logger.GetWithSession("webhook_handler"),
	}
}

// SendWebhook envia um webhook manualmente
// POST /webhooks/send
func (h *WebhookHandler) SendWebhook(c *fiber.Ctx) error {
	// Verificar permissões globais
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || !authCtx.IsGlobalKey {
		return h.sendError(c, "Global access required", "GLOBAL_ACCESS_REQUIRED", fiber.StatusForbidden)
	}

	var req map[string]interface{}
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	return c.JSON(fiber.Map{
		"status":  "sent",
		"message": "Webhook endpoint (mock)",
		"payload": req,
	})
}

// GetWebhookStats obtém estatísticas de webhooks
// GET /webhooks/stats
func (h *WebhookHandler) GetWebhookStats(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"total_sent":   0,
		"total_failed": 0,
		"message":      "Webhook stats endpoint",
	})
}

// GetSessionWebhookStats obtém estatísticas de webhooks de uma sessão específica
// GET /webhooks/sessions/:sessionId/stats
func (h *WebhookHandler) GetSessionWebhookStats(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id":   sessionID,
		"total_sent":   0,
		"total_failed": 0,
		"message":      "Session webhook stats endpoint",
	})
}

// StartWebhookService inicia o serviço de webhooks
// POST /webhooks/start
func (h *WebhookHandler) StartWebhookService(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "started",
		"message": "Webhook service start endpoint",
	})
}

// StopWebhookService para o serviço de webhooks
// POST /webhooks/stop
func (h *WebhookHandler) StopWebhookService(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "stopped",
		"message": "Webhook service stop endpoint",
	})
}

// GetWebhookServiceStatus obtém o status do serviço de webhooks
// GET /webhooks/status
func (h *WebhookHandler) GetWebhookServiceStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"running": true,
		"message": "Webhook service status endpoint",
	})
}

// hasSessionAccess verifica se o usuário tem acesso à sessão
func (h *WebhookHandler) hasSessionAccess(c *fiber.Ctx, sessionID string) bool {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		return false
	}

	// Global key tem acesso a todas as sessões
	if authCtx.IsGlobalKey {
		return true
	}

	// Verificar se a sessão pertence ao usuário autenticado
	return authCtx.SessionID == sessionID
}

// sendError envia uma resposta de erro JSON
func (h *WebhookHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	errorResp := fiber.Map{
		"error":   code,
		"message": message,
		"code":    status,
	}

	return c.Status(status).JSON(errorResp)
}
