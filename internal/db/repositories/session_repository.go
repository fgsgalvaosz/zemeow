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

// SessionRepository interface define as operações do repositório de sessões
type SessionRepository interface {
	Create(session *models.Session) error
	GetByID(id uuid.UUID) (*models.Session, error)
	GetBySessionID(sessionID string) (*models.Session, error)
	GetByAPIKey(apiKey string) (*models.Session, error)
	GetAll(filter *models.SessionFilter) (*models.SessionListResponse, error)
	Update(session *models.Session) error
	UpdateStatus(sessionID string, status models.SessionStatus) error
	Delete(id uuid.UUID) error
	DeleteBySessionID(sessionID string) error
	Exists(sessionID string) (bool, error)
	Count() (int, error)
	GetActiveConnections() ([]*models.Session, error)
}

// sessionRepository implementa SessionRepository
type sessionRepository struct {
	db     *sql.DB
	logger logger.Logger
}

// NewSessionRepository cria uma nova instância do repositório de sessões
func NewSessionRepository(db *sql.DB) SessionRepository {
	return &sessionRepository{
		db:     db,
		logger: logger.Get(),
	}
}

// Create cria uma nova sessão no banco de dados
func (r *sessionRepository) Create(session *models.Session) error {
	if err := session.Validate(); err != nil {
		return fmt.Errorf("invalid session data: %w", err)
	}

	// Gerar ID se não fornecido
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}

	// Definir timestamps
	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	query := `
		INSERT INTO sessions (
			id, session_id, name, api_key, jid, status,
			proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
			webhook_url, webhook_events, created_at, updated_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	_, err := r.db.Exec(query,
		session.ID, session.SessionID, session.Name, session.APIKey, session.JID, session.Status,
		session.ProxyEnabled, session.ProxyHost, session.ProxyPort, session.ProxyUsername, session.ProxyPassword,
		session.WebhookURL, session.WebhookEvents, session.CreatedAt, session.UpdatedAt, session.Metadata,
	)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", session.SessionID).Msg("Failed to create session")
		return fmt.Errorf("failed to create session: %w", err)
	}

	r.logger.Info().Str("session_id", session.SessionID).Msg("Session created successfully")
	return nil
}

// GetByID busca uma sessão pelo ID
func (r *sessionRepository) GetByID(id uuid.UUID) (*models.Session, error) {
	query := `
		SELECT id, session_id, name, api_key, jid, status,
		       proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
		       webhook_url, webhook_events, created_at, updated_at, last_connected_at, metadata
		FROM sessions WHERE id = $1
	`

	session := &models.Session{}
	err := r.db.QueryRow(query, id).Scan(
		&session.ID, &session.SessionID, &session.Name, &session.APIKey, &session.JID, &session.Status,
		&session.ProxyEnabled, &session.ProxyHost, &session.ProxyPort, &session.ProxyUsername, &session.ProxyPassword,
		&session.WebhookURL, &session.WebhookEvents, &session.CreatedAt, &session.UpdatedAt, &session.LastConnectedAt, &session.Metadata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		r.logger.Error().Err(err).Str("id", id.String()).Msg("Failed to get session by ID")
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// GetBySessionID busca uma sessão pelo session_id
func (r *sessionRepository) GetBySessionID(sessionID string) (*models.Session, error) {
	query := `
		SELECT id, session_id, name, api_key, jid, status,
		       proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
		       webhook_url, webhook_events, created_at, updated_at, last_connected_at, metadata
		FROM sessions WHERE session_id = $1
	`

	session := &models.Session{}
	err := r.db.QueryRow(query, sessionID).Scan(
		&session.ID, &session.SessionID, &session.Name, &session.APIKey, &session.JID, &session.Status,
		&session.ProxyEnabled, &session.ProxyHost, &session.ProxyPort, &session.ProxyUsername, &session.ProxyPassword,
		&session.WebhookURL, &session.WebhookEvents, &session.CreatedAt, &session.UpdatedAt, &session.LastConnectedAt, &session.Metadata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		r.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get session by session_id")
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// GetByAPIKey busca uma sessão pela API key
func (r *sessionRepository) GetByAPIKey(apiKey string) (*models.Session, error) {
	query := `
		SELECT id, session_id, name, api_key, jid, status,
		       proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
		       webhook_url, webhook_events, created_at, updated_at, last_connected_at, metadata
		FROM sessions WHERE api_key = $1
	`

	session := &models.Session{}
	err := r.db.QueryRow(query, apiKey).Scan(
		&session.ID, &session.SessionID, &session.Name, &session.APIKey, &session.JID, &session.Status,
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

// GetAll busca todas as sessões com filtros e paginação
func (r *sessionRepository) GetAll(filter *models.SessionFilter) (*models.SessionListResponse, error) {
	if filter == nil {
		filter = &models.SessionFilter{}
	}

	// Definir valores padrão
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

	// Construir query com filtros
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

	// Query para contar total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sessions %s", whereClause)
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to count sessions")
		return nil, fmt.Errorf("failed to count sessions: %w", err)
	}

	// Query principal com paginação
	offset := (filter.Page - 1) * filter.PerPage
	query := fmt.Sprintf(`
		SELECT id, session_id, name, api_key, jid, status,
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
			&session.ID, &session.SessionID, &session.Name, &session.APIKey, &session.JID, &session.Status,
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

