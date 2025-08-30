package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"

	"github.com/felipe/zemeow/internal/api/dto"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/session"
)

// GroupHandler gerencia endpoints relacionados a grupos
type GroupHandler struct {
	sessionService session.Service
	logger         logger.Logger
}

// NewGroupHandler cria uma nova instância do handler de grupo
func NewGroupHandler(sessionService session.Service) *GroupHandler {
	return &GroupHandler{
		sessionService: sessionService,
		logger:         logger.GetWithSession("group_handler"),
	}
}

// CreateGroup cria um novo grupo
// POST /sessions/:sessionId/group/create
func (h *GroupHandler) CreateGroup(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	var req dto.CreateGroupRequest
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

	// Preparar participantes
	var participants []types.JID
	var invalidPhones []string
	
	for _, phone := range req.Participants {
		// Limpar número de telefone
		cleanPhone := strings.ReplaceAll(phone, "+", "")
		cleanPhone = strings.ReplaceAll(cleanPhone, " ", "")
		cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")
		
		if len(cleanPhone) < 10 {
			invalidPhones = append(invalidPhones, phone)
			continue
		}
		
		jid, err := types.ParseJID(cleanPhone + "@s.whatsapp.net")
		if err != nil {
			invalidPhones = append(invalidPhones, phone)
			continue
		}
		participants = append(participants, jid)
	}

	if len(participants) == 0 {
		return h.sendError(c, "No valid participants provided", "NO_PARTICIPANTS", fiber.StatusBadRequest)
	}

	// Verificar se os participantes estão no WhatsApp
	onWhatsApp, err := client.IsOnWhatsApp(participants)
	if err != nil {
		h.logger.Warn().Err(err).Str("session_id", sessionID).Msg("Could not verify participants on WhatsApp")
	}

	// Filtrar apenas participantes válidos
	var validParticipants []types.JID
	for _, result := range onWhatsApp {
		if result.IsIn {
			validParticipants = append(validParticipants, result.JID)
		}
	}

	if len(validParticipants) == 0 {
		return h.sendError(c, "No participants found on WhatsApp", "NO_VALID_PARTICIPANTS", fiber.StatusBadRequest)
	}

	// Criar grupo
	resp, err := client.CreateGroup(whatsmeow.ReqCreateGroup{
		Name:         req.Name,
		Participants: validParticipants,
		CreateKey:    client.GenerateMessageID(),
	})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_name", req.Name).Msg("Failed to create group")
		return h.sendError(c, fmt.Sprintf("Failed to create group: %v", err), "CREATE_GROUP_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("group_id", resp.JID.String()).
		Str("group_name", req.Name).
		Int("participants", len(validParticipants)).
		Msg("Group created successfully")

	response := fiber.Map{
		"success":     true,
		"group_id":    resp.JID.String(),
		"name":        req.Name,
		"created_at":  time.Now(),
		"participants": len(validParticipants),
		"timestamp":   time.Now().Unix(),
	}

	// Adicionar informações sobre participantes inválidos se houver
	if len(invalidPhones) > 0 {
		response["invalid_phones"] = invalidPhones
	}

	if len(participants) != len(validParticipants) {
		response["note"] = fmt.Sprintf("Some participants were not added (not on WhatsApp): %d of %d added", 
			len(validParticipants), len(participants))
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// ListGroups lista grupos do usuário
// GET /sessions/:sessionId/group/list
func (h *GroupHandler) ListGroups(c *fiber.Ctx) error {
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

	// Obter grupos
	groups, err := client.GetJoinedGroups()
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get joined groups")
		return h.sendError(c, fmt.Sprintf("Failed to get groups: %v", err), "GET_GROUPS_FAILED", fiber.StatusInternalServerError)
	}

	// Processar grupos
	var groupList []map[string]interface{}
	for _, group := range groups {
		groupInfo, err := client.GetGroupInfo(group)
		if err != nil {
			h.logger.Warn().Err(err).Str("group_id", group.String()).Msg("Failed to get group info")
			continue // Pular grupos com erro
		}

		groupData := map[string]interface{}{
			"id":             group.String(),
			"name":           groupInfo.Name,
			"topic":          groupInfo.Topic,
			"owner":          groupInfo.Owner.String(),
			"created_at":     time.Unix(int64(groupInfo.CreationTime), 0),
			"participants":   len(groupInfo.Participants),
			"announce_mode":  groupInfo.IsAnnounce,
			"locked":         groupInfo.IsLocked,
			"ephemeral":      groupInfo.DisappearingTimer,
		}

		groupList = append(groupList, groupData)
	}

	h.logger.Info().Str("session_id", sessionID).Int("groups_count", len(groupList)).Msg("Groups listed")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"groups":    groupList,
		"total":     len(groupList),
		"timestamp": time.Now().Unix(),
	})
}

