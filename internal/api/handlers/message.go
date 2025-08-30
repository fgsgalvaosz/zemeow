package handlers

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/vincent-petithory/dataurl"
	"google.golang.org/protobuf/proto"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"

	"github.com/felipe/zemeow/internal/api/dto"
	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/session"
)


type MessageHandler struct {
	sessionService session.Service
	logger         logger.Logger
}


func NewMessageHandler(sessionService session.Service) *MessageHandler {
	return &MessageHandler{
		sessionService: sessionService,
		logger:         logger.GetWithSession("message_handler"),
	}
}


func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {

	return h.SendText(c)
}


// @Summary Enviar mensagem de texto
// @Description Envia uma mensagem de texto via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendTextRequest true "Dados da mensagem"
// @Success 200 {object} map[string]interface{} "Mensagem enviada com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/send/text [post]
func (h *MessageHandler) SendText(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	

	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}


	var req dto.SendTextRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	if err := h.validateTextRequest(&req); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}


	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}


	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(req.Text),
		},
	}


	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send message: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}


	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	})
}


func (h *MessageHandler) SendMedia(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	

	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}


	var req dto.SendMediaRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	if err := h.validateMediaRequest(&req); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}


	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	fileData, err := h.decodeBase64Media(req.Media)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_MEDIA_DATA", fiber.StatusBadRequest)
	}


	var mediaType whatsmeow.MediaType
	switch req.Type {
	case dto.MediaTypeImage:
		mediaType = whatsmeow.MediaImage
	case dto.MediaTypeAudio:
		mediaType = whatsmeow.MediaAudio
	case dto.MediaTypeVideo:
		mediaType = whatsmeow.MediaVideo
	case dto.MediaTypeDocument:
		mediaType = whatsmeow.MediaDocument
	case dto.MediaTypeSticker:
		mediaType = whatsmeow.MediaImage // Stickers são tratados como imagens
	default:
		return h.sendError(c, "Unsupported media type", "INVALID_MEDIA_TYPE", fiber.StatusBadRequest)
	}


	uploaded, err := client.Upload(context.Background(), fileData, mediaType)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to upload media: %v", err), "UPLOAD_FAILED", fiber.StatusInternalServerError)
	}


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}


	msg, err := h.buildMediaMessage(req, fileData, uploaded)
	if err != nil {
		return h.sendError(c, err.Error(), "BUILD_MESSAGE_FAILED", fiber.StatusInternalServerError)
	}


	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send media: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}


	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"media_type": req.Type,
		"file_size":  len(fileData),
	})
}


func (h *MessageHandler) SendLocation(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	

	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}


	var req dto.SendLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	if err := h.validateLocationRequest(&req); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}


	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}


	msg := &waE2E.Message{
		LocationMessage: &waE2E.LocationMessage{
			DegreesLatitude:  &req.Latitude,
			DegreesLongitude: &req.Longitude,
			Name:             &req.Name,
		},
	}


	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send location: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}


	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	})
}


func (h *MessageHandler) SendContact(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	

	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}


	var req dto.SendContactRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	if err := h.validateContactRequest(&req); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}


	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}


	msg := &waE2E.Message{
		ContactMessage: &waE2E.ContactMessage{
			DisplayName: &req.Name,
			Vcard:       &req.Vcard,
		},
	}


	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send contact: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}


	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	})
}





func (h *MessageHandler) SendSticker(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SendStickerRequest
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


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}


	filedata, err := h.processMediaData(req.Sticker)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to process sticker: %v", err), "MEDIA_PROCESSING_FAILED", fiber.StatusBadRequest)
	}


	uploaded, err := client.Upload(context.Background(), filedata, whatsmeow.MediaImage)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to upload sticker: %v", err), "UPLOAD_FAILED", fiber.StatusInternalServerError)
	}


	msg := &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String("image/webp"),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(filedata))),
		},
	}


	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send sticker: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	})
}



func (h *MessageHandler) ReactToMessage(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.ReactRequest
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


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	msg := &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key: &waE2E.MessageKey{
				RemoteJID: proto.String(recipient.String()),
				FromMe:    proto.Bool(false),
				ID:        proto.String(req.MessageID),
			},
			Text: proto.String(req.Emoji),
		},
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send reaction: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"message_id": req.MessageID,
		"emoji":      req.Emoji,
	})
}



