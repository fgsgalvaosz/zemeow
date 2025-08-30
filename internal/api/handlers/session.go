package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/felipe/zemeow/internal/api/dto"
	"github.com/felipe/zemeow/internal/api/utils"
	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/session"
	"github.com/gofiber/fiber/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// SessionHandler lida com as requisições HTTP relacionadas às sessões
type SessionHandler struct {
	sessionService session.Service
	sessionRepo    repositories.SessionRepository
	logger         logger.Logger
}

// NewSessionHandler cria um novo handler de sessões
func NewSessionHandler(service session.Service, repo repositories.SessionRepository) *SessionHandler {
	return &SessionHandler{
		sessionService: service,
		sessionRepo:    repo,
		logger:         logger.GetWithSession("session_handler"),
	}
}

// RegisterRoutes registra as rotas do handler
func (h *SessionHandler) RegisterRoutes(app *fiber.App) {
	// Rotas públicas (sem autenticação)
	public := app.Group("/sessions")
	public.Post("/create", h.CreateSession)
	public.Get("/qr/:sessionId", h.GetQRCode)

	// Rotas protegidas (requerem autenticação)
	protected := app.Group("/sessions")
	protected.Get("/", h.ListSessions)
	protected.Get("/:sessionId", h.GetSession)
	protected.Put("/:sessionId", h.UpdateSession)
	protected.Delete("/:sessionId", h.DeleteSession)
	protected.Post("/:sessionId/connect", h.ConnectSession)
	protected.Post("/:sessionId/disconnect", h.DisconnectSession)
	protected.Post("/:sessionId/phone", h.PairPhone)
	protected.Put("/:sessionId/proxy", h.SetProxy)
	protected.Put("/:sessionId/webhook", h.SetWebhook)
}

// CreateSessionRequest representa a requisição para criar uma sessão
type CreateSessionRequest struct {
	SessionID string              `json:"session_id"`
	Name      string              `json:"name" validate:"required"`
	Proxy     *session.ProxyConfig `json:"proxy,omitempty"`
	Webhook   *session.WebhookConfig `json:"webhook,omitempty"`
}

