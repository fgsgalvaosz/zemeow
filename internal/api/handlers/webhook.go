package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/auth"
	"github.com/felipe/zemeow/internal/service/webhook"
)

// WebhookHandler gerencia endpoints de webhooks
type WebhookHandler struct {
	webhookService *webhook.Service
	authService    *auth.AuthService
	logger         logger.Logger
}

// NewWebhookHandler cria uma nova instância do handler de webhooks
func NewWebhookHandler(
	webhookService *webhook.Service,
	authService *auth.AuthService,
) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
		authService:    authService,
		logger:         logger.GetWithSession("webhook_handler"),
	}
}

// SendWebhook envia um webhook manualmente
// POST /webhooks/send
func (h *WebhookHandler) SendWebhook(c *fiber.Ctx) error {
	// Verificar permissões de admin
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || authCtx.Role != auth.RoleAdmin {
		return h.sendError(c, "Admin access required", "ADMIN_REQUIRED", fiber.StatusForbidden)
	}

	var req webhook.WebhookPayload
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar payload
	if req.Event == "" {
		return h.sendError(c, "Event is required", "VALIDATION_ERROR", fiber.StatusBadRequest)
	}
	if req.SessionID == "" {
		return h.sendError(c, "Session ID is required", "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	// Enviar webhook
	response, err := h.webhookService.SendWebhook(req)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", req.SessionID).Str("event", req.Event).Msg("Failed to send webhook")
		return h.sendError(c, "Failed to send webhook", "WEBHOOK_SEND_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().Str("session_id", req.SessionID).Str("event", req.Event).Msg("Webhook sent successfully")
	return c.Status(fiber.StatusOK).JSON(response)
}

// GetWebhookStats obtém estatísticas de webhooks
// GET /webhooks/stats
func (h *WebhookHandler) GetWebhookStats(c *fiber.Ctx) error {
	// Verificar permissões de admin
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || authCtx.Role != auth.RoleAdmin {
		return h.sendError(c, "Admin access required", "ADMIN_REQUIRED", fiber.StatusForbidden)
	}

	// Obter estatísticas
	stats := h.webhookService.GetStats()

	return c.Status(fiber.StatusOK).JSON(stats)
}

// GetSessionWebhookStats obtém estatísticas de webhooks de uma sessão específica
// GET /webhooks/sessions/:sessionId/stats
func (h *WebhookHandler) GetSessionWebhookStats(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Obter estatísticas da sessão
	stats := h.webhookService.GetSessionStats(sessionID)
	if stats == nil {
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	return c.Status(fiber.StatusOK).JSON(stats)
}

// StartWebhookService inicia o serviço de webhooks
// POST /webhooks/start
func (h *WebhookHandler) StartWebhookService(c *fiber.Ctx) error {
	// Verificar permissões de admin
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || authCtx.Role != auth.RoleAdmin {
		return h.sendError(c, "Admin access required", "ADMIN_REQUIRED", fiber.StatusForbidden)
	}

	// Iniciar serviço
	if err := h.webhookService.Start(); err != nil {
		h.logger.Error().Err(err).Msg("Failed to start webhook service")
		return h.sendError(c, "Failed to start webhook service", "SERVICE_START_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"message":    "Webhook service started successfully",
		"started_at": time.Now(),
	}

	h.logger.Info().Msg("Webhook service started")
	return c.Status(fiber.StatusOK).JSON(response)
}

// StopWebhookService para o serviço de webhooks
// POST /webhooks/stop
func (h *WebhookHandler) StopWebhookService(c *fiber.Ctx) error {
	// Verificar permissões de admin
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || authCtx.Role != auth.RoleAdmin {
		return h.sendError(c, "Admin access required", "ADMIN_REQUIRED", fiber.StatusForbidden)
	}

	// Parar serviço
	h.webhookService.Stop()

	response := fiber.Map{
		"message":   "Webhook service stopped successfully",
		"stopped_at": time.Now(),
	}

	h.logger.Info().Msg("Webhook service stopped")
	return c.Status(fiber.StatusOK).JSON(response)
}

// GetWebhookServiceStatus obtém o status do serviço de webhooks
// GET /webhooks/status
func (h *WebhookHandler) GetWebhookServiceStatus(c *fiber.Ctx) error {
	// Verificar permissões de admin
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || authCtx.Role != auth.RoleAdmin {
		return h.sendError(c, "Admin access required", "ADMIN_REQUIRED", fiber.StatusForbidden)
	}

	// Obter status do serviço
	isRunning := h.webhookService.IsRunning()
	stats := h.webhookService.GetStats()

	response := fiber.Map{
		"running":      isRunning,
		"stats":        stats,
		"checked_at":   time.Now(),
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// hasSessionAccess verifica se o usuário tem acesso à sessão
func (h *WebhookHandler) hasSessionAccess(c *fiber.Ctx, sessionID string) bool {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		return false
	}

	// Admin tem acesso a todas as sessões
	if authCtx.Role == auth.RoleAdmin {
		return true
	}

	// Usuário comum só tem acesso à própria sessão
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