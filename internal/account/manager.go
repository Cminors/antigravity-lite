package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Manager handles account operations
type Manager struct {
	storage        *Storage
	currentIndex   int
	mu             sync.RWMutex
	rateLimiter    *RateLimitTracker
	sessionManager *SessionManager
}

// NewManager creates a new account manager
func NewManager(storage *Storage) *Manager {
	m := &Manager{
		storage:        storage,
		currentIndex:   0,
		rateLimiter:    NewRateLimitTracker(),
		sessionManager: NewSessionManager(60 * time.Minute), // 60 min session TTL
	}

	// Start cleanup goroutine
	go m.periodicCleanup()

	return m
}

// periodicCleanup cleans up expired rate limits and sessions
func (m *Manager) periodicCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		m.rateLimiter.ClearExpired()
		m.sessionManager.CleanupExpired()
	}
}

// List returns all accounts
func (m *Manager) List() ([]Account, error) {
	return m.storage.List()
}

// Get returns an account by ID
func (m *Manager) Get(id int64) (*Account, error) {
	return m.storage.Get(id)
}

// Create creates a new account
func (m *Manager) Create(input AccountInput) (*Account, error) {
	if input.AccountType == "" {
		input.AccountType = "free"
	}
	return m.storage.Create(input)
}

// Update updates an account
func (m *Manager) Update(id int64, input AccountInput) (*Account, error) {
	return m.storage.Update(id, input)
}

// Delete deletes an account
func (m *Manager) Delete(id int64) error {
	return m.storage.Delete(id)
}

// GetNextActive returns the next active account using intelligent selection
// Priority: 1. Session-bound account (if valid)
//  2. Non-rate-limited account with highest quota
//  3. Any available account
func (m *Manager) GetNextActive() (*Account, error) {
	return m.GetNextActiveWithSession("")
}

// GetNextActiveWithSession returns the next active account with session stickiness
func (m *Manager) GetNextActiveWithSession(sessionID string) (*Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	accounts, err := m.storage.GetActiveAccounts()
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, errors.New("no active accounts available")
	}

	// 1. Check session binding first (for stability)
	if sessionID != "" {
		if boundAccountID, ok := m.sessionManager.GetBoundAccount(sessionID); ok {
			// Check if bound account is still active and not rate limited
			for _, acc := range accounts {
				if acc.ID == boundAccountID && !m.rateLimiter.IsRateLimited(acc.ID) {
					// Reuse bound account
					_ = m.storage.UpdateLastUsed(acc.ID)
					return &acc, nil
				}
			}
			// Bound account is no longer valid, unbind
			m.sessionManager.UnbindSession(sessionID)
		}
	}

	// 2. Find best non-rate-limited account (already sorted by tier/quota)
	for _, acc := range accounts {
		if !m.rateLimiter.IsRateLimited(acc.ID) {
			// Bind to session if provided
			if sessionID != "" {
				m.sessionManager.BindSession(sessionID, acc.ID)
			}
			_ = m.storage.UpdateLastUsed(acc.ID)
			return &acc, nil
		}
	}

	// 3. All accounts rate limited - find the one with shortest wait
	var bestAccount *Account
	minWait := int(^uint(0) >> 1) // Max int

	for i := range accounts {
		wait := m.rateLimiter.GetRemainingWait(accounts[i].ID)
		if wait < minWait {
			minWait = wait
			bestAccount = &accounts[i]
		}
	}

	if bestAccount != nil {
		if sessionID != "" {
			m.sessionManager.BindSession(sessionID, bestAccount.ID)
		}
		_ = m.storage.UpdateLastUsed(bestAccount.ID)
		return bestAccount, nil
	}

	return nil, errors.New("no accounts available")
}

// GetBestAccount returns the account with most remaining quota (not rate limited)
func (m *Manager) GetBestAccount() (*Account, error) {
	accounts, err := m.storage.GetActiveAccounts()
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, errors.New("no active accounts available")
	}

	// Already sorted by tier and (quota_limit - quota_used) DESC
	// Return first non-rate-limited account
	for i := range accounts {
		if !m.rateLimiter.IsRateLimited(accounts[i].ID) {
			return &accounts[i], nil
		}
	}

	// If all rate limited, return the first one anyway
	return &accounts[0], nil
}

// CheckAccountStatus checks and updates account status
func (m *Manager) CheckAccountStatus(id int64) (*Account, error) {
	account, err := m.storage.Get(id)
	if err != nil {
		return nil, err
	}

	// Mark as checking
	_ = m.storage.UpdateStatus(id, StatusChecking)

	// Refresh token if needed
	if account.AccessToken == "" || time.Now().After(account.TokenExpiry) {
		accessToken, expiry, err := m.refreshAccessToken(account.RefreshToken)
		if err != nil {
			_ = m.storage.UpdateStatus(id, StatusExpired)
			account.Status = StatusExpired
			return account, nil
		}
		_ = m.storage.UpdateToken(id, accessToken, expiry)
		account.AccessToken = accessToken
		account.TokenExpiry = expiry
	}

	// Test API call
	status := m.testAPICall(account.AccessToken)
	_ = m.storage.UpdateStatus(id, status)
	account.Status = status

	return account, nil
}

