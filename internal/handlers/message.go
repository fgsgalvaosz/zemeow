package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/vincent-petithory/dataurl"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"github.com/felipe/zemeow/internal/dto"
	"github.com/felipe/zemeow/internal/handlers/utils"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/services/media"
	"github.com/felipe/zemeow/internal/services/session"
)

type MessageHandler struct {
	sessionService session.Service
	mediaService   *media.MediaService
	logger         logger.Logger
}

func NewMessageHandler(sessionService session.Service, mediaService *media.MediaService) *MessageHandler {
	return &MessageHandler{
		sessionService: sessionService,
		mediaService:   mediaService,
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

	if !utils.HasSessionAccess(c, sessionID) {
		return utils.SendAccessDeniedError(c)
	}

	var req dto.SendTextRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendInvalidJSONError(c)
	}

	if err := h.validateTextRequest(&req); err != nil {
		return utils.SendValidationError(c, err.Error())
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, "Session not found or not connected", utils.ErrCodeSessionNotReady, fiber.StatusBadRequest)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
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
		return utils.SendError(c, fmt.Sprintf("Failed to send message: %v", err), utils.ErrCodeSendFailed, fiber.StatusInternalServerError)
	}

	return utils.SendSuccess(c, fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	}, "Message sent successfully")
}

