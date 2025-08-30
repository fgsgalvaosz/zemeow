package repositories

import (
	"fmt"
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/db/models"
)

// MockSessionRepository implementa SessionRepository em memória para testes
type MockSessionRepository struct {
	sessions map[string]*models.Session
	mutex    sync.RWMutex
}

// NewMockSessionRepository cria um novo repositório mock
func NewMockSessionRepository() SessionRepository {
	return &MockSessionRepository{
		sessions: make(map[string]*models.Session),
	}
}

// Create cria uma nova sessão
func (r *MockSessionRepository) Create(session *models.Session) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.sessions[session.SessionID]; exists {
		return fmt.Errorf("session already exists")
	}

	// Simular ID auto-incremento
	session.ID = uint(len(r.sessions) + 1)
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()

	r.sessions[session.SessionID] = session
	return nil
}

// GetByID obtém uma sessão por ID
func (r *MockSessionRepository) GetByID(id uint) (*models.Session, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, session := range r.sessions {
		if session.ID == id {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found")
}

// GetBySessionID obtém uma sessão por SessionID
func (r *MockSessionRepository) GetBySessionID(sessionID string) (*models.Session, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if session, exists := r.sessions[sessionID]; exists {
		return session, nil
	}

	return nil, fmt.Errorf("session not found")
}

// GetAll obtém todas as sessões
func (r *MockSessionRepository) GetAll() ([]*models.Session, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	sessions := make([]*models.Session, 0, len(r.sessions))
	for _, session := range r.sessions {
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Update atualiza uma sessão
func (r *MockSessionRepository) Update(session *models.Session) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.sessions[session.SessionID]; !exists {
		return fmt.Errorf("session not found")
	}

	session.UpdatedAt = time.Now()
	r.sessions[session.SessionID] = session
	return nil
}

// Delete remove uma sessão
func (r *MockSessionRepository) Delete(sessionID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found")
	}

	delete(r.sessions, sessionID)
	return nil
}

// Exists verifica se uma sessão existe
func (r *MockSessionRepository) Exists(sessionID string) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.sessions[sessionID]
	return exists, nil
}

// GetByStatus obtém sessões por status
func (r *MockSessionRepository) GetByStatus(status models.SessionStatus) ([]*models.Session, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var sessions []*models.Session
	for _, session := range r.sessions {
		if session.Status == status {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// Count retorna o número total de sessões
func (r *MockSessionRepository) Count() (int64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return int64(len(r.sessions)), nil
}

// GetConnectedSessions obtém sessões conectadas
func (r *MockSessionRepository) GetConnectedSessions() ([]*models.Session, error) {
	return r.GetByStatus(models.SessionStatusConnected)
}

// UpdateStatus atualiza apenas o status de uma sessão
func (r *MockSessionRepository) UpdateStatus(sessionID string, status models.SessionStatus) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.Status = status
	session.UpdatedAt = time.Now()
	return nil
}

// UpdateJID atualiza o JID de uma sessão
func (r *MockSessionRepository) UpdateJID(sessionID string, jid string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	if jid == "" {
		session.JID = nil
	} else {
		session.JID = &jid
	}
	session.UpdatedAt = time.Now()
	return nil
}

// GetStats retorna estatísticas das sessões
func (r *MockSessionRepository) GetStats() (map[string]interface{}, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := map[string]interface{}{
		"total":       len(r.sessions),
		"connected":   0,
		"disconnected": 0,
		"connecting":  0,
	}

	for _, session := range r.sessions {
		switch session.Status {
		case models.SessionStatusConnected:
			stats["connected"] = stats["connected"].(int) + 1
		case models.SessionStatusDisconnected:
			stats["disconnected"] = stats["disconnected"].(int) + 1
		case models.SessionStatusConnecting:
			stats["connecting"] = stats["connecting"].(int) + 1
		}
	}

	return stats, nil
}
