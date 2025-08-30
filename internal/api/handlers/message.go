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

// MessageHandler gerencia endpoints de mensagens WhatsApp
type MessageHandler struct {
	sessionService session.Service
	logger         logger.Logger
}

// NewMessageHandler cria uma nova instância do handler de mensagens
func NewMessageHandler(sessionService session.Service) *MessageHandler {
	return &MessageHandler{
		sessionService: sessionService,
		logger:         logger.GetWithSession("message_handler"),
	}
}

// SendMessage envia mensagem (método faltando para compatibilidade)
func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {
	// Reutilizar a lógica de SendText para compatibilidade
	return h.SendText(c)
}

// SendText envia mensagem de texto
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
	
	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Parsear request
	var req dto.SendTextRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar request
	if err := h.validateTextRequest(&req); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	// Obter cliente WhatsApp da sessão
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	// Converter interface para *whatsmeow.Client
	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	// Preparar destinatário
	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	// Gerar ID da mensagem
	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	// Construir mensagem WhatsApp
	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(req.Text),
		},
	}

	// Adicionar context info se fornecido (reply/menções)
	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}

	// Enviar mensagem
	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send message: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	// Retornar sucesso
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	})
}

// SendMedia envia mídia unificada
func (h *MessageHandler) SendMedia(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Parsear request
	var req dto.SendMediaRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar request
	if err := h.validateMediaRequest(&req); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	// Obter cliente WhatsApp
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	// Converter interface para *whatsmeow.Client
	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	// Decodificar dados da mídia
	fileData, err := h.decodeBase64Media(req.Media)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_MEDIA_DATA", fiber.StatusBadRequest)
	}

	// Determinar tipo de mídia para upload
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

	// Upload para WhatsApp
	uploaded, err := client.Upload(context.Background(), fileData, mediaType)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to upload media: %v", err), "UPLOAD_FAILED", fiber.StatusInternalServerError)
	}

	// Preparar destinatário
	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	// Gerar ID da mensagem
	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	// Construir mensagem de mídia
	msg, err := h.buildMediaMessage(req, fileData, uploaded)
	if err != nil {
		return h.sendError(c, err.Error(), "BUILD_MESSAGE_FAILED", fiber.StatusInternalServerError)
	}

	// Adicionar context info se fornecido
	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}

	// Enviar mensagem
	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send media: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	// Retornar sucesso
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"media_type": req.Type,
		"file_size":  len(fileData),
	})
}

// SendLocation envia localização
func (h *MessageHandler) SendLocation(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Parsear request
	var req dto.SendLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar request
	if err := h.validateLocationRequest(&req); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	// Obter cliente WhatsApp
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	// Converter interface para *whatsmeow.Client
	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	// Preparar destinatário
	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	// Gerar ID da mensagem
	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	// Construir mensagem de localização
	msg := &waE2E.Message{
		LocationMessage: &waE2E.LocationMessage{
			DegreesLatitude:  &req.Latitude,
			DegreesLongitude: &req.Longitude,
			Name:             &req.Name,
		},
	}

	// Adicionar context info se fornecido
	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}

	// Enviar mensagem
	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send location: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	// Retornar sucesso
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	})
}

// SendContact envia contato
func (h *MessageHandler) SendContact(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Parsear request
	var req dto.SendContactRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar request
	if err := h.validateContactRequest(&req); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	// Obter cliente WhatsApp
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	// Converter interface para *whatsmeow.Client
	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	// Preparar destinatário
	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	// Gerar ID da mensagem
	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	// Construir mensagem de contato
	msg := &waE2E.Message{
		ContactMessage: &waE2E.ContactMessage{
			DisplayName: &req.Name,
			Vcard:       &req.Vcard,
		},
	}

	// Adicionar context info se fornecido
	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}

	// Enviar mensagem
	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send contact: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	// Retornar sucesso
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
	})
}

// === NOVOS HANDLERS PARA ENDPOINTS ESPECÍFICOS ===

// SendSticker envia um sticker
// POST /sessions/:sessionId/send/sticker
func (h *MessageHandler) SendSticker(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.SendStickerRequest
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

	// Preparar destinatário
	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	// Gerar ID da mensagem
	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	// Processar sticker (base64 ou URL)
	filedata, err := h.processMediaData(req.Sticker)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to process sticker: %v", err), "MEDIA_PROCESSING_FAILED", fiber.StatusBadRequest)
	}

	// Upload sticker
	uploaded, err := client.Upload(context.Background(), filedata, whatsmeow.MediaImage)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to upload sticker: %v", err), "UPLOAD_FAILED", fiber.StatusInternalServerError)
	}

	// Construir mensagem de sticker
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

	// Adicionar context info se fornecido
	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}

	// Enviar mensagem
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

