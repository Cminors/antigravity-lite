package account

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Storage handles account persistence
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new storage instance
func NewStorage(dbPath string) (*Storage, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}

	return s, nil
}

// migrate creates necessary tables
func (s *Storage) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT,
		refresh_token TEXT NOT NULL,
		access_token TEXT,
		token_expiry DATETIME,
		status TEXT DEFAULT 'unknown',
		account_type TEXT DEFAULT 'free',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME,
		quota_used INTEGER DEFAULT 0,
		quota_limit INTEGER DEFAULT 0,
		quota_reset_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS request_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER,
		model TEXT,
		tokens_in INTEGER,
		tokens_out INTEGER,
		latency_ms INTEGER,
		status_code INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (account_id) REFERENCES accounts(id)
	);

	CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
	CREATE INDEX IF NOT EXISTS idx_request_logs_account ON request_logs(account_id);
	CREATE INDEX IF NOT EXISTS idx_request_logs_created ON request_logs(created_at);
	`
	_, err := s.db.Exec(query)
	return err
}

// List returns all accounts
func (s *Storage) List() ([]Account, error) {
	rows, err := s.db.Query(`
		SELECT id, name, email, refresh_token, access_token, token_expiry,
		       status, account_type, created_at, updated_at, last_used_at,
		       quota_used, quota_limit, quota_reset_at
		FROM accounts ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		var tokenExpiry, lastUsedAt, quotaResetAt sql.NullTime
		var accessToken sql.NullString

		err := rows.Scan(
			&a.ID, &a.Name, &a.Email, &a.RefreshToken, &accessToken, &tokenExpiry,
			&a.Status, &a.AccountType, &a.CreatedAt, &a.UpdatedAt, &lastUsedAt,
			&a.QuotaUsed, &a.QuotaLimit, &quotaResetAt,
		)
		if err != nil {
			return nil, err
		}

		if accessToken.Valid {
			a.AccessToken = accessToken.String
		}
		if tokenExpiry.Valid {
			a.TokenExpiry = tokenExpiry.Time
		}
		if lastUsedAt.Valid {
			a.LastUsedAt = lastUsedAt.Time
		}
		if quotaResetAt.Valid {
			a.QuotaResetAt = quotaResetAt.Time
		}

		accounts = append(accounts, a)
	}

	return accounts, nil
}

