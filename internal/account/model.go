package account

import "time"

// Account represents a Google/API account
type Account struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	RefreshToken string    `json:"-"` // Never expose in JSON
	AccessToken  string    `json:"-"`
	TokenExpiry  time.Time `json:"token_expiry"`
	Status       Status    `json:"status"`
	AccountType  string    `json:"account_type"` // free, pro, ultra
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastUsedAt   time.Time `json:"last_used_at"`

	// Quota info
	QuotaUsed    int64 `json:"quota_used"`
	QuotaLimit   int64 `json:"quota_limit"`
	QuotaResetAt time.Time `json:"quota_reset_at"`
}

// Status represents account status
type Status string

const (
	StatusActive   Status = "active"
	StatusExpired  Status = "expired"
	StatusBanned   Status = "banned"
	StatusUnknown  Status = "unknown"
	StatusChecking Status = "checking"
)

// AccountInput represents input for creating/updating account
type AccountInput struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email"`
	RefreshToken string `json:"refresh_token" binding:"required"`
	AccountType  string `json:"account_type"`
}

// AccountExport represents exportable account data
type AccountExport struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	RefreshToken string `json:"refresh_token"`
	AccountType  string `json:"account_type"`
}

// QuotaInfo represents quota information
type QuotaInfo struct {
	Model       string    `json:"model"`
	Used        int64     `json:"used"`
	Limit       int64     `json:"limit"`
	Remaining   int64     `json:"remaining"`
	Percentage  float64   `json:"percentage"`
	ResetAt     time.Time `json:"reset_at"`
}