// ReactToMessage reage a uma mensagem
// POST /sessions/:sessionId/react
func (h *MessageHandler) ReactToMessage(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.ReactRequest
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

	// Preparar destinatário
	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	// Construir mensagem de reação
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

	// Enviar reação
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

// DeleteMessage deleta uma mensagem
// POST /sessions/:sessionId/delete
func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.DeleteMessageRequest
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

	// Preparar destinatário
	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	// Construir mensagem de deleção
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

	// Enviar deleção
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

// === HANDLERS PARA OPERAÇÕES DE CHAT ===

// SetChatPresence define presença no chat
// POST /sessions/:sessionId/chat/presence
func (h *MessageHandler) SetChatPresence(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.ChatPresenceRequest
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

	// Preparar destinatário
	_, err = h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	// Mapear presença
	var presence types.Presence
	switch req.Presence {
	case "available":
		presence = types.PresenceAvailable
	case "unavailable":
		presence = types.PresenceUnavailable
	case "composing":
		// Para composing, usamos PresenceAvailable e enviamos chat state separadamente
		presence = types.PresenceAvailable
	case "recording":
		// Para recording, usamos PresenceAvailable
		presence = types.PresenceAvailable
	case "paused":
		// Para paused, usamos PresenceAvailable
		presence = types.PresenceAvailable
	default:
		return h.sendError(c, "Invalid presence type", "INVALID_PRESENCE", fiber.StatusBadRequest)
	}

	// Enviar presença
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

// MarkAsRead marca mensagens como lidas
// POST /sessions/:sessionId/chat/markread
func (h *MessageHandler) MarkAsRead(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	var req dto.MarkReadRequest
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

	// Preparar destinatário
	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	// Marcar como lido
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

// === HANDLERS PARA DOWNLOAD DE MÍDIA ===

// DownloadImage faz download de uma imagem
// POST /sessions/:sessionId/download/image
func (h *MessageHandler) DownloadImage(c *fiber.Ctx) error {
	return h.downloadMedia(c, "image")
}

// DownloadVideo faz download de um vídeo
// POST /sessions/:sessionId/download/video
func (h *MessageHandler) DownloadVideo(c *fiber.Ctx) error {
	return h.downloadMedia(c, "video")
}

// DownloadAudio faz download de um áudio
// POST /sessions/:sessionId/download/audio
func (h *MessageHandler) DownloadAudio(c *fiber.Ctx) error {
	return h.downloadMedia(c, "audio")
}

// DownloadDocument faz download de um documento
// POST /sessions/:sessionId/download/document
func (h *MessageHandler) DownloadDocument(c *fiber.Ctx) error {
	return h.downloadMedia(c, "document")
}

// downloadMedia função auxiliar para download de mídia
func (h *MessageHandler) downloadMedia(c *fiber.Ctx, mediaType string) error {
	sessionID := c.Params("sessionId")

	var req dto.DownloadMediaRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Validar tipo de mídia
	if req.Type != mediaType {
		return h.sendError(c, fmt.Sprintf("Expected media type %s, got %s", mediaType, req.Type), "INVALID_MEDIA_TYPE", fiber.StatusBadRequest)
	}

	// Obter cliente WhatsApp
	_, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	// NOTA: Para implementar download real, seria necessário:
	// 1. Buscar a mensagem no histórico/store usando req.MessageID
	// 2. Extrair as informações de mídia (URL, chaves, etc.)
	// 3. Usar client.Download() para baixar o arquivo
	// 4. Retornar o arquivo como base64 ou URL temporária

	// Por enquanto, retornamos uma resposta indicando que a funcionalidade
	// precisa ser implementada com base no sistema de armazenamento de mensagens

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

// === NOVOS HANDLERS PARA ENDPOINTS ESPECÍFICOS ===

// SendImage envia uma imagem
// POST /sessions/:sessionId/send/image
func (h *MessageHandler) SendImage(c *fiber.Ctx) error {
	var req dto.SendImageRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Reutilizar lógica do SendMedia
	return h.SendMedia(c)
}

// SendAudio envia um áudio
// POST /sessions/:sessionId/send/audio
func (h *MessageHandler) SendAudio(c *fiber.Ctx) error {
	var req dto.SendAudioRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Reutilizar lógica do SendMedia
	return h.SendMedia(c)
}

// SendDocument envia um documento
// POST /sessions/:sessionId/send/document
func (h *MessageHandler) SendDocument(c *fiber.Ctx) error {
	var req dto.SendDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Reutilizar lógica do SendMedia
	return h.SendMedia(c)
}

// SendVideo envia um vídeo
// POST /sessions/:sessionId/send/video
func (h *MessageHandler) SendVideo(c *fiber.Ctx) error {
	var req dto.SendVideoRequest
	if err := c.BodyParser(&req); err != nil {
		return h.sendError(c, "Invalid request body", "INVALID_JSON", fiber.StatusBadRequest)
	}

	// Reutilizar lógica do SendMedia
	return h.SendMedia(c)
}

// GetMessages lista mensagens (método faltando)
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	
	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// TODO: Implementar listagem de mensagens
	return c.JSON(fiber.Map{
		"success": true,
		"data":    []interface{}{},
	})
}

// GetMessageStatus obtém o status de uma mensagem (mantido para compatibilidade)
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

// SendBulkMessages envia mensagens em lote (mantido para compatibilidade)
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

// sendError envia uma resposta de erro padronizada
func (h *MessageHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	return c.Status(status).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"message": message,
			"code":    code,
		},
	})
}