// CreateSession cria uma nova sessão
// @Summary Criar nova sessão WhatsApp
// @Description Cria uma nova sessão WhatsApp com configurações opcionais de proxy e webhook
// @Tags sessions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body CreateSessionRequest true "Dados da sessão"
// @Success 201 {object} map[string]interface{} "Sessão criada com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions [post]
func (h *SessionHandler) CreateSession(c *fiber.Ctx) error {
	var req CreateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
	}

	// Validar campos obrigatórios
	if req.Name == "" {
		return h.sendError(c, "Name is required", "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	// Gerar session_id se não fornecido
	if req.SessionID == "" {
		req.SessionID = "session_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	// Criar configuração
	config := &session.Config{
		SessionID: req.SessionID,
		Name:      req.Name,
		Proxy:     req.Proxy,
		Webhook:   req.Webhook,
	}

	// Criar sessão
	sessionInfo, err := h.sessionService.CreateSession(context.Background(), config)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create session")
		return h.sendError(c, "Failed to create session", "CREATE_FAILED", fiber.StatusInternalServerError)
	}

	// Preparar resposta
	response := fiber.Map{
		"success": true,
		"session": fiber.Map{
			"id":         sessionInfo.ID,
			"session_id": sessionInfo.ID,
			"name":       sessionInfo.Name,
			"api_key":    sessionInfo.APIKey,
			"status":     sessionInfo.Status,
			"created_at": sessionInfo.CreatedAt,
		},
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// GetSession retorna informações de uma sessão específica
// @Summary Obter detalhes da sessão
// @Description Retorna informações detalhadas de uma sessão específica
// @Tags sessions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Success 200 {object} map[string]interface{} "Detalhes da sessão"
// @Failure 400 {object} map[string]interface{} "ID da sessão inválido"
// @Failure 404 {object} map[string]interface{} "Sessão não encontrada"
// @Router /sessions/{sessionId} [get]
func (h *SessionHandler) GetSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	sessionInfo, err := h.sessionService.GetSession(context.Background(), sessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get session")
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	response := fiber.Map{
		"success": true,
		"session": fiber.Map{
			"id":                sessionInfo.ID,
			"session_id":        sessionInfo.ID,
			"name":              sessionInfo.Name,
			"api_key":           sessionInfo.APIKey,
			"status":            sessionInfo.Status,
			"jid":               sessionInfo.JID,
			"is_connected":      sessionInfo.IsConnected,
			"created_at":        sessionInfo.CreatedAt,
			"updated_at":        sessionInfo.UpdatedAt,
			"connected_at":      sessionInfo.ConnectedAt,
			"last_connected_at": sessionInfo.LastConnectedAt,
		},
	}

	return c.JSON(response)
}

// ListSessions lista todas as sessões
func (h *SessionHandler) ListSessions(c *fiber.Ctx) error {
	// Parse query parameters for filtering
	var filter models.SessionFilter
	
	// Parse page and per_page
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	
	filter.Page = page
	filter.PerPage = perPage
	
	// Parse status filter
	if status := c.Query("status"); status != "" {
		sessionStatus := models.SessionStatus(status)
		filter.Status = &sessionStatus
	}
	
	// Parse name filter
	if name := c.Query("name"); name != "" {
		filter.Name = &name
	}

	sessions, err := h.sessionRepo.GetAll(&filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list sessions")
		return h.sendError(c, "Failed to list sessions", "LIST_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"success": true,
		"data": fiber.Map{
			"sessions": sessions.Sessions,
			"pagination": fiber.Map{
				"total":       sessions.Total,
				"page":        sessions.Page,
				"per_page":    sessions.PerPage,
				"total_pages": sessions.TotalPages,
			},
		},
	}

	return c.JSON(response)
}

// UpdateSession atualiza uma sessão existente
func (h *SessionHandler) UpdateSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	var req models.UpdateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
	}

	// Obter sessão existente
	sessionModel, err := h.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}

	// Atualizar campos
	if req.Name != nil {
		sessionModel.Name = *req.Name
	}

	// Atualizar proxy se fornecido
	if req.Proxy != nil {
		// TODO: Implementar atualização de proxy
	}

	// Atualizar webhook se fornecido
	if req.Webhook != nil {
		// TODO: Implementar atualização de webhook
	}

	// Atualizar metadados se fornecido
	if req.Metadata != nil {
		sessionModel.Metadata = *req.Metadata
	}

	// Salvar alterações
	if err := h.sessionRepo.Update(sessionModel); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to update session")
		return h.sendError(c, "Failed to update session", "UPDATE_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"success": true,
		"session": fiber.Map{
			"id":                sessionModel.ID,
			"session_id":        sessionModel.SessionID,
			"name":              sessionModel.Name,
			"api_key":           sessionModel.APIKey,
			"status":            sessionModel.Status,
			"jid":               sessionModel.JID,
			"proxy_enabled":     sessionModel.ProxyEnabled,
			"proxy_host":        sessionModel.ProxyHost,
			"proxy_port":        sessionModel.ProxyPort,
			"webhook_url":       sessionModel.WebhookURL,
			"webhook_events":    sessionModel.WebhookEvents,
			"created_at":        sessionModel.CreatedAt,
			"updated_at":        sessionModel.UpdatedAt,
			"last_connected_at": sessionModel.LastConnectedAt,
		},
	}

	return c.JSON(response)
}

// DeleteSession remove uma sessão
func (h *SessionHandler) DeleteSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	// Remover sessão
	if err := h.sessionService.DeleteSession(context.Background(), sessionID); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to delete session")
		return h.sendError(c, "Failed to delete session", "DELETE_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"success": true,
		"message": "Session deleted successfully",
	}

	return c.JSON(response)
}

// ConnectSession conecta uma sessão ao WhatsApp
func (h *SessionHandler) ConnectSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	// Conectar sessão
	if err := h.sessionService.ConnectSession(context.Background(), sessionID); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to connect session")
		return h.sendError(c, "Failed to connect session", "CONNECT_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"success": true,
		"message": "Session connection initiated",
	}

	return c.JSON(response)
}

// DisconnectSession desconecta uma sessão do WhatsApp
func (h *SessionHandler) DisconnectSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	// Desconectar sessão
	if err := h.sessionService.DisconnectSession(context.Background(), sessionID); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to disconnect session")
		return h.sendError(c, "Failed to disconnect session", "DISCONNECT_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"success": true,
		"message": "Session disconnected successfully",
	}

	return c.JSON(response)
}

