package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Anthropic provider tests ---

func TestCallAnthropic_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key = %q, want test-key", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("anthropic-version = %q", r.Header.Get("anthropic-version"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}

		// Verify request body
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		if req["model"] != "claude-sonnet-4-5-20250929" {
			t.Errorf("model = %v", req["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"content": [{"text": "Hello from Anthropic!"}],
			"usage": {"input_tokens": 100, "output_tokens": 50}
		}`)
	}))
	defer server.Close()

	model := Model{
		ID:            "claude-sonnet-4-5-20250929",
		Name:          "Claude Sonnet 4.5",
		InputPerMTok:  3.0,
		OutputPerMTok: 15.0,
		Provider:      "anthropic",
	}

	result, err := callAnthropic(server.URL, "test-key", model, "hello")
	if err != nil {
		t.Fatalf("callAnthropic error: %v", err)
	}
	if result.Content != "Hello from Anthropic!" {
		t.Errorf("Content = %q", result.Content)
	}
	if result.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", result.InputTokens)
	}
	if result.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", result.OutputTokens)
	}
	if result.ActualCost <= 0 {
		t.Error("ActualCost should be positive")
	}
}

func TestCallAnthropic_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"error": {"message": "Invalid API key"}}`)
	}))
	defer server.Close()

	model := Model{ID: "claude-sonnet-4-5-20250929", Provider: "anthropic"}
	_, err := callAnthropic(server.URL, "bad-key", model, "hello")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain status code: %v", err)
	}
}

func TestCallAnthropic_RateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		fmt.Fprint(w, `{"error": {"message": "Rate limit exceeded"}}`)
	}))
	defer server.Close()

	model := Model{ID: "claude-sonnet-4-5-20250929", Provider: "anthropic"}
	_, err := callAnthropic(server.URL, "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should contain 429: %v", err)
	}
}

func TestCallAnthropic_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not json")
	}))
	defer server.Close()

	model := Model{ID: "claude-sonnet-4-5-20250929", Provider: "anthropic"}
	_, err := callAnthropic(server.URL, "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestCallAnthropic_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"content": [], "usage": {"input_tokens": 10, "output_tokens": 0}}`)
	}))
	defer server.Close()

	model := Model{ID: "claude-sonnet-4-5-20250929", Provider: "anthropic", InputPerMTok: 3.0, OutputPerMTok: 15.0}
	result, err := callAnthropic(server.URL, "key", model, "hello")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Content != "" {
		t.Errorf("Content should be empty for empty content array, got %q", result.Content)
	}
}

// --- OpenAI compatible provider tests ---

func TestCallOpenAICompatible_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("Authorization = %q, want Bearer prefix", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"choices": [{"message": {"content": "Hello from OpenAI!"}}],
			"usage": {"prompt_tokens": 80, "completion_tokens": 40}
		}`)
	}))
	defer server.Close()

	model := Model{
		ID:            "gpt-5",
		Name:          "GPT-5",
		InputPerMTok:  1.25,
		OutputPerMTok: 10.0,
		Provider:      "openai",
	}

	result, err := callOpenAICompatible(server.URL, "test-key", model, "hello")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Content != "Hello from OpenAI!" {
		t.Errorf("Content = %q", result.Content)
	}
	if result.InputTokens != 80 {
		t.Errorf("InputTokens = %d, want 80", result.InputTokens)
	}
	if result.OutputTokens != 40 {
		t.Errorf("OutputTokens = %d, want 40", result.OutputTokens)
	}
}

func TestCallOpenAICompatible_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No auth header should be set
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Authorization should be empty when no API key, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices": [{"message": {"content": "ok"}}], "usage": {"prompt_tokens": 1, "completion_tokens": 1}}`)
	}))
	defer server.Close()

	model := Model{ID: "atlas", Provider: "promptctl"}
	result, err := callOpenAICompatible(server.URL, "", model, "hello")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Content != "ok" {
		t.Errorf("Content = %q", result.Content)
	}
}

func TestCallOpenAICompatible_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"error": {"message": "Internal server error"}}`)
	}))
	defer server.Close()

	model := Model{ID: "gpt-5", Provider: "openai"}
	_, err := callOpenAICompatible(server.URL, "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain 500: %v", err)
	}
}

func TestCallOpenAICompatible_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices": [], "usage": {"prompt_tokens": 10, "completion_tokens": 0}}`)
	}))
	defer server.Close()

	model := Model{ID: "gpt-5", Provider: "openai"}
	result, err := callOpenAICompatible(server.URL, "key", model, "hello")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Content != "" {
		t.Errorf("Content should be empty for no choices, got %q", result.Content)
	}
}

