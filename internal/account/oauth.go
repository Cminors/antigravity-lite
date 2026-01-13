package account

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuth configuration - same as Antigravity-Manager
const (
	OAuthClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	OAuthClientSecret = "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"
	OAuthAuthURL      = "https://accounts.google.com/o/oauth2/v2/auth"
	OAuthTokenURL     = "https://oauth2.googleapis.com/token"
	OAuthUserInfoURL  = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// OAuthScopes required for Google AI Studio
var OAuthScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/cclog",
	"https://www.googleapis.com/auth/experimentsandconfigs",
}

// OAuthHandler handles OAuth authorization flow
type OAuthHandler struct {
	manager     *Manager
	callbackURL string
	server      *http.Server
	serverMu    sync.Mutex
	resultChan  chan *OAuthResult
}

// OAuthResult represents the result of OAuth flow
type OAuthResult struct {
	Account *Account
	Error   error
}

// TokenResponse from Google OAuth
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// UserInfo from Google
type UserInfo struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(manager *Manager) *OAuthHandler {
	return &OAuthHandler{
		manager:    manager,
		resultChan: make(chan *OAuthResult, 1),
	}
}

// GetAuthURL generates the OAuth authorization URL
func (h *OAuthHandler) GetAuthURL(redirectURI string) string {
	h.callbackURL = redirectURI

	params := url.Values{}
	params.Set("client_id", OAuthClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(OAuthScopes, " "))
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	params.Set("include_granted_scopes", "true")

	return OAuthAuthURL + "?" + params.Encode()
}

// ExchangeCode exchanges authorization code for tokens
func (h *OAuthHandler) ExchangeCode(code string, redirectURI string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", OAuthClientID)
	data.Set("client_secret", OAuthClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.Post(
		OAuthTokenURL,
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response failed: %w", err)
	}

	if tokenResp.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh_token returned (account may have already authorized this app)")
	}

	return &tokenResp, nil
}

// GetUserInfo fetches user information using access token
func (h *OAuthHandler) GetUserInfo(accessToken string) (*UserInfo, error) {
	req, _ := http.NewRequest("GET", OAuthUserInfoURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get userinfo failed: %s", string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("parse userinfo failed: %w", err)
	}

	return &userInfo, nil
}

// ProcessCallback processes OAuth callback and creates account
func (h *OAuthHandler) ProcessCallback(code string, redirectURI string) (*Account, error) {
	// Exchange code for tokens
	tokenResp, err := h.ExchangeCode(code, redirectURI)
	if err != nil {
		return nil, err
	}

	// Get user info
	userInfo, err := h.GetUserInfo(tokenResp.AccessToken)
	if err != nil {
		return nil, err
	}

	// Create account
	input := AccountInput{
		Name:         userInfo.Name,
		Email:        userInfo.Email,
		RefreshToken: tokenResp.RefreshToken,
		AccountType:  "free", // Will be determined by quota check later
	}

	account, err := h.manager.Create(input)
	if err != nil {
		return nil, fmt.Errorf("create account failed: %w", err)
	}

	// Update access token and expiry
	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	_ = h.manager.storage.UpdateToken(account.ID, tokenResp.AccessToken, expiry)

	account.AccessToken = tokenResp.AccessToken
	account.TokenExpiry = expiry
	account.Status = StatusActive

	_ = h.manager.storage.UpdateStatus(account.ID, StatusActive)

	return account, nil
}

// StartCallbackServer starts a temporary HTTP server for OAuth callback
func (h *OAuthHandler) StartCallbackServer(port int) (string, error) {
	h.serverMu.Lock()
	defer h.serverMu.Unlock()

	if h.server != nil {
		return "", fmt.Errorf("callback server already running")
	}

	mux := http.NewServeMux()

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/oauth/callback", port)

	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			return
		}

		account, err := h.ProcessCallback(code, callbackURL)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Authorization Failed</title></head>
<body style="text-align:center;padding:50px;font-family:sans-serif;">
<h1 style="color:red;">❌ Authorization Failed</h1>
<p>%s</p>
<p>Please close this window and try again.</p>
</body></html>`, err.Error())
			h.resultChan <- &OAuthResult{Error: err}
			return
		}

		// Success response
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Authorization Successful</title></head>
<body style="text-align:center;padding:50px;font-family:sans-serif;">
<h1 style="color:green;">✅ Authorization Successful!</h1>
<p>Account <strong>%s</strong> has been added.</p>
<p>You can now close this window.</p>
<script>setTimeout(function(){window.close();}, 3000);</script>
</body></html>`, account.Email)

		h.resultChan <- &OAuthResult{Account: account}

		// Stop server after successful callback
		go func() {
			time.Sleep(time.Second)
			h.StopCallbackServer()
		}()
	})

	h.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := h.server.ListenAndServe(); err != http.ErrServerClosed {
			h.resultChan <- &OAuthResult{Error: err}
		}
	}()

	return callbackURL, nil
}

// StopCallbackServer stops the callback server
func (h *OAuthHandler) StopCallbackServer() {
	h.serverMu.Lock()
	defer h.serverMu.Unlock()

	if h.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.server.Shutdown(ctx)
		h.server = nil
	}
}

// WaitForCallback waits for OAuth callback result with timeout
func (h *OAuthHandler) WaitForCallback(timeout time.Duration) (*OAuthResult, error) {
	select {
	case result := <-h.resultChan:
		return result, nil
	case <-time.After(timeout):
		h.StopCallbackServer()
		return nil, fmt.Errorf("OAuth callback timeout")
	}
}
