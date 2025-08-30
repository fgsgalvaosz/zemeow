package session

import (
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/rs/zerolog"
)

// CacheEntry representa uma entrada no cache
type CacheEntry struct {
	Session   *models.Session
	Token     string
	Status    models.SessionStatus
	LastSeen  time.Time
	ExpiresAt time.Time
}

// SessionCache implementa um cache thread-safe para sessões
type SessionCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry // sessionID -> CacheEntry
	tokens  map[string]string      // token -> sessionID
	logger  zerolog.Logger

	// Configurações
	defaultTTL      time.Duration
	cleanupInterval time.Duration

	// Controle de cleanup
	stopCleanup chan struct{}
}

// NewSessionCache cria um novo cache de sessões
func NewSessionCache(defaultTTL, cleanupInterval time.Duration) *SessionCache {
	cache := &SessionCache{
		entries:         make(map[string]*CacheEntry),
		tokens:          make(map[string]string),
		logger:          logger.Get().With().Str("component", "session_cache").Logger(),
		defaultTTL:      defaultTTL,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Iniciar limpeza automática
	go cache.startCleanup()

	return cache
}

// Set adiciona ou atualiza uma sessão no cache
func (c *SessionCache) Set(sessionID string, session *models.Session, ttl ...time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiration := c.defaultTTL
	if len(ttl) > 0 {
		expiration = ttl[0]
	}

	entry := &CacheEntry{
		Session:   session,
		Token:     session.Token,
		Status:    session.Status,
		LastSeen:  time.Now(),
		ExpiresAt: time.Now().Add(expiration),
	}

	// Remover token antigo se existir
	if oldEntry, exists := c.entries[sessionID]; exists {
		delete(c.tokens, oldEntry.Token)
	}

	// Adicionar nova entrada
	c.entries[sessionID] = entry
	c.tokens[session.Token] = sessionID

	c.logger.Debug().Str("session_id", sessionID).Msg("Session cached")
}

// Get recupera uma sessão do cache
func (c *SessionCache) Get(sessionID string) (*models.Session, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[sessionID]
	if !exists {
		return nil, false
	}

	// Verificar se expirou
	if time.Now().After(entry.ExpiresAt) {
		c.logger.Debug().Str("session_id", sessionID).Msg("Cache entry expired")
		return nil, false
	}

	// Atualizar last seen
	entry.LastSeen = time.Now()

	return entry.Session, true
}

// GetByToken recupera uma sessão pelo token
func (c *SessionCache) GetByToken(token string) (*models.Session, bool) {
	c.mu.RLock()
	sessionID, exists := c.tokens[token]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	return c.Get(sessionID)
}

// UpdateStatus atualiza o status de uma sessão no cache
func (c *SessionCache) UpdateStatus(sessionID string, status models.SessionStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[sessionID]
	if !exists {
		return
	}

	entry.Status = status
	entry.Session.Status = status
	entry.LastSeen = time.Now()

	c.logger.Debug().Str("session_id", sessionID).Str("status", string(status)).Msg("Session status updated in cache")
}

// Delete remove uma sessão do cache
func (c *SessionCache) Delete(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[sessionID]
	if !exists {
		return
	}

	// Remover token
	delete(c.tokens, entry.Token)

	// Remover entrada
	delete(c.entries, sessionID)

	c.logger.Debug().Str("session_id", sessionID).Msg("Session removed from cache")
}

// Exists verifica se uma sessão existe no cache
func (c *SessionCache) Exists(sessionID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[sessionID]
	if !exists {
		return false
	}

	// Verificar se expirou
	return !time.Now().After(entry.ExpiresAt)
}

// List retorna todas as sessões no cache
func (c *SessionCache) List() []*models.Session {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sessions := make([]*models.Session, 0, len(c.entries))
	now := time.Now()

	for _, entry := range c.entries {
		// Pular entradas expiradas
		if now.After(entry.ExpiresAt) {
			continue
		}

		sessions = append(sessions, entry.Session)
	}

	return sessions
}

// GetStats retorna estatísticas do cache
func (c *SessionCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		TotalEntries: len(c.entries),
		TotalTokens:  len(c.tokens),
	}

	now := time.Now()
	for _, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			stats.ExpiredEntries++
		} else {
			stats.ActiveEntries++
		}

		switch entry.Status {
		case models.SessionStatusConnected, models.SessionStatusAuthenticated:
			stats.ConnectedSessions++
		case models.SessionStatusDisconnected:
			stats.DisconnectedSessions++
		case models.SessionStatusConnecting:
			stats.ConnectingSessions++
		case models.SessionStatusError:
			stats.ErrorSessions++
		}
	}

	return stats
}

// Refresh atualiza o TTL de uma sessão
func (c *SessionCache) Refresh(sessionID string, ttl ...time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[sessionID]
	if !exists {
		return false
	}

	expiration := c.defaultTTL
	if len(ttl) > 0 {
		expiration = ttl[0]
	}

	entry.ExpiresAt = time.Now().Add(expiration)
	entry.LastSeen = time.Now()

	c.logger.Debug().Str("session_id", sessionID).Msg("Session cache refreshed")
	return true
}

// Clear limpa todo o cache
func (c *SessionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.tokens = make(map[string]string)

	c.logger.Info().Msg("Session cache cleared")
}

// startCleanup inicia o processo de limpeza automática
func (c *SessionCache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	c.logger.Info().Dur("interval", c.cleanupInterval).Msg("Starting cache cleanup routine")

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			c.logger.Info().Msg("Cache cleanup routine stopped")
			return
		}
	}
}

// cleanup remove entradas expiradas
func (c *SessionCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	// Identificar entradas expiradas
	for sessionID, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			expired = append(expired, sessionID)
		}
	}

	// Remover entradas expiradas
	for _, sessionID := range expired {
		entry := c.entries[sessionID]
		delete(c.tokens, entry.Token)
		delete(c.entries, sessionID)
	}

	if len(expired) > 0 {
		c.logger.Debug().Int("expired_count", len(expired)).Msg("Cleaned up expired cache entries")
	}
}

// Stop para o processo de limpeza
func (c *SessionCache) Stop() {
	close(c.stopCleanup)
}

// CacheStats representa estatísticas do cache
type CacheStats struct {
	TotalEntries         int `json:"total_entries"`
	ActiveEntries        int `json:"active_entries"`
	ExpiredEntries       int `json:"expired_entries"`
	TotalTokens          int `json:"total_tokens"`
	ConnectedSessions    int `json:"connected_sessions"`
	DisconnectedSessions int `json:"disconnected_sessions"`
	ConnectingSessions   int `json:"connecting_sessions"`
	ErrorSessions        int `json:"error_sessions"`
}
