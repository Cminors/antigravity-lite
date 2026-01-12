package account

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// Manager handles account operations
type Manager struct {
	storage      *Storage
	currentIndex int
	mu           sync.RWMutex
}

// NewManager creates a new account manager
func NewManager(storage *Storage) *Manager {
	return &Manager{
		storage:      storage,
		currentIndex: 0,
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

// GetNextActive returns the next active account using round-robin
func (m *Manager) GetNextActive() (*Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	accounts, err := m.storage.GetActiveAccounts()
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, errors.New("no active accounts available")
	}

	// Round-robin selection
	if m.currentIndex >= len(accounts) {
		m.currentIndex = 0
	}

	account := accounts[m.currentIndex]
	m.currentIndex++

	// Update last used
	_ = m.storage.UpdateLastUsed(account.ID)

	return &account, nil
}

// GetBestAccount returns the account with most remaining quota
func (m *Manager) GetBestAccount() (*Account, error) {
	accounts, err := m.storage.GetActiveAccounts()
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, errors.New("no active accounts available")
	}

	// Already sorted by quota_used ASC
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
	// Get OAuth credentials from environment variables
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	
	if clientID == "" || clientSecret == "" {
		return "", time.Time{}, errors.New("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET environment variables are required")
	}

	// Google OAuth token refresh
	data := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": refreshToken,
		"grant_type":    "refresh_token",
	}

	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(
		"https://oauth2.googleapis.com/token",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", time.Time{}, fmt.Errorf("token refresh failed: %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
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
func (m *Manager) MarkAccountError(id int64, statusCode int) {
	switch statusCode {
	case 401:
		_ = m.storage.UpdateStatus(id, StatusExpired)
	case 403:
		_ = m.storage.UpdateStatus(id, StatusBanned)
	}
}

// GetStorage returns the underlying storage
func (m *Manager) GetStorage() *Storage {
	return m.storage
}
