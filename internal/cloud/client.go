package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const requestTimeout = 5 * time.Second

// Client wraps optional cloud interactions behind an explicit opt-in boundary.
type Client interface {
	Enabled() bool
	PostRating(rating int, intentLen int) error
	SubmitFeedback(text string) error
}

type disabledClient struct{}

func (c disabledClient) Enabled() bool { return false }
func (c disabledClient) PostRating(rating int, intentLen int) error {
	return nil
}
func (c disabledClient) SubmitFeedback(text string) error {
	return nil
}

type httpClient struct {
	baseURL string
	client  *http.Client
}

// New returns an enabled HTTP client only when cloud opt-in is enabled and baseURL is present.
func New(enabled bool, baseURL string) Client {
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if !enabled || baseURL == "" {
		return disabledClient{}
	}
	return &httpClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: requestTimeout},
	}
}

func (c *httpClient) Enabled() bool { return true }

func (c *httpClient) PostRating(rating int, intentLen int) error {
	body, err := json.Marshal(map[string]int{
		"rating":     rating,
		"intent_len": intentLen,
	})
	if err != nil {
		return err
	}
	return c.postJSON("/rating", body)
}

func (c *httpClient) SubmitFeedback(text string) error {
	body, err := json.Marshal(map[string]string{
		"feedback": text,
	})
	if err != nil {
		return err
	}
	return c.postJSON("/feedback", body)
}

func (c *httpClient) postJSON(path string, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cloud request failed: %s", resp.Status)
	}
	return nil
}
