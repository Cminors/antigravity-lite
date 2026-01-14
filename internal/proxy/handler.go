package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"antigravity-lite/config"
	"antigravity-lite/internal/account"
	"antigravity-lite/internal/router"

	"github.com/gin-gonic/gin"
)

// Handler handles proxy requests
type Handler struct {
	accountMgr *account.Manager
	router     *router.Router
	cfg        *config.Config
	client     *http.Client
}

// NewHandler creates a new proxy handler
func NewHandler(accountMgr *account.Manager, rt *router.Router, cfg *config.Config) *Handler {
	return &Handler{
		accountMgr: accountMgr,
		router:     rt,
		cfg:        cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.Proxy.Timeout) * time.Second,
		},
	}
}

// OpenAI request/response types
type ChatCompletionRequest struct {
	Model       string                   `json:"model"`
	Messages    []map[string]interface{} `json:"messages"`
	Stream      bool                     `json:"stream"`
	Temperature float64                  `json:"temperature,omitempty"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int                    `json:"index"`
		Message      map[string]interface{} `json:"message"`
		FinishReason string                 `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// HandleChatCompletions handles OpenAI-style chat completions
func (h *Handler) HandleChatCompletions(c *gin.Context) {
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
		return
	}

	// Route model
	originalModel := req.Model
	targetModel := h.router.Route(originalModel)

	// Check for background request
	if h.router.IsBackgroundRequest(req.Messages) {
		targetModel = h.router.GetLightModel()
	}

	// Get account
	acct, err := h.accountMgr.GetNextActive()
	if err != nil {
		c.JSON(503, gin.H{"error": gin.H{"message": "no available accounts", "type": "service_unavailable"}})
		return
	}

	// Ensure valid token
	if err := h.accountMgr.EnsureValidToken(acct); err != nil {
		c.JSON(503, gin.H{"error": gin.H{"message": "token refresh failed", "type": "authentication_error"}})
		return
	}

	start := time.Now()

	// Convert to Gemini format and call API
	resp, statusCode, err := h.callGeminiAPI(acct, targetModel, req)
	if err != nil {
		// Try with another account on error
		if h.cfg.Proxy.AutoRotate && (statusCode == 429 || statusCode == 401 || statusCode == 403) {
			h.accountMgr.MarkAccountError(acct.ID, statusCode)

			for i := 0; i < h.cfg.Proxy.MaxRetries; i++ {
				acct, err = h.accountMgr.GetNextActive()
				if err != nil {
					break
				}
				if err := h.accountMgr.EnsureValidToken(acct); err != nil {
					continue
				}
				resp, statusCode, err = h.callGeminiAPI(acct, targetModel, req)
				if err == nil {
					break
				}
				h.accountMgr.MarkAccountError(acct.ID, statusCode)
			}
		}

		if err != nil {
			c.JSON(statusCode, gin.H{"error": gin.H{"message": err.Error(), "type": "api_error"}})
			return
		}
	}

	latency := time.Since(start).Milliseconds()

	// Log request
	_ = h.accountMgr.GetStorage().LogRequest(
		acct.ID, targetModel,
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens,
		int(latency), statusCode,
	)

	// Return OpenAI format response
	c.JSON(200, resp)
}

