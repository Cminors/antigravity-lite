package quota

import (
	"database/sql"
	"time"
)

// Tracker tracks quota and usage statistics
type Tracker struct {
	db *sql.DB
}

// NewTracker creates a new quota tracker
func NewTracker(db *sql.DB) *Tracker {
	return &Tracker{db: db}
}

// Stats represents usage statistics
type Stats struct {
	TotalRequests     int64   `json:"total_requests"`
	TotalTokensIn     int64   `json:"total_tokens_in"`
	TotalTokensOut    int64   `json:"total_tokens_out"`
	AvgLatencyMs      float64 `json:"avg_latency_ms"`
	SuccessRate       float64 `json:"success_rate"`
	RequestsToday     int64   `json:"requests_today"`
	RequestsThisWeek  int64   `json:"requests_this_week"`
	RequestsThisMonth int64   `json:"requests_this_month"`
}

// ModelStats represents per-model statistics
type ModelStats struct {
	Model        string  `json:"model"`
	Requests     int64   `json:"requests"`
	TokensIn     int64   `json:"tokens_in"`
	TokensOut    int64   `json:"tokens_out"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

// AccountStats represents per-account statistics
type AccountStats struct {
	AccountID   int64   `json:"account_id"`
	AccountName string  `json:"account_name"`
	Requests    int64   `json:"requests"`
	TokensIn    int64   `json:"tokens_in"`
	TokensOut   int64   `json:"tokens_out"`
	SuccessRate float64 `json:"success_rate"`
}

// GetStats returns overall usage statistics
func (t *Tracker) GetStats() (*Stats, error) {
	var stats Stats

	// Total requests
	err := t.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(tokens_in), 0), COALESCE(SUM(tokens_out), 0), 
		       COALESCE(AVG(latency_ms), 0)
		FROM request_logs
	`).Scan(&stats.TotalRequests, &stats.TotalTokensIn, &stats.TotalTokensOut, &stats.AvgLatencyMs)
	if err != nil {
		return nil, err
	}

	// Success rate
	var successCount int64
	_ = t.db.QueryRow(`
		SELECT COUNT(*) FROM request_logs WHERE status_code = 200
	`).Scan(&successCount)
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalRequests) * 100
	}

	// Today's requests
	today := time.Now().Truncate(24 * time.Hour)
	_ = t.db.QueryRow(`
		SELECT COUNT(*) FROM request_logs WHERE created_at >= ?
	`, today).Scan(&stats.RequestsToday)

	// This week's requests
	weekAgo := time.Now().AddDate(0, 0, -7)
	_ = t.db.QueryRow(`
		SELECT COUNT(*) FROM request_logs WHERE created_at >= ?
	`, weekAgo).Scan(&stats.RequestsThisWeek)

	// This month's requests
	monthAgo := time.Now().AddDate(0, -1, 0)
	_ = t.db.QueryRow(`
		SELECT COUNT(*) FROM request_logs WHERE created_at >= ?
	`, monthAgo).Scan(&stats.RequestsThisMonth)

	return &stats, nil
}

// GetModelStats returns per-model statistics
func (t *Tracker) GetModelStats() ([]ModelStats, error) {
	rows, err := t.db.Query(`
		SELECT model, COUNT(*), COALESCE(SUM(tokens_in), 0), 
		       COALESCE(SUM(tokens_out), 0), COALESCE(AVG(latency_ms), 0)
		FROM request_logs
		GROUP BY model
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ModelStats
	for rows.Next() {
		var s ModelStats
		if err := rows.Scan(&s.Model, &s.Requests, &s.TokensIn, &s.TokensOut, &s.AvgLatencyMs); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// GetAccountStats returns per-account statistics
func (t *Tracker) GetAccountStats() ([]AccountStats, error) {
	rows, err := t.db.Query(`
		SELECT r.account_id, a.name, COUNT(*), COALESCE(SUM(r.tokens_in), 0),
		       COALESCE(SUM(r.tokens_out), 0),
		       CAST(SUM(CASE WHEN r.status_code = 200 THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*) * 100
		FROM request_logs r
		JOIN accounts a ON r.account_id = a.id
		GROUP BY r.account_id
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []AccountStats
	for rows.Next() {
		var s AccountStats
		if err := rows.Scan(&s.AccountID, &s.AccountName, &s.Requests, &s.TokensIn, &s.TokensOut, &s.SuccessRate); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// GetHourlyStats returns hourly request counts for the last 24 hours
func (t *Tracker) GetHourlyStats() ([]map[string]interface{}, error) {
	rows, err := t.db.Query(`
		SELECT strftime('%Y-%m-%d %H:00', created_at) as hour, COUNT(*)
		FROM request_logs
		WHERE created_at >= datetime('now', '-24 hours')
		GROUP BY hour
		ORDER BY hour
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var hour string
		var count int64
		if err := rows.Scan(&hour, &count); err != nil {
			return nil, err
		}
		stats = append(stats, map[string]interface{}{
			"hour":     hour,
			"requests": count,
		})
	}

	return stats, nil
}

// GetRecentRequests returns recent request logs
func (t *Tracker) GetRecentRequests(limit int) ([]map[string]interface{}, error) {
	rows, err := t.db.Query(`
		SELECT r.id, r.account_id, a.name, r.model, r.tokens_in, r.tokens_out, 
		       r.latency_ms, r.status_code, r.created_at
		FROM request_logs r
		JOIN accounts a ON r.account_id = a.id
		ORDER BY r.created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var (
			id, accountID, tokensIn, tokensOut, latencyMs, statusCode int64
			accountName, model                                        string
			createdAt                                                 time.Time
		)
		if err := rows.Scan(&id, &accountID, &accountName, &model, &tokensIn, &tokensOut, &latencyMs, &statusCode, &createdAt); err != nil {
			return nil, err
		}
		logs = append(logs, map[string]interface{}{
			"id":           id,
			"account_id":   accountID,
			"account_name": accountName,
			"model":        model,
			"tokens_in":    tokensIn,
			"tokens_out":   tokensOut,
			"latency_ms":   latencyMs,
			"status_code":  statusCode,
			"created_at":   createdAt.Format(time.RFC3339),
		})
	}

	return logs, nil
}