// --- Gemini provider tests ---

func TestCallGemini_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL contains model and key
		if !strings.Contains(r.URL.String(), "generateContent") {
			t.Error("URL should contain generateContent")
		}
		if !strings.Contains(r.URL.String(), "key=test-key") {
			t.Error("URL should contain API key")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"candidates": [{"content": {"parts": [{"text": "Hello from Gemini!"}]}}],
			"usageMetadata": {"promptTokenCount": 60, "candidatesTokenCount": 30, "totalTokenCount": 90}
		}`)
	}))
	defer server.Close()

	model := Model{
		ID:            "gemini-3.1-pro",
		Name:          "Gemini 3.1 Pro",
		InputPerMTok:  1.25,
		OutputPerMTok: 5.0,
		Provider:      "google",
	}

	result, err := callGemini(server.URL, "test-key", model, "hello", nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Content != "Hello from Gemini!" {
		t.Errorf("Content = %q", result.Content)
	}
	if result.InputTokens != 60 {
		t.Errorf("InputTokens = %d, want 60", result.InputTokens)
	}
	if result.OutputTokens != 30 {
		t.Errorf("OutputTokens = %d, want 30", result.OutputTokens)
	}
}

func TestCallGemini_WithOptions(t *testing.T) {
	var requestBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &requestBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"candidates": [{"content": {"parts": [{"text": "ok"}]}}],
			"usageMetadata": {"promptTokenCount": 10, "candidatesTokenCount": 5}
		}`)
	}))
	defer server.Close()

	model := Model{ID: "gemini-3.1-pro", Provider: "google", InputPerMTok: 1.25, OutputPerMTok: 5.0}
	opts := &CompleteOptions{
		ThinkingLevel:   "high",
		MediaResolution: "medium",
	}

	_, err := callGemini(server.URL, "key", model, "hello", opts)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	// Verify options were sent in request
	genConfig, ok := requestBody["generationConfig"].(map[string]interface{})
	if !ok {
		t.Fatal("generationConfig not found in request body")
	}
	if genConfig["mediaResolution"] != "medium" {
		t.Errorf("mediaResolution = %v, want medium", genConfig["mediaResolution"])
	}
}

func TestCallGemini_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		fmt.Fprint(w, `{"error": {"message": "Forbidden"}}`)
	}))
	defer server.Close()

	model := Model{ID: "gemini-3.1-pro", Provider: "google"}
	_, err := callGemini(server.URL, "bad-key", model, "hello", nil)
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestCallGemini_FallbackTokenCounting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Only totalTokenCount, no individual counts
		fmt.Fprint(w, `{
			"candidates": [{"content": {"parts": [{"text": "result"}]}}],
			"usageMetadata": {"totalTokenCount": 100}
		}`)
	}))
	defer server.Close()

	model := Model{ID: "gemini-3.1-pro", Provider: "google", InputPerMTok: 1.25, OutputPerMTok: 5.0}
	result, err := callGemini(server.URL, "key", model, "hello", nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// When only totalTokenCount is given, should split evenly
	if result.InputTokens != 50 {
		t.Errorf("InputTokens = %d, want 50 (half of total)", result.InputTokens)
	}
	if result.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50 (half of total)", result.OutputTokens)
	}
}

