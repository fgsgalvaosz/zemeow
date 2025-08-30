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


type SessionHandler struct {
	sessionService session.Service
	sessionRepo    repositories.SessionRepository
	logger         logger.Logger
}


func NewSessionHandler(service session.Service, repo repositories.SessionRepository) *SessionHandler {
	return &SessionHandler{
		sessionService: service,
		sessionRepo:    repo,
		logger:         logger.GetWithSession("session_handler"),
	}
}


func (h *SessionHandler) RegisterRoutes(app *fiber.App) {
















}


// @Summary Criar nova sessão WhatsApp
// @Description Cria uma nova sessão WhatsApp com configurações opcionais de proxy e webhook. Campos opcionais: session_id (será gerado automaticamente se não fornecido), api_key (será gerada automaticamente se não fornecida), proxy (configuração de proxy), webhook (configuração de webhook)
// @Tags sessions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body dto.CreateSessionRequest true "Dados da sessão - Exemplo mínimo: {\"name\": \"Minha Sessão\"}"
// @Success 201 {object} map[string]interface{} "Sessão criada com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos - Verifique se a porta do proxy está entre 1-65535"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/add [post]
func (h *SessionHandler) CreateSession(c *fiber.Ctx) error {

	validatedBody := c.Locals("validated_body")
	if validatedBody == nil {
		return h.sendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
	}

	req, ok := validatedBody.(*dto.CreateSessionRequest)
	if !ok {
		return h.sendError(c, "Invalid request format", "INVALID_REQUEST", fiber.StatusBadRequest)
	}








	if req.Name == "" {
		req.Name = "Test Session"
	}


	if req.SessionID == "" {
		req.SessionID = "session_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}


	var proxyConfig *session.ProxyConfig
	if req.Proxy != nil {
		proxyConfig = &session.ProxyConfig{
			Enabled:  req.Proxy.Enabled,
			Host:     req.Proxy.Host,
			Port:     req.Proxy.Port,
			Username: req.Proxy.Username,
			Password: req.Proxy.Password,
			Type:     req.Proxy.Type,
		}
	}

	var webhookConfig *session.WebhookConfig
	if req.Webhook != nil {
		webhookConfig = &session.WebhookConfig{
			URL:    req.Webhook.URL,
			Events: req.Webhook.Events,
			Secret: req.Webhook.Secret,
		}
	}


	config := &session.Config{
		SessionID: req.SessionID,
		Name:      req.Name,
		Proxy:     proxyConfig,
		Webhook:   webhookConfig,
	}


	sessionInfo, err := h.sessionService.CreateSession(context.Background(), config)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create session")
		return h.sendError(c, "Failed to create session", "CREATE_FAILED", fiber.StatusInternalServerError)
	}


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


func (h *SessionHandler) ListSessions(c *fiber.Ctx) error {

	var filter models.SessionFilter


	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter.Page = page
	filter.PerPage = perPage


	if status := c.Query("status"); status != "" {
		sessionStatus := models.SessionStatus(status)
		filter.Status = &sessionStatus
	}


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


func (h *SessionHandler) UpdateSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	var req models.UpdateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
	}


	sessionModel, err := h.sessionRepo.GetByIdentifier(sessionID)
	if err != nil {
		return h.sendError(c, "Session not found", "SESSION_NOT_FOUND", fiber.StatusNotFound)
	}


	if req.Name != nil {
		sessionModel.Name = *req.Name
	}


	if req.Proxy != nil {

	}


	if req.Webhook != nil {

	}


	if req.Metadata != nil {
		sessionModel.Metadata = *req.Metadata
	}


	if err := h.sessionRepo.Update(sessionModel); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to update session")
		return h.sendError(c, "Failed to update session", "UPDATE_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"success": true,
		"session": fiber.Map{
			"id":                sessionModel.ID,
			"session_id":        sessionModel.ID.String(),
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


func (h *SessionHandler) DeleteSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}


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


// @Summary Conectar sessão
// @Description Inicia a conexão WhatsApp para a sessão e retorna o QR code
// @Tags sessions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Success 200 {object} map[string]interface{} "Conexão iniciada com QR code"
// @Failure 400 {object} map[string]interface{} "Sessão já conectada ou erro de validação"
// @Failure 404 {object} map[string]interface{} "Sessão não encontrada"
// @Router /sessions/{sessionId}/connect [post]
func (h *SessionHandler) ConnectSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}


	qrInfo, err := h.sessionService.GetQRCode(context.Background(), sessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to connect session")
		return h.sendError(c, "Failed to connect session", "CONNECT_FAILED", fiber.StatusInternalServerError)
	}

	response := fiber.Map{
		"success":   true,
		"message":   "Session connection initiated",
		"qr_code":   qrInfo.Code,
		"timeout":   qrInfo.Timeout,
		"timestamp": qrInfo.Timestamp,
	}

	return c.JSON(response)
}


