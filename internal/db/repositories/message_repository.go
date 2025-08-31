package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/logger"
)

type MessageRepository interface {
	Create(message *models.Message) error
	GetByID(id uuid.UUID) (*models.Message, error)
	GetByMessageID(sessionID uuid.UUID, messageID string) (*models.Message, error)
	Update(message *models.Message) error
	Delete(id uuid.UUID) error

	List(filter *models.MessageFilter) ([]*models.Message, error)
	ListByChat(sessionID uuid.UUID, chatJID string, limit, offset int) ([]*models.Message, error)
	Search(sessionID uuid.UUID, query string, limit, offset int) ([]*models.Message, error)

	GetStatistics(sessionID uuid.UUID) (*models.MessageStatistics, error)
	GetChatStatistics(sessionID uuid.UUID, chatJID string) (*models.MessageStatistics, error)

	MarkAsRead(sessionID uuid.UUID, chatJID string, messageIDs []string) error
	GetUnreadCount(sessionID uuid.UUID, chatJID *string) (int64, error)
	CleanupEphemeralMessages() (int64, error)

	GetReplies(sessionID uuid.UUID, messageID string) ([]*models.Message, error)
	GetReactions(sessionID uuid.UUID, messageID string) ([]*models.Message, error)

	UpdateMinIOReferences(messageID uuid.UUID, minioMediaID, minioPath, minioURL, minioBucket string) error
	ClearMinIOReferences(minioPath string) error
	GetSessionMediaMessages(sessionID string, page, limit int, mediaType, direction string) ([]*models.Message, int, error)
	GetMediaStatistics(sessionID uuid.UUID) (map[string]interface{}, error)
}

type messageRepository struct {
	db     *sqlx.DB
	logger logger.Logger
}

func NewMessageRepository(db *sqlx.DB) MessageRepository {
	return &messageRepository{
		db:     db,
		logger: logger.GetWithSession("message_repository"),
	}
}

func (r *messageRepository) Create(message *models.Message) error {
	if message.ID == uuid.Nil {
		message.ID = uuid.New()
	}

	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()

	query := `
		INSERT INTO messages (
			id, session_id, message_id, whatsapp_message_id, chat_jid, sender_jid, recipient_jid,
			message_type, content, raw_message, media_url, media_type, media_size, media_filename,
			media_sha256, media_key, caption, quoted_message_id, quoted_content, reply_to_message_id,
			context_info, direction, status, is_from_me, is_ephemeral, is_view_once, is_forwarded,
			is_edit, edit_version, mentions, reaction_emoji, reaction_timestamp, location_latitude,
			location_longitude, location_name, location_address, contact_name, contact_phone,
			contact_vcard, sticker_pack_id, sticker_pack_name, group_invite_code, 
			group_invite_expiration, poll_name, poll_options, poll_selectable_count,
			error_message, retry_count, timestamp, created_at, updated_at
		) VALUES (
			:id, :session_id, :message_id, :whatsapp_message_id, :chat_jid, :sender_jid, :recipient_jid,
			:message_type, :content, :raw_message, :media_url, :media_type, :media_size, :media_filename,
			:media_sha256, :media_key, :caption, :quoted_message_id, :quoted_content, :reply_to_message_id,
			:context_info, :direction, :status, :is_from_me, :is_ephemeral, :is_view_once, :is_forwarded,
			:is_edit, :edit_version, :mentions, :reaction_emoji, :reaction_timestamp, :location_latitude,
			:location_longitude, :location_name, :location_address, :contact_name, :contact_phone,
			:contact_vcard, :sticker_pack_id, :sticker_pack_name, :group_invite_code,
			:group_invite_expiration, :poll_name, :poll_options, :poll_selectable_count,
			:error_message, :retry_count, :timestamp, :created_at, :updated_at
		)`

	_, err := r.db.NamedExec(query, message)
	if err != nil {
		r.logger.Error().Err(err).Str("message_id", message.MessageID).Msg("Failed to create message")
		return fmt.Errorf("failed to create message: %w", err)
	}

	r.logger.Debug().Str("message_id", message.MessageID).Str("chat_jid", message.ChatJID).Msg("Message created successfully")
	return nil
}

