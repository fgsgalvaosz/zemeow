package message

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waE2E "go.mau.fi/whatsmeow/binary/proto"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
)

// PersistenceService gerencia a persistência de mensagens do WhatsApp
type PersistenceService struct {
	messageRepo repositories.MessageRepository
	logger      logger.Logger
}

// NewPersistenceService cria uma nova instância do PersistenceService
func NewPersistenceService(messageRepo repositories.MessageRepository) *PersistenceService {
	return &PersistenceService{
		messageRepo: messageRepo,
		logger:      logger.GetWithSession("message_persistence"),
	}
}

// ProcessMessageEvent processa um evento de mensagem do WhatsApp e persiste no banco
func (s *PersistenceService) ProcessMessageEvent(sessionID uuid.UUID, evt *events.Message) error {
	message, err := s.convertEventToMessage(sessionID, evt)
	if err != nil {
		s.logger.Error().Err(err).Str("message_id", evt.Info.ID).Msg("Failed to convert event to message")
		return fmt.Errorf("failed to convert event to message: %w", err)
	}

	// Verificar se a mensagem já existe
	existing, err := s.messageRepo.GetByMessageID(sessionID, evt.Info.ID)
	if err == nil && existing != nil {
		// Mensagem já existe, atualizar se necessário
		s.logger.Debug().Str("message_id", evt.Info.ID).Msg("Message already exists, updating")
		message.ID = existing.ID
		message.CreatedAt = existing.CreatedAt
		return s.messageRepo.Update(message)
	}

	// Criar nova mensagem
	err = s.messageRepo.Create(message)
	if err != nil {
		s.logger.Error().Err(err).Str("message_id", evt.Info.ID).Msg("Failed to persist message")
		return fmt.Errorf("failed to persist message: %w", err)
	}

	s.logger.Debug().
		Str("message_id", evt.Info.ID).
		Str("chat_jid", evt.Info.Chat.String()).
		Str("sender_jid", evt.Info.Sender.String()).
		Str("type", message.MessageType).
		Msg("Message persisted successfully")

	return nil
}

// ProcessReceiptEvent processa um evento de confirmação de leitura/entrega
func (s *PersistenceService) ProcessReceiptEvent(sessionID uuid.UUID, evt *events.Receipt) error {
	if len(evt.MessageIDs) == 0 {
		return nil
	}

	status := s.convertReceiptTypeToStatus(evt.Type)
	
	for _, messageID := range evt.MessageIDs {
		message, err := s.messageRepo.GetByMessageID(sessionID, messageID)
		if err != nil {
			s.logger.Warn().Err(err).Str("message_id", messageID).Msg("Message not found for receipt")
			continue
		}

		// Atualizar apenas se o novo status for "superior" ao atual
		if s.shouldUpdateStatus(message.Status, status) {
			message.Status = status
			err = s.messageRepo.Update(message)
			if err != nil {
				s.logger.Error().Err(err).Str("message_id", messageID).Msg("Failed to update message status")
			} else {
				s.logger.Debug().
					Str("message_id", messageID).
					Str("status", status).
					Msg("Message status updated")
			}
		}
	}

	return nil
}