// Get returns an account by ID
func (s *Storage) Get(id int64) (*Account, error) {
	var a Account
	var tokenExpiry, lastUsedAt, quotaResetAt sql.NullTime
	var accessToken sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, email, refresh_token, access_token, token_expiry,
		       status, account_type, created_at, updated_at, last_used_at,
		       quota_used, quota_limit, quota_reset_at
		FROM accounts WHERE id = ?
	`, id).Scan(
		&a.ID, &a.Name, &a.Email, &a.RefreshToken, &accessToken, &tokenExpiry,
		&a.Status, &a.AccountType, &a.CreatedAt, &a.UpdatedAt, &lastUsedAt,
		&a.QuotaUsed, &a.QuotaLimit, &quotaResetAt,
	)
	if err != nil {
		return nil, err
	}

	if accessToken.Valid {
		a.AccessToken = accessToken.String
	}
	if tokenExpiry.Valid {
		a.TokenExpiry = tokenExpiry.Time
	}
	if lastUsedAt.Valid {
		a.LastUsedAt = lastUsedAt.Time
	}
	if quotaResetAt.Valid {
		a.QuotaResetAt = quotaResetAt.Time
	}

	return &a, nil
}

// Create creates a new account
func (s *Storage) Create(input AccountInput) (*Account, error) {
	result, err := s.db.Exec(`
		INSERT INTO accounts (name, email, refresh_token, account_type, status)
		VALUES (?, ?, ?, ?, ?)
	`, input.Name, input.Email, input.RefreshToken, input.AccountType, StatusUnknown)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return s.Get(id)
}

// Update updates an account
func (s *Storage) Update(id int64, input AccountInput) (*Account, error) {
	_, err := s.db.Exec(`
		UPDATE accounts 
		SET name = ?, email = ?, refresh_token = ?, account_type = ?, updated_at = ?
		WHERE id = ?
	`, input.Name, input.Email, input.RefreshToken, input.AccountType, time.Now(), id)
	if err != nil {
		return nil, err
	}

	return s.Get(id)
}

// Delete deletes an account
func (s *Storage) Delete(id int64) error {
	_, err := s.db.Exec("DELETE FROM accounts WHERE id = ?", id)
	return err
}

// UpdateStatus updates account status
func (s *Storage) UpdateStatus(id int64, status Status) error {
	_, err := s.db.Exec(`
		UPDATE accounts SET status = ?, updated_at = ? WHERE id = ?
	`, status, time.Now(), id)
	return err
}

// UpdateToken updates access token
func (s *Storage) UpdateToken(id int64, accessToken string, expiry time.Time) error {
	_, err := s.db.Exec(`
		UPDATE accounts SET access_token = ?, token_expiry = ?, updated_at = ? WHERE id = ?
	`, accessToken, expiry, time.Now(), id)
	return err
}

// UpdateLastUsed updates last used time
func (s *Storage) UpdateLastUsed(id int64) error {
	_, err := s.db.Exec(`
		UPDATE accounts SET last_used_at = ? WHERE id = ?
	`, time.Now(), id)
	return err
}

// UpdateQuota updates quota info
func (s *Storage) UpdateQuota(id int64, used, limit int64, resetAt time.Time) error {
	_, err := s.db.Exec(`
		UPDATE accounts SET quota_used = ?, quota_limit = ?, quota_reset_at = ?, updated_at = ? WHERE id = ?
	`, used, limit, resetAt, time.Now(), id)
	return err
}

// UpdateAccountType updates account subscription type
func (s *Storage) UpdateAccountType(id int64, accountType string) error {
	_, err := s.db.Exec(`
		UPDATE accounts SET account_type = ?, updated_at = ? WHERE id = ?
	`, accountType, time.Now(), id)
	return err
}

// GetActiveAccounts returns accounts with active status
// Sorted by: 1. Account type (ultra > pro > free)
//  2. Remaining quota (higher first)
//  3. Least recently used
func (s *Storage) GetActiveAccounts() ([]Account, error) {
	rows, err := s.db.Query(`
		SELECT id, name, email, refresh_token, access_token, token_expiry,
		       status, account_type, created_at, updated_at, last_used_at,
		       quota_used, quota_limit, quota_reset_at
		FROM accounts 
		WHERE status = ?
		ORDER BY 
			CASE account_type 
				WHEN 'ultra' THEN 1 
				WHEN 'pro' THEN 2 
				WHEN 'free' THEN 3 
				ELSE 4 
			END ASC,
			(quota_limit - quota_used) DESC,
			last_used_at ASC NULLS FIRST
	`, StatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		var tokenExpiry, lastUsedAt, quotaResetAt sql.NullTime
		var accessToken sql.NullString

		err := rows.Scan(
			&a.ID, &a.Name, &a.Email, &a.RefreshToken, &accessToken, &tokenExpiry,
			&a.Status, &a.AccountType, &a.CreatedAt, &a.UpdatedAt, &lastUsedAt,
			&a.QuotaUsed, &a.QuotaLimit, &quotaResetAt,
		)
		if err != nil {
			return nil, err
		}

		if accessToken.Valid {
			a.AccessToken = accessToken.String
		}
		if tokenExpiry.Valid {
			a.TokenExpiry = tokenExpiry.Time
		}
		if lastUsedAt.Valid {
			a.LastUsedAt = lastUsedAt.Time
		}
		if quotaResetAt.Valid {
			a.QuotaResetAt = quotaResetAt.Time
		}

		accounts = append(accounts, a)
	}

	return accounts, nil
}

// LogRequest logs a request
func (s *Storage) LogRequest(accountID int64, model string, tokensIn, tokensOut, latencyMs, statusCode int) error {
	_, err := s.db.Exec(`
		INSERT INTO request_logs (account_id, model, tokens_in, tokens_out, latency_ms, status_code)
		VALUES (?, ?, ?, ?, ?, ?)
	`, accountID, model, tokensIn, tokensOut, latencyMs, statusCode)
	return err
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection
func (s *Storage) DB() *sql.DB {
	return s.db
}
