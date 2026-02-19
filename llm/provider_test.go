package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	// ~4 chars per token heuristic
	text := "Hello world this is a test"
	n := EstimateTokens(text)
	if n <= 0 {
		t.Errorf("EstimateTokens(%q) = %d", text, n)
	}
	if n > len(text) {
		t.Errorf("EstimateTokens too high: %d for %d chars", n, len(text))
	}
}

func TestEstimateOutputTokens(t *testing.T) {
	in := 100
	for _, pt := range []string{"business_analysis", "code_review", "debugging", "general", "unknown"} {
		out := EstimateOutputTokens(in, pt)
		if out <= 0 {
			t.Errorf("EstimateOutputTokens(100, %q) = %d", pt, out)
		}
	}
}

func TestFindModel_Found(t *testing.T) {
	m, err := FindModel("claude-sonnet-4-5-20250929")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Claude Sonnet 4.5" {
		t.Errorf("Name = %q", m.Name)
	}
	m2, _ := FindModel("Claude Sonnet 4.5")
	if m2.ID != m.ID {
		t.Error("FindModel by name should return same model")
	}
}

func TestFindModel_NotFound(t *testing.T) {
	_, err := FindModel("nonexistent-model-xyz")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnstructuredMultiplier(t *testing.T) {
	if got := UnstructuredMultiplier(15); got != 2.2 {
		t.Errorf("InputPerMTok 15 (expensive): got %v, want 2.2", got)
	}
	if got := UnstructuredMultiplier(10.1); got != 2.2 {
		t.Errorf("InputPerMTok 10.1: got %v, want 2.2", got)
	}
	if got := UnstructuredMultiplier(5); got != 2.8 {
		t.Errorf("InputPerMTok 5 (mid): got %v, want 2.8", got)
	}
	if got := UnstructuredMultiplier(2.1); got != 2.8 {
		t.Errorf("InputPerMTok 2.1: got %v, want 2.8", got)
	}
	if got := UnstructuredMultiplier(1); got != 3.5 {
		t.Errorf("InputPerMTok 1 (cheap): got %v, want 3.5", got)
	}
	if got := UnstructuredMultiplier(0.2); got != 3.5 {
		t.Errorf("InputPerMTok 0.2: got %v, want 3.5", got)
	}
}

func TestEstimateCost(t *testing.T) {
	est, err := EstimateCost("Hello world prompt", "gpt-5", "general")
	if err != nil {
		t.Fatal(err)
	}
	if est.TotalEstCost <= 0 {
		t.Error("TotalEstCost should be positive")
	}
	if est.SavingsPercent <= 0 {
		t.Error("SavingsPercent should be positive")
	}
	// gpt-5 is cheap tier (InputPerMTok 1.25) → 3.5x → ~71%
	if est.SavingsPercent < 70 || est.SavingsPercent > 72 {
		t.Errorf("gpt-5 savings percent should be ~71, got %.1f", est.SavingsPercent)
	}
}

func TestEstimateCost_InvalidModel(t *testing.T) {
	_, err := EstimateCost("prompt", "invalid", "general")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatCostEstimate(t *testing.T) {
	est := &CostEstimate{
		ModelName: "Test", ModelID: "test",
		InputTokens: 100, EstOutputTokens: 200,
		TotalEstCost: 0.01, WastedWithout: 0.03, Savings: 0.02, SavingsPercent: 66.67,
	}
	s := FormatCostEstimate(est)
	if s == "" {
		t.Fatal("empty string")
	}
	if len(s) < 20 {
		t.Error("expected substantial output")
	}
}

func TestFormatCostEstimate_LargeTokens(t *testing.T) {
	// Hit formatNum with >= 1000 for thousand separators
	est := &CostEstimate{
		ModelName: "Test", ModelID: "test",
		InputTokens: 15000, EstOutputTokens: 25000,
		TotalEstCost: 0.1, WastedWithout: 0.3, Savings: 0.2, SavingsPercent: 66.67,
	}
	s := FormatCostEstimate(est)
	if !strings.Contains(s, "15,000") || !strings.Contains(s, "25,000") {
		t.Errorf("expected comma-separated numbers: %s", s)
	}
}

func TestFormatCostComparison(t *testing.T) {
	s := FormatCostComparison("Short prompt", "general")
	if s == "" {
		t.Fatal("empty")
	}
}

func TestAnnualSavingsProjection(t *testing.T) {
	low, high := AnnualSavingsProjection(0.01, 30)
	annual := 0.01 * 30 * 365 // 109.5
	if low >= high {
		t.Error("low should be < high")
	}
	if low > annual || high < annual {
		t.Errorf("range should bracket %v: got low=%v high=%v", annual, low, high)
	}
}

func TestProviderKeys(t *testing.T) {
	keys := ProviderKeys()
	if len(keys) != 6 {
		t.Errorf("len(ProviderKeys) = %d, want 6", len(keys))
	}
	if keys[0] != "promptctl" {
		t.Errorf("first provider key = %q, want promptctl", keys[0])
	}
}

func TestListModels(t *testing.T) {
	models := ListModels()
	if len(models) == 0 {
		t.Fatal("no models")
	}
}

func TestFormatModelList(t *testing.T) {
	s := FormatModelList()
	if s == "" {
		t.Fatal("empty")
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("nil config")
	}
	if cfg.APIKeys == nil {
		t.Error("APIKeys should be non-nil")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	_ = os.MkdirAll(filepath.Join(dir, ".promptctl"), 0755)
	path := filepath.Join(dir, ".promptctl", "llm.json")
	_ = os.WriteFile(path, []byte("not json"), 0600)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadConfig_WithFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	_ = os.MkdirAll(filepath.Join(dir, ".promptctl"), 0755)
	path := filepath.Join(dir, ".promptctl", "llm.json")
	data, _ := json.Marshal(Config{
		DefaultProvider: "openai",
		DefaultModel:    "gpt-5",
		APIKeys:        map[string]string{"openai": "sk-test"},
	})
	_ = os.WriteFile(path, data, 0600)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultProvider != "openai" || cfg.DefaultModel != "gpt-5" {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.APIKeys["openai"] != "sk-test" {
		t.Error("APIKeys not loaded")
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfg := &Config{
		DefaultProvider: "anthropic",
		DefaultModel:    "claude-sonnet-4-5-20250929",
		APIKeys:        make(map[string]string),
	}
	err := SaveConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".promptctl", "llm.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("config file not created")
	}
}

func TestComplete_NoAPIKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	os.Unsetenv("ANTHROPIC_API_KEY")
	_, err := Complete("hello", "claude-sonnet-4-5-20250929")
	if err == nil {
		t.Fatal("expected error when no API key")
	}
}

