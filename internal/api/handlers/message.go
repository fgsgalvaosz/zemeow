package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/felipe/zemeow/internal/logger"
)

// MessageHandler gerencia endpoints de mensagens WhatsApp
type MessageHandler struct {
	logger logger.Logger
}

// NewMessageHandler cria uma nova instância do handler de mensagens
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		logger: logger.GetWithSession("message_handler"),
	}
}

// SendMessageRequest representa uma requisição de envio de mensagem
type SendMessageRequest struct {
	To      string `json:"to"`
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Media   string `json:"media,omitempty"`
	Caption string `json:"caption,omitempty"`
}

// SendMessage envia uma mensagem WhatsApp
// POST /sessions/:sessionId/messages
func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar campos obrigatórios
	if req.To == "" {
		return h.sendError(c, "recipient 'to' is required", "MISSING_FIELD", fiber.StatusBadRequest)
	}

	if req.Type == "" {
		req.Type = "text"
	}

	if req.Type == "text" && req.Text == "" {
		return h.sendError(c, "text content is required for text messages", "MISSING_FIELD", fiber.StatusBadRequest)
	}

	// Mock: Verificação simplificada de sessão

	// Mock: Simular envio de mensagem
	messageID := "msg_" + sessionID + "_" + req.To

	response := fiber.Map{
		"message_id": messageID,
		"session_id": sessionID,
		"to":         req.To,
		"type":       req.Type,
		"status":     "sent",
		"sent_at":    time.Now(),
		"message":    "Message sending endpoint (mock)",
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetMessages obtém mensagens de uma sessão
// GET /sessions/:sessionId/messages
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Parâmetros de paginação
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	if limit > 100 {
		limit = 100
	}

	// Mock: Simular lista de mensagens
	messages := []fiber.Map{
		{
			"id":        "msg_1",
			"from":      "5511999999999",
			"text":      "Hello World",
			"timestamp": time.Now().Unix(),
		},
	}

	response := fiber.Map{
		"session_id": sessionID,
		"messages":   messages,
		"limit":      limit,
		"offset":     offset,
		"total":      len(messages),
		"message":    "Messages list endpoint (mock)",
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetMessageStatus obtém o status de uma mensagem
// GET /sessions/:sessionId/messages/:messageId/status
func (h *MessageHandler) GetMessageStatus(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	messageID := c.Params("messageId")

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"message_id": messageID,
		"status":     "delivered",
		"message":    "Message status endpoint",
	})
}

// SendBulkMessages envia mensagens em lote
// POST /sessions/:sessionId/messages/bulk
func (h *MessageHandler) SendBulkMessages(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "sent",
		"message":    "Bulk messages endpoint",
	})
}

// hasSessionAccess verifica se o usuário tem acesso à sessão
func (h *MessageHandler) hasSessionAccess(c *fiber.Ctx, sessionID string) bool {
	return true // Simplificado para permitir acesso
}

// sendError envia uma resposta de erro JSON
func (h *MessageHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	errorResp := fiber.Map{
		"error":   code,
		"message": message,
		"status":  status,
	}

	return c.Status(status).JSON(errorResp)
}
