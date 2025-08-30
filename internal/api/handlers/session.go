package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/session"
)

// SessionHandler gerencia endpoints de sessões WhatsApp
type SessionHandler struct {
	sessionService session.Service
	logger         logger.Logger
}

// NewSessionHandler cria uma nova instância do handler de sessões
func NewSessionHandler(sessionService session.Service) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
		logger:         logger.GetWithSession("session_handler"),
	}
}

// CreateSession cria uma nova sessão WhatsApp
// POST /sessions
func (h *SessionHandler) CreateSession(c *fiber.Ctx) error {
	// Verificar permissões globais
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || !authCtx.IsGlobalKey {
		return h.sendError(c, "Global access required", "GLOBAL_ACCESS_REQUIRED", fiber.StatusForbidden)
	}

	var req models.CreateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse create session request")
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar campos obrigatórios
	if req.Name == "" {
		return h.sendError(c, "name is required", "MISSING_FIELD", fiber.StatusBadRequest)
	}

	// Criar configuração da sessão
	config := &session.Config{
		SessionID: req.SessionID,
		Name:      req.Name,
		APIKey:    req.APIKey,
		Proxy:     req.Proxy,
		Webhook:   req.Webhook,
	}

	// Criar sessão
	ctx := context.Background()
	sessionInfo, err := h.sessionService.CreateSession(ctx, config)
	if err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create session")
		return h.sendError(c, err.Error(), "CREATE_SESSION_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().Str("session_id", sessionInfo.ID).Str("name", sessionInfo.Name).Msg("Session created successfully")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"session_id": sessionInfo.ID,
		"name":       sessionInfo.Name,
		"api_key":    sessionInfo.APIKey,
		"status":     sessionInfo.Status,
		"created_at": sessionInfo.CreatedAt,
		"message":    "Session created successfully",
	})
}

// GetSession obtém informações de uma sessão específica
// GET /sessions/:sessionId
func (h *SessionHandler) GetSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Buscar sessão
	ctx := context.Background()
	sessionInfo, err := h.sessionService.GetSession(ctx, sessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get session")
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	return c.Status(fiber.StatusOK).JSON(sessionInfo)
}

// GetAllSessions lista todas as sessões (apenas acesso global)
// GET /sessions
func (h *SessionHandler) GetAllSessions(c *fiber.Ctx) error {
	// Verificar permissões globais
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || !authCtx.IsGlobalKey {
		return h.sendError(c, "Global access required", "GLOBAL_ACCESS_REQUIRED", fiber.StatusForbidden)
	}

	// Listar sessões
	ctx := context.Background()
	sessions, err := h.sessionService.ListSessions(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list sessions")
		return h.sendError(c, "Failed to list sessions", "LIST_SESSIONS_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// UpdateSession atualiza uma sessão existente
// PUT /sessions/:sessionId
func (h *SessionHandler) UpdateSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "updated",
		"message":    "Session update endpoint",
	})
}

// DeleteSession remove uma sessão
// DELETE /sessions/:sessionId
func (h *SessionHandler) DeleteSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Deletar sessão
	ctx := context.Background()
	if err := h.sessionService.DeleteSession(ctx, sessionID); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to delete session")
		return h.sendError(c, "Failed to delete session", "DELETE_SESSION_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().Str("session_id", sessionID).Msg("Session deleted successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "deleted",
		"message":    "Session deleted successfully",
	})
}

// ConnectSession conecta uma sessão WhatsApp
// POST /sessions/:sessionId/connect
func (h *SessionHandler) ConnectSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "connecting",
		"message":    "Session connection endpoint",
	})
}

// DisconnectSession desconecta uma sessão WhatsApp
// POST /sessions/:sessionId/disconnect
func (h *SessionHandler) DisconnectSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "disconnected",
		"message":    "Session disconnection endpoint",
	})
}

// GetSessionStatus obtém o status de uma sessão
// GET /sessions/:sessionId/status
func (h *SessionHandler) GetSessionStatus(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "active",
		"message":    "Session status endpoint",
	})
}

// GetSessionQRCode obtém o QR code de uma sessão
// GET /sessions/:sessionId/qr
func (h *SessionHandler) GetSessionQRCode(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"qr_code":    "sample_qr_code_data",
		"message":    "QR code endpoint",
	})
}

// GetSessionStats obtém estatísticas de uma sessão
// GET /sessions/:sessionId/stats
func (h *SessionHandler) GetSessionStats(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id":        sessionID,
		"messages_sent":     0,
		"messages_received": 0,
		"message":           "Session stats endpoint",
	})
}

// GetActiveConnections obtém todas as conexões ativas (apenas admin)
// GET /sessions/active
func (h *SessionHandler) GetActiveConnections(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"active_connections": []fiber.Map{},
		"total":              0,
		"message":            "Active connections endpoint",
	})
}

// hasSessionAccess verifica se o usuário tem acesso à sessão
func (h *SessionHandler) hasSessionAccess(c *fiber.Ctx, sessionID string) bool {
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

// LogoutSession faz logout de uma sessão WhatsApp
// POST /sessions/:sessionId/logout
func (h *SessionHandler) LogoutSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "logged_out",
		"message":    "Session logout endpoint",
	})
}

// PairPhone inicia pareamento por telefone
// POST /sessions/:sessionId/pairphone
func (h *SessionHandler) PairPhone(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req struct {
		PhoneNumber string `json:"phone_number"`
	}

	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar número de telefone
	if req.PhoneNumber == "" {
		return h.sendError(c, "phone_number is required", "MISSING_FIELD", fiber.StatusBadRequest)
	}

	return c.JSON(fiber.Map{
		"session_id":   sessionID,
		"phone_number": req.PhoneNumber,
		"pairing_code": "123456",
		"message":      "Phone pairing endpoint - use pairing code to complete authentication",
		"initiated_at": time.Now(),
	})
}

// SetProxy configura proxy para uma sessão
// POST /sessions/:sessionId/proxy
func (h *SessionHandler) SetProxy(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req struct {
		Enabled  bool   `json:"enabled"`
		Type     string `json:"type"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"proxy": fiber.Map{
			"enabled": req.Enabled,
			"type":    req.Type,
			"host":    req.Host,
			"port":    req.Port,
		},
		"message": "Proxy configuration endpoint",
	})
}

// GetProxy obtém configuração de proxy de uma sessão
// GET /sessions/:sessionId/proxy
func (h *SessionHandler) GetProxy(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"proxy": fiber.Map{
			"enabled": false,
			"type":    "",
			"host":    "",
			"port":    0,
		},
		"message": "Proxy configuration endpoint",
	})
}

// TestProxy testa conectividade do proxy de uma sessão
// POST /sessions/:sessionId/proxy/test
func (h *SessionHandler) TestProxy(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"test_result": fiber.Map{
			"success":       true,
			"response_time": "150ms",
			"ip_address":    "192.168.1.100",
		},
		"message": "Proxy test endpoint",
	})
}

// generateToken gera um token simples para a sessão
func (h *SessionHandler) generateToken() string {
	return "token_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// sendError envia uma resposta de erro JSON
func (h *SessionHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	errorResp := fiber.Map{
		"error":   code,
		"message": message,
		"code":    status,
	}

	return c.Status(status).JSON(errorResp)
}