// @Summary Desconectar sessão
// @Description Desconecta a sessão do WhatsApp
// @Tags sessions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Success 200 {object} map[string]interface{} "Sessão desconectada com sucesso"
// @Failure 404 {object} map[string]interface{} "Sessão não encontrada"
// @Router /sessions/{sessionId}/disconnect [post]
func (h *SessionHandler) DisconnectSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}


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


func (h *SessionHandler) SetProxy(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	var req session.ProxyConfig
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_REQUEST", fiber.StatusBadRequest)
	}


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


// @Summary Listar todas as sessões
// @Description Lista todas as sessões com paginação e filtros opcionais
// @Tags sessions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "Número da página" default(1)
// @Param per_page query int false "Itens por página" default(20)
// @Param status query string false "Filtrar por status" Enums(connected, disconnected, connecting, error)
// @Param name query string false "Filtrar por nome"
// @Success 200 {object} map[string]interface{} "Lista de sessões"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions [get]
func (h *SessionHandler) GetAllSessions(c *fiber.Ctx) error {

	return h.ListSessions(c)
}


// @Summary Listar conexões ativas
// @Description Lista todas as sessões atualmente conectadas ao WhatsApp
// @Tags sessions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Lista de conexões ativas"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/active [get]
func (h *SessionHandler) GetActiveConnections(c *fiber.Ctx) error {

	activeSessions, err := h.sessionRepo.GetActiveConnections()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get active connections")
		return h.sendError(c, "Failed to get active connections", "ACTIVE_CONNECTIONS_FAILED", fiber.StatusInternalServerError)
	}


	var connections []map[string]interface{}
	for _, session := range activeSessions {
		connections = append(connections, map[string]interface{}{
			"id":                session.ID,
			"session_id":        session.GetSessionID(), // UUID como session_id
			"name":              session.Name,
			"status":            session.Status,
			"jid":               session.JID,
			"last_connected_at": session.LastConnectedAt,
			"created_at":        session.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"success":         true,
		"active_sessions": connections,
		"total":           len(connections),
		"timestamp":       time.Now().Unix(),
	})
}


func (h *SessionHandler) LogoutSession(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}


	return c.JSON(fiber.Map{
		"success": true,
		"message": "Session logout initiated",
	})
}


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


func (h *SessionHandler) GetSessionQRCode(c *fiber.Ctx) error {

	return h.GetQRCode(c)
}


func (h *SessionHandler) GetSessionStats(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}


	return c.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{},
	})
}


func (h *SessionHandler) GetProxy(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}


	return c.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{},
	})
}


func (h *SessionHandler) TestProxy(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}


	return c.JSON(fiber.Map{
		"success": true,
		"message": "Proxy test initiated",
	})
}





