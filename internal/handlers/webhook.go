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
// @Router /sessions/{sessionId}/webhooks/find [get]
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
// @Router /sessions/{sessionId}/webhooks/set [post]
func (h *WebhookHandler) SetWebhook(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	if !utils.HasSessionAccess(c, sessionID) {
		return utils.SendAccessDeniedError(c)
	}

	var req dto.WebhookConfigRequest

	if err := c.BodyParser(&req); err != nil {
		return utils.SendInvalidJSONError(c)
	}

	validEvents := h.getAllAvailableEvents()
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

	} else {
		session.WebhookURL = nil
		session.WebhookEvents = nil
	}

	if err := h.sessionRepo.Update(session); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to update webhook configuration")
		return utils.SendError(c, "Failed to save webhook configuration", "WEBHOOK_SAVE_FAILED", fiber.StatusInternalServerError)
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
			"id":           time.Now().Unix(),
			"url":          req.URL,
			"events":       req.Events,
			"payload_mode": "processed", // Valor padrão fixo
			"active":       req.Active,
			"created_at":   time.Now().Unix(),
		},
		"timestamp": time.Now().Unix(),
	})
}

// @Summary Lista eventos disponíveis para webhook
// @Description Retorna lista completa de todos os eventos de webhook disponíveis por categoria
// @Tags webhooks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Success 200 {object} map[string]interface{} "Lista de eventos disponíveis"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Router /sessions/{sessionId}/webhooks/events [get]
func (h *WebhookHandler) GetWebhookEvents(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	if !utils.HasSessionAccess(c, sessionID) {
		return utils.SendAccessDeniedError(c)
	}

	events := h.getAllAvailableEvents()

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"events":     events,
		"total":      len(events),
		"categories": h.getEventCategories(),
		"payload_modes": map[string]interface{}{
			"raw": map[string]interface{}{
				"description": "Eventos originais da whatsmeow sem modificações",
				"format":      "Estrutura original da biblioteca whatsmeow",
				"benefit":     "Zero perda de dados, máxima compatibilidade",
			},
		},
		"timestamp": time.Now().Unix(),
	})
}

