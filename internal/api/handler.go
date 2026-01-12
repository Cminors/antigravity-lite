package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"antigravity-lite/config"
	"antigravity-lite/internal/account"
	"antigravity-lite/internal/quota"
	"antigravity-lite/internal/router"

	"github.com/gin-gonic/gin"
)

// Handler handles management API requests
type Handler struct {
	accountMgr *account.Manager
	router     *router.Router
	tracker    *quota.Tracker
	cfg        *config.Config
}

// NewHandler creates a new API handler
func NewHandler(accountMgr *account.Manager, rt *router.Router, tracker *quota.Tracker, cfg *config.Config) *Handler {
	return &Handler{
		accountMgr: accountMgr,
		router:     rt,
		tracker:    tracker,
		cfg:        cfg,
	}
}

// ListAccounts returns all accounts
func (h *Handler) ListAccounts(c *gin.Context) {
	accounts, err := h.accountMgr.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, accounts)
}

// CreateAccount creates a new account
func (h *Handler) CreateAccount(c *gin.Context) {
	var input account.AccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	acc, err := h.accountMgr.Create(input)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, acc)
}

// GetAccount returns an account by ID
func (h *Handler) GetAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	acc, err := h.accountMgr.Get(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "account not found"})
		return
	}

	c.JSON(200, acc)
}

// UpdateAccount updates an account
func (h *Handler) UpdateAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	var input account.AccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	acc, err := h.accountMgr.Update(id, input)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, acc)
}

// DeleteAccount deletes an account
func (h *Handler) DeleteAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	if err := h.accountMgr.Delete(id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(204, nil)
}

// CheckAccount checks account status
func (h *Handler) CheckAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	acc, err := h.accountMgr.CheckAccountStatus(id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, acc)
}

// CheckAllAccounts checks all accounts
func (h *Handler) CheckAllAccounts(c *gin.Context) {
	if err := h.accountMgr.CheckAllAccounts(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	accounts, _ := h.accountMgr.List()
	c.JSON(200, accounts)
}

// ImportAccounts imports accounts from JSON
func (h *Handler) ImportAccounts(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	count, err := h.accountMgr.Import(body)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"imported": count})
}

// ExportAccounts exports all accounts
func (h *Handler) ExportAccounts(c *gin.Context) {
	data, err := h.accountMgr.Export()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=accounts.json")
	c.Writer.Write(data)
}

// GetRoutes returns model routes
func (h *Handler) GetRoutes(c *gin.Context) {
	routes := h.router.GetRoutes()
	c.JSON(200, routes)
}

// UpdateRoutes updates model routes
func (h *Handler) UpdateRoutes(c *gin.Context) {
	var routes map[string]string
	if err := c.ShouldBindJSON(&routes); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	h.router.SetRoutes(routes)

	// Update config
	h.cfg.Routes = nil
	for pattern, target := range routes {
		h.cfg.Routes = append(h.cfg.Routes, config.RouteConfig{
			Pattern: pattern,
			Target:  target,
		})
	}

	c.JSON(200, routes)
}

// GetStats returns usage statistics
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.tracker.GetStats()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, stats)
}

// GetModelStats returns per-model statistics
func (h *Handler) GetModelStats(c *gin.Context) {
	stats, err := h.tracker.GetModelStats()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, stats)
}

// GetAccountStats returns per-account statistics
func (h *Handler) GetAccountStats(c *gin.Context) {
	stats, err := h.tracker.GetAccountStats()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, stats)
}

// GetHourlyStats returns hourly statistics
func (h *Handler) GetHourlyStats(c *gin.Context) {
	stats, err := h.tracker.GetHourlyStats()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, stats)
}

// GetRecentLogs returns recent request logs
func (h *Handler) GetRecentLogs(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	logs, err := h.tracker.GetRecentRequests(limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, logs)
}

// GetConfig returns current configuration
func (h *Handler) GetConfig(c *gin.Context) {
	c.JSON(200, h.cfg)
}

// UpdateConfig updates configuration
func (h *Handler) UpdateConfig(c *gin.Context) {
	var newCfg config.Config
	if err := c.ShouldBindJSON(&newCfg); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Update relevant fields
	h.cfg.Server.LogLevel = newCfg.Server.LogLevel
	h.cfg.Proxy.Timeout = newCfg.Proxy.Timeout
	h.cfg.Proxy.MaxRetries = newCfg.Proxy.MaxRetries
	h.cfg.Proxy.AutoRotate = newCfg.Proxy.AutoRotate

	c.JSON(200, h.cfg)
}

// Dashboard returns dashboard data
func (h *Handler) Dashboard(c *gin.Context) {
	accounts, _ := h.accountMgr.List()
	stats, _ := h.tracker.GetStats()
	modelStats, _ := h.tracker.GetModelStats()
	hourlyStats, _ := h.tracker.GetHourlyStats()

	activeCount := 0
	for _, acc := range accounts {
		if acc.Status == account.StatusActive {
			activeCount++
		}
	}

	c.JSON(200, gin.H{
		"accounts": gin.H{
			"total":  len(accounts),
			"active": activeCount,
		},
		"stats":        stats,
		"model_stats":  modelStats,
		"hourly_stats": hourlyStats,
	})
}

// Health returns health status
func (h *Handler) Health(c *gin.Context) {
	accounts, _ := h.accountMgr.List()
	activeCount := 0
	for _, acc := range accounts {
		if acc.Status == account.StatusActive {
			activeCount++
		}
	}

	status := "healthy"
	if activeCount == 0 && len(accounts) > 0 {
		status = "degraded"
	} else if len(accounts) == 0 {
		status = "no_accounts"
	}

	c.JSON(200, gin.H{
		"status":          status,
		"total_accounts":  len(accounts),
		"active_accounts": activeCount,
		"uptime":          "ok",
	})
}

// Helper to parse JSON body
func parseJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
