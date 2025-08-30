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


type GroupHandler struct {
	sessionService session.Service
	logger         logger.Logger
}


func NewGroupHandler(sessionService session.Service) *GroupHandler {
	return &GroupHandler{
		sessionService: sessionService,
		logger:         logger.GetWithSession("group_handler"),
	}
}


// @Summary Criar novo grupo WhatsApp
// @Description Cria um novo grupo WhatsApp com participantes especificados
// @Tags groups
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.CreateGroupRequest true "Dados do grupo"
// @Success 200 {object} map[string]interface{} "Grupo criado com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/group/create [post]
func (h *GroupHandler) CreateGroup(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	var req dto.CreateGroupRequest
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


	var participants []types.JID
	var invalidPhones []string
	
	for _, phone := range req.Participants {

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


	var phoneNumbers []string
	for _, jid := range participants {
		phoneNumbers = append(phoneNumbers, "+"+jid.User)
	}


	onWhatsApp, err := client.IsOnWhatsApp(phoneNumbers)
	if err != nil {
		h.logger.Warn().Err(err).Str("session_id", sessionID).Msg("Could not verify participants on WhatsApp")

		onWhatsApp = make([]types.IsOnWhatsAppResponse, len(participants))
		for i, jid := range participants {
			onWhatsApp[i] = types.IsOnWhatsAppResponse{
				JID:  jid,
				IsIn: true, // Assumir que estão no WhatsApp
			}
		}
	}


	var validParticipants []types.JID
	for _, result := range onWhatsApp {
		if result.IsIn {
			validParticipants = append(validParticipants, result.JID)
		}
	}

	if len(validParticipants) == 0 {
		return h.sendError(c, "No participants found on WhatsApp", "NO_VALID_PARTICIPANTS", fiber.StatusBadRequest)
	}


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


	if len(invalidPhones) > 0 {
		response["invalid_phones"] = invalidPhones
	}

	if len(participants) != len(validParticipants) {
		response["note"] = fmt.Sprintf("Some participants were not added (not on WhatsApp): %d of %d added", 
			len(validParticipants), len(participants))
	}

	return c.Status(fiber.StatusOK).JSON(response)
}



func (h *GroupHandler) ListGroups(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	groups, err := client.GetJoinedGroups()
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get joined groups")
		return h.sendError(c, fmt.Sprintf("Failed to get groups: %v", err), "GET_GROUPS_FAILED", fiber.StatusInternalServerError)
	}


	var groupList []map[string]interface{}
	for _, groupInfo := range groups {
		groupData := map[string]interface{}{
			"id":             groupInfo.JID.String(),
			"name":           groupInfo.Name,
			"topic":          groupInfo.Topic,
			"owner":          groupInfo.OwnerJID.String(),
			"created_at":     groupInfo.GroupCreated,
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



func (h *GroupHandler) GetGroupInfo(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	var req dto.GroupInfoRequest
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


	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}


	groupInfo, err := client.GetGroupInfo(groupJID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Failed to get group info")
		return h.sendError(c, fmt.Sprintf("Failed to get group info: %v", err), "GROUP_INFO_FAILED", fiber.StatusInternalServerError)
	}


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
			"owner":          groupInfo.OwnerJID.String(),
			"created_at":     groupInfo.GroupCreated,
			"participants":   participants,
			"announce_mode":  groupInfo.IsAnnounce,
			"locked":         groupInfo.IsLocked,
			"ephemeral":      groupInfo.DisappearingTimer,
		},
		"timestamp": time.Now().Unix(),
	})
}



func (h *GroupHandler) GetInviteLink(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	var req dto.GroupInviteLinkRequest
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


	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}


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



func (h *GroupHandler) LeaveGroup(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.LeaveGroupRequest
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


	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}


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



func (h *GroupHandler) SetGroupPhoto(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SetGroupPhotoRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	h.logger.Info().Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Group photo update requested")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"group_id":  req.GroupID,
		"message":   "Group photo functionality - implementation needed",
		"note":      "Requires media upload and processing implementation",
		"timestamp": time.Now().Unix(),
	})
}