// convertEventToMessage converte um evento de mensagem do WhatsApp para o modelo interno
func (s *PersistenceService) convertEventToMessage(sessionID uuid.UUID, evt *events.Message) (*models.Message, error) {
	message := &models.Message{
		SessionID:         sessionID,
		MessageID:         evt.Info.ID,
		ChatJID:           evt.Info.Chat.String(),
		SenderJID:         evt.Info.Sender.String(),
		Direction:         s.getMessageDirection(evt.Info.IsFromMe),
		Status:            string(models.MessageStatusReceived),
		IsFromMe:          evt.Info.IsFromMe,
		IsEphemeral:       evt.IsEphemeral,
		IsViewOnce:        evt.IsViewOnce,
		IsForwarded:       s.isForwardedMessage(evt.Message),
		Timestamp:         evt.Info.Timestamp,
		ContextInfo:       make(models.JSONB),
	}

	// Serializar mensagem completa como JSON
	rawMessage, err := json.Marshal(evt.Message)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal raw message")
	} else {
		message.RawMessage = rawMessage
	}

	// Processar diferentes tipos de mensagem
	err = s.processMessageContent(message, evt.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to process message content: %w", err)
	}

	// Processar informações de contexto (reply, mentions, etc.)
	s.processContextInfo(message, evt.Message)

	// Definir recipient_jid para mensagens diretas
	if evt.Info.Chat.Server != types.GroupServer && evt.Info.Chat.Server != types.BroadcastServer {
		if evt.Info.IsFromMe {
			message.RecipientJID = &message.ChatJID
		} else {
			ourJID := evt.Info.Chat.String() // Em chat direto, chat_jid é o JID do outro usuário
			message.RecipientJID = &ourJID
		}
	}

	return message, nil
}

// processMessageContent processa o conteúdo específico de cada tipo de mensagem
func (s *PersistenceService) processMessageContent(message *models.Message, waMsg *waE2E.Message) error {
	switch {
	case waMsg.Conversation != nil:
		message.MessageType = string(models.MessageTypeText)
		message.Content = waMsg.Conversation

	case waMsg.ExtendedTextMessage != nil:
		message.MessageType = string(models.MessageTypeText)
		message.Content = waMsg.ExtendedTextMessage.Text
		if waMsg.ExtendedTextMessage.ContextInfo != nil {
			s.processExtendedContextInfo(message, waMsg.ExtendedTextMessage.ContextInfo)
		}

	case waMsg.ImageMessage != nil:
		message.MessageType = string(models.MessageTypeImage)
		s.processMediaMessage(message, &MediaInfo{
			URL:      waMsg.ImageMessage.URL,
			MimeType: waMsg.ImageMessage.Mimetype,
			Size:     waMsg.ImageMessage.FileLength,
			SHA256:   waMsg.ImageMessage.FileSHA256,
			Caption:  waMsg.ImageMessage.Caption,
		})

	case waMsg.AudioMessage != nil:
		message.MessageType = string(models.MessageTypeAudio)
		s.processMediaMessage(message, &MediaInfo{
			URL:      waMsg.AudioMessage.URL,
			MimeType: waMsg.AudioMessage.Mimetype,
			Size:     waMsg.AudioMessage.FileLength,
			SHA256:   waMsg.AudioMessage.FileSHA256,
		})

	case waMsg.VideoMessage != nil:
		message.MessageType = string(models.MessageTypeVideo)
		s.processMediaMessage(message, &MediaInfo{
			URL:      waMsg.VideoMessage.URL,
			MimeType: waMsg.VideoMessage.Mimetype,
			Size:     waMsg.VideoMessage.FileLength,
			SHA256:   waMsg.VideoMessage.FileSHA256,
			Caption:  waMsg.VideoMessage.Caption,
		})

	case waMsg.DocumentMessage != nil:
		message.MessageType = string(models.MessageTypeDocument)
		s.processMediaMessage(message, &MediaInfo{
			URL:      waMsg.DocumentMessage.URL,
			MimeType: waMsg.DocumentMessage.Mimetype,
			Size:     waMsg.DocumentMessage.FileLength,
			SHA256:   waMsg.DocumentMessage.FileSHA256,
			Caption:  waMsg.DocumentMessage.Caption,
			Filename: waMsg.DocumentMessage.FileName,
		})

	case waMsg.StickerMessage != nil:
		message.MessageType = string(models.MessageTypeSticker)
		s.processMediaMessage(message, &MediaInfo{
			URL:      waMsg.StickerMessage.URL,
			MimeType: waMsg.StickerMessage.Mimetype,
			Size:     waMsg.StickerMessage.FileLength,
			SHA256:   waMsg.StickerMessage.FileSHA256,
		})

	case waMsg.LocationMessage != nil:
		message.MessageType = string(models.MessageTypeLocation)
		s.processLocationMessage(message, waMsg.LocationMessage)

	case waMsg.ContactMessage != nil:
		message.MessageType = string(models.MessageTypeContact)
		s.processContactMessage(message, waMsg.ContactMessage)

	case waMsg.GroupInviteMessage != nil:
		message.MessageType = string(models.MessageTypeGroupInvite)
		s.processGroupInviteMessage(message, waMsg.GroupInviteMessage)

	case waMsg.PollCreationMessage != nil:
		message.MessageType = string(models.MessageTypePoll)
		s.processPollMessage(message, waMsg.PollCreationMessage)

	case waMsg.ReactionMessage != nil:
		message.MessageType = string(models.MessageTypeReaction)
		s.processReactionMessage(message, waMsg.ReactionMessage)

	default:
		message.MessageType = string(models.MessageTypeUnknown)
		s.logger.Warn().Msg("Unknown message type received")
	}

	return nil
}

