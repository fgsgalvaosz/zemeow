package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/google/uuid"
)


type SessionRepository interface {
	Create(session *models.Session) error
	GetByID(id uuid.UUID) (*models.Session, error)
	GetBySessionID(sessionID string) (*models.Session, error)             // Busca por UUID string
	GetByName(name string) (*models.Session, error) // Busca por nome
	GetByIdentifier(identifier string) (*models.Session, error) // Busca por UUID ou nome (dual mode)
	GetByAPIKey(apiKey string) (*models.Session, error)
	GetAll(filter *models.SessionFilter) (*models.SessionListResponse, error)
	Update(session *models.Session) error
	UpdateStatus(identifier string, status models.SessionStatus) error
	UpdateStatusAndJID(identifier string, status models.SessionStatus, jid *string) error
	UpdateJID(identifier string, jid *string) error
	UpdateQRCode(identifier string, qrCode string) error
	ClearQRCode(identifier string) error
	Delete(id uuid.UUID) error
	DeleteByIdentifier(identifier string) error
	DeleteBySessionID(sessionID string) error // Remove por UUID string
	Exists(identifier string) (bool, error)
	Count() (int, error)
	GetActiveConnections() ([]*models.Session, error)
	Close() error
}


type sessionRepository struct {
	db     *sql.DB
	logger logger.Logger
}


func NewSessionRepository(db *sql.DB) SessionRepository {
	return &sessionRepository{
		db:     db,
		logger: logger.Get(),
	}
}


func (r *sessionRepository) Create(session *models.Session) error {
	if err := session.Validate(); err != nil {
		return fmt.Errorf("invalid session data: %w", err)
	}


	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}


	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	query := `
		INSERT INTO sessions (
			id, name, api_key, jid, status,
			proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
			webhook_url, webhook_events, created_at, updated_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
	`

	_, err := r.db.Exec(query,
		session.ID, session.Name, session.APIKey, session.JID, string(session.Status),
		session.ProxyEnabled, session.ProxyHost, session.ProxyPort, session.ProxyUsername, session.ProxyPassword,
		session.WebhookURL, session.WebhookEvents, session.CreatedAt, session.UpdatedAt, session.Metadata,
	)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", session.ID.String()).Msg("Failed to create session")
		return fmt.Errorf("failed to create session: %w", err)
	}

	r.logger.Info().Str("session_id", session.ID.String()).Msg("Session created successfully")
	return nil
}


func (r *sessionRepository) GetByID(id uuid.UUID) (*models.Session, error) {
	query := `
		SELECT id, name, api_key, jid, status,
		       proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
		       webhook_url, webhook_events, created_at, updated_at, last_connected_at, metadata, qrcode
		FROM sessions WHERE id = $1
	`

	session := &models.Session{}
	var qrcode sql.NullString
	err := r.db.QueryRow(query, id).Scan(
		&session.ID, &session.Name, &session.APIKey, &session.JID, &session.Status,
		&session.ProxyEnabled, &session.ProxyHost, &session.ProxyPort, &session.ProxyUsername, &session.ProxyPassword,
		&session.WebhookURL, &session.WebhookEvents, &session.CreatedAt, &session.UpdatedAt, &session.LastConnectedAt, &session.Metadata, &qrcode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		r.logger.Error().Err(err).Str("id", id.String()).Msg("Failed to get session by ID")
		return nil, fmt.Errorf("failed to get session: %w", err)
	}


	if qrcode.Valid {
		session.QRCode = &qrcode.String
	}

	return session, nil
}


func (r *sessionRepository) GetBySessionID(sessionID string) (*models.Session, error) {

	id, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID format: %w", err)
	}


	return r.GetByID(id)
}


