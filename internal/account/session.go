package account

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// RateLimitEntry tracks rate limit status for an account
type RateLimitEntry struct {
	AccountID int64
	Email     string
	LimitedAt time.Time
	ResetAt   time.Time
	FailCount int
	LastError string
}

// RateLimitTracker tracks rate-limited accounts
type RateLimitTracker struct {
	mu      sync.RWMutex
	entries map[int64]*RateLimitEntry
}

// NewRateLimitTracker creates a new rate limit tracker
func NewRateLimitTracker() *RateLimitTracker {
	return &RateLimitTracker{
		entries: make(map[int64]*RateLimitEntry),
	}
}

// MarkRateLimited marks an account as rate limited
func (t *RateLimitTracker) MarkRateLimited(accountID int64, email string, resetSeconds int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	resetAt := now.Add(time.Duration(resetSeconds) * time.Second)

	if entry, exists := t.entries[accountID]; exists {
		entry.LimitedAt = now
		entry.ResetAt = resetAt
		entry.FailCount++
	} else {
		t.entries[accountID] = &RateLimitEntry{
			AccountID: accountID,
			Email:     email,
			LimitedAt: now,
			ResetAt:   resetAt,
			FailCount: 1,
		}
	}
}

// IsRateLimited checks if an account is currently rate limited
func (t *RateLimitTracker) IsRateLimited(accountID int64) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entry, exists := t.entries[accountID]
	if !exists {
		return false
	}

	// Check if reset time has passed
	if time.Now().After(entry.ResetAt) {
		return false
	}

	return true
}

// GetRemainingWait returns remaining wait time in seconds
func (t *RateLimitTracker) GetRemainingWait(accountID int64) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entry, exists := t.entries[accountID]
	if !exists {
		return 0
	}

	remaining := time.Until(entry.ResetAt)
	if remaining <= 0 {
		return 0
	}

	return int(remaining.Seconds())
}

// ClearRateLimit clears rate limit for an account
func (t *RateLimitTracker) ClearRateLimit(accountID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, accountID)
}

// ClearExpired removes expired rate limit entries
func (t *RateLimitTracker) ClearExpired() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for id, entry := range t.entries {
		if now.After(entry.ResetAt) {
			delete(t.entries, id)
		}
	}
}

// SessionBinding tracks session to account binding for stickiness
type SessionBinding struct {
	SessionID string
	AccountID int64
	BoundAt   time.Time
}

// SessionManager manages session stickiness
type SessionManager struct {
	mu       sync.RWMutex
	bindings map[string]*SessionBinding
	ttl      time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(ttl time.Duration) *SessionManager {
	return &SessionManager{
		bindings: make(map[string]*SessionBinding),
		ttl:      ttl,
	}
}

// GenerateSessionID generates a stable session ID from the first user message
// This ensures the same conversation always uses the same account
func GenerateSessionID(firstMessage string) string {
	if firstMessage == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(firstMessage))
	return hex.EncodeToString(hash[:8]) // First 8 bytes = 16 hex chars
}

// GetBoundAccount returns the account bound to a session, if any and still valid
func (m *SessionManager) GetBoundAccount(sessionID string) (int64, bool) {
	if sessionID == "" {
		return 0, false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	binding, exists := m.bindings[sessionID]
	if !exists {
		return 0, false
	}

	// Check TTL
	if time.Since(binding.BoundAt) > m.ttl {
		return 0, false
	}

	return binding.AccountID, true
}

// BindSession binds a session to an account
func (m *SessionManager) BindSession(sessionID string, accountID int64) {
	if sessionID == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.bindings[sessionID] = &SessionBinding{
		SessionID: sessionID,
		AccountID: accountID,
		BoundAt:   time.Now(),
	}
}

// UnbindSession removes session binding
func (m *SessionManager) UnbindSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.bindings, sessionID)
}

// CleanupExpired removes expired session bindings
func (m *SessionManager) CleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, binding := range m.bindings {
		if now.Sub(binding.BoundAt) > m.ttl {
			delete(m.bindings, id)
		}
	}
}
