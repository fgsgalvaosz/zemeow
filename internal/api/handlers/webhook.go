package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"

	"github.com/felipe/zemeow/internal/api/dto"
	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/webhook"
)

// WebhookHandler gerencia operações relacionadas a webhooks
type WebhookHandler struct {
	logger         logger.Logger
	webhookService *webhook.WebhookService
	sessionRepo    repositories.SessionRepository
}

// NewWebhookHandler cria uma nova instância do WebhookHandler
func NewWebhookHandler(webhookService *webhook.WebhookService, sessionRepo repositories.SessionRepository) *WebhookHandler {
	return &WebhookHandler{
		logger:         logger.GetWithSession("webhook_handler"),
		webhookService: webhookService,
		sessionRepo:    sessionRepo,
	}
}

// FindWebhook busca webhooks configurados para uma sessão
// @Summary Buscar webhooks configurados
// @Description Retorna lista de webhooks configurados para uma sessão específica
// @Tags webhooks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Success 200 {object} map[string]interface{} "Lista de webhooks configurados"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Router /webhooks/sessions/{sessionId}/find [get]
func (h *WebhookHandler) FindWebhook(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied to this session", "SESSION_ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Buscar a sessão no banco de dados
	session, err := h.sessionRepo.GetByIdentifier(sessionID)
	if err != nil {
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	webhooks := []map[string]interface{}{}

	// Se há webhook configurado, adicionar à lista
	if session.WebhookURL != nil && *session.WebhookURL != "" {
		webhooks = append(webhooks, map[string]interface{}{
			"id":         1,
			"url":        *session.WebhookURL,
			"events":     []string(session.WebhookEvents),
			"active":     true,
			"created_at": session.CreatedAt.Unix(),
			"updated_at": session.UpdatedAt.Unix(),
		})
	}

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"webhooks":   webhooks,
		"total":      len(webhooks),
		"timestamp":  time.Now().Unix(),
	})
}

// SetWebhook configura um webhook para uma sessão
// @Summary Configurar webhook
// @Description Configura um novo webhook para uma sessão específica
// @Tags webhooks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param webhook body dto.WebhookConfigRequest true "Configuração do webhook"
// @Success 200 {object} map[string]interface{} "Webhook configurado com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Router /webhooks/sessions/{sessionId}/set [post]
func (h *WebhookHandler) SetWebhook(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied to this session", "SESSION_ACCESS_DENIED", fiber.StatusForbidden)
	}

	var req dto.WebhookConfigRequest
	
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}
	
	// Validar eventos
	validEvents := h.getValidEvents()
	for _, event := range req.Events {
		if !h.isValidEvent(event, validEvents) {
			return h.sendError(c, fmt.Sprintf("Invalid event: %s", event), "INVALID_EVENT", fiber.StatusBadRequest)
		}
	}

	// Buscar a sessão no banco de dados
	session, err := h.sessionRepo.GetByIdentifier(sessionID)
	if err != nil {
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	// Atualizar configuração do webhook
	if req.Active {
		session.WebhookURL = &req.URL
		session.WebhookEvents = pq.StringArray(req.Events)
	} else {
		session.WebhookURL = nil
		session.WebhookEvents = nil
	}

	// Salvar no banco de dados
	if err := h.sessionRepo.Update(session); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to update webhook configuration")
		return h.sendError(c, "Failed to save webhook configuration", "WEBHOOK_SAVE_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("url", req.URL).
		Strs("events", req.Events).
		Bool("active", req.Active).
		Msg("Webhook configured successfully")
	
	return c.JSON(fiber.Map{
		"status":     "configured",
		"message":    "Webhook configured successfully",
		"session_id": sessionID,
		"webhook": map[string]interface{}{
			"id":         time.Now().Unix(), // ID temporário
			"url":        req.URL,
			"events":     req.Events,
			"active":     req.Active,
			"created_at": time.Now().Unix(),
		},
		"timestamp": time.Now().Unix(),
	})
}

// GetWebhookEvents retorna lista completa de eventos disponíveis
// @Summary Listar eventos de webhook
// @Description Retorna lista completa de todos os eventos de webhook disponíveis organizados por categoria
// @Tags webhooks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Lista de eventos disponíveis"
// @Router /webhooks/events [get]
func (h *WebhookHandler) GetWebhookEvents(c *fiber.Ctx) error {
	events := h.getValidEvents()
	
	return c.JSON(fiber.Map{
		"events": events,
		"total":  len(events),
		"categories": map[string][]string{
			"connection": {"connected", "disconnected", "connect_failure", "reconnect"},
			"messages":   {"message", "receipt", "undecryptable_message"},
			"presence":   {"presence", "chat_presence"},
			"groups":     {"group_info", "joined_group"},
			"calls":      {"call_offer", "call_accept", "call_terminate"},
			"media":      {"picture"},
			"system":     {"app_state", "history_sync"},
		},
		"timestamp": time.Now().Unix(),
	})
}

// getValidEvents retorna lista de todos os eventos válidos
func (h *WebhookHandler) getValidEvents() []map[string]interface{} {
	return []map[string]interface{}{
		// Connection events
		{"name": "connected", "category": "connection", "description": "WhatsApp connection established"},
		{"name": "disconnected", "category": "connection", "description": "WhatsApp connection lost"},
		{"name": "connect_failure", "category": "connection", "description": "Failed to connect to WhatsApp"},
		{"name": "reconnect", "category": "connection", "description": "Reconnection attempt"},
		
		// Message events
		{"name": "message", "category": "messages", "description": "New message received"},
		{"name": "receipt", "category": "messages", "description": "Message delivery receipt"},
		{"name": "undecryptable_message", "category": "messages", "description": "Message could not be decrypted"},
		
		// Presence events
		{"name": "presence", "category": "presence", "description": "User presence status change"},
		{"name": "chat_presence", "category": "presence", "description": "Chat typing status"},
		
		// Group events
		{"name": "group_info", "category": "groups", "description": "Group information updated"},
		{"name": "joined_group", "category": "groups", "description": "Joined a new group"},
		
		// Call events
		{"name": "call_offer", "category": "calls", "description": "Incoming call offer"},
		{"name": "call_accept", "category": "calls", "description": "Call accepted"},
		{"name": "call_terminate", "category": "calls", "description": "Call terminated"},
		
		// Media events
		{"name": "picture", "category": "media", "description": "Profile picture updated"},
		
		// System events
		{"name": "app_state", "category": "system", "description": "App state synchronization"},
		{"name": "history_sync", "category": "system", "description": "Message history synchronization"},
	}
}

// isValidEvent verifica se um evento é válido
func (h *WebhookHandler) isValidEvent(event string, validEvents []map[string]interface{}) bool {
	for _, validEvent := range validEvents {
		if validEvent["name"].(string) == event {
			return true
		}
	}
	return false
}

func (h *WebhookHandler) hasSessionAccess(c *fiber.Ctx, sessionID string) bool {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		return false
	}

	// Acesso global sempre permitido
	if authCtx.IsGlobalKey {
		return true
	}

	// Verificar se o usuário tem acesso à sessão específica
	return authCtx.SessionID == sessionID
}

func (h *WebhookHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	return c.Status(status).JSON(fiber.Map{
		"error":   code,
		"message": message,
		"code":    status,
	})
}