// MediaInfo estrutura auxiliar para informações de mídia
type MediaInfo struct {
	URL      *string
	MimeType *string
	Size     *uint64
	SHA256   []byte
	Caption  *string
	Filename *string
}

// processMediaMessage processa mensagens de mídia
func (s *PersistenceService) processMediaMessage(message *models.Message, media *MediaInfo) {
	if media.URL != nil {
		message.MediaURL = media.URL
	}
	if media.MimeType != nil {
		message.MediaType = media.MimeType
	}
	if media.Size != nil {
		size := int64(*media.Size)
		message.MediaSize = &size
	}
	if media.SHA256 != nil {
		sha256 := fmt.Sprintf("%x", media.SHA256)
		message.MediaSHA256 = &sha256
	}
	if media.Caption != nil {
		message.Caption = media.Caption
	}
	if media.Filename != nil {
		message.MediaFilename = media.Filename
	}
}

// processLocationMessage processa mensagens de localização
func (s *PersistenceService) processLocationMessage(message *models.Message, loc *waE2E.LocationMessage) {
	if loc.DegreesLatitude != nil {
		lat := float64(*loc.DegreesLatitude)
		message.LocationLatitude = &lat
	}
	if loc.DegreesLongitude != nil {
		lng := float64(*loc.DegreesLongitude)
		message.LocationLongitude = &lng
	}
	if loc.Name != nil {
		message.LocationName = loc.Name
	}
	if loc.Address != nil {
		message.LocationAddress = loc.Address
	}
}

// processContactMessage processa mensagens de contato
func (s *PersistenceService) processContactMessage(message *models.Message, contact *waE2E.ContactMessage) {
	if contact.DisplayName != nil {
		message.ContactName = contact.DisplayName
	}
	if contact.Vcard != nil {
		message.ContactVCard = contact.Vcard
		// Extrair telefone do vCard se possível
		if phone := s.extractPhoneFromVCard(*contact.Vcard); phone != "" {
			message.ContactPhone = &phone
		}
	}
}

// processGroupInviteMessage processa mensagens de convite de grupo
func (s *PersistenceService) processGroupInviteMessage(message *models.Message, invite *waE2E.GroupInviteMessage) {
	if invite.InviteCode != nil {
		message.GroupInviteCode = invite.InviteCode
	}
	if invite.InviteExpiration != nil {
		expiration := time.Unix(int64(*invite.InviteExpiration), 0)
		message.GroupInviteExpiration = &expiration
	}
	if invite.GroupName != nil {
		message.Content = invite.GroupName
	}
}