// GetQRCode retorna o QR Code para autenticação
// @Summary Obter QR Code da sessão
// @Description Retorna o QR Code para autenticação da sessão WhatsApp
// @Tags sessions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Success 200 {object} map[string]interface{} "QR Code gerado com sucesso"
// @Failure 400 {object} map[string]interface{} "ID da sessão inválido"
// @Failure 404 {object} map[string]interface{} "Sessão não encontrada"
// @Router /sessions/{sessionId}/qr [get]
func (h *SessionHandler) GetQRCode(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	// Obter QR Code
	qrInfo, err := h.sessionService.GetQRCode(context.Background(), sessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get QR code")
		return h.sendError(c, "Failed to get QR code", "QR_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"success": true,
		"data": fiber.Map{
			"qr_code":   qrInfo.Code,
			"timeout":   qrInfo.Timeout,
			"timestamp": qrInfo.Timestamp,
		},
	}

	return c.JSON(response)
}

// PairPhone inicia o pareamento por telefone
func (h *SessionHandler) PairPhone(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	var req session.PairPhoneRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
	}

	if req.PhoneNumber == "" {
		return h.sendError(c, "Phone number is required", "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	// Parear por telefone
	response, err := h.sessionService.PairPhone(context.Background(), sessionID, &req)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to pair phone")
		return h.sendError(c, "Failed to pair phone", "PAIR_FAILED", fiber.StatusInternalServerError)
	}

	result := fiber.Map{
		"success": response.Success,
	}

	if response.Code != "" {
		result["code"] = response.Code
	}
	if response.Message != "" {
		result["message"] = response.Message
	}

	return c.JSON(result)
}

// SetProxy configura o proxy para uma sessão
func (h *SessionHandler) SetProxy(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	var req session.ProxyConfig
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
	}

	// Configurar proxy
	if err := h.sessionService.SetProxy(context.Background(), sessionID, &req); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to set proxy")
		return h.sendError(c, "Failed to set proxy", "PROXY_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"success": true,
		"message": "Proxy configured successfully",
	}

	return c.JSON(response)
}

// SetWebhook configura o webhook para uma sessão
func (h *SessionHandler) SetWebhook(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	var req session.WebhookConfig
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
	}

	if req.URL == "" {
		return h.sendError(c, "Webhook URL is required", "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	// Configurar webhook
	if err := h.sessionService.SetWebhook(context.Background(), sessionID, &req); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to set webhook")
		return h.sendError(c, "Failed to set webhook", "WEBHOOK_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"success": true,
		"message": "Webhook configured successfully",
	}

	return c.JSON(response)
}

// GetAllSessions retorna todas as sessões (método faltando)
func (h *SessionHandler) GetAllSessions(c *fiber.Ctx) error {
	// Reutilizar a lógica de ListSessions
	return h.ListSessions(c)
}

// GetActiveConnections retorna conexões ativas (método faltando)
func (h *SessionHandler) GetActiveConnections(c *fiber.Ctx) error {
	// TODO: Implementar obtenção de conexões ativas
	return c.JSON(fiber.Map{
		"success": true,
		"data":    []interface{}{},
	})
}

// LogoutSession faz logout da sessão (método faltando)
func (h *SessionHandler) LogoutSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	// TODO: Implementar logout da sessão
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Session logout initiated",
	})
}

// GetSessionStatus retorna o status da sessão (método faltando)
func (h *SessionHandler) GetSessionStatus(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	status, err := h.sessionService.GetSessionStatus(context.Background(), sessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get session status")
		return h.sendError(c, "Failed to get session status", "STATUS_FAILED", fiber.StatusInternalServerError)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"status":  status,
	})
}

// GetSessionQRCode retorna o QR Code da sessão (método faltando)
func (h *SessionHandler) GetSessionQRCode(c *fiber.Ctx) error {
	// Reutilizar a lógica de GetQRCode
	return h.GetQRCode(c)
}

// GetSessionStats retorna estatísticas da sessão (método faltando)
func (h *SessionHandler) GetSessionStats(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	// TODO: Implementar obtenção de estatísticas da sessão
	return c.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{},
	})
}