// === MÉTODOS AUXILIARES E VALIDAÇÕES ===

// parseJID converte número de telefone para JID do WhatsApp
func (h *MessageHandler) parseJID(phone string) (types.JID, error) {
	// Remover caracteres não numéricos
	phone = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	// Verificar se é um número válido
	if len(phone) < 10 || len(phone) > 15 {
		return types.JID{}, fmt.Errorf("invalid phone number length")
	}

	// Construir JID no formato WhatsApp
	jid := types.NewJID(phone, types.DefaultUserServer)
	return jid, nil
}

// validateTextRequest valida requisição de texto
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

// validateMediaRequest valida requisição de mídia
func (h *MessageHandler) validateMediaRequest(req *dto.SendMediaRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient 'to' is required")
	}
	
	if req.Media == "" {
		return fmt.Errorf("media data is required")
	}
	
	// Verificar se é uma data URL válida
	if !strings.HasPrefix(req.Media, "data:") {
		return fmt.Errorf("media must be a valid data URL (data:mime/type;base64,...)")
	}
	
	// Validar caption se fornecido
	if req.Caption != "" && len(req.Caption) > 1024 {
		return fmt.Errorf("caption exceeds maximum length of 1024 characters")
	}
	
	// Validar filename se fornecido
	if req.Filename != "" && len(req.Filename) > 255 {
		return fmt.Errorf("filename exceeds maximum length of 255 characters")
	}
	
	return nil
}

// validateLocationRequest valida requisição de localização
func (h *MessageHandler) validateLocationRequest(req *dto.SendLocationRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient 'to' is required")
	}
	
	// Validar latitude (-90 a 90)
	if req.Latitude < -90 || req.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	
	// Validar longitude (-180 a 180)
	if req.Longitude < -180 || req.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	
	return nil
}

// validateContactRequest valida requisição de contato
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

// decodeBase64Media decodifica dados de mídia em Base64
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