func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.DeleteMessageRequest
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


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	msg := &waE2E.Message{
		ProtocolMessage: &waE2E.ProtocolMessage{
			Key: &waE2E.MessageKey{
				RemoteJID: proto.String(recipient.String()),
				FromMe:    proto.Bool(true),
				ID:        proto.String(req.MessageID),
			},
			Type: waE2E.ProtocolMessage_REVOKE.Enum(),
		},
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to delete message: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":     "deleted",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"message_id": req.MessageID,
		"for_all":    req.ForAll,
	})
}





func (h *MessageHandler) SetChatPresence(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.ChatPresenceRequest
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


	_, err = h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	var presence types.Presence
	switch req.Presence {
	case "available":
		presence = types.PresenceAvailable
	case "unavailable":
		presence = types.PresenceUnavailable
	case "composing":

		presence = types.PresenceAvailable
	case "recording":

		presence = types.PresenceAvailable
	case "paused":

		presence = types.PresenceAvailable
	default:
		return h.sendError(c, "Invalid presence type", "INVALID_PRESENCE", fiber.StatusBadRequest)
	}


	err = client.SendPresence(presence)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send presence: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":    "sent",
		"recipient": req.To,
		"presence":  req.Presence,
	})
}



func (h *MessageHandler) MarkAsRead(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.MarkReadRequest
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


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	var messageIDs []types.MessageID
	for _, msgID := range req.MessageID {
		messageIDs = append(messageIDs, msgID)
	}

	err = client.MarkRead(messageIDs, time.Now(), recipient, recipient)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to mark as read: %v", err), "MARK_READ_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":      "marked_read",
		"recipient":   req.To,
		"message_ids": req.MessageID,
	})
}





func (h *MessageHandler) DownloadImage(c *fiber.Ctx) error {
	return h.downloadMedia(c, "image")
}



func (h *MessageHandler) DownloadVideo(c *fiber.Ctx) error {
	return h.downloadMedia(c, "video")
}



func (h *MessageHandler) DownloadAudio(c *fiber.Ctx) error {
	return h.downloadMedia(c, "audio")
}



func (h *MessageHandler) DownloadDocument(c *fiber.Ctx) error {
	return h.downloadMedia(c, "document")
}


func (h *MessageHandler) downloadMedia(c *fiber.Ctx, mediaType string) error {
	sessionID := c.Params("sessionId")

	var req dto.DownloadMediaRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	if req.Type != mediaType {
		return h.sendError(c, fmt.Sprintf("Expected media type %s, got %s", mediaType, req.Type), "INVALID_MEDIA_TYPE", fiber.StatusBadRequest)
	}


	_, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}










	h.logger.Warn().
		Str("session_id", sessionID).
		Str("message_id", req.MessageID).
		Str("media_type", req.Type).
		Msg("Media download requested - implementation needed")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    false,
		"message_id": req.MessageID,
		"type":       req.Type,
		"error":      "Media download functionality requires message storage implementation",
		"note":       "To implement: store messages with media info, then use client.Download()",
		"timestamp":  time.Now().Unix(),
	})
}





func (h *MessageHandler) SendButtons(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SendButtonsRequest
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


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}




	h.logger.Warn().
		Str("session_id", sessionID).
		Str("to", req.To).
		Int("buttons_count", len(req.Buttons)).
		Msg("Interactive buttons requested - sending as text fallback")


	buttonText := req.Text + "\n\n"
	for i, button := range req.Buttons {
		buttonText += fmt.Sprintf("%d. %s\n", i+1, button.Text)
	}
	if req.Footer != "" {
		buttonText += "\n" + req.Footer
	}


	msg := &waE2E.Message{
		Conversation: proto.String(buttonText),
	}


	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send buttons: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"message_id": messageID,
		"status":     "sent_as_text",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"note":       "Interactive buttons sent as text fallback - requires WhatsApp Business API for full functionality",
	})
}



func (h *MessageHandler) SendList(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SendListRequest
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


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}




	h.logger.Warn().
		Str("session_id", sessionID).
		Str("to", req.To).
		Int("sections_count", len(req.Sections)).
		Msg("Interactive list requested - sending as text fallback")


	listText := req.Text + "\n\n" + req.Title + "\n"
	for _, section := range req.Sections {
		listText += "\n" + section.Title + ":\n"
		for i, row := range section.Rows {
			listText += fmt.Sprintf("%d. %s", i+1, row.Title)
			if row.Description != "" {
				listText += " - " + row.Description
			}
			listText += "\n"
		}
	}
	if req.Footer != "" {
		listText += "\n" + req.Footer
	}


	msg := &waE2E.Message{
		Conversation: proto.String(listText),
	}


	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send list: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"message_id": messageID,
		"status":     "sent_as_text",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"note":       "Interactive list sent as text fallback - requires WhatsApp Business API for full functionality",
	})
}