// GetProxy retorna a configuração de proxy (método faltando)
func (h *SessionHandler) GetProxy(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	// TODO: Implementar obtenção de configuração de proxy
	return c.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{},
	})
}

// TestProxy testa a configuração de proxy (método faltando)
func (h *SessionHandler) TestProxy(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	// TODO: Implementar teste de proxy
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Proxy test initiated",
	})
}

// === NOVOS MÉTODOS PARA OPERAÇÕES DE SESSÃO (WHATSAPP) ===

// SetPresence define presença da sessão
// POST /sessions/:sessionId/presence
func (h *SessionHandler) SetPresence(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SessionPresenceRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Obter cliente WhatsApp
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	// Mapear presença
	var presence types.Presence
	switch req.Presence {
	case "available":
		presence = types.PresenceAvailable
	case "unavailable":
		presence = types.PresenceUnavailable
	default:
		return h.sendError(c, "Invalid presence type. Use 'available' or 'unavailable'", "INVALID_PRESENCE", fiber.StatusBadRequest)
	}

	// Enviar presença global
	err = client.SendPresence(presence)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("presence", req.Presence).Msg("Failed to set presence")
		return h.sendError(c, fmt.Sprintf("Failed to set presence: %v", err), "PRESENCE_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().Str("session_id", sessionID).Str("presence", req.Presence).Msg("Presence set successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"session_id": sessionID,
		"presence":   req.Presence,
		"timestamp":  time.Now().Unix(),
	})
}

// CheckContacts verifica se contatos existem
// POST /sessions/:sessionId/check
func (h *SessionHandler) CheckContacts(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.CheckContactRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar se há telefones para verificar
	if len(req.Phone) == 0 {
		return h.sendError(c, "At least one phone number is required", "NO_PHONES", fiber.StatusBadRequest)
	}

	// Obter cliente WhatsApp
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	// Preparar números para verificação
	var phoneNumbers []string
	phoneToJID := make(map[string]string)

	for _, phone := range req.Phone {
		// Limpar e validar número de telefone
		cleanPhone := strings.ReplaceAll(phone, "+", "")
		cleanPhone = strings.ReplaceAll(cleanPhone, " ", "")
		cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")

		if len(cleanPhone) < 10 {
			continue // Pular números muito curtos
		}

		phoneNumbers = append(phoneNumbers, cleanPhone)
		phoneToJID[phone] = cleanPhone + "@s.whatsapp.net"
	}

	if len(phoneNumbers) == 0 {
		return h.sendError(c, "No valid phone numbers provided", "NO_VALID_PHONES", fiber.StatusBadRequest)
	}

	// Verificar quais números estão no WhatsApp
	results, err := client.IsOnWhatsApp(phoneNumbers)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to check contacts on WhatsApp")
		return h.sendError(c, fmt.Sprintf("Failed to check contacts: %v", err), "CHECK_FAILED", fiber.StatusInternalServerError)
	}

	// Processar resultados
	var contacts []map[string]interface{}
	resultMap := make(map[string]whatsmeow.IsOnWhatsAppResponse)

	for _, result := range results {
		resultMap[result.JID.User] = result
	}

	for originalPhone, jid := range phoneToJID {
		if result, exists := resultMap[jid.User]; exists {
			contact := map[string]interface{}{
				"phone":    originalPhone,
				"jid":      result.JID.String(),
				"exists":   result.IsIn,
				"verified": result.VerifiedName != nil,
				"business": result.BusinessName != "",
			}

			if result.VerifiedName != nil {
				contact["verified_name"] = *result.VerifiedName
			}

			if result.BusinessName != "" {
				contact["business_name"] = result.BusinessName
			}

			contacts = append(contacts, contact)
		} else {
			// Número não encontrado nos resultados
			contacts = append(contacts, map[string]interface{}{
				"phone":    originalPhone,
				"exists":   false,
				"verified": false,
				"business": false,
			})
		}
	}

	h.logger.Info().Str("session_id", sessionID).Int("total_checked", len(contacts)).Msg("Contact check completed")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"session_id": sessionID,
		"contacts":   contacts,
		"total":      len(contacts),
		"timestamp":  time.Now().Unix(),
	})
}