// processPollMessage processa mensagens de enquete
func (s *PersistenceService) processPollMessage(message *models.Message, poll *waE2E.PollCreationMessage) {
	if poll.Name != nil {
		message.PollName = poll.Name
		message.Content = poll.Name
	}
	if poll.SelectableOptionsCount != nil {
		count := int(*poll.SelectableOptionsCount)
		message.PollSelectableCount = &count
	}

	// Processar opções da enquete
	if len(poll.Options) > 0 {
		options := make([]map[string]interface{}, len(poll.Options))
		for i, option := range poll.Options {
			optionMap := make(map[string]interface{})
			if option.OptionName != nil {
				optionMap["name"] = *option.OptionName
			}
			options[i] = optionMap
		}
		message.PollOptions = models.JSONB{"options": options}
	}
}

// processReactionMessage processa mensagens de reação
func (s *PersistenceService) processReactionMessage(message *models.Message, reaction *waE2E.ReactionMessage) {
	if reaction.Text != nil {
		message.ReactionEmoji = reaction.Text
	}
	if reaction.Key != nil && reaction.Key.ID != nil {
		message.ReplyToMessageID = reaction.Key.ID
	}

	// Timestamp da reação
	reactionTime := time.Now()
	message.ReactionTimestamp = &reactionTime
}

// processContextInfo processa informações de contexto da mensagem
func (s *PersistenceService) processContextInfo(message *models.Message, waMsg *waE2E.Message) {
	var contextInfo *waE2E.ContextInfo

	// Extrair ContextInfo de diferentes tipos de mensagem
	switch {
	case waMsg.ExtendedTextMessage != nil:
		contextInfo = waMsg.ExtendedTextMessage.ContextInfo
	case waMsg.ImageMessage != nil:
		contextInfo = waMsg.ImageMessage.ContextInfo
	case waMsg.VideoMessage != nil:
		contextInfo = waMsg.VideoMessage.ContextInfo
	case waMsg.DocumentMessage != nil:
		contextInfo = waMsg.DocumentMessage.ContextInfo
	case waMsg.AudioMessage != nil:
		contextInfo = waMsg.AudioMessage.ContextInfo
	case waMsg.LocationMessage != nil:
		contextInfo = waMsg.LocationMessage.ContextInfo
	case waMsg.ContactMessage != nil:
		contextInfo = waMsg.ContactMessage.ContextInfo
	}

	if contextInfo != nil {
		s.processExtendedContextInfo(message, contextInfo)
	}
}

// processExtendedContextInfo processa informações de contexto estendidas
func (s *PersistenceService) processExtendedContextInfo(message *models.Message, contextInfo *waE2E.ContextInfo) {
	// Mensagem citada
	if contextInfo.StanzaID != nil {
		message.ReplyToMessageID = contextInfo.StanzaID
		if contextInfo.QuotedMessage != nil {
			// Extrair conteúdo da mensagem citada
			quotedContent := s.extractQuotedContent(contextInfo.QuotedMessage)
			if quotedContent != "" {
				message.QuotedContent = &quotedContent
			}
		}
	}

	// Menções
	if len(contextInfo.MentionedJID) > 0 {
		mentions := make([]string, len(contextInfo.MentionedJID))
		for i, jid := range contextInfo.MentionedJID {
			mentions[i] = jid
		}
		message.Mentions = mentions
	}

	// Informações adicionais no contexto
	if contextInfo.IsForwarded != nil && *contextInfo.IsForwarded {
		message.IsForwarded = true
	}
}

