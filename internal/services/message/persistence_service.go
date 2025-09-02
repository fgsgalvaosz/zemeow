package message

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/models"
	"github.com/felipe/zemeow/internal/repositories"
	"github.com/felipe/zemeow/internal/services/media"
)

type ClientProvider interface {
	GetClient(sessionID string) *whatsmeow.Client
}

type PersistenceService struct {
	messageRepo    repositories.MessageRepository
	mediaService   *media.MediaService
	clientProvider ClientProvider
	logger         logger.Logger
}

func NewPersistenceService(messageRepo repositories.MessageRepository, mediaService *media.MediaService, clientProvider ClientProvider) *PersistenceService {
	return &PersistenceService{
		messageRepo:    messageRepo,
		mediaService:   mediaService,
		clientProvider: clientProvider,
		logger:         logger.GetWithSession("message_persistence"),
	}
}

func (s *PersistenceService) ProcessMessageEvent(sessionID uuid.UUID, evt *events.Message) error {
	message, err := s.convertEventToMessage(sessionID, evt)
	if err != nil {
		s.logger.Error().Err(err).Str("message_id", evt.Info.ID).Msg("Failed to convert event to message")
		return fmt.Errorf("failed to convert event to message: %w", err)
	}

	existing, err := s.messageRepo.GetByMessageID(sessionID, evt.Info.ID)
	if err == nil && existing != nil {

		s.logger.Debug().Str("message_id", evt.Info.ID).Msg("Message already exists, updating")
		message.ID = existing.ID
		message.CreatedAt = existing.CreatedAt
		return s.messageRepo.Update(message)
	}

	err = s.messageRepo.Create(message)
	if err != nil {
		s.logger.Error().Err(err).Str("message_id", evt.Info.ID).Msg("Failed to persist message")
		return fmt.Errorf("failed to persist message: %w", err)
	}

	if s.hasMediaContent(message) {
		go s.processMediaContent(sessionID, evt, message)
	}

	s.logger.Debug().
		Str("message_id", evt.Info.ID).
		Str("chat_jid", evt.Info.Chat.String()).
		Str("sender_jid", evt.Info.Sender.String()).
		Str("type", message.MessageType).
		Msg("Message persisted successfully")

	return nil
}

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

func (s *PersistenceService) convertEventToMessage(sessionID uuid.UUID, evt *events.Message) (*models.Message, error) {
	message := &models.Message{
		SessionID:   sessionID,
		MessageID:   evt.Info.ID,
		ChatJID:     evt.Info.Chat.String(),
		SenderJID:   evt.Info.Sender.String(),
		Direction:   s.getMessageDirection(evt.Info.IsFromMe),
		Status:      string(models.MessageStatusReceived),
		IsFromMe:    evt.Info.IsFromMe,
		IsEphemeral: evt.IsEphemeral,
		IsViewOnce:  evt.IsViewOnce,
		IsForwarded: s.isForwardedMessage(evt.Message),
		Timestamp:   evt.Info.Timestamp,
		ContextInfo: make(models.JSONB),
	}

	rawMessage, err := json.Marshal(evt.Message)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal raw message")
	} else {
		message.RawMessage = rawMessage
	}

	err = s.processMessageContent(message, evt.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to process message content: %w", err)
	}

	s.processContextInfo(message, evt.Message)

	if evt.Info.Chat.Server != types.GroupServer && evt.Info.Chat.Server != types.BroadcastServer {
		if evt.Info.IsFromMe {
			message.RecipientJID = &message.ChatJID
		} else {
			ourJID := evt.Info.Chat.String()
			message.RecipientJID = &ourJID
		}
	}

	return message, nil
}

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

type MediaInfo struct {
	URL      *string
	MimeType *string
	Size     *uint64
	SHA256   []byte
	Caption  *string
	Filename *string
}

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

func (s *PersistenceService) processContactMessage(message *models.Message, contact *waE2E.ContactMessage) {
	if contact.DisplayName != nil {
		message.ContactName = contact.DisplayName
	}
	if contact.Vcard != nil {
		message.ContactVCard = contact.Vcard

		if phone := s.extractPhoneFromVCard(*contact.Vcard); phone != "" {
			message.ContactPhone = &phone
		}
	}
}

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

