package llm

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// testdataDir returns the absolute path to the testdata/responses directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "..", "testdata", "responses")
}

// loadFixture reads a fixture file from testdata/responses/.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(testdataDir(t), name))
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}
	return data
}

// fixtureServer creates an httptest.Server that responds with the given fixture
// file content and HTTP status code.
func fixtureServer(t *testing.T, fixture string, statusCode int) *httptest.Server {
	t.Helper()
	body := loadFixture(t, fixture)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server
}

// --- Anthropic fixture-based tests ---

func TestFixture_Anthropic_SuccessResponse(t *testing.T) {
	server := fixtureServer(t, "anthropic_success.json", 200)

	model := Model{
		ID:            "claude-sonnet-4-5-20250929",
		Name:          "Claude Sonnet 4.5",
		InputPerMTok:  3.0,
		OutputPerMTok: 15.0,
		Provider:      "anthropic",
	}

	result, err := callAnthropic(server.URL, "test-key", model, "review this code")
	if err != nil {
		t.Fatalf("callAnthropic error: %v", err)
	}

	if !strings.Contains(result.Content, "authentication flow") {
		t.Errorf("Content missing expected text, got: %s", result.Content[:80])
	}
	if result.InputTokens != 1250 {
		t.Errorf("InputTokens = %d, want 1250", result.InputTokens)
	}
	if result.OutputTokens != 340 {
		t.Errorf("OutputTokens = %d, want 340", result.OutputTokens)
	}
	if result.ActualCost <= 0 {
		t.Error("ActualCost should be positive")
	}

	// Verify cost calculation: (1250/1M * 3.0) + (340/1M * 15.0)
	expectedCost := (1250.0/1_000_000)*3.0 + (340.0/1_000_000)*15.0
	if result.ActualCost < expectedCost*0.99 || result.ActualCost > expectedCost*1.01 {
		t.Errorf("ActualCost = %f, want ~%f", result.ActualCost, expectedCost)
	}
}

func TestFixture_Anthropic_SuccessWithHighTokens(t *testing.T) {
	server := fixtureServer(t, "anthropic_success_with_cost.json", 200)

	model := Model{
		ID:            "claude-opus-4-6",
		Name:          "Claude Opus 4.6",
		InputPerMTok:  15.0,
		OutputPerMTok: 75.0,
		Provider:      "anthropic",
	}

	result, err := callAnthropic(server.URL, "test-key", model, "analyze architecture")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if result.InputTokens != 5000 {
		t.Errorf("InputTokens = %d, want 5000", result.InputTokens)
	}
	if result.OutputTokens != 1500 {
		t.Errorf("OutputTokens = %d, want 1500", result.OutputTokens)
	}

	// Cost for expensive model: (5000/1M * 15.0) + (1500/1M * 75.0) = 0.075 + 0.1125 = 0.1875
	expectedCost := (5000.0/1_000_000)*15.0 + (1500.0/1_000_000)*75.0
	if result.ActualCost < expectedCost*0.99 || result.ActualCost > expectedCost*1.01 {
		t.Errorf("ActualCost = %f, want ~%f", result.ActualCost, expectedCost)
	}
}

// --- OpenAI fixture-based tests ---

func TestFixture_OpenAI_SuccessResponse(t *testing.T) {
	server := fixtureServer(t, "openai_success.json", 200)

	model := Model{
		ID:            "gpt-5",
		Name:          "GPT-5",
		InputPerMTok:  1.25,
		OutputPerMTok: 10.0,
		Provider:      "openai",
	}

	result, err := callOpenAICompatible(server.URL, "test-key", model, "review this code")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if !strings.Contains(result.Content, "SQL injection") {
		t.Errorf("Content missing expected text, got: %s", result.Content[:80])
	}
	if result.InputTokens != 980 {
		t.Errorf("InputTokens = %d, want 980", result.InputTokens)
	}
	if result.OutputTokens != 275 {
		t.Errorf("OutputTokens = %d, want 275", result.OutputTokens)
	}

	expectedCost := (980.0/1_000_000)*1.25 + (275.0/1_000_000)*10.0
	if result.ActualCost < expectedCost*0.99 || result.ActualCost > expectedCost*1.01 {
		t.Errorf("ActualCost = %f, want ~%f", result.ActualCost, expectedCost)
	}
}

func TestFixture_OpenAI_SuccessWithHighTokens(t *testing.T) {
	server := fixtureServer(t, "openai_success_with_cost.json", 200)

	model := Model{
		ID:            "gpt-5",
		Name:          "GPT-5",
		InputPerMTok:  1.25,
		OutputPerMTok: 10.0,
		Provider:      "openai",
	}

	result, err := callOpenAICompatible(server.URL, "test-key", model, "debug this code")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if result.InputTokens != 3200 {
		t.Errorf("InputTokens = %d, want 3200", result.InputTokens)
	}
	if result.OutputTokens != 890 {
		t.Errorf("OutputTokens = %d, want 890", result.OutputTokens)
	}
}

