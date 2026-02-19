package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// Provider represents a supported LLM provider
type Provider struct {
	Name      string
	Models    []Model
	BaseURL   string
	EnvKey    string // environment variable name for API key
	KeyURL    string // URL where users create API keys
	Order     int    // display order in menus
}

// Model represents a specific model with its pricing
type Model struct {
	ID            string
	Name          string  // human-friendly name
	InputPerMTok  float64 // cost per 1M input tokens in USD
	OutputPerMTok float64 // cost per 1M output tokens in USD
	ContextWindow int     // max context window size in tokens
	Provider      string
}

// UnstructuredMultiplier returns the rework multiplier for unstructured prompting
// by price tier: expensive models (users careful) 2.2x, mid 2.8x, cheap 3.5x.
func UnstructuredMultiplier(inputPerMTok float64) float64 {
	switch {
	case inputPerMTok > 10.0:
		return 2.2
	case inputPerMTok > 2.0:
		return 2.8
	default:
		return 3.5
	}
}

// CostEstimate holds token and cost breakdown for a prompt
type CostEstimate struct {
	InputTokens       int
	EstOutputTokens   int     // estimated based on prompt complexity
	InputCost         float64 // USD
	EstOutputCost     float64 // USD
	TotalEstCost      float64 // USD
	ModelID           string
	ModelName         string
	WastedWithout     float64 // estimated cost of unstructured prompting (typically 2-4x)
	Savings           float64 // money saved by using promptctl
	SavingsPercent    float64
}

// CompletionResult holds the response from an LLM call
type CompletionResult struct {
	Content       string
	InputTokens   int
	OutputTokens  int
	ActualCost    float64
	Model         string
	LatencyMs     int64
}

// Config holds LLM configuration stored in ~/.promptctl/llm.json
type Config struct {
	DefaultProvider string            `json:"default_provider"`
	DefaultModel    string            `json:"default_model"`
	APIKeys         map[string]string `json:"api_keys"`
}

// MaxLLMResponseBytes is the maximum response body size for LLM API calls (memory safety).
const MaxLLMResponseBytes = 8 * 1024 * 1024 // 8 MB

// -----------------------------------------------------------------------
// Supported providers and models with current pricing (as of Feb 2026)
// -----------------------------------------------------------------------