// refreshAccessToken refreshes the access token using refresh token
func (m *Manager) refreshAccessToken(refreshToken string) (string, time.Time, error) {
	// Built-in OAuth credentials (same as Antigravity-Manager)
	// These are Google's official Cloud Code OAuth credentials
	const defaultClientID = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	const defaultClientSecret = "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"

	// Allow environment variable override if user wants custom credentials
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if clientID == "" {
		clientID = defaultClientID
	}
	if clientSecret == "" {
		clientSecret = defaultClientSecret
	}

	// Google OAuth token refresh using form-urlencoded (required format)
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	resp, err := http.Post(
		"https://oauth2.googleapis.com/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", time.Time{}, fmt.Errorf("token refresh failed: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", time.Time{}, err
	}

	expiry := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return result.AccessToken, expiry, nil
}

// testAPICall tests if the account can make API calls
func (m *Manager) testAPICall(accessToken string) Status {
	client := &http.Client{Timeout: 10 * time.Second}

	req, _ := http.NewRequest("GET",
		"https://generativelanguage.googleapis.com/v1beta/models",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnknown
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		return StatusActive
	case 401, 403:
		return StatusBanned
	case 429:
		return StatusActive // Rate limited but valid
	default:
		return StatusUnknown
	}
}

// Import imports accounts from JSON
func (m *Manager) Import(data []byte) (int, error) {
	var exports []AccountExport
	if err := json.Unmarshal(data, &exports); err != nil {
		return 0, err
	}

	count := 0
	for _, e := range exports {
		input := AccountInput{
			Name:         e.Name,
			Email:        e.Email,
			RefreshToken: e.RefreshToken,
			AccountType:  e.AccountType,
		}
		if input.Name == "" {
			input.Name = fmt.Sprintf("Account %d", count+1)
		}
		if input.AccountType == "" {
			input.AccountType = "free"
		}

		if _, err := m.storage.Create(input); err == nil {
			count++
		}
	}

	return count, nil
}

// Export exports all accounts
func (m *Manager) Export() ([]byte, error) {
	accounts, err := m.storage.List()
	if err != nil {
		return nil, err
	}

	exports := make([]AccountExport, len(accounts))
	for i, a := range accounts {
		exports[i] = AccountExport{
			Name:         a.Name,
			Email:        a.Email,
			RefreshToken: a.RefreshToken,
			AccountType:  a.AccountType,
		}
	}

	return json.MarshalIndent(exports, "", "  ")
}

// CheckAllAccounts checks status of all accounts
func (m *Manager) CheckAllAccounts() error {
	accounts, err := m.storage.List()
	if err != nil {
		return err
	}

	for _, a := range accounts {
		_, _ = m.CheckAccountStatus(a.ID)
	}

	return nil
}

// EnsureValidToken ensures the account has a valid access token
func (m *Manager) EnsureValidToken(account *Account) error {
	if account.AccessToken != "" && time.Now().Before(account.TokenExpiry.Add(-5*time.Minute)) {
		return nil // Token still valid
	}

	accessToken, expiry, err := m.refreshAccessToken(account.RefreshToken)
	if err != nil {
		_ = m.storage.UpdateStatus(account.ID, StatusExpired)
		return err
	}

	_ = m.storage.UpdateToken(account.ID, accessToken, expiry)
	account.AccessToken = accessToken
	account.TokenExpiry = expiry

	return nil
}

// MarkAccountError marks an account as having an error
// For 429 errors, it marks the account as rate limited temporarily
// For 401/403 errors, it updates the account status
func (m *Manager) MarkAccountError(id int64, statusCode int) {
	account, err := m.storage.Get(id)
	if err != nil {
		return
	}

	switch statusCode {
	case 429:
		// Rate limited - mark for 60 seconds default
		// In production, parse Retry-After header from response
		m.rateLimiter.MarkRateLimited(id, account.Email, 60)
	case 401:
		_ = m.storage.UpdateStatus(id, StatusExpired)
	case 403:
		_ = m.storage.UpdateStatus(id, StatusBanned)
	case 500, 503:
		// Server error - brief rate limit to avoid hammering
		m.rateLimiter.MarkRateLimited(id, account.Email, 10)
	}
}

// MarkAccountSuccess clears rate limit for an account after successful request
func (m *Manager) MarkAccountSuccess(id int64) {
	m.rateLimiter.ClearRateLimit(id)
}

// GetStorage returns the underlying storage
func (m *Manager) GetStorage() *Storage {
	return m.storage
}