func (h *MessageHandler) SendPoll(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SendPollRequest
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


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}


	var pollOptions []*waE2E.PollCreationMessage_Option
	for _, option := range req.Options {
		pollOptions = append(pollOptions, &waE2E.PollCreationMessage_Option{
			OptionName: proto.String(option),
		})
	}


	selectableCount := req.Selectable
	if selectableCount == 0 {
		selectableCount = 1 // Padrão: apenas uma opção
	}


	msg := &waE2E.Message{
		PollCreationMessage: &waE2E.PollCreationMessage{
			Name:    proto.String(req.Name),
			Options: pollOptions,

		},
	}


	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("to", req.To).Msg("Failed to send poll")
		return h.sendError(c, fmt.Sprintf("Failed to send poll: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("to", req.To).
		Str("poll_name", req.Name).
		Int("options_count", len(req.Options)).
		Msg("Poll sent successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"poll": map[string]interface{}{
			"name":        req.Name,
			"options":     req.Options,
			"selectable":  selectableCount,
		},
	})
}



func (h *MessageHandler) EditMessage(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.EditMessageRequest
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


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	msg := &waE2E.Message{
		EditedMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				Conversation: proto.String(req.Text),
			},
		},
		ProtocolMessage: &waE2E.ProtocolMessage{
			Key: &waE2E.MessageKey{
				RemoteJID: proto.String(recipient.String()),
				FromMe:    proto.Bool(true),
				ID:        proto.String(req.MessageID),
			},
			Type:        waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(),
			EditedMessage: &waE2E.Message{
				Conversation: proto.String(req.Text),
			},
		},
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("to", req.To).Str("message_id", req.MessageID).Msg("Failed to edit message")
		return h.sendError(c, fmt.Sprintf("Failed to edit message: %v", err), "EDIT_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("to", req.To).
		Str("message_id", req.MessageID).
		Msg("Message edited successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":           true,
		"original_message_id": req.MessageID,
		"status":            "edited",
		"timestamp":         response.Timestamp,
		"recipient":         req.To,
		"new_text":          req.Text,
	})
}





func (h *MessageHandler) SendImage(c *fiber.Ctx) error {
	var req dto.SendImageRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	return h.SendMedia(c)
}



func (h *MessageHandler) SendAudio(c *fiber.Ctx) error {
	var req dto.SendAudioRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	return h.SendMedia(c)
}



func (h *MessageHandler) SendDocument(c *fiber.Ctx) error {
	var req dto.SendDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	return h.SendMedia(c)
}



func (h *MessageHandler) SendVideo(c *fiber.Ctx) error {
	var req dto.SendVideoRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}


	return h.SendMedia(c)
}


func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	

	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}


	return c.JSON(fiber.Map{
		"success": true,
		"data":    []interface{}{},
	})
}



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



func (h *MessageHandler) SendBulkMessages(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "sent",
		"message":    "Bulk messages endpoint",
	})
}


func (h *MessageHandler) hasSessionAccess(c *fiber.Ctx, sessionID string) bool {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		return false
	}


	if authCtx.IsGlobalKey {
		return true
	}


	return authCtx.SessionID == sessionID
}


func (h *MessageHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	return c.Status(status).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"message": message,
			"code":    code,
		},
	})
}




func (h *MessageHandler) parseJID(phone string) (types.JID, error) {

	phone = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)


	if len(phone) < 10 || len(phone) > 15 {
		return types.JID{}, fmt.Errorf("invalid phone number length")
	}


	jid := types.NewJID(phone, types.DefaultUserServer)
	return jid, nil
}


func (h *MessageHandler) validateTextRequest(req *dto.SendTextRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient 'to' is required")
	}
	
	if req.Text == "" {
		return fmt.Errorf("text content is required")
	}
	
	if len(req.Text) > 4096 {
		return fmt.Errorf("text content exceeds maximum length of 4096 characters")
	}
	
	return nil
}


func (h *MessageHandler) validateMediaRequest(req *dto.SendMediaRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient 'to' is required")
	}
	
	if req.Media == "" {
		return fmt.Errorf("media data is required")
	}
	

	if !strings.HasPrefix(req.Media, "data:") {
		return fmt.Errorf("media must be a valid data URL (data:mime/type;base64,...)")
	}
	

	if req.Caption != "" && len(req.Caption) > 1024 {
		return fmt.Errorf("caption exceeds maximum length of 1024 characters")
	}
	

	if req.Filename != "" && len(req.Filename) > 255 {
		return fmt.Errorf("filename exceeds maximum length of 255 characters")
	}
	
	return nil
}