func (h *GroupHandler) SetGroupName(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SetGroupNameRequest
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


	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}


	err = client.SetGroupName(groupJID, req.Name)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Failed to set group name")
		return h.sendError(c, fmt.Sprintf("Failed to set group name: %v", err), "SET_NAME_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("group_id", req.GroupID).
		Str("new_name", req.Name).
		Msg("Group name updated")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"group_id":  req.GroupID,
		"name":      req.Name,
		"message":   "Group name updated successfully",
		"timestamp": time.Now().Unix(),
	})
}



func (h *GroupHandler) SetGroupTopic(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SetGroupTopicRequest
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


	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}


	err = client.SetGroupTopic(groupJID, "", "", req.Topic)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Failed to set group topic")
		return h.sendError(c, fmt.Sprintf("Failed to set group topic: %v", err), "SET_TOPIC_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("group_id", req.GroupID).
		Str("new_topic", req.Topic).
		Msg("Group topic updated")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"group_id":  req.GroupID,
		"topic":     req.Topic,
		"message":   "Group topic updated successfully",
		"timestamp": time.Now().Unix(),
	})
}



func (h *GroupHandler) SetGroupAnnounce(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SetGroupAnnounceRequest
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


	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}


	err = client.SetGroupAnnounce(groupJID, req.AnnounceMode)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Failed to set group announce mode")
		return h.sendError(c, fmt.Sprintf("Failed to set group announce mode: %v", err), "SET_ANNOUNCE_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("group_id", req.GroupID).
		Bool("announce_mode", req.AnnounceMode).
		Msg("Group announce mode updated")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":      true,
		"group_id":     req.GroupID,
		"announce":     req.AnnounceMode,
		"message":      "Group announce mode updated successfully",
		"timestamp":    time.Now().Unix(),
	})
}



func (h *GroupHandler) SetGroupLocked(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SetGroupLockedRequest
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


	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}


	err = client.SetGroupLocked(groupJID, req.Locked)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Failed to set group locked mode")
		return h.sendError(c, fmt.Sprintf("Failed to set group locked mode: %v", err), "SET_LOCKED_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("group_id", req.GroupID).
		Bool("locked_mode", req.Locked).
		Msg("Group locked mode updated")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"group_id":  req.GroupID,
		"locked":    req.Locked,
		"message":   "Group locked mode updated successfully",
		"timestamp": time.Now().Unix(),
	})
}



func (h *GroupHandler) JoinGroup(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.JoinGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	_, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}




	h.logger.Warn().Str("session_id", sessionID).Str("invite_code", req.InviteCode).Msg("Join group functionality needs implementation")

	return h.sendError(c, "Join group functionality requires updated implementation", "NOT_IMPLEMENTED", fiber.StatusNotImplemented)


}



