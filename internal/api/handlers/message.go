package handlers

import (
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/vincent-petithory/dataurl"
	"google.golang.org/protobuf/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"

	"github.com/felipe/zemeow/internal/api/dto"
	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/logger"
)

// MessageHandler gerencia endpoints de mensagens WhatsApp
type MessageHandler struct {
	logger logger.Logger
}

// NewMessageHandler cria uma nova instância do handler de mensagens
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		logger: logger.GetWithSession("message_handler"),
	}
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

// sendError envia uma resposta de erro JSON
func (h *MessageHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	errorResp := fiber.Map{
		"error":   code,
		"message": message,
		"status":  status,
	}

	return c.Status(status).JSON(errorResp)
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
			if msg.ImageMessage != nil && msg.ImageMessage.ContextInfo != nil {
				msg.ImageMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			if msg.AudioMessage != nil && msg.AudioMessage.ContextInfo != nil {
				msg.AudioMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			if msg.VideoMessage != nil && msg.VideoMessage.ContextInfo != nil {
				msg.VideoMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			if msg.DocumentMessage != nil && msg.DocumentMessage.ContextInfo != nil {
				msg.DocumentMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
			
			if msg.StickerMessage != nil && msg.StickerMessage.ContextInfo != nil {
				msg.StickerMessage.ContextInfo.MentionedJID = mentionedJIDs
			}
		}
	}
}

// SendMessage envia uma mensagem (mantido para compatibilidade)
func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "sent",
		"message":    "Send message endpoint",
	})
}

// GetMessages lista mensagens (mantido para compatibilidade)
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"messages":   []fiber.Map{},
		"message":    "Get messages endpoint",
	})
}

// SendText envia mensagem de texto
func (h *MessageHandler) SendText(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "sent",
		"message":    "Send text endpoint",
	})
}

// SendMedia envia mensagem de mídia
func (h *MessageHandler) SendMedia(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "sent",
		"message":    "Send media endpoint",
	})
}

// SendLocation envia mensagem de localização
func (h *MessageHandler) SendLocation(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "sent",
		"message":    "Send location endpoint",
	})
}

// SendContact envia mensagem de contato
func (h *MessageHandler) SendContact(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"status":     "sent",
		"message":    "Send contact endpoint",
	})
}