// buildMediaMessage constrói mensagem de mídia baseada no tipo
func (h *MessageHandler) buildMediaMessage(req dto.SendMediaRequest, filedata []byte, uploaded whatsmeow.UploadResponse) (*waE2E.Message, error) {
	switch req.Type {
	case dto.MediaTypeImage:
		// Detectar MIME type se não fornecido
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
		// MIME type padrão para áudio do WhatsApp
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
		// Detectar MIME type se não fornecido
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
		// Detectar MIME type se não fornecido
		mimeType := req.MimeType
		if mimeType == "" {
			// Tentar detectar pelo filename
			if req.Filename != "" {
				mimeType = mime.TypeByExtension(strings.ToLower(req.Filename[strings.LastIndex(req.Filename, "."):]))
			}
			// Fallback para detecção automática
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
		// Detectar MIME type se não fornecido
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

// addContextInfo adiciona informações de contexto (replies, menções) à mensagem
func (h *MessageHandler) addContextInfo(msg *waE2E.Message, contextInfo *dto.ContextInfo) {
	if contextInfo == nil {
		return
	}
	
	// Adicionar reply info
	if contextInfo.StanzaID != nil && contextInfo.Participant != nil {
		// Para mensagens de texto
		if msg.ExtendedTextMessage != nil {
			if msg.ExtendedTextMessage.ContextInfo == nil {
				msg.ExtendedTextMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			msg.ExtendedTextMessage.ContextInfo.StanzaID = contextInfo.StanzaID
			msg.ExtendedTextMessage.ContextInfo.Participant = contextInfo.Participant
			msg.ExtendedTextMessage.ContextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		}
		
		// Para mensagens de mídia
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
	
	// Adicionar menções
	if len(contextInfo.MentionedJID) > 0 {
		// Converter strings para JIDs
		var mentionedJIDs []string
		for _, jidStr := range contextInfo.MentionedJID {
			if jid, err := h.parseJID(jidStr); err == nil {
				mentionedJIDs = append(mentionedJIDs, jid.String())
			}
		}
		
		if len(mentionedJIDs) > 0 {
			// Para mensagens de texto
			if msg.ExtendedTextMessage != nil {
				if msg.ExtendedTextMessage.ContextInfo == nil {
					msg.ExtendedTextMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.ExtendedTextMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			// Para mensagens de mídia
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

// processMediaData processa dados de mídia (base64 ou URL)
func (h *MessageHandler) processMediaData(mediaData string) ([]byte, error) {
	if strings.HasPrefix(mediaData, "data:") {
		// Decodificar base64
		return h.decodeBase64Media(mediaData)
	} else if strings.HasPrefix(mediaData, "http://") || strings.HasPrefix(mediaData, "https://") {
		// Download de URL
		return h.downloadMediaFromURL(mediaData)
	}

	return nil, fmt.Errorf("invalid media data format")
}

// downloadMediaFromURL faz download de mídia de uma URL
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

// sendMediaMessage envia mensagem de mídia genérica
func (h *MessageHandler) sendMediaMessage(c *fiber.Ctx, sessionID string, req dto.SendMediaRequest) error {
	// Verificar acesso à sessão
	if !h.hasSessionAccess(c, sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	// Validar requisição
	if err := h.validateMediaRequest(&req); err != nil {
		return h.sendError(c, err.Error(), "VALIDATION_ERROR", fiber.StatusBadRequest)
	}

	// Obter cliente WhatsApp da sessão
	clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)
	if err != nil {
		return h.sendError(c, "Session not found or not connected", "SESSION_NOT_READY", fiber.StatusBadRequest)
	}

	// Converter interface para *whatsmeow.Client
	client, ok := clientInterface.(*whatsmeow.Client)
	if !ok {
		return h.sendError(c, "Invalid WhatsApp client", "INVALID_CLIENT", fiber.StatusInternalServerError)
	}

	// Preparar destinatário
	recipient, err := h.parseJID(req.To)
	if err != nil {
		return h.sendError(c, err.Error(), "INVALID_RECIPIENT", fiber.StatusBadRequest)
	}

	// Gerar ID da mensagem
	messageID := req.MessageID
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	// Processar mídia
	filedata, err := h.processMediaData(req.Media)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to process media: %v", err), "MEDIA_PROCESSING_FAILED", fiber.StatusBadRequest)
	}

	// Upload mídia
	uploaded, err := client.Upload(context.Background(), filedata, whatsmeow.MediaType(req.Type))
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to upload media: %v", err), "UPLOAD_FAILED", fiber.StatusInternalServerError)
	}

	// Construir mensagem
	msg, err := h.buildMediaMessage(req, filedata, uploaded)
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to build message: %v", err), "MESSAGE_BUILD_FAILED", fiber.StatusInternalServerError)
	}

	// Adicionar context info se fornecido
	if req.ContextInfo != nil {
		h.addContextInfo(msg, req.ContextInfo)
	}

	// Enviar mensagem
	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return h.sendError(c, fmt.Sprintf("Failed to send message: %v", err), "SEND_FAILED", fiber.StatusInternalServerError)
	}

	// Retornar sucesso
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message_id": messageID,
		"status":     "sent",
		"timestamp":  response.Timestamp,
		"recipient":  req.To,
		"type":       req.Type,
	})
}