func (s *PersistenceService) processPollMessage(message *models.Message, poll *waE2E.PollCreationMessage) {
	if poll.Name != nil {
		message.PollName = poll.Name
		message.Content = poll.Name
	}
	if poll.SelectableOptionsCount != nil {
		count := int(*poll.SelectableOptionsCount)
		message.PollSelectableCount = &count
	}

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

func (s *PersistenceService) processReactionMessage(message *models.Message, reaction *waE2E.ReactionMessage) {
	if reaction.Text != nil {
		message.ReactionEmoji = reaction.Text
	}
	if reaction.Key != nil && reaction.Key.ID != nil {
		message.ReplyToMessageID = reaction.Key.ID
	}

	reactionTime := time.Now()
	message.ReactionTimestamp = &reactionTime
}

func (s *PersistenceService) processContextInfo(message *models.Message, waMsg *waE2E.Message) {
	var contextInfo *waE2E.ContextInfo

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

func (s *PersistenceService) processExtendedContextInfo(message *models.Message, contextInfo *waE2E.ContextInfo) {

	if contextInfo.StanzaID != nil {
		message.ReplyToMessageID = contextInfo.StanzaID
		if contextInfo.QuotedMessage != nil {

			quotedContent := s.extractQuotedContent(contextInfo.QuotedMessage)
			if quotedContent != "" {
				message.QuotedContent = &quotedContent
			}
		}
	}

	if len(contextInfo.MentionedJID) > 0 {
		mentions := make([]string, len(contextInfo.MentionedJID))
		copy(mentions, contextInfo.MentionedJID)
		message.Mentions = mentions
	}

	if contextInfo.IsForwarded != nil && *contextInfo.IsForwarded {
		message.IsForwarded = true
	}
}

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

func (s *PersistenceService) getMessageDirection(isFromMe bool) string {
	if isFromMe {
		return string(models.MessageDirectionOutgoing)
	}
	return string(models.MessageDirectionIncoming)
}

func (s *PersistenceService) isForwardedMessage(waMsg *waE2E.Message) bool {

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

func (s *PersistenceService) convertReceiptTypeToStatus(receiptType types.ReceiptType) string {
	switch receiptType {
	case types.ReceiptTypeDelivered:
		return string(models.MessageStatusDelivered)
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf:
		return string(models.MessageStatusRead)
	case types.ReceiptTypePlayed:
		return string(models.MessageStatusRead)
	default:
		return string(models.MessageStatusDelivered)
	}
}

func (s *PersistenceService) shouldUpdateStatus(currentStatus, newStatus string) bool {

	statusHierarchy := map[string]int{
		string(models.MessageStatusSent):      1,
		string(models.MessageStatusDelivered): 2,
		string(models.MessageStatusRead):      3,
	}

	currentLevel, currentExists := statusHierarchy[currentStatus]
	newLevel, newExists := statusHierarchy[newStatus]

	if !currentExists || !newExists {
		return false
	}

	return newLevel > currentLevel
}

func (s *PersistenceService) hasMediaContent(message *models.Message) bool {
	return message.MessageType == "image" ||
		message.MessageType == "video" ||
		message.MessageType == "audio" ||
		message.MessageType == "document" ||
		message.MessageType == "sticker"
}

func (s *PersistenceService) processMediaContent(sessionID uuid.UUID, evt *events.Message, message *models.Message) {
	if s.mediaService == nil {
		s.logger.Warn().Str("message_id", evt.Info.ID).Msg("MediaService not available, skipping media processing")
		return
	}

	client := s.clientProvider.GetClient(sessionID.String())
	if client == nil {
		s.logger.Error().Str("message_id", evt.Info.ID).Msg("WhatsApp client not available for media download")
		return
	}

	mediaData, err := s.downloadMediaFromMessage(client, evt.Message)
	if err != nil {
		s.logger.Error().Err(err).
			Str("message_id", evt.Info.ID).
			Str("message_type", message.MessageType).
			Msg("Failed to download media from WhatsApp")
		return
	}

	if mediaData == nil || len(mediaData.Data) == 0 {
		s.logger.Warn().Str("message_id", evt.Info.ID).Msg("No media data to process")
		return
	}

	direction := "incoming"
	if evt.Info.IsFromMe {
		direction = "outgoing"
	}

	mediaPath := fmt.Sprintf("media/%s/%s/%s", direction, sessionID.String(), evt.Info.ID)
	if mediaData.FileName != "" {
		mediaPath = fmt.Sprintf("%s/%s", mediaPath, mediaData.FileName)
	}

	err = s.mediaService.UploadMedia(
		context.Background(),
		mediaPath,
		bytes.NewReader(mediaData.Data),
		int64(len(mediaData.Data)),
		mediaData.MimeType,
	)
	if err != nil {
		s.logger.Error().Err(err).
			Str("message_id", evt.Info.ID).
			Str("session_id", sessionID.String()).
			Msg("Failed to upload media to MinIO")
		return
	}

	mediaURL, err := s.mediaService.GetMediaURL(context.Background(), mediaPath)
	if err != nil {
		s.logger.Error().Err(err).
			Str("message_id", evt.Info.ID).
			Str("session_id", sessionID.String()).
			Msg("Failed to get media URL from MinIO")
		return
	}

	err = s.messageRepo.UpdateMinIOReferences(
		message.ID,
		"",
		mediaPath,
		mediaURL,
		"zemeow-media",
	)
	if err != nil {
		s.logger.Error().Err(err).
			Str("message_id", evt.Info.ID).
			Str("minio_path", mediaPath).
			Msg("Failed to update MinIO references in database")
		return
	}

	s.logger.Info().
		Str("message_id", evt.Info.ID).
		Str("session_id", sessionID.String()).
		Str("minio_path", mediaPath).
		Int64("size", int64(len(mediaData.Data))).
		Str("direction", direction).
		Msg("Media processed and saved to MinIO successfully")
}

type MediaData struct {
	Data     []byte
	MimeType string
	FileName string
}

func (s *PersistenceService) downloadMediaFromMessage(client *whatsmeow.Client, waMsg *waE2E.Message) (*MediaData, error) {
	var mediaData *MediaData
	var err error

	switch {
	case waMsg.ImageMessage != nil:
		mediaData, err = s.downloadImageMessage(client, waMsg.ImageMessage)
	case waMsg.VideoMessage != nil:
		mediaData, err = s.downloadVideoMessage(client, waMsg.VideoMessage)
	case waMsg.AudioMessage != nil:
		mediaData, err = s.downloadAudioMessage(client, waMsg.AudioMessage)
	case waMsg.DocumentMessage != nil:
		mediaData, err = s.downloadDocumentMessage(client, waMsg.DocumentMessage)
	case waMsg.StickerMessage != nil:
		mediaData, err = s.downloadStickerMessage(client, waMsg.StickerMessage)
	default:
		return nil, fmt.Errorf("unsupported media message type")
	}

	return mediaData, err
}

func (s *PersistenceService) downloadImageMessage(client *whatsmeow.Client, img *waE2E.ImageMessage) (*MediaData, error) {
	data, err := client.Download(context.Background(), img)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	fileName := "image"
	if img.Caption != nil && *img.Caption != "" {
		fileName = *img.Caption
	}
	if !strings.Contains(fileName, ".") {
		fileName += ".jpg"
	}

	mimeType := "image/jpeg"
	if img.Mimetype != nil {
		mimeType = *img.Mimetype
	}

	return &MediaData{
		Data:     data,
		MimeType: mimeType,
		FileName: fileName,
	}, nil
}

func (s *PersistenceService) downloadVideoMessage(client *whatsmeow.Client, video *waE2E.VideoMessage) (*MediaData, error) {
	data, err := client.Download(context.Background(), video)
	if err != nil {
		return nil, fmt.Errorf("failed to download video: %w", err)
	}

	fileName := "video"
	if video.Caption != nil && *video.Caption != "" {
		fileName = *video.Caption
	}
	if !strings.Contains(fileName, ".") {
		fileName += ".mp4"
	}

	mimeType := "video/mp4"
	if video.Mimetype != nil {
		mimeType = *video.Mimetype
	}

	return &MediaData{
		Data:     data,
		MimeType: mimeType,
		FileName: fileName,
	}, nil
}

func (s *PersistenceService) downloadAudioMessage(client *whatsmeow.Client, audio *waE2E.AudioMessage) (*MediaData, error) {
	data, err := client.Download(context.Background(), audio)
	if err != nil {
		return nil, fmt.Errorf("failed to download audio: %w", err)
	}

	fileName := "audio"
	if !strings.Contains(fileName, ".") {
		fileName += ".ogg"
	}

	mimeType := "audio/ogg"
	if audio.Mimetype != nil {
		mimeType = *audio.Mimetype
	}

	return &MediaData{
		Data:     data,
		MimeType: mimeType,
		FileName: fileName,
	}, nil
}

func (s *PersistenceService) downloadDocumentMessage(client *whatsmeow.Client, doc *waE2E.DocumentMessage) (*MediaData, error) {
	data, err := client.Download(context.Background(), doc)
	if err != nil {
		return nil, fmt.Errorf("failed to download document: %w", err)
	}

	fileName := "document"
	if doc.FileName != nil {
		fileName = *doc.FileName
	}

	mimeType := "application/octet-stream"
	if doc.Mimetype != nil {
		mimeType = *doc.Mimetype
	}

	return &MediaData{
		Data:     data,
		MimeType: mimeType,
		FileName: fileName,
	}, nil
}

func (s *PersistenceService) downloadStickerMessage(client *whatsmeow.Client, sticker *waE2E.StickerMessage) (*MediaData, error) {
	data, err := client.Download(context.Background(), sticker)
	if err != nil {
		return nil, fmt.Errorf("failed to download sticker: %w", err)
	}

	fileName := "sticker.webp"
	mimeType := "image/webp"
	if sticker.Mimetype != nil {
		mimeType = *sticker.Mimetype
	}

	return &MediaData{
		Data:     data,
		MimeType: mimeType,
		FileName: fileName,
	}, nil
}