func (h *WebhookHandler) getAllAvailableEvents() []map[string]interface{} {
	return []map[string]interface{}{
		// Conexão
		{"name": "connected", "category": "connection", "description": "Cliente conectado com sucesso"},
		{"name": "disconnected", "category": "connection", "description": "Cliente desconectado"},
		{"name": "logged_out", "category": "connection", "description": "Cliente deslogado/despareado"},
		{"name": "connect_failure", "category": "connection", "description": "Falha na conexão"},
		{"name": "temporary_ban", "category": "connection", "description": "Banimento temporário"},
		{"name": "client_outdated", "category": "connection", "description": "Cliente desatualizado"},
		{"name": "cat_refresh_error", "category": "connection", "description": "Erro de refresh CAT"},
		{"name": "stream_error", "category": "connection", "description": "Erro no stream"},
		{"name": "stream_replaced", "category": "connection", "description": "Stream substituído"},
		{"name": "permanent_disconnect", "category": "connection", "description": "Desconexão permanente"},
		{"name": "manual_login_reconnect", "category": "connection", "description": "Reconexão de login manual"},
		
		// Mensagens
		{"name": "message", "category": "messages", "description": "Nova mensagem recebida"},
		{"name": "receipt", "category": "messages", "description": "Confirmação de entrega/leitura"},
		{"name": "undecryptable_message", "category": "messages", "description": "Mensagem não descriptografável"},
		{"name": "delete_for_me", "category": "messages", "description": "Mensagem deletada localmente"},
		{"name": "history_sync", "category": "messages", "description": "Sincronização de histórico"},
		{"name": "offline_sync_preview", "category": "messages", "description": "Preview de sincronização offline"},
		{"name": "offline_sync_completed", "category": "messages", "description": "Sincronização offline concluída"},
		{"name": "media_retry", "category": "messages", "description": "Nova tentativa de mídia"},
		{"name": "media_retry_error", "category": "messages", "description": "Erro na tentativa de mídia"},
		{"name": "decrypt_fail_mode", "category": "messages", "description": "Modo de falha na descriptografia"},
		
		// Presença
		{"name": "presence", "category": "presence", "description": "Status de presença do usuário"},
		{"name": "chat_presence", "category": "presence", "description": "Status de digitação no chat"},
		{"name": "qr", "category": "presence", "description": "Código QR gerado"},
		{"name": "qr_scanned_without_multidevice", "category": "presence", "description": "QR escaneado sem multidevice"},
		
		// Grupos
		{"name": "joined_group", "category": "groups", "description": "Entrada em grupo"},
		{"name": "group_info", "category": "groups", "description": "Informações do grupo alteradas"},
		{"name": "group_participant_add", "category": "groups", "description": "Participante adicionado"},
		{"name": "group_participant_remove", "category": "groups", "description": "Participante removido"},
		{"name": "group_participant_promote", "category": "groups", "description": "Participante promovido"},
		{"name": "group_participant_demote", "category": "groups", "description": "Participante rebaixado"},
		{"name": "group_participant_leave", "category": "groups", "description": "Participante saiu"},
		{"name": "group_participant_invite", "category": "groups", "description": "Participante convidado"},
		{"name": "group_participant_change_number", "category": "groups", "description": "Participante mudou número"},
		{"name": "group_change_subject", "category": "groups", "description": "Assunto do grupo alterado"},
		{"name": "group_change_description", "category": "groups", "description": "Descrição do grupo alterada"},
		{"name": "group_change_icon", "category": "groups", "description": "Ícone do grupo alterado"},
		{"name": "group_change_announce", "category": "groups", "description": "Configuração de anúncios alterada"},
		{"name": "group_change_restrict", "category": "groups", "description": "Configuração de restrições alterada"},
		{"name": "group_change_invite_link", "category": "groups", "description": "Link de convite alterado"},
		{"name": "group_v4_add_invite_sent", "category": "groups", "description": "Convite enviado para grupo v4"},
		{"name": "group_create", "category": "groups", "description": "Grupo criado"},
		{"name": "group_member_add_mode", "category": "groups", "description": "Modo de adição de membros alterado"},
		{"name": "group_participant_add_request_join", "category": "groups", "description": "Solicitação de entrada em grupo"},
		{"name": "community_link_parent_group", "category": "groups", "description": "Grupo vinculado a comunidade"},
		{"name": "community_participant_add", "category": "groups", "description": "Participante adicionado à comunidade"},
		{"name": "community_change_description", "category": "groups", "description": "Descrição da comunidade alterada"},
		{"name": "community_allow_member_added_groups", "category": "groups", "description": "Permissão de grupos adicionados por membros alterada"},
		
		// Chamadas
		{"name": "call_offer", "category": "calls", "description": "Oferta de chamada recebida"},
		{"name": "call_offer_notice", "category": "calls", "description": "Notificação de chamada oferecida"},
		{"name": "call_accept", "category": "calls", "description": "Chamada aceita"},
		{"name": "call_pre_accept", "category": "calls", "description": "Chamada pré-aceita"},
		{"name": "call_reject", "category": "calls", "description": "Chamada rejeitada"},
		{"name": "call_terminate", "category": "calls", "description": "Chamada encerrada"},
		{"name": "call_transport", "category": "calls", "description": "Transporte de chamada"},
		{"name": "call_relay_latency", "category": "calls", "description": "Latência de retransmissão de chamada"},
		{"name": "unknown_call_event", "category": "calls", "description": "Evento de chamada desconhecido"},
		
		// Contatos
		{"name": "contact", "category": "contacts", "description": "Contato alterado"},
		{"name": "picture", "category": "contacts", "description": "Foto de perfil alterada"},
		{"name": "blocklist", "category": "contacts", "description": "Lista de bloqueados alterada"},
		{"name": "blocklist_change", "category": "contacts", "description": "Alteração na lista de bloqueio"},
		{"name": "business_name", "category": "contacts", "description": "Nome comercial alterado"},
		{"name": "user_about", "category": "contacts", "description": "Sobre o usuário alterado"},
		{"name": "user_status_mute", "category": "contacts", "description": "Silenciamento de status do usuário"},
		{"name": "push_name", "category": "contacts", "description": "Nome de exibição alterado"},
		{"name": "archive", "category": "contacts", "description": "Arquivamento de chat"},
		{"name": "unarchive", "category": "contacts", "description": "Desarquivamento de chat"},
		{"name": "pin", "category": "contacts", "description": "Fixação de chat"},
		{"name": "unpin", "category": "contacts", "description": "Desfixação de chat"},
		{"name": "mute", "category": "contacts", "description": "Silenciamento de chat"},
		{"name": "unmute", "category": "contacts", "description": "Dessilenciamento de chat"},
		{"name": "clear_chat", "category": "contacts", "description": "Limpeza de chat"},
		{"name": "delete_chat", "category": "contacts", "description": "Exclusão de chat"},
		{"name": "mark_chat_as_read", "category": "contacts", "description": "Marcação de chat como lido"},
		
		// Sistema
		{"name": "app_state", "category": "system", "description": "Estado da aplicação"},
		{"name": "app_state_sync_complete", "category": "system", "description": "Sincronização de estado completa"},
		{"name": "keepalive_timeout", "category": "system", "description": "Timeout de keep-alive"},
		{"name": "keepalive_restored", "category": "system", "description": "Keep-alive restaurado"},
		{"name": "stream_error", "category": "system", "description": "Erro no stream"},
		{"name": "stream_replaced", "category": "system", "description": "Stream substituído"},
		{"name": "manual_login_reconnect", "category": "system", "description": "Reconexão de login manual"},
		{"name": "privacy_settings", "category": "system", "description": "Configurações de privacidade alteradas"},
		{"name": "unarchive_chats_setting", "category": "system", "description": "Configuração de desarquivamento de chats"},
		{"name": "push_name_setting", "category": "system", "description": "Configuração de nome de exibição"},
		{"name": "identity_change", "category": "system", "description": "Alteração de identidade"},
		{"name": "label_edit", "category": "system", "description": "Edição de etiqueta"},
		{"name": "label_association_chat", "category": "system", "description": "Associação de etiqueta a chat"},
		{"name": "label_association_message", "category": "system", "description": "Associação de etiqueta a mensagem"},
		
		// Newsletters
		{"name": "newsletter_join", "category": "newsletters", "description": "Entrada em newsletter"},
		{"name": "newsletter_leave", "category": "newsletters", "description": "Saída de newsletter"},
		{"name": "newsletter_mute_change", "category": "newsletters", "description": "Status de silenciamento alterado"},
		{"name": "newsletter_live_update", "category": "newsletters", "description": "Atualização ao vivo"},
		{"name": "newsletter_message_meta", "category": "newsletters", "description": "Metadados de mensagem"},
		
		// Mídia
		{"name": "fb_message", "category": "media", "description": "Mensagem do Facebook"},
		{"name": "star", "category": "media", "description": "Mensagem marcada com estrela"},
	}
}