// extractQuotedContent extrai o conteúdo de uma mensagem citada
func (s *PersistenceService) extractQuotedContent(quotedMsg *waE2E.Message) string {
	switch {
	case quotedMsg.Conversation != nil:
		return *quotedMsg.Conversation
	case quotedMsg.ExtendedTextMessage != nil && quotedMsg.ExtendedTextMessage.Text != nil:
		return *quotedMsg.ExtendedTextMessage.Text
	case quotedMsg.ImageMessage != nil && quotedMsg.ImageMessage.Caption != nil:
		return *quotedMsg.ImageMessage.Caption
	case quotedMsg.VideoMessage != nil && quotedMsg.VideoMessage.Caption != nil:
		return *quotedMsg.VideoMessage.Caption
	case quotedMsg.DocumentMessage != nil && quotedMsg.DocumentMessage.Caption != nil:
		return *quotedMsg.DocumentMessage.Caption
	case quotedMsg.LocationMessage != nil && quotedMsg.LocationMessage.Name != nil:
		return *quotedMsg.LocationMessage.Name
	case quotedMsg.ContactMessage != nil && quotedMsg.ContactMessage.DisplayName != nil:
		return *quotedMsg.ContactMessage.DisplayName
	default:
		return "[Media]"
	}
}

// extractPhoneFromVCard extrai o número de telefone de um vCard
func (s *PersistenceService) extractPhoneFromVCard(vcard string) string {
	lines := strings.Split(vcard, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "TEL:") {
			return strings.TrimPrefix(line, "TEL:")
		}
		if strings.HasPrefix(line, "TEL;") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return parts[1]
			}
		}
	}
	return ""
}

// getMessageDirection determina a direção da mensagem
func (s *PersistenceService) getMessageDirection(isFromMe bool) string {
	if isFromMe {
		return string(models.MessageDirectionOutgoing)
	}
	return string(models.MessageDirectionIncoming)
}

// isForwardedMessage verifica se a mensagem foi encaminhada
func (s *PersistenceService) isForwardedMessage(waMsg *waE2E.Message) bool {
	// Verificar em diferentes tipos de mensagem
	switch {
	case waMsg.ExtendedTextMessage != nil && waMsg.ExtendedTextMessage.ContextInfo != nil:
		return waMsg.ExtendedTextMessage.ContextInfo.IsForwarded != nil && *waMsg.ExtendedTextMessage.ContextInfo.IsForwarded
	case waMsg.ImageMessage != nil && waMsg.ImageMessage.ContextInfo != nil:
		return waMsg.ImageMessage.ContextInfo.IsForwarded != nil && *waMsg.ImageMessage.ContextInfo.IsForwarded
	case waMsg.VideoMessage != nil && waMsg.VideoMessage.ContextInfo != nil:
		return waMsg.VideoMessage.ContextInfo.IsForwarded != nil && *waMsg.VideoMessage.ContextInfo.IsForwarded
	case waMsg.DocumentMessage != nil && waMsg.DocumentMessage.ContextInfo != nil:
		return waMsg.DocumentMessage.ContextInfo.IsForwarded != nil && *waMsg.DocumentMessage.ContextInfo.IsForwarded
	}
	return false
}

// convertReceiptTypeToStatus converte o tipo de confirmação para status da mensagem
func (s *PersistenceService) convertReceiptTypeToStatus(receiptType types.ReceiptType) string {
	switch receiptType {
	case types.ReceiptTypeDelivered:
		return string(models.MessageStatusDelivered)
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf:
		return string(models.MessageStatusRead)
	case types.ReceiptTypePlayed:
		return string(models.MessageStatusRead) // Áudio/vídeo reproduzido = lido
	default:
		return string(models.MessageStatusDelivered)
	}
}

// shouldUpdateStatus verifica se o status deve ser atualizado
func (s *PersistenceService) shouldUpdateStatus(currentStatus, newStatus string) bool {
	// Hierarquia de status: sent < delivered < read
	statusHierarchy := map[string]int{
		string(models.MessageStatusSent):      1,
		string(models.MessageStatusDelivered): 2,
		string(models.MessageStatusRead):      3,
	}

	currentLevel, currentExists := statusHierarchy[currentStatus]
	newLevel, newExists := statusHierarchy[newStatus]

	// Se algum status não está na hierarquia, não atualizar
	if !currentExists || !newExists {
		return false
	}

	// Atualizar apenas se o novo status for superior
	return newLevel > currentLevel
}