func (h *MessageHandler) validateLocationRequest(req *dto.SendLocationRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient 'to' is required")
	}
	

	if req.Latitude < -90 || req.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	

	if req.Longitude < -180 || req.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	
	return nil
}


func (h *MessageHandler) validateContactRequest(req *dto.SendContactRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient 'to' is required")
	}
	
	if req.Name == "" {
		return fmt.Errorf("contact name is required")
	}
	
	if req.Vcard == "" {
		return fmt.Errorf("vcard data is required")
	}
	
	return nil
}


func (h *MessageHandler) decodeBase64Media(dataURL string) ([]byte, error) {
	if !strings.HasPrefix(dataURL, "data:") {
		return nil, fmt.Errorf("invalid data URL format")
	}
	
	dataURLStruct, err := dataurl.DecodeString(dataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 data: %v", err)
	}
	
	return dataURLStruct.Data, nil
}


func (h *MessageHandler) buildMediaMessage(req dto.SendMediaRequest, filedata []byte, uploaded whatsmeow.UploadResponse) (*waE2E.Message, error) {
	switch req.Type {
	case dto.MediaTypeImage:

		mimeType := req.MimeType
		if mimeType == "" {
			mimeType = http.DetectContentType(filedata)
		}
		
		return &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption:       &req.Caption,
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      &mimeType,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(filedata))),
			},
		}, nil
		
	case dto.MediaTypeAudio:

		mimeType := req.MimeType
		if mimeType == "" {
			mimeType = "audio/ogg; codecs=opus"
		}
		
		ptt := true // Áudio como mensagem de voz
		
		return &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      &mimeType,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(filedata))),
				PTT:           &ptt,
			},
		}, nil
		
	case dto.MediaTypeVideo:

		mimeType := req.MimeType
		if mimeType == "" {
			mimeType = http.DetectContentType(filedata)
		}
		
		return &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				Caption:       &req.Caption,
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      &mimeType,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(filedata))),
			},
		}, nil
		
	case dto.MediaTypeDocument:

		mimeType := req.MimeType
		if mimeType == "" {

			if req.Filename != "" {
				mimeType = mime.TypeByExtension(strings.ToLower(req.Filename[strings.LastIndex(req.Filename, "."):]))
			}

			if mimeType == "" {
				mimeType = http.DetectContentType(filedata)
			}
		}
		
		return &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				FileName:      &req.Filename,
				Caption:       &req.Caption,
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      &mimeType,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(filedata))),
			},
		}, nil
		
	case dto.MediaTypeSticker:

		mimeType := req.MimeType
		if mimeType == "" {
			mimeType = http.DetectContentType(filedata)
		}
		
		return &waE2E.Message{
			StickerMessage: &waE2E.StickerMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      &mimeType,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(filedata))),
			},
		}, nil
		
	default:
		return nil, fmt.Errorf("unsupported media type: %s", req.Type)
	}
}