// GetGroupInfo obtém informações do grupo
// POST /sessions/:sessionId/group/info
func (h *GroupHandler) GetGroupInfo(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	var req dto.GroupInfoRequest
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

	// Preparar JID do grupo
	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}

	// Obter informações do grupo
	groupInfo, err := client.GetGroupInfo(groupJID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Failed to get group info")
		return h.sendError(c, fmt.Sprintf("Failed to get group info: %v", err), "GROUP_INFO_FAILED", fiber.StatusInternalServerError)
	}

	// Processar participantes
	var participants []map[string]interface{}
	for _, participant := range groupInfo.Participants {
		participants = append(participants, map[string]interface{}{
			"jid":            participant.JID.String(),
			"phone":          participant.JID.User,
			"is_admin":       participant.IsAdmin,
			"is_super_admin": participant.IsSuperAdmin,
		})
	}

	h.logger.Info().Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Group info retrieved")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"group": map[string]interface{}{
			"id":             groupJID.String(),
			"name":           groupInfo.Name,
			"topic":          groupInfo.Topic,
			"owner":          groupInfo.Owner.String(),
			"created_at":     time.Unix(int64(groupInfo.CreationTime), 0),
			"participants":   participants,
			"announce_mode":  groupInfo.IsAnnounce,
			"locked":         groupInfo.IsLocked,
			"ephemeral":      groupInfo.DisappearingTimer,
		},
		"timestamp": time.Now().Unix(),
	})
}

// GetInviteLink obtém link de convite do grupo
// POST /sessions/:sessionId/group/invitelink
func (h *GroupHandler) GetInviteLink(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	var req dto.GroupInviteLinkRequest
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

	// Preparar JID do grupo
	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}

	// Obter link de convite
	inviteLink, err := client.GetGroupInviteLink(groupJID, false)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Failed to get invite link")
		return h.sendError(c, fmt.Sprintf("Failed to get invite link: %v", err), "INVITE_LINK_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Group invite link retrieved")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":      true,
		"group_id":     req.GroupID,
		"invite_link":  inviteLink,
		"timestamp":    time.Now().Unix(),
	})
}

// LeaveGroup sai do grupo
// POST /sessions/:sessionId/group/leave
func (h *GroupHandler) LeaveGroup(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.LeaveGroupRequest
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

	// Preparar JID do grupo
	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}

	// Sair do grupo
	err = client.LeaveGroup(groupJID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Failed to leave group")
		return h.sendError(c, fmt.Sprintf("Failed to leave group: %v", err), "LEAVE_GROUP_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Left group successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"group_id":  req.GroupID,
		"message":   "Left group successfully",
		"timestamp": time.Now().Unix(),
	})
}

// SetGroupPhoto define foto do grupo (placeholder)
// POST /sessions/:sessionId/group/photo
func (h *GroupHandler) SetGroupPhoto(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SetGroupPhotoRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// TODO: Implementar upload de foto do grupo
	h.logger.Info().Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Group photo update requested")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"group_id":  req.GroupID,
		"message":   "Group photo functionality - implementation needed",
		"note":      "Requires media upload and processing implementation",
		"timestamp": time.Now().Unix(),
	})
}

// SetGroupName define nome do grupo (placeholder)
// POST /sessions/:sessionId/group/name
func (h *GroupHandler) SetGroupName(c *fiber.Ctx) error {
	return h.sendError(c, "Group name functionality - implementation needed", "NOT_IMPLEMENTED", fiber.StatusNotImplemented)
}

// SetGroupTopic define tópico do grupo (placeholder)
// POST /sessions/:sessionId/group/topic
func (h *GroupHandler) SetGroupTopic(c *fiber.Ctx) error {
	return h.sendError(c, "Group topic functionality - implementation needed", "NOT_IMPLEMENTED", fiber.StatusNotImplemented)
}

// SetGroupAnnounce configura anúncios (placeholder)
// POST /sessions/:sessionId/group/announce
func (h *GroupHandler) SetGroupAnnounce(c *fiber.Ctx) error {
	return h.sendError(c, "Group announce functionality - implementation needed", "NOT_IMPLEMENTED", fiber.StatusNotImplemented)
}

// SetGroupLocked bloqueia grupo (placeholder)
// POST /sessions/:sessionId/group/locked
func (h *GroupHandler) SetGroupLocked(c *fiber.Ctx) error {
	return h.sendError(c, "Group locked functionality - implementation needed", "NOT_IMPLEMENTED", fiber.StatusNotImplemented)
}

// JoinGroup entra no grupo (placeholder)
// POST /sessions/:sessionId/group/join
func (h *GroupHandler) JoinGroup(c *fiber.Ctx) error {
	return h.sendError(c, "Join group functionality - implementation needed", "NOT_IMPLEMENTED", fiber.StatusNotImplemented)
}

// UpdateParticipants atualiza participantes (placeholder)
// POST /sessions/:sessionId/group/updateparticipants
func (h *GroupHandler) UpdateParticipants(c *fiber.Ctx) error {
	return h.sendError(c, "Update participants functionality - implementation needed", "NOT_IMPLEMENTED", fiber.StatusNotImplemented)
}

// Métodos auxiliares

func (h *GroupHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	h.logger.Error().Str("error", message).Str("code", code).Int("status", status).Msg("Group handler error")
	return c.Status(status).JSON(fiber.Map{
		"success": false,
		"error":   message,
		"code":    code,
	})
}