// callGeminiAPI calls Gemini API with converted request
func (h *Handler) callGeminiAPI(acct *account.Account, model string, req ChatCompletionRequest) (*ChatCompletionResponse, int, error) {
	// Convert messages to Gemini format
	geminiReq := h.convertToGeminiRequest(req.Messages, req.Temperature, req.MaxTokens)

	jsonBody, _ := json.Marshal(geminiReq)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)

	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+acct.AccessToken)

	resp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, 500, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, fmt.Errorf("API error: %s", string(body))
	}

	// Parse Gemini response
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, 500, err
	}

	// Convert to OpenAI format
	content := ""
	finishReason := "stop"
	if len(geminiResp.Candidates) > 0 {
		cand := geminiResp.Candidates[0]
		if len(cand.Content.Parts) > 0 {
			content = cand.Content.Parts[0].Text
		}
		if cand.FinishReason != "" {
			finishReason = strings.ToLower(cand.FinishReason)
		}
	}

	return &ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []struct {
			Index        int                    `json:"index"`
			Message      map[string]interface{} `json:"message"`
			FinishReason string                 `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
				FinishReason: finishReason,
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		},
	}, 200, nil
}

// convertToGeminiRequest converts OpenAI messages to Gemini format
func (h *Handler) convertToGeminiRequest(messages []map[string]interface{}, temp float64, maxTokens int) map[string]interface{} {
	contents := make([]map[string]interface{}, 0)
	systemInstruction := ""

	for _, msg := range messages {
		role := msg["role"].(string)
		content := msg["content"]

		if role == "system" {
			// Handle system message
			if c, ok := content.(string); ok {
				systemInstruction = c
			}
			continue
		}

		geminiRole := "user"
		if role == "assistant" {
			geminiRole = "model"
		}

		// Handle content (could be string or array for multimodal)
		var parts []map[string]interface{}
		switch c := content.(type) {
		case string:
			parts = []map[string]interface{}{{"text": c}}
		case []interface{}:
			for _, part := range c {
				if p, ok := part.(map[string]interface{}); ok {
					if p["type"] == "text" {
						parts = append(parts, map[string]interface{}{"text": p["text"]})
					} else if p["type"] == "image_url" {
						// Handle image URL
						if imgURL, ok := p["image_url"].(map[string]interface{}); ok {
							if url, ok := imgURL["url"].(string); ok {
								if strings.HasPrefix(url, "data:") {
									// Base64 encoded image
									urlParts := strings.SplitN(url, ",", 2)
									if len(urlParts) == 2 {
										mimeType := strings.TrimPrefix(strings.Split(urlParts[0], ";")[0], "data:")
										parts = append(parts, map[string]interface{}{
											"inline_data": map[string]interface{}{
												"mime_type": mimeType,
												"data":      urlParts[1],
											},
										})
									}
								}
							}
						}
					}
				}
			}
		}

		contents = append(contents, map[string]interface{}{
			"role":  geminiRole,
			"parts": parts,
		})
	}

	req := map[string]interface{}{
		"contents": contents,
	}

	// Add system instruction
	if systemInstruction != "" {
		req["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{{"text": systemInstruction}},
		}
	}

	// Add generation config
	genConfig := map[string]interface{}{}
	if temp > 0 {
		genConfig["temperature"] = temp
	}
	if maxTokens > 0 {
		genConfig["maxOutputTokens"] = maxTokens
	}
	if len(genConfig) > 0 {
		req["generationConfig"] = genConfig
	}

	return req
}

// HandleModels returns available models
func (h *Handler) HandleModels(c *gin.Context) {
	models := []map[string]interface{}{
		// Gemini 3.x (Latest 2025)
		{"id": "gemini-3-pro-high", "object": "model", "owned_by": "google"},
		{"id": "gemini-3-pro", "object": "model", "owned_by": "google"},
		{"id": "gemini-3-flash", "object": "model", "owned_by": "google"},

		// Gemini 2.5.x
		{"id": "gemini-2.5-pro", "object": "model", "owned_by": "google"},
		{"id": "gemini-2.5-flash", "object": "model", "owned_by": "google"},
		{"id": "gemini-2.5-flash-lite", "object": "model", "owned_by": "google"},

		// Gemini 2.0.x
		{"id": "gemini-2.0-flash", "object": "model", "owned_by": "google"},
		{"id": "gemini-2.0-flash-lite", "object": "model", "owned_by": "google"},
		{"id": "gemini-2.0-pro", "object": "model", "owned_by": "google"},

		// Gemini 1.5.x (Legacy)
		{"id": "gemini-1.5-flash", "object": "model", "owned_by": "google"},
		{"id": "gemini-1.5-pro", "object": "model", "owned_by": "google"},

		// Claude 4.x (Latest 2025)
		{"id": "claude-opus-4-5-thinking", "object": "model", "owned_by": "anthropic-alias"},
		{"id": "claude-opus-4-5", "object": "model", "owned_by": "anthropic-alias"},
		{"id": "claude-sonnet-4-5", "object": "model", "owned_by": "anthropic-alias"},
		{"id": "claude-sonnet-4", "object": "model", "owned_by": "anthropic-alias"},

		// Claude 3.x (Legacy aliases)
		{"id": "claude-3-opus", "object": "model", "owned_by": "anthropic-alias"},
		{"id": "claude-3-5-sonnet", "object": "model", "owned_by": "anthropic-alias"},
		{"id": "claude-3-sonnet", "object": "model", "owned_by": "anthropic-alias"},
		{"id": "claude-3-haiku", "object": "model", "owned_by": "anthropic-alias"},

		// OpenAI Aliases (mapped to Gemini)
		{"id": "gpt-4o", "object": "model", "owned_by": "openai-alias"},
		{"id": "gpt-4o-mini", "object": "model", "owned_by": "openai-alias"},
		{"id": "gpt-4-turbo", "object": "model", "owned_by": "openai-alias"},
		{"id": "gpt-4", "object": "model", "owned_by": "openai-alias"},
		{"id": "gpt-3.5-turbo", "object": "model", "owned_by": "openai-alias"},
		{"id": "o1-preview", "object": "model", "owned_by": "openai-alias"},
		{"id": "o1-mini", "object": "model", "owned_by": "openai-alias"},
		{"id": "o3-mini", "object": "model", "owned_by": "openai-alias"},
	}

	c.JSON(200, gin.H{"object": "list", "data": models})
}

// HandleGeminiModels returns models in Gemini API native format
func (h *Handler) HandleGeminiModels(c *gin.Context) {
	geminiModels := []map[string]interface{}{
		{
			"name":                       "models/gemini-3-pro-high",
			"displayName":                "Gemini 3 Pro High",
			"description":                "Most capable Gemini 3 model for complex reasoning",
			"supportedGenerationMethods": []string{"generateContent", "countTokens"},
		},
		{
			"name":                       "models/gemini-3-pro",
			"displayName":                "Gemini 3 Pro",
			"description":                "Balanced Gemini 3 model for general tasks",
			"supportedGenerationMethods": []string{"generateContent", "countTokens"},
		},
		{
			"name":                       "models/gemini-3-flash",
			"displayName":                "Gemini 3 Flash",
			"description":                "Fast Gemini 3 model for quick responses",
			"supportedGenerationMethods": []string{"generateContent", "countTokens"},
		},
		{
			"name":                       "models/gemini-2.5-pro",
			"displayName":                "Gemini 2.5 Pro",
			"description":                "Advanced Gemini 2.5 model",
			"supportedGenerationMethods": []string{"generateContent", "countTokens"},
		},
		{
			"name":                       "models/gemini-2.5-flash",
			"displayName":                "Gemini 2.5 Flash",
			"description":                "Fast Gemini 2.5 model",
			"supportedGenerationMethods": []string{"generateContent", "countTokens"},
		},
		{
			"name":                       "models/gemini-2.5-flash-lite",
			"displayName":                "Gemini 2.5 Flash Lite",
			"description":                "Lightweight Gemini 2.5 model",
			"supportedGenerationMethods": []string{"generateContent", "countTokens"},
		},
		{
			"name":                       "models/gemini-2.0-flash",
			"displayName":                "Gemini 2.0 Flash",
			"description":                "Fast Gemini 2.0 model",
			"supportedGenerationMethods": []string{"generateContent", "countTokens"},
		},
		{
			"name":                       "models/gemini-2.0-pro",
			"displayName":                "Gemini 2.0 Pro",
			"description":                "Advanced Gemini 2.0 model",
			"supportedGenerationMethods": []string{"generateContent", "countTokens"},
		},
		{
			"name":                       "models/gemini-1.5-flash",
			"displayName":                "Gemini 1.5 Flash",
			"description":                "Legacy fast model",
			"supportedGenerationMethods": []string{"generateContent", "countTokens"},
		},
		{
			"name":                       "models/gemini-1.5-pro",
			"displayName":                "Gemini 1.5 Pro",
			"description":                "Legacy advanced model",
			"supportedGenerationMethods": []string{"generateContent", "countTokens"},
		},
	}

	c.JSON(200, gin.H{"models": geminiModels})
}