func (h *WebhookHandler) getEventCategories() map[string][]string {
	return map[string][]string{
		"connection": {
			"connected", "disconnected", "logged_out", "connect_failure", 
			"temporary_ban", "client_outdated", "cat_refresh_error", 
			"stream_error", "stream_replaced", "permanent_disconnect", 
			"manual_login_reconnect",
		},
		"messages": {
			"message", "receipt", "undecryptable_message", "delete_for_me",
			"history_sync", "offline_sync_preview", "offline_sync_completed",
			"media_retry", "media_retry_error", "decrypt_fail_mode",
		},
		"presence": {
			"presence", "chat_presence", "qr", "qr_scanned_without_multidevice",
		},
		"groups": {
			"joined_group", "group_info", "group_participant_add",
			"group_participant_remove", "group_participant_promote",
			"group_participant_demote", "group_participant_leave",
			"group_participant_invite", "group_participant_change_number",
			"group_change_subject", "group_change_description",
			"group_change_icon", "group_change_announce",
			"group_change_restrict", "group_change_invite_link",
			"group_v4_add_invite_sent", "group_create",
			"group_member_add_mode", "group_participant_add_request_join",
			"community_link_parent_group", "community_participant_add",
			"community_change_description", "community_allow_member_added_groups",
		},
		"calls": {
			"call_offer", "call_offer_notice", "call_accept",
			"call_pre_accept", "call_reject", "call_terminate",
			"call_transport", "call_relay_latency", "unknown_call_event",
		},
		"contacts": {
			"contact", "picture", "blocklist", "blocklist_change",
			"business_name", "user_about", "user_status_mute",
			"push_name", "archive", "unarchive", "pin", "unpin",
			"mute", "unmute", "clear_chat", "delete_chat",
			"mark_chat_as_read",
		},
		"system": {
			"app_state", "app_state_sync_complete", "keepalive_timeout",
			"keepalive_restored", "stream_error", "stream_replaced",
			"manual_login_reconnect", "privacy_settings",
			"unarchive_chats_setting", "push_name_setting",
			"identity_change", "label_edit", "label_association_chat",
			"label_association_message",
		},
		"newsletters": {
			"newsletter_join", "newsletter_leave", "newsletter_mute_change",
			"newsletter_live_update", "newsletter_message_meta",
		},
		"media": {
			"fb_message", "star",
		},
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