func (r *messageRepository) GetByID(id uuid.UUID) (*models.Message, error) {
	var message models.Message

	query := `SELECT * FROM messages WHERE id = $1`

	err := r.db.Get(&message, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message not found")
		}
		r.logger.Error().Err(err).Str("id", id.String()).Msg("Failed to get message by ID")
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &message, nil
}

func (r *messageRepository) GetByMessageID(sessionID uuid.UUID, messageID string) (*models.Message, error) {
	var message models.Message

	query := `SELECT * FROM messages WHERE session_id = $1 AND message_id = $2`

	err := r.db.Get(&message, query, sessionID, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message not found")
		}
		r.logger.Error().Err(err).Str("message_id", messageID).Msg("Failed to get message by message_id")
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &message, nil
}

func (r *messageRepository) Update(message *models.Message) error {
	message.UpdatedAt = time.Now()

	query := `
		UPDATE messages SET
			whatsapp_message_id = :whatsapp_message_id, chat_jid = :chat_jid, sender_jid = :sender_jid,
			recipient_jid = :recipient_jid, message_type = :message_type, content = :content,
			raw_message = :raw_message, media_url = :media_url, media_type = :media_type,
			media_size = :media_size, media_filename = :media_filename, media_sha256 = :media_sha256,
			media_key = :media_key, caption = :caption, quoted_message_id = :quoted_message_id,
			quoted_content = :quoted_content, reply_to_message_id = :reply_to_message_id,
			context_info = :context_info, direction = :direction, status = :status,
			is_from_me = :is_from_me, is_ephemeral = :is_ephemeral, is_view_once = :is_view_once,
			is_forwarded = :is_forwarded, is_edit = :is_edit, edit_version = :edit_version,
			mentions = :mentions, reaction_emoji = :reaction_emoji, reaction_timestamp = :reaction_timestamp,
			location_latitude = :location_latitude, location_longitude = :location_longitude,
			location_name = :location_name, location_address = :location_address,
			contact_name = :contact_name, contact_phone = :contact_phone, contact_vcard = :contact_vcard,
			sticker_pack_id = :sticker_pack_id, sticker_pack_name = :sticker_pack_name,
			group_invite_code = :group_invite_code, group_invite_expiration = :group_invite_expiration,
			poll_name = :poll_name, poll_options = :poll_options, poll_selectable_count = :poll_selectable_count,
			error_message = :error_message, retry_count = :retry_count, updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExec(query, message)
	if err != nil {
		r.logger.Error().Err(err).Str("message_id", message.MessageID).Msg("Failed to update message")
		return fmt.Errorf("failed to update message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("message not found")
	}

	r.logger.Debug().Str("message_id", message.MessageID).Msg("Message updated successfully")
	return nil
}

func (r *messageRepository) Delete(id uuid.UUID) error {
	query := `DELETE FROM messages WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("id", id.String()).Msg("Failed to delete message")
		return fmt.Errorf("failed to delete message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("message not found")
	}

	r.logger.Debug().Str("id", id.String()).Msg("Message deleted successfully")
	return nil
}

