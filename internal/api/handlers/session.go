package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/service/auth"
	"github.com/felipe/zemeow/internal/service/session"
	"github.com/rs/zerolog"
)

// SessionHandler gerencia endpoints de sessões WhatsApp
type SessionHandler struct {
	sessionManager *session.Manager
	authService    *auth.AuthService
	logger         zerolog.Logger
}

// NewSessionHandler cria uma nova instância do handler de sessões
func NewSessionHandler(
	sessionManager *session.Manager,
	authService *auth.AuthService,
) *SessionHandler {
	return &SessionHandler{
		sessionManager: sessionManager,
		authService:    authService,
		logger:         zerolog.New(nil).With().Str("component", "session_handler").Logger(),
	}
}

// CreateSession cria uma nova sessão WhatsApp
// POST /sessions
func (h *SessionHandler) CreateSession(c *fiber.Ctx) error {
	// Verificar permissões de admin
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || !authCtx.IsAdmin {
		return h.sendError(c, "Admin access required", "ADMIN_REQUIRED", fiber.StatusForbidden)
	}

	var req struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
	}

	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar campos obrigatórios
	if req.SessionID == "" {
		return h.sendError(c, "session_id is required", "MISSING_FIELD", fiber.StatusBadRequest)
	}

	if req.Name == "" {
		req.Name = req.SessionID
	}

	// Gerar token para a sessão
	token := h.generateToken()

	// Criar sessão
	session, err := h.sessionManager.CreateSession(req.SessionID, req.Name, token)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", req.SessionID).Msg("Failed to create session")
		return h.sendError(c, "Failed to create session", "CREATE_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().Str("session_id", req.SessionID).Msg("Session created successfully")
	return c.Status(fiber.StatusCreated).JSON(session)
}

// GetSession obtém informações de uma sessão específica
// GET /sessions/:sessionId
func (h *SessionHandler) GetSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	session, err := h.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get session")
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	return c.Status(fiber.StatusOK).JSON(session)
}