func (r *sessionRepository) GetByName(name string) (*models.Session, error) {
	query := `
		SELECT id, name, api_key, jid, status,
		       proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
		       webhook_url, webhook_events, created_at, updated_at, last_connected_at, metadata
		FROM sessions WHERE name = $1
	`

	session := &models.Session{}
	err := r.db.QueryRow(query, name).Scan(
		&session.ID, &session.Name, &session.APIKey, &session.JID, &session.Status,
		&session.ProxyEnabled, &session.ProxyHost, &session.ProxyPort, &session.ProxyUsername, &session.ProxyPassword,
		&session.WebhookURL, &session.WebhookEvents, &session.CreatedAt, &session.UpdatedAt, &session.LastConnectedAt, &session.Metadata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		r.logger.Error().Err(err).Str("name", name).Msg("Failed to get session by name")
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}


func (r *sessionRepository) GetByIdentifier(identifier string) (*models.Session, error) {

	if id, err := uuid.Parse(identifier); err == nil {
		return r.GetByID(id)
	}


	return r.GetByName(identifier)
}


func (r *sessionRepository) GetByAPIKey(apiKey string) (*models.Session, error) {
	query := `
		SELECT id, name, api_key, jid, status,
		       proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
		       webhook_url, webhook_events, created_at, updated_at, last_connected_at, metadata
		FROM sessions WHERE api_key = $1
	`

	session := &models.Session{}
	err := r.db.QueryRow(query, apiKey).Scan(
		&session.ID, &session.Name, &session.APIKey, &session.JID, &session.Status,
		&session.ProxyEnabled, &session.ProxyHost, &session.ProxyPort, &session.ProxyUsername, &session.ProxyPassword,
		&session.WebhookURL, &session.WebhookEvents, &session.CreatedAt, &session.UpdatedAt, &session.LastConnectedAt, &session.Metadata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		r.logger.Error().Err(err).Msg("Failed to get session by API key")
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}


func (r *sessionRepository) GetAll(filter *models.SessionFilter) (*models.SessionListResponse, error) {
	if filter == nil {
		filter = &models.SessionFilter{}
	}


	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "created_at"
	}
	if filter.OrderDir == "" {
		filter.OrderDir = "DESC"
	}


	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.Name != nil {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIndex))
		args = append(args, "%"+*filter.Name+"%")
		argIndex++
	}

	if filter.JID != nil {
		conditions = append(conditions, fmt.Sprintf("jid = $%d", argIndex))
		args = append(args, *filter.JID)
		argIndex++
	}

	if filter.CreatedAt != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.CreatedAt)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}


	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sessions %s", whereClause)
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to count sessions")
		return nil, fmt.Errorf("failed to count sessions: %w", err)
	}


	offset := (filter.Page - 1) * filter.PerPage
	query := fmt.Sprintf(`
		SELECT id, name, api_key, jid, status,
		       proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
		       webhook_url, webhook_events, created_at, updated_at, last_connected_at, metadata
		FROM sessions %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, filter.OrderBy, filter.OrderDir, argIndex, argIndex+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to query sessions")
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		session := models.Session{}
		err := rows.Scan(
			&session.ID, &session.Name, &session.APIKey, &session.JID, &session.Status,
			&session.ProxyEnabled, &session.ProxyHost, &session.ProxyPort, &session.ProxyUsername, &session.ProxyPassword,
			&session.WebhookURL, &session.WebhookEvents, &session.CreatedAt, &session.UpdatedAt, &session.LastConnectedAt, &session.Metadata,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan session")
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating sessions")
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	totalPages := (total + filter.PerPage - 1) / filter.PerPage

	return &models.SessionListResponse{
		Sessions:   sessions,
		Total:      total,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		TotalPages: totalPages,
	}, nil
}


func (r *sessionRepository) Update(session *models.Session) error {
	if err := session.Validate(); err != nil {
		return fmt.Errorf("invalid session data: %w", err)
	}

	session.UpdatedAt = time.Now()

	query := `
		UPDATE sessions SET
			name = $2, jid = $3, status = $4,
			proxy_enabled = $5, proxy_host = $6, proxy_port = $7, proxy_username = $8, proxy_password = $9,
			webhook_url = $10, webhook_events = $11, updated_at = $12, last_connected_at = $13, metadata = $14
		WHERE id = $1
	`

	result, err := r.db.Exec(query,
		session.ID, session.Name, session.JID, string(session.Status),
		session.ProxyEnabled, session.ProxyHost, session.ProxyPort, session.ProxyUsername, session.ProxyPassword,
		session.WebhookURL, session.WebhookEvents, session.UpdatedAt, session.LastConnectedAt, session.Metadata,
	)

	if err != nil {
		r.logger.Error().Err(err).Str("session_name", session.Name).Msg("Failed to update session")
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	r.logger.Info().Str("session_name", session.Name).Msg("Session updated successfully")
	return nil
}


func (r *sessionRepository) UpdateStatus(sessionID string, status models.SessionStatus) error {
	now := time.Now()
	var lastConnectedAt *time.Time

	if status == models.SessionStatusConnected || status == models.SessionStatusAuthenticated {
		lastConnectedAt = &now
	}


	var query string
	if _, err := uuid.Parse(sessionID); err == nil {
		query = `
			UPDATE sessions SET
				status = $2, updated_at = $3, last_connected_at = COALESCE($4, last_connected_at)
			WHERE id = $1
		`
	} else {
		query = `
			UPDATE sessions SET
				status = $2, updated_at = $3, last_connected_at = COALESCE($4, last_connected_at)
			WHERE name = $1
		`
	}

	result, err := r.db.Exec(query, sessionID, string(status), now, lastConnectedAt)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID).Str("status", string(status)).Msg("Failed to update session status")
		return fmt.Errorf("failed to update session status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	r.logger.Info().Str("session_id", sessionID).Str("status", string(status)).Msg("Session status updated successfully")
	return nil
}


func (r *sessionRepository) UpdateStatusAndJID(identifier string, status models.SessionStatus, jid *string) error {
	var query string
	var args []interface{}

	statusStr := string(status)
	now := time.Now()
	var lastConnectedAt *time.Time

	// Set last_connected_at only for connected/authenticated status
	if status == models.SessionStatusConnected || status == models.SessionStatusAuthenticated {
		lastConnectedAt = &now
	}

	if id, err := uuid.Parse(identifier); err == nil {
		query = `
			UPDATE sessions
			SET status = $1, jid = $2, updated_at = $3, last_connected_at = COALESCE($4, last_connected_at)
			WHERE id = $5
		`
		args = []interface{}{statusStr, jid, now, lastConnectedAt, id}
	} else {
		query = `
			UPDATE sessions
			SET status = $1, jid = $2, updated_at = $3, last_connected_at = COALESCE($4, last_connected_at)
			WHERE name = $5
		`
		args = []interface{}{statusStr, jid, now, lastConnectedAt, identifier}
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		r.logger.Error().Err(err).Str("session_identifier", identifier).Str("status", string(status)).Msg("Failed to update session status and JID")
		return fmt.Errorf("failed to update session status and JID: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	jidStr := "nil"
	if jid != nil {
		jidStr = *jid
	}
	r.logger.Info().Str("session_identifier", identifier).Str("status", string(status)).Str("jid", jidStr).Msg("Session status and JID updated successfully")
	return nil
}


func (r *sessionRepository) UpdateJID(identifier string, jid *string) error {
	var query string
	var args []interface{}


	if id, err := uuid.Parse(identifier); err == nil {
		query = `
			UPDATE sessions
			SET jid = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`
		args = []interface{}{jid, id}
	} else {

		query = `
			UPDATE sessions
			SET jid = $1, updated_at = CURRENT_TIMESTAMP
			WHERE name = $2
		`
		args = []interface{}{jid, identifier}
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		r.logger.Error().Err(err).Str("session_identifier", identifier).Msg("Failed to update session JID")
		return fmt.Errorf("failed to update session JID: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	jidStr := "nil"
	if jid != nil {
		jidStr = *jid
	}
	r.logger.Info().Str("session_identifier", identifier).Str("jid", jidStr).Msg("Session JID updated successfully")
	return nil
}


func (r *sessionRepository) UpdateQRCode(identifier string, qrCode string) error {
	var query string
	var args []interface{}


	if id, err := uuid.Parse(identifier); err == nil {
		query = `
			UPDATE sessions
			SET qrcode = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`
		args = []interface{}{qrCode, id}
	} else {

		query = `
			UPDATE sessions
			SET qrcode = $1, updated_at = CURRENT_TIMESTAMP
			WHERE name = $2
		`
		args = []interface{}{qrCode, identifier}
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		r.logger.Error().Err(err).Str("session_identifier", identifier).Msg("Failed to update session QR code")
		return fmt.Errorf("failed to update session QR code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	r.logger.Info().Str("session_identifier", identifier).Msg("Session QR code updated successfully")
	return nil
}

func (r *sessionRepository) ClearQRCode(identifier string) error {
	var query string
	var args []interface{}


	if id, err := uuid.Parse(identifier); err == nil {
		query = `
			UPDATE sessions
			SET qrcode = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		args = []interface{}{id}
	} else {

		query = `
			UPDATE sessions
			SET qrcode = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE name = $1
		`
		args = []interface{}{identifier}
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		r.logger.Error().Err(err).Str("session_identifier", identifier).Msg("Failed to clear session QR code")
		return fmt.Errorf("failed to clear session QR code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	r.logger.Info().Str("session_identifier", identifier).Msg("Session QR code cleared successfully")
	return nil
}


func (r *sessionRepository) Delete(id uuid.UUID) error {
	// Primeiro, obter o JID da sessão para limpar dados do WhatsApp
	var jid sql.NullString
	jidQuery := "SELECT jid FROM sessions WHERE id = $1"
	err := r.db.QueryRow(jidQuery, id).Scan(&jid)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error().Err(err).Str("id", id.String()).Msg("Failed to get session JID for cleanup")
		return fmt.Errorf("failed to get session JID: %w", err)
	}

	// Iniciar transação para garantir consistência
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Se há JID, limpar dados do WhatsApp primeiro
	if jid.Valid && jid.String != "" {
		r.logger.Info().Str("id", id.String()).Str("jid", jid.String).Msg("Cleaning up WhatsApp data for session")

		// Deletar apenas whatsmeow_device - as outras tabelas serão limpas automaticamente
		// devido às foreign keys CASCADE que já existem no whatsmeow
		deleteQuery := "DELETE FROM whatsmeow_device WHERE jid = $1"
		result, err := tx.Exec(deleteQuery, jid.String)
		if err != nil {
			r.logger.Warn().Err(err).Str("jid", jid.String).Msg("Failed to delete WhatsApp device (continuing)")
		} else {
			if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
				r.logger.Info().Str("jid", jid.String).Int64("rows", rowsAffected).Msg("WhatsApp device and related data cleaned via CASCADE")
			}
		}
	}

	// Deletar a sessão
	sessionQuery := "DELETE FROM sessions WHERE id = $1"
	result, err := tx.Exec(sessionQuery, id)
	if err != nil {
		r.logger.Error().Err(err).Str("id", id.String()).Msg("Failed to delete session")
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	// Commit da transação
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info().Str("id", id.String()).Msg("Session and associated WhatsApp data deleted successfully")
	return nil
}


