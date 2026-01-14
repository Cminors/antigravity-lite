package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"antigravity-lite/config"
	"antigravity-lite/internal/account"
	"antigravity-lite/internal/api"
	"antigravity-lite/internal/proxy"
	"antigravity-lite/internal/quota"
	"antigravity-lite/internal/router"

	"github.com/gin-gonic/gin"
)

//go:embed web/*
var webFS embed.FS

func main() {
	// Determine config path
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		execPath, _ := os.Executable()
		configPath = filepath.Join(filepath.Dir(execPath), "config.yaml")
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("Warning: Could not load config: %v, using defaults", err)
		cfg = config.DefaultConfig()
	}

	// Initialize storage
	dbPath := cfg.Storage.DBPath
	if !filepath.IsAbs(dbPath) {
		execPath, _ := os.Executable()
		dbPath = filepath.Join(filepath.Dir(execPath), dbPath)
	}

	storage, err := account.NewStorage(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Initialize components
	accountMgr := account.NewManager(storage)
	modelRouter := router.NewRouter(cfg)
	quotaTracker := quota.NewTracker(storage.DB())
	proxyHandler := proxy.NewHandler(accountMgr, modelRouter, cfg)
	apiHandler := api.NewHandler(accountMgr, modelRouter, quotaTracker, cfg, configPath)

	// Setup Gin
	if cfg.Server.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API Proxy endpoints (OpenAI compatible)
	r.POST("/v1/chat/completions", proxyHandler.HandleChatCompletions)
	r.GET("/v1/models", proxyHandler.HandleModels)

	// API Proxy endpoints (Gemini compatible)
	r.GET("/v1beta/models", proxyHandler.HandleGeminiModels)

	// API Proxy endpoints (Anthropic compatible)
	r.POST("/v1/messages", proxyHandler.HandleAnthropicMessages)

	// Health check
	r.GET("/health", apiHandler.Health)

	// Management API
	apiGroup := r.Group("/api")
	{
		// Dashboard
		apiGroup.GET("/dashboard", apiHandler.Dashboard)

		// Accounts
		apiGroup.GET("/accounts", apiHandler.ListAccounts)
		apiGroup.POST("/accounts", apiHandler.CreateAccount)
		apiGroup.GET("/accounts/:id", apiHandler.GetAccount)
		apiGroup.PUT("/accounts/:id", apiHandler.UpdateAccount)
		apiGroup.DELETE("/accounts/:id", apiHandler.DeleteAccount)
		apiGroup.POST("/accounts/:id/check", apiHandler.CheckAccount)
		apiGroup.POST("/accounts/check-all", apiHandler.CheckAllAccounts)
		apiGroup.POST("/accounts/import", apiHandler.ImportAccounts)
		apiGroup.GET("/accounts/export", apiHandler.ExportAccounts)
		apiGroup.POST("/accounts/:id/quota", apiHandler.RefreshQuota)
		apiGroup.POST("/accounts/refresh-quotas", apiHandler.RefreshAllQuotas)

		// Routes
		apiGroup.GET("/routes", apiHandler.GetRoutes)
		apiGroup.PUT("/routes", apiHandler.UpdateRoutes)

		// Stats
		apiGroup.GET("/stats", apiHandler.GetStats)
		apiGroup.GET("/stats/models", apiHandler.GetModelStats)
		apiGroup.GET("/stats/accounts", apiHandler.GetAccountStats)
		apiGroup.GET("/stats/hourly", apiHandler.GetHourlyStats)
		apiGroup.GET("/logs", apiHandler.GetRecentLogs)

		// Config
		apiGroup.GET("/config", apiHandler.GetConfig)
		apiGroup.PUT("/config", apiHandler.UpdateConfig)

		// OAuth
		apiGroup.GET("/oauth/start", apiHandler.StartOAuth)
		apiGroup.GET("/oauth/callback", apiHandler.OAuthCallback)
	}

	// Serve embedded web UI
	webContent, _ := fs.Sub(webFS, "web")
	fileServer := http.FileServer(http.FS(webContent))

	r.NoRoute(func(c *gin.Context) {
		// Try to serve static files
		path := c.Request.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists
		file, err := webContent.Open(path[1:])
		if err != nil {
			// Serve index.html for SPA routing
			c.Request.URL.Path = "/index.html"
		} else {
			file.Close()
		}

		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("🚀 Antigravity Lite starting on http://%s", addr)
	log.Printf("📊 Dashboard: http://%s/", addr)
	log.Printf("🔌 OpenAI API: http://%s/v1/chat/completions", addr)
	log.Printf("🔌 Anthropic API: http://%s/v1/messages", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