// Update atualiza uma sessão existente
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
		session.ID, session.Name, session.JID, session.Status,
		session.ProxyEnabled, session.ProxyHost, session.ProxyPort, session.ProxyUsername, session.ProxyPassword,
		session.WebhookURL, session.WebhookEvents, session.UpdatedAt, session.LastConnectedAt, session.Metadata,
	)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", session.SessionID).Msg("Failed to update session")
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	r.logger.Info().Str("session_id", session.SessionID).Msg("Session updated successfully")
	return nil
}

// UpdateStatus atualiza apenas o status de uma sessão
func (r *sessionRepository) UpdateStatus(sessionID string, status models.SessionStatus) error {
	now := time.Now()
	var lastConnectedAt *time.Time

	if status == models.SessionStatusConnected || status == models.SessionStatusAuthenticated {
		lastConnectedAt = &now
	}

	query := `
		UPDATE sessions SET
			status = $2, updated_at = $3, last_connected_at = COALESCE($4, last_connected_at)
		WHERE session_id = $1
	`

	result, err := r.db.Exec(query, sessionID, status, now, lastConnectedAt)
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

// Delete remove uma sessão pelo ID
func (r *sessionRepository) Delete(id uuid.UUID) error {
	query := "DELETE FROM sessions WHERE id = $1"

	result, err := r.db.Exec(query, id)
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

	r.logger.Info().Str("id", id.String()).Msg("Session deleted successfully")
	return nil
}

// DeleteBySessionID remove uma sessão pelo session_id
func (r *sessionRepository) DeleteBySessionID(sessionID string) error {
	query := "DELETE FROM sessions WHERE session_id = $1"

	result, err := r.db.Exec(query, sessionID)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to delete session")
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	r.logger.Info().Str("session_id", sessionID).Msg("Session deleted successfully")
	return nil
}

// Exists verifica se uma sessão existe
func (r *sessionRepository) Exists(sessionID string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM sessions WHERE session_id = $1)"

	var exists bool
	err := r.db.QueryRow(query, sessionID).Scan(&exists)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to check if session exists")
		return false, fmt.Errorf("failed to check if session exists: %w", err)
	}

	return exists, nil
}

// Count retorna o número total de sessões
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

// GetActiveConnections retorna todas as sessões conectadas
func (r *sessionRepository) GetActiveConnections() ([]*models.Session, error) {
	query := `
		SELECT id, session_id, name, api_key, jid, status,
		       proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password,
		       webhook_url, webhook_events, created_at, updated_at, last_connected_at, metadata
		FROM sessions 
		WHERE status IN ('connected', 'authenticated')
		ORDER BY last_connected_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get active connections")
		return nil, fmt.Errorf("failed to get active connections: %w", err)
	}
	defer rows.Close()

	var sessions []*models.Session
	for rows.Next() {
		session := &models.Session{}
		err := rows.Scan(
			&session.ID, &session.SessionID, &session.Name, &session.APIKey, &session.JID, &session.Status,
			&session.ProxyEnabled, &session.ProxyHost, &session.ProxyPort, &session.ProxyUsername, &session.ProxyPassword,
			&session.WebhookURL, &session.WebhookEvents, &session.CreatedAt, &session.UpdatedAt, &session.LastConnectedAt, &session.Metadata,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan active session")
			return nil, fmt.Errorf("failed to scan active session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating active sessions")
		return nil, fmt.Errorf("error iterating active sessions: %w", err)
	}

	r.logger.Info().Int("count", len(sessions)).Msg("Retrieved active connections")
	return sessions, nil
}