func TestCallGemini_EmptyCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"candidates": [], "usageMetadata": {"totalTokenCount": 0}}`)
	}))
	defer server.Close()

	model := Model{ID: "gemini-3.1-pro", Provider: "google"}
	result, err := callGemini(server.URL, "key", model, "hello", nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Content != "" {
		t.Errorf("Content should be empty for empty candidates, got %q", result.Content)
	}
}

// --- Token estimation tests ---

func TestEstimateTokens_Empty(t *testing.T) {
	n := EstimateTokens("")
	if n != 0 {
		t.Errorf("empty string should have 0 tokens, got %d", n)
	}
}

func TestEstimateTokens_Unicode(t *testing.T) {
	// Unicode characters should be counted properly
	n := EstimateTokens("Hello 世界")
	if n <= 0 {
		t.Errorf("unicode text should have positive tokens, got %d", n)
	}
}

func TestEstimateTokens_LongText(t *testing.T) {
	text := strings.Repeat("hello world ", 1000)
	n := EstimateTokens(text)
	if n <= 0 {
		t.Error("long text should have positive tokens")
	}
	// Should be roughly charCount/3.8
	expected := len(text) * 100 / 380
	tolerance := expected / 5
	if n < expected-tolerance || n > expected+tolerance {
		t.Errorf("tokens = %d, expected ~%d (tolerance %d)", n, expected, tolerance)
	}
}

// --- Cost calculation accuracy ---

func TestEstimateCost_CheapModel_HighSavings(t *testing.T) {
	est, err := EstimateCost("test prompt", "deepseek-chat", "general")
	if err != nil {
		t.Fatal(err)
	}
	// Cheap model (0.28 InputPerMTok) should have 3.5x multiplier => ~71% savings
	if est.SavingsPercent < 70 {
		t.Errorf("cheap model savings = %.1f%%, want >= 70%%", est.SavingsPercent)
	}
}

func TestEstimateCost_ExpensiveModel_LowerSavings(t *testing.T) {
	est, err := EstimateCost("test prompt", "claude-opus-4-6", "general")
	if err != nil {
		t.Fatal(err)
	}
	// Expensive model (5.0 InputPerMTok > 2.0) should have 2.8x multiplier => ~64% savings
	if est.SavingsPercent > 70 {
		t.Errorf("expensive model savings = %.1f%%, want < 70%%", est.SavingsPercent)
	}
}

// --- GetAPIKey ---

func TestGetAPIKey_FromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("ANTHROPIC_API_KEY", "sk-test-env-key")

	key := GetAPIKey("anthropic", "ANTHROPIC_API_KEY")
	if key != "sk-test-env-key" {
		t.Errorf("GetAPIKey = %q, want sk-test-env-key", key)
	}
}

func TestGetAPIKey_FromConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	os.Unsetenv("OPENAI_API_KEY")

	// Save key to config
	configDir := filepath.Join(dir, ".promptctl")
	os.MkdirAll(configDir, 0755)
	cfg := &Config{
		DefaultProvider: "openai",
		DefaultModel:    "gpt-5",
		APIKeys:         map[string]string{"openai": "sk-from-config"},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(configDir, "llm.json"), data, 0600)

	key := GetAPIKey("openai", "OPENAI_API_KEY")
	if key != "sk-from-config" {
		t.Errorf("GetAPIKey from config = %q, want sk-from-config", key)
	}
}

func TestGetAPIKey_NoKeyAvailable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	os.Unsetenv("NONEXISTENT_KEY")

	key := GetAPIKey("nonexistent", "NONEXISTENT_KEY")
	if key != "" {
		t.Errorf("GetAPIKey with no key = %q, want empty", key)
	}
}

// --- SetAPIKey ---

func TestSetAPIKey_SavesAndReads(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	err := SetAPIKey("openai", "sk-new-key")
	if err != nil {
		// On macOS, keychain operations may fail in test contexts (e.g. CI, sandboxed).
		// This is expected - the keychain integration is tested via manual/integration tests.
		t.Skipf("SetAPIKey failed (expected in non-keychain environments): %v", err)
	}

	// Read back
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKeys == nil {
		t.Error("APIKeys should not be nil after SetAPIKey")
	}
}

// --- FormatNum ---

func TestFormatNum_Values(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
	}
	for _, tt := range tests {
		got := formatNum(tt.in)
		if got != tt.want {
			t.Errorf("formatNum(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// --- Complete function ---

func TestComplete_InvalidModel(t *testing.T) {
	_, err := Complete("hello", "nonexistent-model-xyz")
	if err == nil {
		t.Error("expected error for invalid model")
	}
}

func TestCompleteWithOptions_InvalidModel(t *testing.T) {
	_, err := CompleteWithOptions("hello", "nonexistent-model-xyz", nil)
	if err == nil {
		t.Error("expected error for invalid model")
	}
}

// --- Network error simulation ---

func TestCallAnthropic_ConnectionRefused(t *testing.T) {
	model := Model{ID: "test", Provider: "anthropic"}
	_, err := callAnthropic("http://127.0.0.1:1", "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestCallOpenAICompatible_ConnectionRefused(t *testing.T) {
	model := Model{ID: "test", Provider: "openai"}
	_, err := callOpenAICompatible("http://127.0.0.1:1", "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestCallGemini_ConnectionRefused(t *testing.T) {
	model := Model{ID: "test", Provider: "google"}
	_, err := callGemini("http://127.0.0.1:1", "key", model, "hello", nil)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}