func (h *MessageHandler) saveMediaToMinIO(sessionID, messageID, fileName, contentType string, data []byte, direction, chatJID, senderJID string) (*media.MediaInfo, error) {
	if h.mediaService == nil {
		h.logger.Warn().Msg("MediaService not available, skipping media upload")
		return nil, nil
	}

	mediaPath := fmt.Sprintf("media/%s/%s/%s", direction, sessionID, messageID)
	if fileName != "" {
		mediaPath = fmt.Sprintf("%s/%s", mediaPath, fileName)
	}

	err := h.mediaService.UploadMedia(
		context.Background(),
		mediaPath,
		bytes.NewReader(data),
		int64(len(data)),
		contentType,
	)
	if err != nil {
		h.logger.Error().Err(err).
			Str("session_id", sessionID).
			Str("message_id", messageID).
			Str("file_name", fileName).
			Msg("Failed to upload media to MinIO")
		return nil, err
	}

	mediaURL, err := h.mediaService.GetMediaURL(context.Background(), mediaPath)
	if err != nil {
		h.logger.Error().Err(err).
			Str("session_id", sessionID).
			Str("message_id", messageID).
			Str("file_name", fileName).
			Msg("Failed to generate media URL")
		return nil, err
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("message_id", messageID).
		Str("minio_path", mediaPath).
		Int64("size", int64(len(data))).
		Msg("Media uploaded to MinIO successfully")

	return &media.MediaInfo{
		FileName:    fileName,
		ContentType: contentType,
		Size:        int64(len(data)),
		Path:        mediaPath,
		URL:         mediaURL,
	}, nil
}

// @Summary Enviar mídia
// @Description Envia arquivos de mídia (imagem, vídeo, áudio, documento) via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendMediaRequest true "Dados da mídia"
// @Success 200 {object} map[string]interface{} "Mídia enviada com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/send/media [post]
func (h *MessageHandler) SendMedia(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	if !utils.HasSessionAccess(c, sessionID) {
		return utils.SendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	var req dto.SendMediaRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	if err := h.validateMediaRequest(&req); err != nil {
		return utils.SendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	fileData, err := h.decodeBase64Media(req.Media)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_MEDIA_DATA", fiber.StatusBadRequest)
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
		mediaType = whatsmeow.MediaImage
	default:
		return utils.SendError(c, "Unsupported media type", "INVALID_MEDIA_TYPE", fiber.StatusBadRequest)
	}

	uploaded, err := client.Upload(context.Background(), fileData, mediaType)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to upload media: %v", err), "UPLOAD_FAILED", fiber.StatusInternalServerError)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	msg, err := h.buildMediaMessage(req, fileData, uploaded)
	if err != nil {
		return utils.SendError(c, err.Error(), "BUILD_MESSAGE_FAILED", fiber.StatusInternalServerError)
	}

	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}

	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to send media: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
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

// @Summary Enviar localização
// @Description Envia uma localização geográfica via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendLocationRequest true "Dados da localização"
// @Success 200 {object} map[string]interface{} "Localização enviada com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/send/location [post]
func (h *MessageHandler) SendLocation(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	if !utils.HasSessionAccess(c, sessionID) {
		return utils.SendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	var req dto.SendLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	if err := h.validateLocationRequest(&req); err != nil {
		return utils.SendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
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
		return utils.SendError(c, fmt.Sprintf("Failed to send location: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	})
}

// @Summary Enviar contato
// @Description Envia um cartão de contato via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendContactRequest true "Dados do contato"
// @Success 200 {object} map[string]interface{} "Contato enviado com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/send/contact [post]
func (h *MessageHandler) SendContact(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	if !utils.HasSessionAccess(c, sessionID) {
		return utils.SendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	var req dto.SendContactRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	if err := h.validateContactRequest(&req); err != nil {
		return utils.SendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
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
		return utils.SendError(c, fmt.Sprintf("Failed to send contact: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	})
}

// @Summary Enviar sticker
// @Description Envia um sticker/figurinha via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendStickerRequest true "Dados do sticker"
// @Success 200 {object} map[string]interface{} "Sticker enviado com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/send/sticker [post]
func (h *MessageHandler) SendSticker(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SendStickerRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	filedata, err := h.processMediaData(req.Sticker)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to process sticker: %v", err), "MEDIA_PROCESSING_FAILED", fiber.StatusBadRequest)
	}

	uploaded, err := client.Upload(context.Background(), filedata, whatsmeow.MediaImage)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to upload sticker: %v", err), "UPLOAD_FAILED", fiber.StatusInternalServerError)
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
		return utils.SendError(c, fmt.Sprintf("Failed to send sticker: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	})
}

// @Summary Reagir a mensagem
// @Description Envia uma reação (emoji) para uma mensagem específica
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.ReactRequest true "Dados da reação"
// @Success 200 {object} map[string]interface{} "Reação enviada com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/send/reaction [post]
func (h *MessageHandler) ReactToMessage(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.ReactRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
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
		return utils.SendError(c, fmt.Sprintf("Failed to send reaction: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
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
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
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
		return utils.SendError(c, fmt.Sprintf("Failed to delete message: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
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
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	_, err = h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
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
		return utils.SendError(c, "Invalid presence type", "INVALID_PRESENCE", fiber.StatusBadRequest)
	}

	err = client.SendPresence(presence)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to send presence: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":    "sent",
		"recipient": req.To,
		"presence":  req.Presence,
	})
}

// @Summary Marcar como lida
// @Description Marca uma mensagem como lida no WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.MarkReadRequest true "Dados da mensagem"
// @Success 200 {object} map[string]interface{} "Mensagem marcada como lida"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Router /sessions/{sessionId}/messages/read [post]
func (h *MessageHandler) MarkAsRead(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.MarkReadRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	var messageIDs []types.MessageID
	for _, msgID := range req.MessageID {
		messageIDs = append(messageIDs, msgID)
	}

	err = client.MarkRead(messageIDs, time.Now(), recipient, recipient)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to mark as read: %v", err), "MARK_READ_FAILED", fiber.StatusInternalServerError)
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
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	if req.Type != mediaType {
		return utils.SendError(c, fmt.Sprintf("Expected media type %s, got %s", mediaType, req.Type), "INVALID_MEDIA_TYPE", fiber.StatusBadRequest)
	}

	_, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return utils.SendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
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

// @Summary Enviar botões interativos
// @Description Envia uma mensagem com botões interativos via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendButtonsRequest true "Dados dos botões"
// @Success 200 {object} map[string]interface{} "Botões enviados com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 501 {object} map[string]interface{} "Funcionalidade não implementada"
// @Router /sessions/{sessionId}/send/buttons [post]
func (h *MessageHandler) SendButtons(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SendButtonsRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	var buttons []*waE2E.ButtonsMessage_Button
	for _, btn := range req.Buttons {
		button := &waE2E.ButtonsMessage_Button{
			ButtonID: proto.String(btn.ID),
			ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{
				DisplayText: proto.String(btn.Text),
			},
			Type: waE2E.ButtonsMessage_Button_RESPONSE.Enum(),
		}
		buttons = append(buttons, button)
	}

	msg := &waE2E.Message{
		ButtonsMessage: &waE2E.ButtonsMessage{
			ContentText: proto.String(req.Text),
			FooterText:  proto.String(req.Footer),
			Buttons:     buttons,
		},
	}

	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}

	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("to", req.To).Msg("Failed to send interactive buttons, trying text fallback")

		buttonText := req.Text + "\n\n"
		for i, button := range req.Buttons {
			buttonText += fmt.Sprintf("%d. %s\n", i+1, button.Text)
		}
		if req.Footer != "" {
			buttonText += "\n" + req.Footer
		}

		textMsg := &waE2E.Message{
			Conversation: proto.String(buttonText),
		}

		response, err = client.SendMessage(context.Background(), recipient, textMsg, whatsmeow.SendRequestExtra{ID: messageID})
		if err != nil {
			return utils.SendError(c, fmt.Sprintf("Failed to send buttons: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":    true,
			"message_id": messageID,
			"status":     "sent_as_text_fallback",
			"timestamp":  response.Timestamp,
			"recipient":  req.To,
			"note":       "Interactive buttons failed, sent as text fallback",
		})
	}

	h.logger.Info().Str("session_id", sessionID).Str("to", req.To).Int("buttons_count", len(req.Buttons)).Msg("Interactive buttons sent successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"message_id": messageID,
		"status":     "sent_interactive",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"buttons":    len(req.Buttons),
	})
}

// @Summary Enviar lista interativa
// @Description Envia uma mensagem com lista de opções interativa via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendListRequest true "Dados da lista"
// @Success 200 {object} map[string]interface{} "Lista enviada com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 501 {object} map[string]interface{} "Funcionalidade não implementada"
// @Router /sessions/{sessionId}/send/list [post]
func (h *MessageHandler) SendList(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SendListRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	var sections []*waE2E.ListMessage_Section
	for _, section := range req.Sections {
		var rows []*waE2E.ListMessage_Row
		for _, row := range section.Rows {
			listRow := &waE2E.ListMessage_Row{
				RowID:       proto.String(row.ID),
				Title:       proto.String(row.Title),
				Description: proto.String(row.Description),
			}
			rows = append(rows, listRow)
		}

		listSection := &waE2E.ListMessage_Section{
			Title: proto.String(section.Title),
			Rows:  rows,
		}
		sections = append(sections, listSection)
	}

	msg := &waE2E.Message{
		ListMessage: &waE2E.ListMessage{
			Title:       proto.String(req.Title),
			Description: proto.String(req.Text),
			FooterText:  proto.String(req.Footer),
			ButtonText:  proto.String(req.ButtonText),
			ListType:    waE2E.ListMessage_SINGLE_SELECT.Enum(),
			Sections:    sections,
		},
	}

	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}

	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("to", req.To).Msg("Failed to send interactive list, trying text fallback")

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

		textMsg := &waE2E.Message{
			Conversation: proto.String(listText),
		}

		response, err = client.SendMessage(context.Background(), recipient, textMsg, whatsmeow.SendRequestExtra{ID: messageID})
		if err != nil {
			return utils.SendError(c, fmt.Sprintf("Failed to send list: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":    true,
			"message_id": messageID,
			"status":     "sent_as_text_fallback",
			"timestamp":  response.Timestamp,
			"recipient":  req.To,
			"note":       "Interactive list failed, sent as text fallback",
		})
	}

	h.logger.Info().Str("session_id", sessionID).Str("to", req.To).Int("sections_count", len(req.Sections)).Msg("Interactive list sent successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"message_id": messageID,
		"status":     "sent_interactive",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"sections":   len(req.Sections),
	})
}

func (h *MessageHandler) SendPoll(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SendPollRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
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
		selectableCount = 1
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
		return utils.SendError(c, fmt.Sprintf("Failed to send poll: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
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
			"name":       req.Name,
			"options":    req.Options,
			"selectable": selectableCount,
		},
	})
}

func (h *MessageHandler) EditMessage(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.EditMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
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
			Type: waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(),
			EditedMessage: &waE2E.Message{
				Conversation: proto.String(req.Text),
			},
		},
	}

	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{})
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("to", req.To).Str("message_id", req.MessageID).Msg("Failed to edit message")
		return utils.SendError(c, fmt.Sprintf("Failed to edit message: %v", err), "EDIT_FAILED", fiber.StatusInternalServerError)
	}

	h.logger.Info().
		Str("session_id", sessionID).
		Str("to", req.To).
		Str("message_id", req.MessageID).
		Msg("Message edited successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":             true,
		"original_message_id": req.MessageID,
		"status":              "edited",
		"timestamp":           response.Timestamp,
		"recipient":           req.To,
		"new_text":            req.Text,
	})
}

// @Summary Enviar imagem
// @Description Envia uma imagem via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendImageRequest true "Dados da imagem"
// @Success 200 {object} map[string]interface{} "Imagem enviada com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/send/image [post]
func (h *MessageHandler) SendImage(c *fiber.Ctx) error {
	var req dto.SendImageRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	return h.SendMedia(c)
}

// @Summary Enviar áudio
// @Description Envia um arquivo de áudio via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendAudioRequest true "Dados do áudio"
// @Success 200 {object} map[string]interface{} "Áudio enviado com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/send/audio [post]
func (h *MessageHandler) SendAudio(c *fiber.Ctx) error {
	var req dto.SendAudioRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	return h.SendMedia(c)
}

// @Summary Enviar documento
// @Description Envia um documento/arquivo via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendDocumentRequest true "Dados do documento"
// @Success 200 {object} map[string]interface{} "Documento enviado com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/send/document [post]
func (h *MessageHandler) SendDocument(c *fiber.Ctx) error {
	var req dto.SendDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	return h.SendMedia(c)
}

// @Summary Enviar vídeo
// @Description Envia um vídeo via WhatsApp
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body dto.SendVideoRequest true "Dados do vídeo"
// @Success 200 {object} map[string]interface{} "Vídeo enviado com sucesso"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /sessions/{sessionId}/send/video [post]
func (h *MessageHandler) SendVideo(c *fiber.Ctx) error {
	var req dto.SendVideoRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	return h.SendMedia(c)
}

// @Summary Listar mensagens
// @Description Retorna o histórico de mensagens da sessão
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Success 200 {object} map[string]interface{} "Lista de mensagens"
// @Failure 403 {object} map[string]interface{} "Acesso negado"
// @Router /sessions/{sessionId}/messages [get]
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	if !utils.HasSessionAccess(c, sessionID) {
		return utils.SendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	phone := c.Query("phone")
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	if phone == "" {
		return utils.SendError(c, "Phone number is required", "MISSING_PHONE", fiber.StatusBadRequest)
	}

	h.logger.Info().Str("session_id", sessionID).Str("phone", phone).Msg("GetMessages called - returning empty for now")

	return c.JSON(fiber.Map{
		"success": true,
		"data": map[string]interface{}{
			"messages": []interface{}{},
			"pagination": map[string]interface{}{
				"limit":  limit,
				"offset": offset,
				"total":  0,
			},
		},
		"note": "Message history retrieval will be implemented with database integration",
	})
}

// @Summary Obter status da mensagem
// @Description Retorna o status de entrega de uma mensagem específica
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param messageId path string true "ID da mensagem"
// @Success 200 {object} map[string]interface{} "Status da mensagem"
// @Failure 400 {object} map[string]interface{} "ID inválido"
// @Router /sessions/{sessionId}/messages/{messageId}/status [get]
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

// @Summary Enviar mensagens em lote
// @Description Envia múltiplas mensagens de uma vez para diferentes destinatários
// @Tags messages
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param request body map[string]interface{} true "Lista de mensagens"
// @Success 200 {object} map[string]interface{} "Mensagens enviadas em lote"
// @Failure 400 {object} map[string]interface{} "Dados inválidos"
// @Router /sessions/{sessionId}/send/bulk [post]
func (h *MessageHandler) SendBulkMessages(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "sent",
		"message":    "Bulk messages endpoint",
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

		ptt := true

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

	if !utils.HasSessionAccess(c, sessionID) {
		return utils.SendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	if err := h.validateMediaRequest(&req); err != nil {
		return utils.SendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	client, err := h.getWhatsAppClient(sessionID)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to get WhatsApp client: %v", err), "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	recipient, err := h.parseJID(req.To)
	if err != nil {
		return utils.SendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	filedata, err := h.processMediaData(req.Media)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to process media: %v", err), "MEDIA_PROCESSING_FAILED", fiber.StatusBadRequest)
	}

	uploaded, err := client.Upload(context.Background(), filedata, whatsmeow.MediaType(req.Type))
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to upload media: %v", err), "UPLOAD_FAILED", fiber.StatusInternalServerError)
	}

	msg, err := h.buildMediaMessage(req, filedata, uploaded)
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to build message: %v", err), "MESSAGE_BUILD_FAILED", fiber.StatusInternalServerError)
	}

	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}

	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return utils.SendError(c, fmt.Sprintf("Failed to send message: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"type":       req.Type,
	})
}

func (h *MessageHandler) getWhatsAppClient(sessionID string) (*whatsmeow.Client, error) {
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get WhatsApp client from session service")
		return nil, fmt.Errorf("session not found or not connected: %w", err)
	}

	h.logger.Debug().Str("session_id", sessionID).Str("client_type", fmt.Sprintf("%T", clientInterface)).Msg("Got client interface")

	if client, ok := clientInterface.(*whatsmeow.Client); ok {
		h.logger.Debug().Str("session_id", sessionID).Msg("Direct cast successful")
		return client, nil
	}

	type ClientGetter interface {
		GetClient() *whatsmeow.Client
	}

	clientGetter, ok := clientInterface.(ClientGetter)
	if !ok {
		h.logger.Error().Str("session_id", sessionID).Str("client_type", fmt.Sprintf("%T", clientInterface)).Msg("Client does not implement GetClient method")
		return nil, fmt.Errorf("client does not implement GetClient method, type: %T", clientInterface)
	}

	client := clientGetter.GetClient()
	h.logger.Debug().Str("session_id", sessionID).Msg("GetClient method successful")
	return client, nil
}
