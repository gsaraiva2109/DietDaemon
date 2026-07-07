package hevy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// BaseURL is Hevy's public REST API base URL.
const BaseURL = "https://api.hevyapp.com"

// Client is a minimal Hevy API client. Construct with NewClient — no network
// calls happen until a method is invoked (same "cheap to construct" convention
// as the LLM adapters).
type Client struct {
	apiKey string
	http   *http.Client
}

// NewClient returns a Hevy API client for the given API key. The caller is
// responsible for decrypting the stored key before passing it here.
func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey, http: &http.Client{}}
}

// listResponse is the envelope for GET /v1/workouts.
type listResponse struct {
	Page      int           `json:"page"`
	PageCount int           `json:"page_count"`
	Workouts  []HevyWorkout `json:"workouts"`
}

// ListWorkouts fetches a single page of workouts. page is 1-based; pageSize is
// silently clamped to 10 (Hevy's hard max). Returns the workouts, total page
// count, and any error.
func (c *Client) ListWorkouts(ctx context.Context, page, pageSize int) ([]HevyWorkout, int, error) {
	if pageSize > 10 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}

	u, err := url.Parse(BaseURL + "/v1/workouts")
	if err != nil {
		return nil, 0, fmt.Errorf("hevy: parse url: %w", err)
	}
	q := u.Query()
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("pageSize", fmt.Sprintf("%d", pageSize))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("hevy: new request: %w", err)
	}
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("hevy: list workouts: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, 0, fmt.Errorf("hevy: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("hevy: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var lr listResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, 0, fmt.Errorf("hevy: unmarshal: %w", err)
	}

	return lr.Workouts, lr.PageCount, nil
}