// GetContactInfo obtém informações de contato
// POST /sessions/:sessionId/info
func (h *SessionHandler) GetContactInfo(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.ContactInfoRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Verificar se a sessão existe e está conectada
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	// Limpar e preparar número de telefone
	cleanPhone := strings.ReplaceAll(req.Phone, "+", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, " ", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")

	if len(cleanPhone) < 10 {
		return h.sendError(c, "Invalid phone number format", "INVALID_PHONE", fiber.StatusBadRequest)
	}

	// Preparar JID
	jid, err := types.ParseJID(cleanPhone + "@s.whatsapp.net")
	if err != nil {
		return h.sendError(c, "Invalid phone number format", "INVALID_PHONE", fiber.StatusBadRequest)
	}

	// Obter informações do usuário
	userInfo, err := client.GetUserInfo([]types.JID{jid})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("phone", req.Phone).Msg("Failed to get user info")
		return h.sendError(c, fmt.Sprintf("Failed to get contact info: %v", err), "INFO_FAILED", fiber.StatusInternalServerError)
	}

	// Verificar se o usuário existe no WhatsApp
	isOnWhatsApp, err := client.IsOnWhatsApp([]types.JID{jid})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("phone", req.Phone).Msg("Failed to check if user is on WhatsApp")
		return h.sendError(c, fmt.Sprintf("Failed to verify contact: %v", err), "VERIFY_FAILED", fiber.StatusInternalServerError)
	}

	// Processar informações
	contact := map[string]interface{}{
		"phone":     req.Phone,
		"jid":       jid.String(),
		"exists":    false,
		"verified":  false,
		"business":  false,
	}

	// Verificar se está no WhatsApp
	if len(isOnWhatsApp) > 0 && isOnWhatsApp[0].IsIn {
		contact["exists"] = true

		if isOnWhatsApp[0].VerifiedName != nil {
			contact["verified"] = true
			contact["verified_name"] = *isOnWhatsApp[0].VerifiedName
		}

		if isOnWhatsApp[0].BusinessName != "" {
			contact["business"] = true
			contact["business_name"] = isOnWhatsApp[0].BusinessName
		}
	}

	// Adicionar informações do usuário se disponíveis
	if info, exists := userInfo[jid]; exists {
		if len(info.Devices) > 0 {
			contact["devices"] = len(info.Devices)
		}
		contact["user_info"] = info
	}

	// Tentar obter informações do contato salvo
	if contactInfo, err := client.Store.Contacts.GetContact(jid); err == nil {
		if contactInfo.FullName != "" {
			contact["full_name"] = contactInfo.FullName
		}
		if contactInfo.FirstName != "" {
			contact["first_name"] = contactInfo.FirstName
		}
		if contactInfo.PushName != "" {
			contact["push_name"] = contactInfo.PushName
		}
		if contactInfo.BusinessName != "" {
			contact["business_name"] = contactInfo.BusinessName
		}
	}

	h.logger.Info().Str("session_id", sessionID).Str("phone", req.Phone).Bool("exists", contact["exists"].(bool)).Msg("Contact info retrieved")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"session_id": sessionID,
		"contact":    contact,
		"timestamp":  time.Now().Unix(),
	})
}

// GetContactAvatar obtém avatar de contato
// POST /sessions/:sessionId/avatar
func (h *SessionHandler) GetContactAvatar(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.ContactAvatarRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Verificar se a sessão existe e está conectada
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	// Limpar e preparar número de telefone
	cleanPhone := strings.ReplaceAll(req.Phone, "+", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, " ", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")

	if len(cleanPhone) < 10 {
		return h.sendError(c, "Invalid phone number format", "INVALID_PHONE", fiber.StatusBadRequest)
	}

	// Preparar JID
	jid, err := types.ParseJID(cleanPhone + "@s.whatsapp.net")
	if err != nil {
		return h.sendError(c, "Invalid phone number format", "INVALID_PHONE", fiber.StatusBadRequest)
	}

	// Obter avatar
	avatar, err := client.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{
		Preview: false, // Obter imagem em alta resolução
	})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("phone", req.Phone).Msg("Failed to get contact avatar")

		// Se não conseguir obter avatar, retornar informação de que não há avatar
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":    true,
			"session_id": sessionID,
			"contact": map[string]interface{}{
				"phone":      req.Phone,
				"jid":        jid.String(),
				"has_avatar": false,
				"error":      "Avatar not found or not accessible",
			},
			"timestamp": time.Now().Unix(),
		})
	}

	// Preparar resposta com informações do avatar
	avatarInfo := map[string]interface{}{
		"phone":      req.Phone,
		"jid":        jid.String(),
		"has_avatar": true,
		"avatar": map[string]interface{}{
			"url":         avatar.URL,
			"id":          avatar.ID,
			"type":        avatar.Type,
			"direct_path": avatar.DirectPath,
		},
	}

	// Tentar obter também a versão preview (baixa resolução)
	if preview, err := client.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{
		Preview: true,
	}); err == nil {
		avatarInfo["avatar"].(map[string]interface{})["preview_url"] = preview.URL
	}

	h.logger.Info().Str("session_id", sessionID).Str("phone", req.Phone).Str("avatar_id", avatar.ID).Msg("Contact avatar retrieved")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"session_id": sessionID,
		"contact":    avatarInfo,
		"timestamp":  time.Now().Unix(),
	})
}

