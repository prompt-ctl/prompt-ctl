package prompt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// EnhanceViaAPI calls the hosted enhance endpoint (e.g. Cloudflare Worker) and returns the result.
// Returns an error on network failure, non-2xx, or invalid JSON.
func EnhanceViaAPI(baseURL string, cfg EnhanceConfig) (*EnhanceResult, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")
	format := cfg.OutputFormat
	if format == "" {
		format = "xml"
	}

	body := struct {
		Intent   string `json:"intent"`
		Format   string `json:"format"`
		Persona  string `json:"persona,omitempty"`
		SaveAs   string `json:"save_as,omitempty"`
		NoConstraints bool `json:"no_constraints,omitempty"`
	}{
		Intent:   cfg.Intent,
		Format:   format,
		Persona:  cfg.Persona,
		SaveAs:   cfg.SaveAs,
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

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("enhance API: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
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
