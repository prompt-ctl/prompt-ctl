package prompt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MaxEnhanceResponseBytes is the maximum response body size for the enhance API (SSRF/memory safety).
const MaxEnhanceResponseBytes = 2 * 1024 * 1024 // 2 MB

// EnhanceViaAPI calls the hosted enhance endpoint (e.g. Cloudflare Worker) and returns the result.
// Returns an error on network failure, non-2xx, or invalid JSON.
// Only HTTPS URLs are allowed (SSRF protection).
func EnhanceViaAPI(baseURL string, cfg EnhanceConfig) (*EnhanceResult, error) {
	return EnhanceViaAPIWithClient(baseURL, cfg, nil)
}

// EnhanceViaAPIWithClient is like EnhanceViaAPI but uses the given http.Client (for tests).
// If client is nil, uses default client.
func EnhanceViaAPIWithClient(baseURL string, cfg EnhanceConfig, client *http.Client) (*EnhanceResult, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("enhance API: invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("enhance API: URL must use HTTPS (got %q)", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("enhance API: URL must have a host")
	}
	format := cfg.OutputFormat
	if format == "" {
		format = "xml"
	}

	body := struct {
		Intent        string `json:"intent"`
		Format        string `json:"format"`
		Persona       string `json:"persona,omitempty"`
		SaveAs        string `json:"save_as,omitempty"`
		NoConstraints bool   `json:"no_constraints,omitempty"`
	}{
		Intent:        cfg.Intent,
		Format:        format,
		Persona:       cfg.Persona,
		SaveAs:        cfg.SaveAs,
		NoConstraints: cfg.NoConstraints,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("enhance API: marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/enhance", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("enhance API: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.ClientVersion != "" {
		req.Header.Set("User-Agent", "promptctl/"+cfg.ClientVersion)
	}

	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("enhance API: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, MaxEnhanceResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("enhance API: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("enhance API: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Prompt   string `json:"prompt"`
		Template string `json:"template,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("enhance API: parse response: %w", err)
	}
	if result.Prompt == "" {
		return nil, fmt.Errorf("enhance API: empty prompt in response")
	}

	return &EnhanceResult{
		Prompt:   result.Prompt,
		Template: result.Template,
	}, nil
}