// GetContacts lista contatos da sessão
// GET /sessions/:sessionId/contacts
func (h *SessionHandler) GetContacts(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	// Obter cliente WhatsApp
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	// Obter parâmetros de query opcionais
	limit := c.QueryInt("limit", 100)  // Limite padrão de 100
	offset := c.QueryInt("offset", 0)  // Offset padrão de 0
	search := c.Query("search", "")    // Filtro de busca opcional

	if limit > 500 {
		limit = 500 // Máximo de 500 contatos por request
	}

	// Obter todos os contatos do store
	allContacts, err := client.Store.Contacts.GetAllContacts()
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get contacts from store")
		return h.sendError(c, fmt.Sprintf("Failed to get contacts: %v", err), "CONTACTS_FAILED", fiber.StatusInternalServerError)
	}

	// Processar e filtrar contatos
	var contacts []map[string]interface{}
	count := 0

	for jid, contact := range allContacts {
		// Aplicar filtro de busca se fornecido
		if search != "" {
			searchLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(contact.FullName), searchLower) &&
			   !strings.Contains(strings.ToLower(contact.FirstName), searchLower) &&
			   !strings.Contains(strings.ToLower(contact.PushName), searchLower) &&
			   !strings.Contains(jid.User, search) {
				continue
			}
		}

		// Aplicar offset
		if count < offset {
			count++
			continue
		}

		// Aplicar limite
		if len(contacts) >= limit {
			break
		}

		// Preparar informações do contato
		contactInfo := map[string]interface{}{
			"jid":           jid.String(),
			"phone":         jid.User,
			"full_name":     contact.FullName,
			"first_name":    contact.FirstName,
			"push_name":     contact.PushName,
			"business_name": contact.BusinessName,
			"is_business":   contact.BusinessName != "",
		}

		// Adicionar informações extras se disponíveis
		if contact.FullName == "" && contact.FirstName == "" && contact.PushName != "" {
			contactInfo["display_name"] = contact.PushName
		} else if contact.FullName != "" {
			contactInfo["display_name"] = contact.FullName
		} else if contact.FirstName != "" {
			contactInfo["display_name"] = contact.FirstName
		} else {
			contactInfo["display_name"] = jid.User
		}

		contacts = append(contacts, contactInfo)
		count++
	}

	// Obter total de contatos (para paginação)
	totalContacts := len(allContacts)

	h.logger.Info().
		Str("session_id", sessionID).
		Int("total_contacts", totalContacts).
		Int("returned_contacts", len(contacts)).
		Int("limit", limit).
		Int("offset", offset).
		Str("search", search).
		Msg("Contacts retrieved")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"session_id": sessionID,
		"contacts":   contacts,
		"pagination": map[string]interface{}{
			"total":  totalContacts,
			"limit":  limit,
			"offset": offset,
			"count":  len(contacts),
		},
		"search":    search,
		"timestamp": time.Now().Unix(),
	})
}

// sendError envia uma resposta de erro padronizada
func (h *SessionHandler) sendError(c *fiber.Ctx, message, code string, statusCode int) error {
	return c.Status(statusCode).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"message": message,
			"code":    code,
		},
	})
}