func (r *sessionRepository) DeleteBySessionID(sessionID string) error {

	id, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID format: %w", err)
	}


	return r.Delete(id)
}


func (r *sessionRepository) DeleteByName(name string) error {
	// Primeiro, obter o ID e JID da sessão
	var id uuid.UUID
	var jid sql.NullString
	query := "SELECT id, jid FROM sessions WHERE name = $1"
	err := r.db.QueryRow(query, name).Scan(&id, &jid)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("session not found")
		}
		r.logger.Error().Err(err).Str("name", name).Msg("Failed to get session info for cleanup")
		return fmt.Errorf("failed to get session info: %w", err)
	}

	// Usar o método Delete principal que já tem a lógica de limpeza
	return r.Delete(id)
}


func (r *sessionRepository) DeleteByIdentifier(identifier string) error {

	if id, err := uuid.Parse(identifier); err == nil {
		return r.Delete(id)
	}


	return r.DeleteByName(identifier)
}



func (r *sessionRepository) Exists(identifier string) (bool, error) {

	if id, err := uuid.Parse(identifier); err == nil {
		query := "SELECT EXISTS(SELECT 1 FROM sessions WHERE id = $1)"
		var exists bool
		err := r.db.QueryRow(query, id).Scan(&exists)
		if err != nil {
			r.logger.Error().Err(err).Str("session_id", identifier).Msg("Failed to check if session exists by ID")
			return false, fmt.Errorf("failed to check if session exists: %w", err)
		}
		return exists, nil
	}


	query := "SELECT EXISTS(SELECT 1 FROM sessions WHERE name = $1)"
	var exists bool
	err := r.db.QueryRow(query, identifier).Scan(&exists)
	if err != nil {
		r.logger.Error().Err(err).Str("session_name", identifier).Msg("Failed to check if session exists by name")
		return false, fmt.Errorf("failed to check if session exists: %w", err)
	}

	return exists, nil
}


