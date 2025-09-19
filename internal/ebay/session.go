package ebay

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session represents a user session
type Session struct {
	ID         string    `json:"id"`
	EBayUserID string    `json:"ebayUserId"` // eBay user identifier from OAuth token
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	IPAddress  string    `json:"ipAddress"`
}

// SessionManager handles user sessions
type SessionManager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session // sessionID -> Session
	userSessions map[string]string   // ebayUserID -> sessionID
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions:     make(map[string]*Session),
		userSessions: make(map[string]string),
	}

	// Start cleanup goroutine
	go sm.startCleanup()

	return sm
}

// CreateSession creates a new session for a user
func (sm *SessionManager) CreateSession(ebayUserID, ipAddress string) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:         sessionID,
		EBayUserID: ebayUserID,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour), // 24 hour sessions
		IPAddress:  ipAddress,
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Remove any existing session for this user
	if oldSessionID, exists := sm.userSessions[ebayUserID]; exists {
		delete(sm.sessions, oldSessionID)
	}

	// Store new session
	sm.sessions[sessionID] = session
	sm.userSessions[ebayUserID] = sessionID

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Session expired, remove it
		sm.mu.RUnlock()
		sm.mu.Lock()
		delete(sm.sessions, sessionID)
		delete(sm.userSessions, session.EBayUserID)
		sm.mu.Unlock()
		sm.mu.RLock()
		return nil, false
	}

	return session, true
}

// GetUserSession retrieves a session by eBay user ID
func (sm *SessionManager) GetUserSession(ebayUserID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessionID, exists := sm.userSessions[ebayUserID]
	if !exists {
		return nil, false
	}

	return sm.GetSession(sessionID)
}

// ExtendSession extends the expiration time of a session
func (sm *SessionManager) ExtendSession(sessionID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}

	// Extend by another 24 hours
	session.ExpiresAt = time.Now().Add(24 * time.Hour)
	return true
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return
	}

	delete(sm.sessions, sessionID)
	delete(sm.userSessions, session.EBayUserID)
}

// DeleteUserSession removes a session by eBay user ID
func (sm *SessionManager) DeleteUserSession(ebayUserID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionID, exists := sm.userSessions[ebayUserID]
	if !exists {
		return
	}

	delete(sm.sessions, sessionID)
	delete(sm.userSessions, ebayUserID)
}

// GetActiveSessionCount returns the number of active sessions
func (sm *SessionManager) GetActiveSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// startCleanup runs periodic cleanup of expired sessions
func (sm *SessionManager) startCleanup() {
	ticker := time.NewTicker(1 * time.Hour) // Clean up every hour
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanupExpiredSessions()
	}
}

// cleanupExpiredSessions removes expired sessions
func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	expiredSessions := []string{}

	// Find expired sessions
	for sessionID, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	// Remove expired sessions
	for _, sessionID := range expiredSessions {
		session := sm.sessions[sessionID]
		delete(sm.sessions, sessionID)
		delete(sm.userSessions, session.EBayUserID)
	}
}

// generateSessionID creates a cryptographically secure session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