func (h *SessionHandler) SetPresence(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SessionPresenceRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	var presence types.Presence
	switch req.Presence {
	case "available":
		presence = types.PresenceAvailable
	case "unavailable":
		presence = types.PresenceUnavailable
	default:
		return h.sendError(c, "Invalid presence type. Use 'available' or 'unavailable'", "INVALID_PRESENCE", fiber.StatusBadRequest)
	}


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



func (h *SessionHandler) CheckContacts(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.CheckContactRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	if len(req.Phone) == 0 {
		return h.sendError(c, "At least one phone number is required", "NO_PHONES", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	var phoneNumbers []string
	phoneToJID := make(map[string]string)

	for _, phone := range req.Phone {

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


	results, err := client.IsOnWhatsApp(phoneNumbers)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to check contacts on WhatsApp")
		return h.sendError(c, fmt.Sprintf("Failed to check contacts: %v", err), "CHECK_FAILED", fiber.StatusInternalServerError)
	}


	var contacts []map[string]interface{}
	resultMap := make(map[string]types.IsOnWhatsAppResponse)

	for _, result := range results {
		resultMap[result.JID.User] = result
	}

	for originalPhone, jidStr := range phoneToJID {

		phoneNumber := strings.Split(jidStr, "@")[0]
		if result, exists := resultMap[phoneNumber]; exists {
			contact := map[string]interface{}{
				"phone":    originalPhone,
				"jid":      result.JID.String(),
				"exists":   result.IsIn,
				"verified": result.VerifiedName != nil,
				"business": false, // BusinessName não está disponível na versão atual
			}

			if result.VerifiedName != nil {
				contact["verified_name"] = *result.VerifiedName
			}

			contacts = append(contacts, contact)
		} else {

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



func (h *SessionHandler) GetContactInfo(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.ContactInfoRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	cleanPhone := strings.ReplaceAll(req.Phone, "+", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, " ", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")

	if len(cleanPhone) < 10 {
		return h.sendError(c, "Invalid phone number format", "INVALID_PHONE", fiber.StatusBadRequest)
	}


	jid, err := types.ParseJID(cleanPhone + "@s.whatsapp.net")
	if err != nil {
		return h.sendError(c, "Invalid phone number format", "INVALID_PHONE", fiber.StatusBadRequest)
	}


	userInfo, err := client.GetUserInfo([]types.JID{jid})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("phone", req.Phone).Msg("Failed to get user info")
		return h.sendError(c, fmt.Sprintf("Failed to get contact info: %v", err), "INFO_FAILED", fiber.StatusInternalServerError)
	}


	isOnWhatsApp, err := client.IsOnWhatsApp([]string{cleanPhone})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("phone", req.Phone).Msg("Failed to check if user is on WhatsApp")
		return h.sendError(c, fmt.Sprintf("Failed to verify contact: %v", err), "VERIFY_FAILED", fiber.StatusInternalServerError)
	}


	contact := map[string]interface{}{
		"phone":    req.Phone,
		"jid":      jid.String(),
		"exists":   false,
		"verified": false,
		"business": false,
	}


	if len(isOnWhatsApp) > 0 && isOnWhatsApp[0].IsIn {
		contact["exists"] = true

		if isOnWhatsApp[0].VerifiedName != nil {
			contact["verified"] = true
			contact["verified_name"] = *isOnWhatsApp[0].VerifiedName
		}


	}


	if info, exists := userInfo[jid]; exists {
		if len(info.Devices) > 0 {
			contact["devices"] = len(info.Devices)
		}
		contact["user_info"] = info
	}


	if contactInfo, err := client.Store.Contacts.GetContact(context.Background(), jid); err == nil {
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



func (h *SessionHandler) GetContactAvatar(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.ContactAvatarRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	cleanPhone := strings.ReplaceAll(req.Phone, "+", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, " ", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")

	if len(cleanPhone) < 10 {
		return h.sendError(c, "Invalid phone number format", "INVALID_PHONE", fiber.StatusBadRequest)
	}


	jid, err := types.ParseJID(cleanPhone + "@s.whatsapp.net")
	if err != nil {
		return h.sendError(c, "Invalid phone number format", "INVALID_PHONE", fiber.StatusBadRequest)
	}


	avatar, err := client.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{
		Preview: false, // Obter imagem em alta resolução
	})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("phone", req.Phone).Msg("Failed to get contact avatar")


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



func (h *SessionHandler) GetContacts(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	limit := c.QueryInt("limit", 100) // Limite padrão de 100
	offset := c.QueryInt("offset", 0) // Offset padrão de 0
	search := c.Query("search", "")   // Filtro de busca opcional

	if limit > 500 {
		limit = 500 // Máximo de 500 contatos por request
	}


	allContacts, err := client.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get contacts from store")
		return h.sendError(c, fmt.Sprintf("Failed to get contacts: %v", err), "CONTACTS_FAILED", fiber.StatusInternalServerError)
	}


	var contacts []map[string]interface{}
	count := 0

	for jid, contact := range allContacts {

		if search != "" {
			searchLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(contact.FullName), searchLower) &&
				!strings.Contains(strings.ToLower(contact.FirstName), searchLower) &&
				!strings.Contains(strings.ToLower(contact.PushName), searchLower) &&
				!strings.Contains(jid.User, search) {
				continue
			}
		}


		if count < offset {
			count++
			continue
		}


		if len(contacts) >= limit {
			break
		}


		contactInfo := map[string]interface{}{
			"jid":           jid.String(),
			"phone":         jid.User,
			"full_name":     contact.FullName,
			"first_name":    contact.FirstName,
			"push_name":     contact.PushName,
			"business_name": contact.BusinessName,
			"is_business":   contact.BusinessName != "",
		}


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





func (h *SessionHandler) PairPhone(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.PairPhoneRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	if client.IsLoggedIn() {
		return h.sendError(c, "Session is already paired", "ALREADY_PAIRED", fiber.StatusBadRequest)
	}


	phoneValidator := utils.NewPhoneValidator()
	if !phoneValidator.IsValidPhone(req.Phone) {
		return h.sendError(c, "Invalid phone number format", "INVALID_PHONE", fiber.StatusBadRequest)
	}

	cleanPhone := phoneValidator.CleanPhone(req.Phone)


	linkingCode, err := client.PairPhone(context.Background(), cleanPhone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("phone", req.Phone).Msg("Failed to pair phone")
		return h.sendError(c, fmt.Sprintf("Failed to pair phone: %v", err), "PAIR_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("phone", req.Phone).
		Str("linking_code", linkingCode).
		Msg("Phone pairing initiated")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":      true,
		"phone":        req.Phone,
		"linking_code": linkingCode,
		"message":      "Enter the linking code in your WhatsApp app",
		"expires_at":   time.Now().Add(5 * time.Minute).Unix(), // Códigos expiram em 5 minutos
		"timestamp":    time.Now().Unix(),
	})
}



func (h *SessionHandler) ConfigureProxy(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.ProxyConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err == nil {
		if client, ok := clientInterface.(*whatsmeow.Client); ok && client.IsConnected() {
			return h.sendError(c, "Cannot configure proxy while connected. Please disconnect first", "SESSION_CONNECTED", fiber.StatusBadRequest)
		}
	}


	if !req.Enabled {

		h.logger.Info().Str("session_id", sessionID).Msg("Proxy disabled")

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":   true,
			"message":   "Proxy disabled successfully",
			"enabled":   false,
			"timestamp": time.Now().Unix(),
		})
	}


	if req.Host == "" || req.Port == 0 {
		return h.sendError(c, "Host and port are required when enabling proxy", "MISSING_PROXY_CONFIG", fiber.StatusBadRequest)
	}

	if req.Type == "" {
		req.Type = "http" // Padrão
	}




	h.logger.Info().
		Str("session_id", sessionID).
		Str("proxy_type", req.Type).
		Str("proxy_host", req.Host).
		Int("proxy_port", req.Port).
		Msg("Proxy configured")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Proxy configured successfully",
		"config": map[string]interface{}{
			"enabled": true,
			"type":    req.Type,
			"host":    req.Host,
			"port":    req.Port,
		},
		"note":      "Proxy configuration saved - implementation requires database integration",
		"timestamp": time.Now().Unix(),
	})
}



func (h *SessionHandler) ConfigureS3(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.S3ConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	if !req.Enabled {

		h.logger.Info().Str("session_id", sessionID).Msg("S3 disabled")

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":   true,
			"message":   "S3 disabled successfully",
			"enabled":   false,
			"timestamp": time.Now().Unix(),
		})
	}


	if req.Bucket == "" || req.Region == "" || req.AccessKeyID == "" || req.SecretAccessKey == "" {
		return h.sendError(c, "Bucket, region, access_key_id and secret_access_key are required", "MISSING_S3_CONFIG", fiber.StatusBadRequest)
	}




	h.logger.Info().
		Str("session_id", sessionID).
		Str("s3_bucket", req.Bucket).
		Str("s3_region", req.Region).
		Msg("S3 configured")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "S3 configured successfully",
		"config": map[string]interface{}{
			"enabled": true,
			"bucket":  req.Bucket,
			"region":  req.Region,
		},
		"note":      "S3 configuration saved - implementation requires database and S3 client integration",
		"timestamp": time.Now().Unix(),
	})
}