// GetAllSessions lista todas as sessões (apenas admin)
// GET /sessions
func (h *SessionHandler) GetAllSessions(c *fiber.Ctx) error {
	// Verificar permissões de admin
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || authCtx.Role != auth.RoleAdmin {
		return h.sendError(c, "Admin access required", "ADMIN_REQUIRED", fiber.StatusForbidden)
	}

	sessions, err := h.sessionRepo.GetAll()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get all sessions")
		return h.sendError(c, "Failed to retrieve sessions", "RETRIEVAL_FAILED", fiber.StatusInternalServerError)
	}

	response := models.SessionListResponse{
		Sessions: sessions,
		Total:    len(sessions),
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// UpdateSession atualiza uma sessão existente
// PUT /sessions/:sessionId
func (h *SessionHandler) UpdateSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão (admin ou própria sessão)
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	var req models.UpdateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar dados da requisição
	if err := req.Validate(); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	// Obter sessão atual
	session, err := h.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	// Atualizar campos
	if req.Name != nil {
		session.Name = *req.Name
	}
	if req.Description != nil {
		session.Description = *req.Description
	}
	if req.Config != nil {
		session.Config = *req.Config
	}
	session.UpdatedAt = time.Now()

	// Salvar alterações
	if err := h.sessionRepo.Update(session); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to update session")
		return h.sendError(c, "Failed to update session", "UPDATE_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().Str("session_id", sessionID).Msg("Session updated successfully")
	return c.Status(fiber.StatusOK).JSON(session)
}

// DeleteSession remove uma sessão
// DELETE /sessions/:sessionId
func (h *SessionHandler) DeleteSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar permissões de admin
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || authCtx.Role != auth.RoleAdmin {
		return h.sendError(c, "Admin access required", "ADMIN_REQUIRED", fiber.StatusForbidden)
	}

	// Verificar se a sessão existe
	exists, err := h.sessionRepo.Exists(sessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to check session existence")
		return h.sendError(c, "Internal server error", "INTERNAL_ERROR", fiber.StatusInternalServerError)
	}
	if !exists {
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	// Desconectar sessão se estiver conectada
	if err := h.whatsappMgr.DisconnectSession(sessionID); err != nil {
		h.logger.Warn().Err(err).Str("session_id", sessionID).Msg("Failed to disconnect session before deletion")
	}

	// Remover sessão do banco de dados
	if err := h.sessionRepo.DeleteBySessionID(sessionID); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to delete session")
		return h.sendError(c, "Failed to delete session", "DELETE_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().Str("session_id", sessionID).Msg("Session deleted successfully")

	response := fiber.Map{
		"message":    "Session deleted successfully",
		"session_id": sessionID,
		"deleted_at": time.Now(),
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// ConnectSession conecta uma sessão WhatsApp
// POST /sessions/:sessionId/connect
func (h *SessionHandler) ConnectSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Verificar se a sessão existe
	session, err := h.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	// Conectar sessão
	qrCode, err := h.whatsappMgr.ConnectSession(sessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to connect session")
		return h.sendError(c, "Failed to connect session", "CONNECTION_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"message":    "Session connection initiated",
		"session_id": sessionID,
		"status":     session.Status,
	}

	// Incluir QR code se disponível
	if qrCode != "" {
		response["qr_code"] = qrCode
	}

	h.logger.Info().Str("session_id", sessionID).Msg("Session connection initiated")
	return c.Status(fiber.StatusOK).JSON(response)
}

// DisconnectSession desconecta uma sessão WhatsApp
// POST /sessions/:sessionId/disconnect
func (h *SessionHandler) DisconnectSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Desconectar sessão
	if err := h.whatsappMgr.DisconnectSession(sessionID); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to disconnect session")
		return h.sendError(c, "Failed to disconnect session", "DISCONNECTION_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"message":         "Session disconnected successfully",
		"session_id":      sessionID,
		"disconnected_at": time.Now(),
	}

	h.logger.Info().Str("session_id", sessionID).Msg("Session disconnected successfully")
	return c.Status(fiber.StatusOK).JSON(response)
}

// GetSessionStatus obtém o status de uma sessão
// GET /sessions/:sessionId/status
func (h *SessionHandler) GetSessionStatus(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Obter informações da sessão
	session, err := h.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	// Obter status do WhatsApp Manager
	connectionInfo := h.whatsappMgr.GetSessionInfo(sessionID)

	response := models.SessionInfoResponse{
		SessionID:      session.SessionID,
		Name:           session.Name,
		Status:         session.Status,
		JID:            session.JID,
		ConnectedAt:    session.ConnectedAt,
		LastSeenAt:     session.LastSeenAt,
		ConnectionInfo: connectionInfo,
		UpdatedAt:      session.UpdatedAt,
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetSessionQRCode obtém o QR code de uma sessão
// GET /sessions/:sessionId/qr
func (h *SessionHandler) GetSessionQRCode(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Obter QR code
	qrCode := h.whatsappMgr.GetQRCode(sessionID)
	if qrCode == "" {
		return h.sendError(c, "QR code not available", "QR_NOT_AVAILABLE", fiber.StatusNotFound)
	}

	response := fiber.Map{
		"session_id":   sessionID,
		"qr_code":      qrCode,
		"generated_at": time.Now(),
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetSessionStats obtém estatísticas de uma sessão
// GET /sessions/:sessionId/stats
func (h *SessionHandler) GetSessionStats(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Obter estatísticas
	stats := h.whatsappMgr.GetSessionStats(sessionID)
	if stats == nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_CONNECTED", fiber.StatusNotFound)
	}

	return c.Status(fiber.StatusOK).JSON(stats)
}

// GetActiveConnections obtém todas as conexões ativas (apenas admin)
// GET /sessions/active
func (h *SessionHandler) GetActiveConnections(c *fiber.Ctx) error {
	// Verificar permissões de admin
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil || authCtx.Role != auth.RoleAdmin {
		return h.sendError(c, "Admin access required", "ADMIN_REQUIRED", fiber.StatusForbidden)
	}

	// Obter conexões ativas do repositório
	activeSessions, err := h.sessionRepo.GetActiveConnections()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get active connections")
		return h.sendError(c, "Failed to retrieve active connections", "RETRIEVAL_FAILED", fiber.StatusInternalServerError)
	}

	// Obter informações detalhadas do WhatsApp Manager
	var connections []fiber.Map
	for _, session := range activeSessions {
		connectionInfo := h.whatsappMgr.GetSessionInfo(session.SessionID)
		connections = append(connections, fiber.Map{
			"session_id":      session.SessionID,
			"name":            session.Name,
			"jid":             session.JID,
			"status":          session.Status,
			"connected_at":    session.ConnectedAt,
			"last_seen_at":    session.LastSeenAt,
			"connection_info": connectionInfo,
		})
	}

	response := fiber.Map{
		"active_connections": connections,
		"total":              len(connections),
		"retrieved_at":       time.Now(),
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// hasSessionAccess verifica se o usuário tem acesso à sessão
func (h *SessionHandler) hasSessionAccess(c *fiber.Ctx, sessionID string) bool {
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