var Providers = map[string]Provider{
	"anthropic": {
		Name:    "Anthropic",
		BaseURL: "https://api.anthropic.com/v1/messages",
		EnvKey:  "ANTHROPIC_API_KEY",
		KeyURL:  "https://console.anthropic.com/settings/keys",
		Order:   1,
		Models: []Model{
			{ID: "claude-sonnet-4-5-20250929", Name: "Claude Sonnet 4.5", InputPerMTok: 3.0, OutputPerMTok: 15.0, ContextWindow: 200000, Provider: "anthropic"},
			{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", InputPerMTok: 1.0, OutputPerMTok: 5.0, ContextWindow: 200000, Provider: "anthropic"},
			{ID: "claude-opus-4-6", Name: "Claude Opus 4.6", InputPerMTok: 5.0, OutputPerMTok: 25.0, ContextWindow: 200000, Provider: "anthropic"},
		},
	},
	"openai": {
		// GPT-4o, GPT-4.1, GPT-4.1 mini, o4-mini retired from ChatGPT Feb 2026 (API unchanged). Prefer GPT-5.1 / GPT-5.2 for new use.
		Name:    "OpenAI",
		BaseURL: "https://api.openai.com/v1/chat/completions",
		EnvKey:  "OPENAI_API_KEY",
		KeyURL:  "https://platform.openai.com/api-keys",
		Order:   2,
		Models: []Model{
			{ID: "gpt-5.1", Name: "GPT-5.1", InputPerMTok: 1.5, OutputPerMTok: 12.0, ContextWindow: 128000, Provider: "openai"},
			{ID: "gpt-5.2", Name: "GPT-5.2", InputPerMTok: 1.75, OutputPerMTok: 14.0, ContextWindow: 128000, Provider: "openai"},
			{ID: "gpt-5", Name: "GPT-5", InputPerMTok: 1.25, OutputPerMTok: 10.0, ContextWindow: 128000, Provider: "openai"},
		},
	},
	"groq": {
		Name:    "Groq",
		BaseURL: "https://api.groq.com/openai/v1/chat/completions",
		EnvKey:  "GROQ_API_KEY",
		KeyURL:  "https://console.groq.com/keys",
		Order:   3,
		Models: []Model{
			{ID: "meta-llama/llama-4-maverick-17b-128e-instruct", Name: "Llama 4 Maverick", InputPerMTok: 0.20, OutputPerMTok: 0.60, ContextWindow: 128000, Provider: "groq"},
			{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B", InputPerMTok: 0.59, OutputPerMTok: 0.79, ContextWindow: 128000, Provider: "groq"},
			{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", InputPerMTok: 0.24, OutputPerMTok: 0.24, ContextWindow: 32768, Provider: "groq"},
		},
	},
	"deepseek": {
		Name:    "DeepSeek",
		BaseURL: "https://api.deepseek.com/v1/chat/completions",
		EnvKey:  "DEEPSEEK_API_KEY",
		KeyURL:  "https://platform.deepseek.com/api_keys",
		Order:   4,
		Models: []Model{
			{ID: "deepseek-chat", Name: "DeepSeek Chat (V3.2)", InputPerMTok: 0.28, OutputPerMTok: 0.42, ContextWindow: 64000, Provider: "deepseek"},
			{ID: "deepseek-reasoner", Name: "DeepSeek R1", InputPerMTok: 0.55, OutputPerMTok: 2.19, ContextWindow: 64000, Provider: "deepseek"},
		},
	},
	// Atlas is the codename for promptctl's deployed LLM (no third-party API key required when using hosted endpoint).
	"promptctl": {
		Name:    "Promptctl",
		BaseURL: "", // use PROMPTCTL_LLM_URL at runtime
		EnvKey:  "PROMPTCTL_API_KEY",
		KeyURL:  "https://github.com/oleg-koval/promptctl",
		Order:   0,
		Models: []Model{
			{ID: "atlas", Name: "Atlas (hosted)", InputPerMTok: 0, OutputPerMTok: 0, ContextWindow: 128000, Provider: "promptctl"},
		},
	},
}

// -----------------------------------------------------------------------
// Token estimation - rough but useful for cost estimation
// -----------------------------------------------------------------------

// EstimateTokens provides a rough token count for a given text.
// Uses the ~4 chars per token heuristic which is reasonably accurate
// for English text across most tokenizers. Not exact, but good enough
// for cost estimation (within ~10% of actual).
func EstimateTokens(text string) int {
	charCount := utf8.RuneCountInString(text)
	// Most modern tokenizers average ~3.5-4.5 chars per token for English.
	// We use 3.8 to slightly overestimate (safer for cost projections).
	return int(float64(charCount) / 3.8)
}

// EstimateOutputTokens guesses how many tokens the response will use
// based on the prompt complexity and type. A structured prompt typically
// gets a more focused (shorter) response than a vague one.
func EstimateOutputTokens(inputTokens int, promptType string) int {
	// Structured prompts produce 1.5-2.5x input in output on average.
	// Unstructured prompts produce 3-5x because the model rambles,
	// repeats itself, and adds unnecessary caveats.
	multipliers := map[string]float64{
		"business_analysis": 2.5,
		"code_review":       1.5,
		"debugging":         1.8,
		"architecture":      2.5,
		"analysis":          2.0,
		"planning":          2.0,
		"writing":           3.0,
		"explanation":       2.0,
		"refactoring":       1.5,
		"testing":           1.5,
		"transformation":    1.5,
		"general":           2.0,
	}

	mult, ok := multipliers[promptType]
	if !ok {
		mult = 2.0
	}

	return int(float64(inputTokens) * mult)
}

// -----------------------------------------------------------------------
// Cost estimation
// -----------------------------------------------------------------------

// EstimateCost calculates the expected cost for running a prompt against a model
func EstimateCost(prompt string, modelID string, promptType string) (*CostEstimate, error) {
	model, err := FindModel(modelID)
	if err != nil {
		return nil, err
	}

	inputTokens := EstimateTokens(prompt)
	outputTokens := EstimateOutputTokens(inputTokens, promptType)

	inputCost := float64(inputTokens) / 1_000_000 * model.InputPerMTok
	outputCost := float64(outputTokens) / 1_000_000 * model.OutputPerMTok
	totalCost := inputCost + outputCost

	// Estimate the cost of unstructured prompting by price tier:
	// expensive (Opus, GPT-4): users careful, fewer rework → 2.2x
	// mid (Sonnet, GPT-4o): average rework → 2.8x
	// cheap (Haiku, Groq, DeepSeek): users sloppy, more rework → 3.5x
	mult := UnstructuredMultiplier(model.InputPerMTok)
	wastedCost := totalCost * mult
	savings := wastedCost - totalCost

	return &CostEstimate{
		InputTokens:     inputTokens,
		EstOutputTokens: outputTokens,
		InputCost:       inputCost,
		EstOutputCost:   outputCost,
		TotalEstCost:    totalCost,
		ModelID:         model.ID,
		ModelName:       model.Name,
		WastedWithout:   wastedCost,
		Savings:         savings,
		SavingsPercent:  (savings / wastedCost) * 100,
	}, nil
}

// FormatCostEstimate produces a human-readable cost breakdown
func FormatCostEstimate(est *CostEstimate) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("  Model:            %s (%s)\n", est.ModelName, est.ModelID))
	sb.WriteString(fmt.Sprintf("  Input tokens:     ~%s\n", formatNum(est.InputTokens)))
	sb.WriteString(fmt.Sprintf("  Est. output:      ~%s tokens\n", formatNum(est.EstOutputTokens)))
	sb.WriteString(fmt.Sprintf("  ─────────────────────────────\n"))
	sb.WriteString(fmt.Sprintf("  Est. cost:        $%.4f\n", est.TotalEstCost))
	sb.WriteString(fmt.Sprintf("  Without promptctl: $%.4f  (rework, rambling, follow-ups)\n", est.WastedWithout))
	sb.WriteString(fmt.Sprintf("  You save:         $%.4f  (%.0f%%)\n", est.Savings, est.SavingsPercent))

	return sb.String()
}

// FormatCostComparison produces a multi-model cost comparison table
func FormatCostComparison(prompt string, promptType string) string {
	var sb strings.Builder

	sb.WriteString("  Model                      Input tok   Est. cost   Without promptctl   Savings\n")
	sb.WriteString("  ───────────────────────────────────────────────────────────────────────────────\n")

	for _, providerName := range []string{"promptctl", "anthropic", "openai", "groq", "deepseek"} {
		provider := Providers[providerName]
		for _, model := range provider.Models {
			est, err := EstimateCost(prompt, model.ID, promptType)
			if err != nil {
				continue
			}
			savPct := fmt.Sprintf("%.0f%%", est.SavingsPercent)
			if est.WastedWithout == 0 || math.IsNaN(est.SavingsPercent) {
				savPct = "N/A"
			}
			sb.WriteString(fmt.Sprintf("  %-26s %7s     $%.4f       $%.4f            %s\n",
				model.Name,
				formatNum(est.InputTokens),
				est.TotalEstCost,
				est.WastedWithout,
				savPct,
			))
		}
	}

	return sb.String()
}

// AnnualSavingsProjection returns estimated annual savings (low, high) in USD
// for given per-call savings and calls per day. Range is ±15% of point estimate.
func AnnualSavingsProjection(savingsPerCall float64, callsPerDay int) (low, high float64) {
	annual := savingsPerCall * float64(callsPerDay*365)
	low = annual * 0.85
	high = annual * 1.15
	return low, high
}

// -----------------------------------------------------------------------
// LLM execution
// -----------------------------------------------------------------------

// Complete sends a prompt to an LLM and returns the response
func Complete(prompt string, modelID string) (*CompletionResult, error) {
	model, err := FindModel(modelID)
	if err != nil {
		return nil, err
	}

	provider, ok := Providers[model.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", model.Provider)
	}

	baseURL := provider.BaseURL
	if model.Provider == "promptctl" {
		if u := os.Getenv("PROMPTCTL_LLM_URL"); u != "" {
			baseURL = u
		}
		if baseURL == "" {
			return nil, fmt.Errorf("Promptctl (Atlas) requires PROMPTCTL_LLM_URL to be set to your deployed LLM endpoint")
		}
	}

	apiKey := GetAPIKey(model.Provider, provider.EnvKey)
	if apiKey == "" && model.Provider != "promptctl" {
		return nil, fmt.Errorf("no API key found for %s. Set it with:\n  promptctl config --provider=%s --api-key=YOUR_KEY\n  or export %s=YOUR_KEY",
			provider.Name, model.Provider, provider.EnvKey)
	}

	start := time.Now()

	var result *CompletionResult
	switch model.Provider {
	case "anthropic":
		result, err = callAnthropic(baseURL, apiKey, *model, prompt)
	case "promptctl":
		result, err = callOpenAICompatible(baseURL, apiKey, *model, prompt)
	default:
		// OpenAI-compatible API (works for OpenAI, Groq, DeepSeek)
		result, err = callOpenAICompatible(baseURL, apiKey, *model, prompt)
	}

	if err != nil {
		return nil, err
	}

	result.LatencyMs = time.Since(start).Milliseconds()
	result.Model = model.Name

	return result, nil
}

// callAnthropic makes a request to the Anthropic Messages API
func callAnthropic(baseURL, apiKey string, model Model, prompt string) (*CompletionResult, error) {
	body := map[string]interface{}{
		"model":      model.ID,
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, MaxLLMResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	text := ""
	if len(result.Content) > 0 {
		text = result.Content[0].Text
	}

	inputCost := float64(result.Usage.InputTokens) / 1_000_000 * model.InputPerMTok
	outputCost := float64(result.Usage.OutputTokens) / 1_000_000 * model.OutputPerMTok

	return &CompletionResult{
		Content:      text,
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
		ActualCost:   inputCost + outputCost,
	}, nil
}

// callOpenAICompatible makes a request to any OpenAI-compatible API
// (OpenAI, Groq, DeepSeek, and many others use this format)
func callOpenAICompatible(baseURL, apiKey string, model Model, prompt string) (*CompletionResult, error) {
	body := map[string]interface{}{
		"model": model.ID,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 4096,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, MaxLLMResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	text := ""
	if len(result.Choices) > 0 {
		text = result.Choices[0].Message.Content
	}

	inputCost := float64(result.Usage.PromptTokens) / 1_000_000 * model.InputPerMTok
	outputCost := float64(result.Usage.CompletionTokens) / 1_000_000 * model.OutputPerMTok

	return &CompletionResult{
		Content:      text,
		InputTokens:  result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
		ActualCost:   inputCost + outputCost,
	}, nil
}

// -----------------------------------------------------------------------
// Configuration management
// -----------------------------------------------------------------------

// LoadConfig reads the LLM config from disk
func LoadConfig() (*Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				DefaultProvider: "anthropic",
				DefaultModel:    "claude-sonnet-4-5-20250929",
				APIKeys:         make(map[string]string),
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse LLM config: %w", err)
	}

	if cfg.APIKeys == nil {
		cfg.APIKeys = make(map[string]string)
	}

	return &cfg, nil
}

// SaveConfig writes the LLM config to disk
func SaveConfig(cfg *Config) error {
	path := configPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Write with restrictive permissions since this contains API keys
	return os.WriteFile(path, data, 0600)
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".promptctl", "llm.json")
}

// GetAPIKey returns the API key for a provider (keychain, config, or env). Used for display and by callers.
func GetAPIKey(provider, envKey string) string {
	if key, ok := keychainGet(provider); ok && key != "" {
		return key
	}
	cfg, err := LoadConfig()
	if err == nil {
		if key, ok := cfg.APIKeys[provider]; ok && key != "" && key != keychainSentinel {
			return key
		}
	}
	return os.Getenv(envKey)
}

// SetAPIKey stores the API key for a provider (keychain on macOS, config file elsewhere)
func SetAPIKey(provider, key string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	if cfg.APIKeys == nil {
		cfg.APIKeys = make(map[string]string)
	}
	if err := keychainStore(provider, key); err != nil {
		return err
	}
	if keychainSentinel != "" {
		cfg.APIKeys[provider] = keychainSentinel
	} else {
		cfg.APIKeys[provider] = key
	}
	return SaveConfig(cfg)
}

// -----------------------------------------------------------------------
// Model lookup helpers
// -----------------------------------------------------------------------

// ProviderKeys returns provider keys in display order
func ProviderKeys() []string {
	return []string{"promptctl", "anthropic", "openai", "groq", "deepseek"}
}

// FindModel searches all providers for a model by ID or name
func FindModel(query string) (*Model, error) {
	query = strings.ToLower(query)

	for _, provider := range Providers {
		for _, model := range provider.Models {
			if strings.ToLower(model.ID) == query || strings.ToLower(model.Name) == query {
				return &model, nil
			}
		}
	}

	return nil, fmt.Errorf("model not found: '%s'. Run 'promptctl models' to see available models", query)
}

// ListModels returns all supported models across all providers
func ListModels() []Model {
	var models []Model
	for _, providerName := range []string{"promptctl", "anthropic", "openai", "groq", "deepseek"} {
		provider := Providers[providerName]
		models = append(models, provider.Models...)
	}
	return models
}

// FormatModelList produces a human-readable table of all models
func FormatModelList() string {
	var sb strings.Builder

	sb.WriteString("  Provider     Model                         In ($/1M tok)   Out ($/1M tok)   Max context\n")
	sb.WriteString("  ──────────────────────────────────────────────────────────────────────────────────────────\n")

	for _, providerName := range []string{"promptctl", "anthropic", "openai", "groq", "deepseek"} {
		provider := Providers[providerName]
		for _, model := range provider.Models {
			sb.WriteString(fmt.Sprintf("  %-12s %-27s $%-14.2f $%-15.2f %sk\n",
				provider.Name,
				model.Name,
				model.InputPerMTok,
				model.OutputPerMTok,
				formatNum(model.ContextWindow/1000),
			))
		}
	}

	return sb.String()
}

// -----------------------------------------------------------------------
// Utilities
// -----------------------------------------------------------------------

func formatNum(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}

	// Add thousand separators
	result := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
