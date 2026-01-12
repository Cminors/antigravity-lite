package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"antigravity-lite/internal/account"

	"github.com/gin-gonic/gin"
)

// AnthropicRequest represents Anthropic API request
type AnthropicRequest struct {
	Model       string                   `json:"model"`
	Messages    []map[string]interface{} `json:"messages"`
	System      string                   `json:"system,omitempty"`
	MaxTokens   int                      `json:"max_tokens"`
	Stream      bool                     `json:"stream"`
	Temperature float64                  `json:"temperature,omitempty"`
}

// AnthropicResponse represents Anthropic API response
type AnthropicResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Content      []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// HandleAnthropicMessages handles Anthropic-style messages endpoint
func (h *Handler) HandleAnthropicMessages(c *gin.Context) {
	var req AnthropicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"type": "error", "error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return
	}

	// Route model
	targetModel := h.router.Route(req.Model)

	// Check for background request
	if h.router.IsBackgroundRequest(req.Messages) {
		targetModel = h.router.GetLightModel()
	}

	// Get account
	acct, err := h.accountMgr.GetNextActive()
	if err != nil {
		c.JSON(503, gin.H{"type": "error", "error": gin.H{"type": "overloaded_error", "message": "no available accounts"}})
		return
	}

	// Ensure valid token
	if err := h.accountMgr.EnsureValidToken(acct); err != nil {
		c.JSON(503, gin.H{"type": "error", "error": gin.H{"type": "authentication_error", "message": "token refresh failed"}})
		return
	}

	if req.Stream {
		h.handleAnthropicStream(c, acct, targetModel, req)
	} else {
		h.handleAnthropicNonStream(c, acct, targetModel, req)
	}
}

// handleAnthropicNonStream handles non-streaming Anthropic requests
func (h *Handler) handleAnthropicNonStream(c *gin.Context, acct *account.Account, model string, req AnthropicRequest) {
	start := time.Now()

	resp, statusCode, err := h.callGeminiForAnthropic(acct, model, req)
	if err != nil {
		// Retry with rotation
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
				resp, statusCode, err = h.callGeminiForAnthropic(acct, model, req)
				if err == nil {
					break
				}
				h.accountMgr.MarkAccountError(acct.ID, statusCode)
			}
		}

		if err != nil {
			c.JSON(statusCode, gin.H{"type": "error", "error": gin.H{"type": "api_error", "message": err.Error()}})
			return
		}
	}

	latency := time.Since(start).Milliseconds()

	// Log request
	_ = h.accountMgr.GetStorage().LogRequest(
		acct.ID, model,
		resp.Usage.InputTokens, resp.Usage.OutputTokens,
		int(latency), statusCode,
	)

	c.JSON(200, resp)
}

// callGeminiForAnthropic calls Gemini API and converts to Anthropic format
func (h *Handler) callGeminiForAnthropic(acct *account.Account, model string, req AnthropicRequest) (*AnthropicResponse, int, error) {
	// Convert to OpenAI format first, then to Gemini
	messages := req.Messages
	if req.System != "" {
		messages = append([]map[string]interface{}{
			{"role": "system", "content": req.System},
		}, messages...)
	}

	geminiReq := h.convertToGeminiRequest(messages, req.Temperature, req.MaxTokens)

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
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, 500, err
	}

	// Convert to Anthropic format
	content := ""
	stopReason := "end_turn"
	if len(geminiResp.Candidates) > 0 {
		cand := geminiResp.Candidates[0]
		if len(cand.Content.Parts) > 0 {
			content = cand.Content.Parts[0].Text
		}
		switch cand.FinishReason {
		case "MAX_TOKENS":
			stopReason = "max_tokens"
		case "STOP":
			stopReason = "end_turn"
		}
	}

	return &AnthropicResponse{
		ID:   fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Type: "message",
		Role: "assistant",
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			{Type: "text", Text: content},
		},
		Model:      model,
		StopReason: stopReason,
		Usage: struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		}{
			InputTokens:  geminiResp.UsageMetadata.PromptTokenCount,
			OutputTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
		},
	}, 200, nil
}

// handleAnthropicStream handles streaming Anthropic requests
func (h *Handler) handleAnthropicStream(c *gin.Context, acct *account.Account, model string, req AnthropicRequest) {
	// Convert to OpenAI format first, then to Gemini
	messages := req.Messages
	if req.System != "" {
		messages = append([]map[string]interface{}{
			{"role": "system", "content": req.System},
		}, messages...)
	}

	geminiReq := h.convertToGeminiRequest(messages, req.Temperature, req.MaxTokens)

	jsonBody, _ := json.Marshal(geminiReq)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse", model)

	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+acct.AccessToken)

	resp, err := h.client.Do(httpReq)
	if err != nil {
		c.JSON(500, gin.H{"type": "error", "error": gin.H{"type": "api_error", "message": err.Error()}})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{"type": "error", "error": gin.H{"type": "api_error", "message": string(body)}})
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	// Send message_start event
	startEvent := map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":    msgID,
			"type":  "message",
			"role":  "assistant",
			"model": model,
			"usage": map[string]int{"input_tokens": 0, "output_tokens": 0},
		},
	}
	startJSON, _ := json.Marshal(startEvent)
	c.Writer.Write([]byte("event: message_start\ndata: " + string(startJSON) + "\n\n"))
	c.Writer.Flush()

	// Send content_block_start
	blockStartEvent := map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]interface{}{
			"type": "text",
			"text": "",
		},
	}
	blockStartJSON, _ := json.Marshal(blockStartEvent)
	c.Writer.Write([]byte("event: content_block_start\ndata: " + string(blockStartJSON) + "\n\n"))
	c.Writer.Flush()

	// Stream response
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "" {
			continue
		}

		var chunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			text := chunk.Candidates[0].Content.Parts[0].Text

			// Send content_block_delta
			deltaEvent := map[string]interface{}{
				"type":  "content_block_delta",
				"index": 0,
				"delta": map[string]interface{}{
					"type": "text_delta",
					"text": text,
				},
			}
			deltaJSON, _ := json.Marshal(deltaEvent)
			c.Writer.Write([]byte("event: content_block_delta\ndata: " + string(deltaJSON) + "\n\n"))
			c.Writer.Flush()
		}
	}

	// Send content_block_stop
	blockStopEvent := map[string]interface{}{
		"type":  "content_block_stop",
		"index": 0,
	}
	blockStopJSON, _ := json.Marshal(blockStopEvent)
	c.Writer.Write([]byte("event: content_block_stop\ndata: " + string(blockStopJSON) + "\n\n"))
	c.Writer.Flush()

	// Send message_delta
	msgDeltaEvent := map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason": "end_turn",
		},
		"usage": map[string]int{"output_tokens": 0},
	}
	msgDeltaJSON, _ := json.Marshal(msgDeltaEvent)
	c.Writer.Write([]byte("event: message_delta\ndata: " + string(msgDeltaJSON) + "\n\n"))
	c.Writer.Flush()

	// Send message_stop
	msgStopEvent := map[string]interface{}{
		"type": "message_stop",
	}
	msgStopJSON, _ := json.Marshal(msgStopEvent)
	c.Writer.Write([]byte("event: message_stop\ndata: " + string(msgStopJSON) + "\n\n"))
	c.Writer.Flush()
}
