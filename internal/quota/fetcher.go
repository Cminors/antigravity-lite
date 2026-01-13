package quota

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	QuotaAPIURL   = "https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels"
	ProjectAPIURL = "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist"
	UserAgent     = "antigravity-lite/1.0 Linux/amd64"
)

// QuotaResponse from Google API
type QuotaResponse struct {
	Models map[string]ModelInfo `json:"models"`
}

// ModelInfo contains quota info for a model
type ModelInfo struct {
	QuotaInfo *QuotaInfo `json:"quotaInfo,omitempty"`
}

// QuotaInfo contains quota details
type QuotaInfo struct {
	RemainingFraction float64 `json:"remainingFraction,omitempty"`
	ResetTime         string  `json:"resetTime,omitempty"`
}

// ProjectResponse from loadCodeAssist API
type ProjectResponse struct {
	ProjectID   string `json:"cloudaicompanionProject,omitempty"`
	CurrentTier *Tier  `json:"currentTier,omitempty"`
	PaidTier    *Tier  `json:"paidTier,omitempty"`
}

// Tier contains subscription tier info
type Tier struct {
	ID        string `json:"id,omitempty"`
	QuotaTier string `json:"quotaTier,omitempty"`
	Name      string `json:"name,omitempty"`
}

// ModelQuota represents quota for a single model
type ModelQuota struct {
	Name       string `json:"name"`
	Percentage int    `json:"percentage"`
	ResetTime  string `json:"reset_time,omitempty"`
}

// AccountQuota represents full quota data for an account
type AccountQuota struct {
	Email            string       `json:"email"`
	ProjectID        string       `json:"project_id,omitempty"`
	SubscriptionTier string       `json:"subscription_tier,omitempty"`
	Models           []ModelQuota `json:"models"`
	IsForbidden      bool         `json:"is_forbidden,omitempty"`
	FetchedAt        time.Time    `json:"fetched_at"`
}

// QuotaFetcher fetches quota from Google API
type QuotaFetcher struct {
	client *http.Client
}

// NewQuotaFetcher creates a new quota fetcher
func NewQuotaFetcher() *QuotaFetcher {
	return &QuotaFetcher{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchProjectInfo fetches project ID and subscription tier
func (f *QuotaFetcher) FetchProjectInfo(accessToken string) (*ProjectResponse, error) {
	body := []byte(`{"metadata":{"ideType":"ANTIGRAVITY"}}`)

	req, err := http.NewRequest("POST", ProjectAPIURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	req.Body = io.NopCloser(bytesReader(body))
	req.ContentLength = int64(len(body))

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("project info request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("project info failed: %d - %s", resp.StatusCode, string(respBody))
	}

	var projectResp ProjectResponse
	if err := json.NewDecoder(resp.Body).Decode(&projectResp); err != nil {
		return nil, fmt.Errorf("parse project info failed: %w", err)
	}

	return &projectResp, nil
}

// FetchQuota fetches quota for an account
func (f *QuotaFetcher) FetchQuota(accessToken, projectID, email string) (*AccountQuota, error) {
	// If no project ID, try to fetch it first
	var subscriptionTier string
	if projectID == "" {
		projectInfo, err := f.FetchProjectInfo(accessToken)
		if err == nil && projectInfo != nil {
			projectID = projectInfo.ProjectID
			// Get subscription tier
			if projectInfo.PaidTier != nil && projectInfo.PaidTier.ID != "" {
				subscriptionTier = projectInfo.PaidTier.ID
			} else if projectInfo.CurrentTier != nil && projectInfo.CurrentTier.ID != "" {
				subscriptionTier = projectInfo.CurrentTier.ID
			}
		}
	}

	// Default project ID if still empty
	if projectID == "" {
		projectID = "bamboo-precept-lgxtn"
	}

	// Fetch quota
	body := []byte(fmt.Sprintf(`{"project":"%s"}`, projectID))

	req, err := http.NewRequest("POST", QuotaAPIURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	req.Body = io.NopCloser(bytesReader(body))
	req.ContentLength = int64(len(body))

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("quota request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle 403 Forbidden
	if resp.StatusCode == 403 {
		return &AccountQuota{
			Email:       email,
			ProjectID:   projectID,
			IsForbidden: true,
			FetchedAt:   time.Now(),
		}, nil
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("quota request failed: %d - %s", resp.StatusCode, string(respBody))
	}

	var quotaResp QuotaResponse
	if err := json.NewDecoder(resp.Body).Decode(&quotaResp); err != nil {
		return nil, fmt.Errorf("parse quota response failed: %w", err)
	}

	// Convert to AccountQuota
	accountQuota := &AccountQuota{
		Email:            email,
		ProjectID:        projectID,
		SubscriptionTier: subscriptionTier,
		Models:           make([]ModelQuota, 0),
		FetchedAt:        time.Now(),
	}

	for name, info := range quotaResp.Models {
		// Only include models we care about
		if !isRelevantModel(name) {
			continue
		}

		percentage := 0
		resetTime := ""
		if info.QuotaInfo != nil {
			percentage = int(info.QuotaInfo.RemainingFraction * 100)
			resetTime = info.QuotaInfo.ResetTime
		}

		accountQuota.Models = append(accountQuota.Models, ModelQuota{
			Name:       name,
			Percentage: percentage,
			ResetTime:  resetTime,
		})
	}

	return accountQuota, nil
}

// isRelevantModel checks if we should include this model
func isRelevantModel(name string) bool {
	// Include all gemini and claude models
	return len(name) > 0 && (name[0] == 'g' || name[0] == 'c')
}

// bytesReader is a simple bytes.Reader wrapper
type bytesReaderWrapper struct {
	data []byte
	pos  int
}

func bytesReader(data []byte) *bytesReaderWrapper {
	return &bytesReaderWrapper{data: data, pos: 0}
}

func (r *bytesReaderWrapper) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