func (h *GroupHandler) UpdateParticipants(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.UpdateGroupParticipantsRequest
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


	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}


	var participantJIDs []types.JID
	var invalidPhones []string

	for _, phone := range req.Participants {
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
		participantJIDs = append(participantJIDs, jid)
	}

	if len(participantJIDs) == 0 {
		return h.sendError(c, "No valid participants provided", "NO_VALID_PARTICIPANTS", fiber.StatusBadRequest)
	}

	var results []types.GroupParticipant
	var operation string


	switch req.Action {
	case "add":
		results, err = client.UpdateGroupParticipants(groupJID, participantJIDs, whatsmeow.ParticipantChangeAdd)
		operation = "added"
	case "remove":
		results, err = client.UpdateGroupParticipants(groupJID, participantJIDs, whatsmeow.ParticipantChangeRemove)
		operation = "removed"
	case "promote":
		results, err = client.UpdateGroupParticipants(groupJID, participantJIDs, whatsmeow.ParticipantChangePromote)
		operation = "promoted"
	case "demote":
		results, err = client.UpdateGroupParticipants(groupJID, participantJIDs, whatsmeow.ParticipantChangeDemote)
		operation = "demoted"
	default:
		return h.sendError(c, "Invalid action. Use: add, remove, promote, demote", "INVALID_ACTION", fiber.StatusBadRequest)
	}

	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Str("action", req.Action).Msg("Failed to update group participants")
		return h.sendError(c, fmt.Sprintf("Failed to update participants: %v", err), "UPDATE_PARTICIPANTS_FAILED", fiber.StatusInternalServerError)
	}


	var successfulChanges []map[string]interface{}
	var failedChanges []map[string]interface{}

	for _, result := range results {
		changeInfo := map[string]interface{}{
			"jid":    result.JID.String(),
			"phone":  result.JID.User,
			"action": operation,
		}

		if result.Error != 0 {
			changeInfo["error"] = fmt.Sprintf("Error code: %d", result.Error)
			failedChanges = append(failedChanges, changeInfo)
		} else {
			successfulChanges = append(successfulChanges, changeInfo)
		}
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("group_id", req.GroupID).
		Str("action", req.Action).
		Int("successful", len(successfulChanges)).
		Int("failed", len(failedChanges)).
		Msg("Group participants updated")

	response := fiber.Map{
		"success":    true,
		"group_id":   req.GroupID,
		"action":     req.Action,
		"results": map[string]interface{}{
			"successful": successfulChanges,
			"failed":     failedChanges,
		},
		"summary": map[string]interface{}{
			"total_requested": len(req.Participants),
			"successful":      len(successfulChanges),
			"failed":          len(failedChanges),
		},
		"timestamp": time.Now().Unix(),
	}


	if len(invalidPhones) > 0 {
		response["invalid_phones"] = invalidPhones
	}

	return c.Status(fiber.StatusOK).JSON(response)
}



func (h *GroupHandler) RemoveGroupPhoto(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.GroupPhotoRemoveRequest
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


	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}




	_, err = client.SetGroupPhoto(groupJID, nil)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Failed to remove group photo")
		return h.sendError(c, fmt.Sprintf("Failed to remove group photo: %v", err), "REMOVE_PHOTO_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("group_id", req.GroupID).
		Msg("Group photo removed")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"group_id":  req.GroupID,
		"message":   "Group photo removed successfully",
		"timestamp": time.Now().Unix(),
	})
}



func (h *GroupHandler) SetGroupEphemeral(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.GroupEphemeralRequest
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


	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		return h.sendError(c, "Invalid group ID format", "INVALID_GROUP_ID", fiber.StatusBadRequest)
	}


	err = client.SetDisappearingTimer(groupJID, time.Duration(req.Duration)*time.Second)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("group_id", req.GroupID).Msg("Failed to set disappearing timer")
		return h.sendError(c, fmt.Sprintf("Failed to set disappearing timer: %v", err), "SET_TIMER_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("group_id", req.GroupID).
		Int64("duration_seconds", req.Duration).
		Msg("Group disappearing timer updated")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":          true,
		"group_id":         req.GroupID,
		"duration_seconds": req.Duration,
		"message":          "Disappearing timer updated successfully",
		"timestamp":        time.Now().Unix(),
	})
}



func (h *GroupHandler) GetInviteInfo(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.GroupInviteInfoRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	_, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}




	h.logger.Warn().Str("session_id", sessionID).Str("invite_code", req.InviteCode).Msg("Get invite info functionality needs implementation")

	return h.sendError(c, "Get invite info functionality requires updated implementation", "NOT_IMPLEMENTED", fiber.StatusNotImplemented)
}



func (h *GroupHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	h.logger.Error().Str("error", message).Str("code", code).Int("status", status).Msg("Group handler error")
	return c.Status(status).JSON(fiber.Map{
		"success": false,
		"error":   message,
		"code":    code,
	})
}