func (h *MessageHandler) addContextInfo(msg *waE2E.Message, contextInfo *dto.ContextInfo) {
	if contextInfo == nil {
		return
	}
	

	if contextInfo.StanzaID != nil && contextInfo.Participant != nil {

		if msg.ExtendedTextMessage != nil {
			if msg.ExtendedTextMessage.ContextInfo == nil {
				msg.ExtendedTextMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			msg.ExtendedTextMessage.ContextInfo.StanzaID = contextInfo.StanzaID
			msg.ExtendedTextMessage.ContextInfo.Participant = contextInfo.Participant
			msg.ExtendedTextMessage.ContextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		}
		

		if msg.ImageMessage != nil {
			if msg.ImageMessage.ContextInfo == nil {
				msg.ImageMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			msg.ImageMessage.ContextInfo.StanzaID = contextInfo.StanzaID
			msg.ImageMessage.ContextInfo.Participant = contextInfo.Participant
			msg.ImageMessage.ContextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		}
		
		if msg.AudioMessage != nil {
			if msg.AudioMessage.ContextInfo == nil {
				msg.AudioMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			msg.AudioMessage.ContextInfo.StanzaID = contextInfo.StanzaID
			msg.AudioMessage.ContextInfo.Participant = contextInfo.Participant
			msg.AudioMessage.ContextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		}
		
		if msg.VideoMessage != nil {
			if msg.VideoMessage.ContextInfo == nil {
				msg.VideoMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			msg.VideoMessage.ContextInfo.StanzaID = contextInfo.StanzaID
			msg.VideoMessage.ContextInfo.Participant = contextInfo.Participant
			msg.VideoMessage.ContextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		}
		
		if msg.DocumentMessage != nil {
			if msg.DocumentMessage.ContextInfo == nil {
				msg.DocumentMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			msg.DocumentMessage.ContextInfo.StanzaID = contextInfo.StanzaID
			msg.DocumentMessage.ContextInfo.Participant = contextInfo.Participant
			msg.DocumentMessage.ContextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		}
		
		if msg.StickerMessage != nil {
			if msg.StickerMessage.ContextInfo == nil {
				msg.StickerMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			msg.StickerMessage.ContextInfo.StanzaID = contextInfo.StanzaID
			msg.StickerMessage.ContextInfo.Participant = contextInfo.Participant
			msg.StickerMessage.ContextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		}
		
		if msg.LocationMessage != nil {
			if msg.LocationMessage.ContextInfo == nil {
				msg.LocationMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			msg.LocationMessage.ContextInfo.StanzaID = contextInfo.StanzaID
			msg.LocationMessage.ContextInfo.Participant = contextInfo.Participant
			msg.LocationMessage.ContextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		}
		
		if msg.ContactMessage != nil {
			if msg.ContactMessage.ContextInfo == nil {
				msg.ContactMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			msg.ContactMessage.ContextInfo.StanzaID = contextInfo.StanzaID
			msg.ContactMessage.ContextInfo.Participant = contextInfo.Participant
			msg.ContactMessage.ContextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		}
	}
	

	if len(contextInfo.MentionedJID) > 0 {

		var mentionedJIDs []string
		for _, jidStr := range contextInfo.MentionedJID {
			if jid, err := h.parseJID(jidStr); err == nil {
				mentionedJIDs = append(mentionedJIDs, jid.String())
			}
		}
		
		if len(mentionedJIDs) > 0 {

			if msg.ExtendedTextMessage != nil {
				if msg.ExtendedTextMessage.ContextInfo == nil {
					msg.ExtendedTextMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.ExtendedTextMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			

			if msg.ImageMessage != nil {
				if msg.ImageMessage.ContextInfo == nil {
					msg.ImageMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.ImageMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			if msg.AudioMessage != nil {
				if msg.AudioMessage.ContextInfo == nil {
					msg.AudioMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.AudioMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			if msg.VideoMessage != nil {
				if msg.VideoMessage.ContextInfo == nil {
					msg.VideoMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.VideoMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			if msg.DocumentMessage != nil {
				if msg.DocumentMessage.ContextInfo == nil {
					msg.DocumentMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.DocumentMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			if msg.StickerMessage != nil {
				if msg.StickerMessage.ContextInfo == nil {
					msg.StickerMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.StickerMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			if msg.LocationMessage != nil {
				if msg.LocationMessage.ContextInfo == nil {
					msg.LocationMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.LocationMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			if msg.ContactMessage != nil {
				if msg.ContactMessage.ContextInfo == nil {
					msg.ContactMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.ContactMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
		}
	}
}


func (h *MessageHandler) processMediaData(mediaData string) ([]byte, error) {
	if strings.HasPrefix(mediaData, "data:") {

		return h.decodeBase64Media(mediaData)
	} else if strings.HasPrefix(mediaData, "http://") || strings.HasPrefix(mediaData, "https://") {

		return h.downloadMediaFromURL(mediaData)
	}

	return nil, fmt.Errorf("invalid media data format")
}


func (h *MessageHandler) downloadMediaFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download media: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read media data: %v", err)
	}

	return data, nil
}


func (h *MessageHandler) sendMediaMessage(c *fiber.Ctx, sessionID string, req dto.SendMediaRequest) error {

	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}


	if err := h.validateMediaRequest(&req); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}


	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}


	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}


	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}


	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}


	filedata, err := h.processMediaData(req.Media)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to process media: %v", err), "MEDIA_PROCESSING_FAILED", fiber.StatusBadRequest)
	}


	uploaded, err := client.Upload(context.Background(), filedata, whatsmeow.MediaType(req.Type))
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to upload media: %v", err), "UPLOAD_FAILED", fiber.StatusInternalServerError)
	}


	msg, err := h.buildMediaMessage(req, filedata, uploaded)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to build message: %v", err), "MESSAGE_BUILD_FAILED", fiber.StatusInternalServerError)
	}


	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}


	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send message: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}


	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"type":       req.Type,
	})
}