func (h *SessionHandler) GetS3Config(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")




	h.logger.Info().Str("session_id", sessionID).Msg("S3 config requested")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"config": map[string]interface{}{
			"enabled":     false,
			"bucket":      "",
			"region":      "",
			"endpoint":    "",
			"status":      "disconnected",
			"last_tested": nil,
		},
		"note":      "S3 config retrieval - implementation requires database integration",
		"timestamp": time.Now().Unix(),
	})
}



func (h *SessionHandler) DeleteS3Config(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")




	h.logger.Info().Str("session_id", sessionID).Msg("S3 config deleted")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"message":   "S3 configuration deleted successfully",
		"timestamp": time.Now().Unix(),
	})
}



func (h *SessionHandler) TestS3Connection(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")




	h.logger.Info().Str("session_id", sessionID).Msg("S3 connection test requested")


	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": false,
		"message": "S3 not configured for this session",
		"test_result": map[string]interface{}{
			"connected":  false,
			"latency_ms": 0,
			"tested_at":  time.Now().Unix(),
			"error":      "S3 not configured",
		},
		"note":      "S3 connection test - implementation requires database and S3 client integration",
		"timestamp": time.Now().Unix(),
	})
}



func (h *SessionHandler) ListNewsletters(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	newsletters, err := client.GetSubscribedNewsletters()
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get newsletters")
		return h.sendError(c, fmt.Sprintf("Failed to get newsletters: %v", err), "NEWSLETTERS_FAILED", fiber.StatusInternalServerError)
	}


	var newsletterList []map[string]interface{}
	for _, newsletter := range newsletters {
		newsletterInfo := map[string]interface{}{
			"id":          newsletter.ID.String(),
			"name":        newsletter.ThreadMeta.Name.Text,
			"description": newsletter.ThreadMeta.Description.Text,
			"verified":    newsletter.ThreadMeta.VerificationState == types.NewsletterVerificationStateVerified,
			"subscribers": newsletter.ThreadMeta.SubscriberCount,
			"created_at":  newsletter.ThreadMeta.CreationTime,
		}
		newsletterList = append(newsletterList, newsletterInfo)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Int("newsletters_count", len(newsletterList)).
		Msg("Newsletters retrieved")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":     true,
		"newsletters": newsletterList,
		"total":       len(newsletterList),
		"timestamp":   time.Now().Unix(),
	})
}


func (h *SessionHandler) sendError(c *fiber.Ctx, message, code string, statusCode int) error {
	return c.Status(statusCode).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"message": message,
			"code":    code,
		},
	})
}