func (r *messageRepository) List(filter *models.MessageFilter) ([]*models.Message, error) {
	var messages []*models.Message
	var conditions []string
	var args []interface{}
	argIndex := 1

	query := `SELECT * FROM messages WHERE 1=1`

	if filter.SessionID != nil {
		conditions = append(conditions, fmt.Sprintf("session_id = $%d", argIndex))
		args = append(args, *filter.SessionID)
		argIndex++
	}

	if filter.ChatJID != nil {
		conditions = append(conditions, fmt.Sprintf("chat_jid = $%d", argIndex))
		args = append(args, *filter.ChatJID)
		argIndex++
	}

	if filter.SenderJID != nil {
		conditions = append(conditions, fmt.Sprintf("sender_jid = $%d", argIndex))
		args = append(args, *filter.SenderJID)
		argIndex++
	}

	if filter.MessageType != nil {
		conditions = append(conditions, fmt.Sprintf("message_type = $%d", argIndex))
		args = append(args, string(*filter.MessageType))
		argIndex++
	}

	if filter.Direction != nil {
		conditions = append(conditions, fmt.Sprintf("direction = $%d", argIndex))
		args = append(args, string(*filter.Direction))
		argIndex++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, string(*filter.Status))
		argIndex++
	}

	if filter.IsFromMe != nil {
		conditions = append(conditions, fmt.Sprintf("is_from_me = $%d", argIndex))
		args = append(args, *filter.IsFromMe)
		argIndex++
	}

	if filter.HasMedia != nil && *filter.HasMedia {
		conditions = append(conditions, "message_type IN ('image', 'audio', 'video', 'document', 'sticker')")
	}

	if filter.IsEphemeral != nil {
		conditions = append(conditions, fmt.Sprintf("is_ephemeral = $%d", argIndex))
		args = append(args, *filter.IsEphemeral)
		argIndex++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filter.DateFrom)
		argIndex++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filter.DateTo)
		argIndex++
	}

	if filter.SearchText != nil && *filter.SearchText != "" {
		conditions = append(conditions, fmt.Sprintf("(content ILIKE $%d OR caption ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+*filter.SearchText+"%")
		argIndex++
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	err := r.db.Select(&messages, query, args...)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to list messages")
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	return messages, nil
}

func (r *messageRepository) ListByChat(sessionID uuid.UUID, chatJID string, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message

	query := `
		SELECT * FROM messages
		WHERE session_id = $1 AND chat_jid = $2
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4`

	err := r.db.Select(&messages, query, sessionID, chatJID, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Str("chat_jid", chatJID).Msg("Failed to list messages by chat")
		return nil, fmt.Errorf("failed to list messages by chat: %w", err)
	}

	return messages, nil
}

func (r *messageRepository) Search(sessionID uuid.UUID, query string, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message

	searchQuery := `
		SELECT * FROM messages
		WHERE session_id = $1
		AND to_tsvector('portuguese', coalesce(content, '') || ' ' || coalesce(caption, '')) @@ plainto_tsquery('portuguese', $2)
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4`

	err := r.db.Select(&messages, searchQuery, sessionID, query, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Str("query", query).Msg("Failed to search messages")
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	return messages, nil
}

func (r *messageRepository) GetStatistics(sessionID uuid.UUID) (*models.MessageStatistics, error) {
	stats := &models.MessageStatistics{
		MessagesByType:   make(map[string]int64),
		MessagesByStatus: make(map[string]int64),
	}

	err := r.db.Get(&stats.TotalMessages,
		"SELECT COUNT(*) FROM messages WHERE session_id = $1", sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total messages: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT message_type, COUNT(*)
		FROM messages
		WHERE session_id = $1
		GROUP BY message_type`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var messageType string
		var count int64
		if err := rows.Scan(&messageType, &count); err != nil {
			return nil, err
		}
		stats.MessagesByType[messageType] = count
	}

	rows, err = r.db.Query(`
		SELECT status, COUNT(*)
		FROM messages
		WHERE session_id = $1
		GROUP BY status`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.MessagesByStatus[status] = count
	}

	err = r.db.Get(&stats.MediaMessages, `
		SELECT COUNT(*) FROM messages
		WHERE session_id = $1 AND message_type IN ('image', 'audio', 'video', 'document', 'sticker')`,
		sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media messages count: %w", err)
	}

	err = r.db.Get(&stats.IncomingMessages,
		"SELECT COUNT(*) FROM messages WHERE session_id = $1 AND direction = 'incoming'", sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming messages count: %w", err)
	}

	err = r.db.Get(&stats.OutgoingMessages,
		"SELECT COUNT(*) FROM messages WHERE session_id = $1 AND direction = 'outgoing'", sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing messages count: %w", err)
	}

	err = r.db.Get(&stats.UnreadMessages, `
		SELECT COUNT(*) FROM messages
		WHERE session_id = $1 AND direction = 'incoming' AND status != 'read'`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread messages count: %w", err)
	}

	err = r.db.Get(&stats.FailedMessages,
		"SELECT COUNT(*) FROM messages WHERE session_id = $1 AND status = 'failed'", sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed messages count: %w", err)
	}

	var lastMessageTime sql.NullTime
	err = r.db.Get(&lastMessageTime,
		"SELECT MAX(timestamp) FROM messages WHERE session_id = $1", sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last message time: %w", err)
	}
	if lastMessageTime.Valid {
		stats.LastMessageTime = &lastMessageTime.Time
	}

	return stats, nil
}

func (r *messageRepository) GetChatStatistics(sessionID uuid.UUID, chatJID string) (*models.MessageStatistics, error) {
	stats := &models.MessageStatistics{
		MessagesByType:   make(map[string]int64),
		MessagesByStatus: make(map[string]int64),
	}

	err := r.db.Get(&stats.TotalMessages,
		"SELECT COUNT(*) FROM messages WHERE session_id = $1 AND chat_jid = $2", sessionID, chatJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total messages: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT message_type, COUNT(*)
		FROM messages
		WHERE session_id = $1 AND chat_jid = $2
		GROUP BY message_type`, sessionID, chatJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var messageType string
		var count int64
		if err := rows.Scan(&messageType, &count); err != nil {
			return nil, err
		}
		stats.MessagesByType[messageType] = count
	}

	err = r.db.Get(&stats.UnreadMessages, `
		SELECT COUNT(*) FROM messages
		WHERE session_id = $1 AND chat_jid = $2 AND direction = 'incoming' AND status != 'read'`,
		sessionID, chatJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread messages count: %w", err)
	}

	return stats, nil
}

func (r *messageRepository) MarkAsRead(sessionID uuid.UUID, chatJID string, messageIDs []string) error {
	if len(messageIDs) == 0 {
		return nil
	}

	query := `
		UPDATE messages
		SET status = 'read', updated_at = NOW()
		WHERE session_id = $1 AND chat_jid = $2 AND message_id = ANY($3) AND direction = 'incoming'`

	result, err := r.db.Exec(query, sessionID, chatJID, messageIDs)
	if err != nil {
		r.logger.Error().Err(err).Str("chat_jid", chatJID).Msg("Failed to mark messages as read")
		return fmt.Errorf("failed to mark messages as read: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.Debug().Int64("rows_affected", rowsAffected).Str("chat_jid", chatJID).Msg("Messages marked as read")

	return nil
}

func (r *messageRepository) GetUnreadCount(sessionID uuid.UUID, chatJID *string) (int64, error) {
	var count int64
	var query string
	var args []interface{}

	if chatJID != nil {
		query = `
			SELECT COUNT(*) FROM messages
			WHERE session_id = $1 AND chat_jid = $2 AND direction = 'incoming' AND status != 'read'`
		args = []interface{}{sessionID, *chatJID}
	} else {
		query = `
			SELECT COUNT(*) FROM messages
			WHERE session_id = $1 AND direction = 'incoming' AND status != 'read'`
		args = []interface{}{sessionID}
	}

	err := r.db.Get(&count, query, args...)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get unread count")
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

func (r *messageRepository) CleanupEphemeralMessages() (int64, error) {
	query := `DELETE FROM messages WHERE is_ephemeral = TRUE AND timestamp < NOW() - INTERVAL '7 days'`

	result, err := r.db.Exec(query)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to cleanup ephemeral messages")
		return 0, fmt.Errorf("failed to cleanup ephemeral messages: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.Info().Int64("deleted_count", rowsAffected).Msg("Ephemeral messages cleaned up")
	return rowsAffected, nil
}

func (r *messageRepository) GetReplies(sessionID uuid.UUID, messageID string) ([]*models.Message, error) {
	var messages []*models.Message

	query := `
		SELECT * FROM messages
		WHERE session_id = $1 AND reply_to_message_id = $2
		ORDER BY timestamp ASC`

	err := r.db.Select(&messages, query, sessionID, messageID)
	if err != nil {
		r.logger.Error().Err(err).Str("message_id", messageID).Msg("Failed to get replies")
		return nil, fmt.Errorf("failed to get replies: %w", err)
	}

	return messages, nil
}

func (r *messageRepository) GetReactions(sessionID uuid.UUID, messageID string) ([]*models.Message, error) {
	var messages []*models.Message

	query := `
		SELECT * FROM messages
		WHERE session_id = $1 AND reply_to_message_id = $2 AND message_type = 'reaction'
		ORDER BY reaction_timestamp ASC`

	err := r.db.Select(&messages, query, sessionID, messageID)
	if err != nil {
		r.logger.Error().Err(err).Str("message_id", messageID).Msg("Failed to get reactions")
		return nil, fmt.Errorf("failed to get reactions: %w", err)
	}

	return messages, nil
}

func (r *messageRepository) UpdateMinIOReferences(messageID uuid.UUID, minioMediaID, minioPath, minioURL, minioBucket string) error {
	query := `
		UPDATE messages
		SET minio_media_id = $2,
		    minio_path = $3,
		    minio_url = $4,
		    minio_bucket = $5,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(query, messageID, minioMediaID, minioPath, minioURL, minioBucket)
	if err != nil {
		r.logger.Error().Err(err).Str("message_id", messageID.String()).Msg("Failed to update MinIO references")
		return fmt.Errorf("failed to update MinIO references: %w", err)
	}

	r.logger.Debug().
		Str("message_id", messageID.String()).
		Str("minio_media_id", minioMediaID).
		Str("minio_path", minioPath).
		Msg("MinIO references updated successfully")

	return nil
}

func (r *messageRepository) ClearMinIOReferences(minioPath string) error {
	query := `
		UPDATE messages
		SET minio_media_id = NULL,
		    minio_path = NULL,
		    minio_url = NULL,
		    updated_at = NOW()
		WHERE minio_path = $1
	`

	result, err := r.db.Exec(query, minioPath)
	if err != nil {
		r.logger.Error().Err(err).Str("minio_path", minioPath).Msg("Failed to clear MinIO references")
		return fmt.Errorf("failed to clear MinIO references: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.Debug().
		Str("minio_path", minioPath).
		Int64("rows_affected", rowsAffected).
		Msg("MinIO references cleared")

	return nil
}

func (r *messageRepository) GetSessionMediaMessages(sessionID string, page, limit int, mediaType, direction string) ([]*models.Message, int, error) {
	offset := (page - 1) * limit

	baseQuery := `
		FROM messages
		WHERE session_id = $1
		AND minio_path IS NOT NULL
		AND message_type IN ('image', 'video', 'audio', 'document', 'sticker')
	`

	args := []interface{}{sessionID}
	argIndex := 2

	if mediaType != "" {
		baseQuery += fmt.Sprintf(" AND message_type = $%d", argIndex)
		args = append(args, mediaType)
		argIndex++
	}

	if direction != "" {
		baseQuery += fmt.Sprintf(" AND direction = $%d", argIndex)
		args = append(args, direction)
		argIndex++
	}

	countQuery := "SELECT COUNT(*) " + baseQuery

	var total int
	err := r.db.Get(&total, countQuery, args...)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to count session media messages")
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	selectQuery := `
		SELECT id, session_id, message_id, whatsapp_message_id, chat_jid, sender_jid, recipient_jid,
		       message_type, content, raw_message, media_url, media_type, media_size, media_filename,
		       media_sha256, media_key, minio_media_id, minio_path, minio_url, minio_bucket,
		       caption, direction, status, is_from_me, is_ephemeral, is_view_once, is_forwarded,
		       is_edit, edit_version, mentions, reply_to_message_id, reaction_emoji, reaction_timestamp,
		       location_latitude, location_longitude, location_name, location_address,
		       contact_name, contact_phone, contact_vcard, sticker_pack_id, sticker_pack_name,
		       group_invite_code, group_invite_expiration, poll_name, poll_options, poll_selectable_count,
		       error_message, retry_count, timestamp, created_at, updated_at
	` + baseQuery + fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

	args = append(args, limit, offset)

	var messages []*models.Message
	err = r.db.Select(&messages, selectQuery, args...)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get session media messages")
		return nil, 0, fmt.Errorf("failed to get messages: %w", err)
	}

	return messages, total, nil
}

func (r *messageRepository) GetMediaStatistics(sessionID uuid.UUID) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_media_count,
			COALESCE(SUM(media_size), 0) as total_media_size,
			COUNT(CASE WHEN message_type = 'image' THEN 1 END) as image_count,
			COUNT(CASE WHEN message_type = 'video' THEN 1 END) as video_count,
			COUNT(CASE WHEN message_type = 'audio' THEN 1 END) as audio_count,
			COUNT(CASE WHEN message_type = 'document' THEN 1 END) as document_count,
			COUNT(CASE WHEN message_type = 'sticker' THEN 1 END) as sticker_count,
			COUNT(CASE WHEN direction = 'incoming' THEN 1 END) as incoming_count,
			COUNT(CASE WHEN direction = 'outgoing' THEN 1 END) as outgoing_count,
			COALESCE(SUM(CASE WHEN message_type = 'image' THEN media_size END), 0) as image_size,
			COALESCE(SUM(CASE WHEN message_type = 'video' THEN media_size END), 0) as video_size,
			COALESCE(SUM(CASE WHEN message_type = 'audio' THEN media_size END), 0) as audio_size,
			COALESCE(SUM(CASE WHEN message_type = 'document' THEN media_size END), 0) as document_size
		FROM messages
		WHERE session_id = $1
		AND minio_path IS NOT NULL
		AND message_type IN ('image', 'video', 'audio', 'document', 'sticker')
	`

	var stats struct {
		TotalMediaCount int64 `db:"total_media_count"`
		TotalMediaSize  int64 `db:"total_media_size"`
		ImageCount      int64 `db:"image_count"`
		VideoCount      int64 `db:"video_count"`
		AudioCount      int64 `db:"audio_count"`
		DocumentCount   int64 `db:"document_count"`
		StickerCount    int64 `db:"sticker_count"`
		IncomingCount   int64 `db:"incoming_count"`
		OutgoingCount   int64 `db:"outgoing_count"`
		ImageSize       int64 `db:"image_size"`
		VideoSize       int64 `db:"video_size"`
		AudioSize       int64 `db:"audio_size"`
		DocumentSize    int64 `db:"document_size"`
	}

	err := r.db.Get(&stats, query, sessionID)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Failed to get media statistics")
		return nil, fmt.Errorf("failed to get media statistics: %w", err)
	}

	result := map[string]interface{}{
		"total_count": stats.TotalMediaCount,
		"total_size":  stats.TotalMediaSize,
		"by_type": map[string]interface{}{
			"image": map[string]interface{}{
				"count": stats.ImageCount,
				"size":  stats.ImageSize,
			},
			"video": map[string]interface{}{
				"count": stats.VideoCount,
				"size":  stats.VideoSize,
			},
			"audio": map[string]interface{}{
				"count": stats.AudioCount,
				"size":  stats.AudioSize,
			},
			"document": map[string]interface{}{
				"count": stats.DocumentCount,
				"size":  stats.DocumentSize,
			},
			"sticker": map[string]interface{}{
				"count": stats.StickerCount,
				"size":  int64(0),
			},
		},
		"by_direction": map[string]interface{}{
			"incoming": stats.IncomingCount,
			"outgoing": stats.OutgoingCount,
		},
	}

	return result, nil
}
