package session

import (
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/rs/zerolog"
)

type CacheEntry struct {
	Session *models.Session

	Status    models.SessionStatus
	LastSeen  time.Time
	ExpiresAt time.Time
}

type SessionCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry

	logger zerolog.Logger

	defaultTTL      time.Duration
	cleanupInterval time.Duration

	stopCleanup chan struct{}
}

func NewSessionCache(defaultTTL, cleanupInterval time.Duration) *SessionCache {
	cache := &SessionCache{
		entries: make(map[string]*CacheEntry),

		logger:          logger.Get().With().Str("component", "session_cache").Logger(),
		defaultTTL:      defaultTTL,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	go cache.startCleanup()

	return cache
}

func (c *SessionCache) Set(sessionID string, session *models.Session, ttl ...time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiration := c.defaultTTL
	if len(ttl) > 0 {
		expiration = ttl[0]
	}

	entry := &CacheEntry{
		Session: session,

		Status:    session.Status,
		LastSeen:  time.Now(),
		ExpiresAt: time.Now().Add(expiration),
	}

	c.entries[sessionID] = entry

	c.logger.Debug().Str("session_id", sessionID).Msg("Session cached")
}

func (c *SessionCache) Get(sessionID string) (*models.Session, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[sessionID]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		c.logger.Debug().Str("session_id", sessionID).Msg("Cache entry expired")
		return nil, false
	}

	entry.LastSeen = time.Now()

	return entry.Session, true
}

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

func (c *SessionCache) Delete(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.entries[sessionID]
	if !exists {
		return
	}

	delete(c.entries, sessionID)

	c.logger.Debug().Str("session_id", sessionID).Msg("Session removed from cache")
}

func (c *SessionCache) Exists(sessionID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[sessionID]
	if !exists {
		return false
	}

	return !time.Now().After(entry.ExpiresAt)
}

func (c *SessionCache) List() []*models.Session {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sessions := make([]*models.Session, 0, len(c.entries))
	now := time.Now()

	for _, entry := range c.entries {

		if now.After(entry.ExpiresAt) {
			continue
		}

		sessions = append(sessions, entry.Session)
	}

	return sessions
}

func (c *SessionCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		TotalEntries: len(c.entries),
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

func (c *SessionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)

	c.logger.Info().Msg("Session cache cleared")
}

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

func (c *SessionCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for sessionID, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			expired = append(expired, sessionID)
		}
	}

	for _, sessionID := range expired {
		delete(c.entries, sessionID)
	}

	if len(expired) > 0 {
		c.logger.Debug().Int("expired_count", len(expired)).Msg("Cleaned up expired cache entries")
	}
}

func (c *SessionCache) Stop() {
	close(c.stopCleanup)
}

type CacheStats struct {
	TotalEntries   int `json:"total_entries"`
	ActiveEntries  int `json:"active_entries"`
	ExpiredEntries int `json:"expired_entries"`

	ConnectedSessions    int `json:"connected_sessions"`
	DisconnectedSessions int `json:"disconnected_sessions"`
	ConnectingSessions   int `json:"connecting_sessions"`
	ErrorSessions        int `json:"error_sessions"`
}