// --- Error fixture tests ---

func TestFixture_Error401_Anthropic(t *testing.T) {
	server := fixtureServer(t, "error_401_unauthorized.json", 401)

	model := Model{ID: "claude-sonnet-4-5-20250929", Provider: "anthropic"}
	_, err := callAnthropic(server.URL, "invalid-key", model, "hello")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain 401: %v", err)
	}
	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("error should contain API key message: %v", err)
	}
}

func TestFixture_Error401_OpenAI(t *testing.T) {
	server := fixtureServer(t, "error_401_unauthorized.json", 401)

	model := Model{ID: "gpt-5", Provider: "openai"}
	_, err := callOpenAICompatible(server.URL, "invalid-key", model, "hello")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain 401: %v", err)
	}
}

func TestFixture_Error429_RateLimited(t *testing.T) {
	server := fixtureServer(t, "error_429_rate_limited.json", 429)

	model := Model{ID: "claude-sonnet-4-5-20250929", Provider: "anthropic"}
	_, err := callAnthropic(server.URL, "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should contain 429: %v", err)
	}
	if !strings.Contains(err.Error(), "Rate limit") {
		t.Errorf("error should contain rate limit message: %v", err)
	}
}

func TestFixture_Error500_ServerError(t *testing.T) {
	server := fixtureServer(t, "error_500_server.json", 500)

	model := Model{ID: "gpt-5", Provider: "openai"}
	_, err := callOpenAICompatible(server.URL, "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain 500: %v", err)
	}
}

func TestFixture_Error500_Anthropic(t *testing.T) {
	server := fixtureServer(t, "error_500_server.json", 500)

	model := Model{ID: "claude-sonnet-4-5-20250929", Provider: "anthropic"}
	_, err := callAnthropic(server.URL, "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain 500: %v", err)
	}
}

func TestFixture_InvalidJSON_Anthropic(t *testing.T) {
	server := fixtureServer(t, "error_invalid_json.txt", 200)

	model := Model{ID: "claude-sonnet-4-5-20250929", Provider: "anthropic"}
	_, err := callAnthropic(server.URL, "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parsing: %v", err)
	}
}

func TestFixture_InvalidJSON_OpenAI(t *testing.T) {
	server := fixtureServer(t, "error_invalid_json.txt", 200)

	model := Model{ID: "gpt-5", Provider: "openai"}
	_, err := callOpenAICompatible(server.URL, "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parsing: %v", err)
	}
}

// --- Connection timeout test ---

func TestFixture_ConnectionTimeout(t *testing.T) {
	// Create a server that delays longer than a short timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(200)
		fmt.Fprint(w, `{"content": [{"text": "too late"}], "usage": {"input_tokens": 1, "output_tokens": 1}}`)
	}))
	t.Cleanup(server.Close)

	// callAnthropic uses a 120s timeout which is too long for a unit test.
	// Instead, verify that a closed server causes a connection error.
	server.Close()

	model := Model{ID: "claude-sonnet-4-5-20250929", Provider: "anthropic"}
	_, err := callAnthropic(server.URL, "key", model, "hello")
	if err == nil {
		t.Fatal("expected error for closed server (simulating connection failure)")
	}
}

// --- Fixture file integrity tests ---

func TestFixture_AllFixturesLoadable(t *testing.T) {
	fixtures := []string{
		"anthropic_success.json",
		"openai_success.json",
		"anthropic_success_with_cost.json",
		"openai_success_with_cost.json",
		"error_401_unauthorized.json",
		"error_429_rate_limited.json",
		"error_500_server.json",
		"error_invalid_json.txt",
	}

	for _, f := range fixtures {
		data := loadFixture(t, f)
		if len(data) == 0 {
			t.Errorf("fixture %s is empty", f)
		}
	}
}

func TestFixture_SuccessResponsesAreValidJSON(t *testing.T) {
	successFixtures := []string{
		"anthropic_success.json",
		"openai_success.json",
		"anthropic_success_with_cost.json",
		"openai_success_with_cost.json",
	}

	for _, f := range successFixtures {
		data := loadFixture(t, f)
		// Quick check: valid JSON starts with { and ends with }
		trimmed := strings.TrimSpace(string(data))
		if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
			t.Errorf("fixture %s does not look like valid JSON", f)
		}
	}
}

func TestFixture_ErrorResponsesContainErrorField(t *testing.T) {
	errorFixtures := []string{
		"error_401_unauthorized.json",
		"error_429_rate_limited.json",
		"error_500_server.json",
	}

	for _, f := range errorFixtures {
		data := loadFixture(t, f)
		if !strings.Contains(string(data), `"error"`) {
			t.Errorf("fixture %s should contain 'error' field", f)
		}
	}
}