func (r *sessionRepository) Count() (int, error) {
	query := "SELECT COUNT(*) FROM sessions"

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to count sessions")
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	return count, nil
}


func (r *sessionRepository) GetActiveConnections() ([]*models.Session, error) {
	query := `
		SELECT id, name, api_key, jid, status,
		       proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
		webhook_url, webhook_events, created_at, updated_at, last_connected_at, metadata
		FROM sessions WHERE status IN ('connected', 'authenticated')
		ORDER BY last_connected_at DESC NULLS LAST, updated_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get active connections")
		return nil, fmt.Errorf("failed to get active connections: %w", err)
	}
	defer rows.Close()

	sessions := make([]*models.Session, 0)
	for rows.Next() {
		session := &models.Session{}
		err := rows.Scan(
			&session.ID, &session.Name, &session.APIKey, &session.JID, &session.Status,
			&session.ProxyEnabled, &session.ProxyHost, &session.ProxyPort, &session.ProxyUsername, &session.ProxyPassword,
			&session.WebhookURL, &session.WebhookEvents, &session.CreatedAt, &session.UpdatedAt, &session.LastConnectedAt, &session.Metadata,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan active connection")
			return nil, fmt.Errorf("failed to scan active connection: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating active connections")
		return nil, fmt.Errorf("error iterating active connections: %w", err)
	}

	r.logger.Debug().Int("count", len(sessions)).Msg("Retrieved active connections")
	return sessions, nil
}


func (r *sessionRepository) Close() error {
	return r.db.Close()
}
