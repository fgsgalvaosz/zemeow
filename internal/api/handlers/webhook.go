package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/logger"
)


type WebhookHandler struct {
	logger logger.Logger
}


func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		logger: logger.GetWithSession("webhook_handler"),
	}
}


// @Summary Enviar webhook manualmente
// @Description Envia um webhook manualmente para teste
// @Tags webhooks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body map[string]interface{} true "Payload do webhook"
// @Success 200 {object} map[string]interface{} "Webhook enviado com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Router /webhooks/send [post]
func (h *WebhookHandler) SendWebhook(c *fiber.Ctx) error {

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


// @Summary Obter estatísticas de webhooks
// @Description Retorna estatísticas globais de webhooks enviados
// @Tags webhooks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Estatísticas de webhooks"
// @Router /webhooks/stats [get]
func (h *WebhookHandler) GetWebhookStats(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"total_sent":   0,
		"total_failed": 0,
		"message":      "Webhook stats endpoint",
	})
}



func (h *WebhookHandler) GetSessionWebhookStats(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id":   sessionID,
		"total_sent":   0,
		"total_failed": 0,
		"message":      "Session webhook stats endpoint",
	})
}



func (h *WebhookHandler) StartWebhookService(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "started",
		"message": "Webhook service start endpoint",
	})
}



func (h *WebhookHandler) StopWebhookService(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "stopped",
		"message": "Webhook service stop endpoint",
	})
}



func (h *WebhookHandler) GetWebhookServiceStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"running": true,
		"message": "Webhook service status endpoint",
	})
}


func (h *WebhookHandler) hasSessionAccess(c *fiber.Ctx, sessionID string) bool {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		return false
	}


	if authCtx.IsGlobalKey {
		return true
	}


	return authCtx.SessionID == sessionID
}


func (h *WebhookHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	errorResp := fiber.Map{
		"error":   code,
		"message": message,
		"code":    status,
	}

	return c.Status(status).JSON(errorResp)
}
