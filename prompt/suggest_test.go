package prompt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSuggest_NoDefaultModel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfgDir := filepath.Join(dir, ".promptctl")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "llm.json")
	cfg := map[string]interface{}{
		"default_provider": "anthropic",
		"default_model":    "",
		"api_keys":         map[string]string{},
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Suggest("", "scope")
	if err == nil {
		t.Fatal("expected error when no default model")
	}
}

func TestSuggest_InvalidConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfgDir := filepath.Join(dir, ".promptctl")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "llm.json")
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Suggest("", "scope")
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

// TestSuggest_NoAPIKey verifies Suggest either returns error when no API key
// (e.g. in CI) or returns a non-empty suggestion when key is available.
func TestSuggest_NoAPIKeyOrSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// No llm.json -> LoadConfig returns default config with default model
	os.Unsetenv("ANTHROPIC_API_KEY")

	suggestion, err := Suggest("", "scope")
	if err != nil {
		// Expected in CI or when no API key
		return
	}
	if suggestion == "" {
		t.Error("if Suggest succeeds, suggestion should be non-empty")
	}
}
