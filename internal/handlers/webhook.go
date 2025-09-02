package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"

	"github.com/felipe/zemeow/internal/dto"
	"github.com/felipe/zemeow/internal/handlers/utils"
	"github.com/felipe/zemeow/internal/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/services/webhook"
)

type WebhookHandler struct {
	logger         logger.Logger
	webhookService *webhook.WebhookService
	sessionRepo    repositories.SessionRepository
}

func NewWebhookHandler(webhookService *webhook.WebhookService, sessionRepo repositories.SessionRepository) *WebhookHandler {
	return &WebhookHandler{
		logger:         logger.GetWithSession("webhook_handler"),
		webhookService: webhookService,
		sessionRepo:    sessionRepo,
	}
}

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

	if !utils.HasSessionAccess(c, sessionID) {
		return utils.SendAccessDeniedError(c)
	}

	session, err := h.sessionRepo.GetByIdentifier(sessionID)
	if err != nil {
		return utils.SendNotFoundError(c, "Session not found")
	}

	webhooks := []map[string]interface{}{}

	if session.WebhookURL != nil && *session.WebhookURL != "" {
		payloadMode := "processed" // padrão
		if session.WebhookPayloadMode != nil {
			payloadMode = *session.WebhookPayloadMode
		}

		webhooks = append(webhooks, map[string]interface{}{
			"id":           1,
			"url":          *session.WebhookURL,
			"events":       []string(session.WebhookEvents),
			"payload_mode": payloadMode,
			"active":       true,
			"created_at":   session.CreatedAt.Unix(),
			"updated_at":   session.UpdatedAt.Unix(),
		})
	}

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"webhooks":   webhooks,
		"total":      len(webhooks),
		"timestamp":  time.Now().Unix(),
	})
}

// @Summary Configurar webhook
// @Description Configura um novo webhook para uma sessão específica com suporte a payload bruto
// @Description Modos de payload disponíveis:
// @Description - "processed": Payload processado e simplificado (padrão)
// @Description - "raw": Payload bruto da whatsmeow sem transformações
// @Description - "both": Ambos os formatos enviados separadamente
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

	if !utils.HasSessionAccess(c, sessionID) {
		return utils.SendAccessDeniedError(c)
	}

	var req dto.WebhookConfigRequest

	if err := c.BodyParser(&req); err != nil {
		return utils.SendInvalidJSONError(c)
	}

	validEvents := h.getValidEvents()
	for _, event := range req.Events {
		if !h.isValidEvent(event, validEvents) {
			return utils.SendError(c, fmt.Sprintf("Invalid event: %s", event), "INVALID_EVENT", fiber.StatusBadRequest)
		}
	}

	session, err := h.sessionRepo.GetByIdentifier(sessionID)
	if err != nil {
		return utils.SendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	if req.Active {
		session.WebhookURL = &req.URL
		session.WebhookEvents = pq.StringArray(req.Events)
		// Configurar PayloadMode (padrão: "processed")
		payloadMode := req.PayloadMode
		if payloadMode == "" {
			payloadMode = "processed"
		}
		session.WebhookPayloadMode = &payloadMode
	} else {
		session.WebhookURL = nil
		session.WebhookEvents = nil
		session.WebhookPayloadMode = nil
	}

	if err := h.sessionRepo.Update(session); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to update webhook configuration")
		return utils.SendError(c, "Failed to save webhook configuration", "WEBHOOK_SAVE_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("url", req.URL).
		Strs("events", req.Events).
		Str("payload_mode", req.PayloadMode).
		Bool("active", req.Active).
		Msg("Webhook configured successfully")

	payloadMode := req.PayloadMode
	if payloadMode == "" {
		payloadMode = "processed"
	}

	return c.JSON(fiber.Map{
		"status":     "configured",
		"message":    "Webhook configured successfully",
		"session_id": sessionID,
		"webhook": map[string]interface{}{
			"id":           time.Now().Unix(),
			"url":          req.URL,
			"events":       req.Events,
			"payload_mode": payloadMode,
			"active":       req.Active,
			"created_at":   time.Now().Unix(),
		},
		"timestamp": time.Now().Unix(),
	})
}

// @Summary Listar eventos de webhook
// @Description Retorna lista completa de todos os eventos de webhook disponíveis organizados por categoria
// @Description Inclui informações sobre os diferentes modos de payload suportados
// @Tags webhooks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Lista de eventos disponíveis e modos de payload"
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
		"payload_modes": map[string]interface{}{
			"processed": map[string]interface{}{
				"description": "Payload processado e simplificado (padrão)",
				"format":      "Estrutura customizada com campos principais",
				"size":        "Menor",
			},
			"raw": map[string]interface{}{
				"description": "Payload bruto da whatsmeow sem transformações",
				"format":      "Estrutura original da biblioteca whatsmeow",
				"size":        "Maior",
			},
			"both": map[string]interface{}{
				"description": "Ambos os formatos enviados separadamente",
				"format":      "Dois webhooks: um processado e um bruto",
				"size":        "Máximo",
			},
		},
		"timestamp": time.Now().Unix(),
	})
}

func (h *WebhookHandler) getValidEvents() []map[string]interface{} {
	return []map[string]interface{}{

		{"name": "connected", "category": "connection", "description": "WhatsApp connection established"},
		{"name": "disconnected", "category": "connection", "description": "WhatsApp connection lost"},
		{"name": "connect_failure", "category": "connection", "description": "Failed to connect to WhatsApp"},
		{"name": "reconnect", "category": "connection", "description": "Reconnection attempt"},

		{"name": "message", "category": "messages", "description": "New message received"},
		{"name": "receipt", "category": "messages", "description": "Message delivery receipt"},
		{"name": "undecryptable_message", "category": "messages", "description": "Message could not be decrypted"},

		{"name": "presence", "category": "presence", "description": "User presence status change"},
		{"name": "chat_presence", "category": "presence", "description": "Chat typing status"},

		{"name": "group_info", "category": "groups", "description": "Group information updated"},
		{"name": "joined_group", "category": "groups", "description": "Joined a new group"},

		{"name": "call_offer", "category": "calls", "description": "Incoming call offer"},
		{"name": "call_accept", "category": "calls", "description": "Call accepted"},
		{"name": "call_terminate", "category": "calls", "description": "Call terminated"},

		{"name": "picture", "category": "media", "description": "Profile picture updated"},

		{"name": "app_state", "category": "system", "description": "App state synchronization"},
		{"name": "history_sync", "category": "system", "description": "Message history synchronization"},
	}
}

func (h *WebhookHandler) isValidEvent(event string, validEvents []map[string]interface{}) bool {
	for _, validEvent := range validEvents {
		if validEvent["name"].(string) == event {
			return true
		}
	}
	return false
}
