package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Use a temp dir as home so we don't touch real config
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	if cfg.GlobalTemplateDir == "" {
		t.Error("GlobalTemplateDir empty")
	}
	if cfg.LocalTemplateDir == "" {
		t.Error("LocalTemplateDir empty")
	}
	if cfg.GlobalConfigFile == "" {
		t.Error("GlobalConfigFile empty")
	}
	if cfg.DefaultVars == nil {
		t.Error("DefaultVars nil")
	}
}

func TestInitGlobal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	err := InitGlobal()
	if err != nil {
		t.Fatalf("InitGlobal() err = %v", err)
	}

	templatesDir := filepath.Join(dir, ".promptctl", "templates")
	for _, name := range []string{"review", "debug", "arch", "commit", "explain"} {
		path := filepath.Join(templatesDir, name+".yaml")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("starter template %s not created", name)
		}
	}
}

func TestInitGlobal_DoesNotOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create templates dir and one existing file
	templatesDir := filepath.Join(dir, ".promptctl", "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatal(err)
	}
	existingContent := "existing content"
	reviewPath := filepath.Join(templatesDir, "review.yaml")
	if err := os.WriteFile(reviewPath, []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := InitGlobal()
	if err != nil {
		t.Fatalf("InitGlobal() err = %v", err)
	}

	data, _ := os.ReadFile(reviewPath)
	if string(data) != existingContent {
		t.Error("InitGlobal overwrote existing review.yaml")
	}
}

func TestLoad_LocalConfigFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	localPromptctl := filepath.Join(dir, ".promptctl")
	if err := os.MkdirAll(localPromptctl, 0755); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	want := filepath.Join(dir, ".promptctl", "templates")
	// Resolve symlinks (e.g. /var -> /private/var on macOS) for comparison
	gotNorm, _ := filepath.EvalSymlinks(cfg.LocalTemplateDir)
	wantNorm, _ := filepath.EvalSymlinks(want)
	if gotNorm != wantNorm {
		t.Errorf("LocalTemplateDir = %s, want %s", cfg.LocalTemplateDir, want)
	}
}

func TestLoad_EnhanceEnvVars(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("PROMPTCTL_ENHANCE_URL", "https://enhance.example.com")
	t.Setenv("PROMPTCTL_ENHANCE", "llm")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg.EnhanceURL != "https://enhance.example.com" {
		t.Errorf("EnhanceURL = %q, want https://enhance.example.com", cfg.EnhanceURL)
	}
	if cfg.EnhanceMode != "llm" {
		t.Errorf("EnhanceMode = %q, want llm", cfg.EnhanceMode)
	}
}

func TestLoad_EnhanceModeDefaultLLM(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("PROMPTCTL_ENHANCE", "") // empty => default to "llm"
	t.Setenv("PROMPTCTL_ENHANCE_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg.EnhanceMode != "llm" {
		t.Errorf("EnhanceMode = %q, want llm when empty", cfg.EnhanceMode)
	}
	if cfg.EnhanceURL != DefaultEnhanceURL {
		t.Errorf("EnhanceURL = %q, want %q", cfg.EnhanceURL, DefaultEnhanceURL)
	}
}

func TestLoad_EnhanceModeRuleExplicit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg.EnhanceMode != "rule" {
		t.Errorf("EnhanceMode = %q, want rule", cfg.EnhanceMode)
	}
}

func TestLoad_EnhanceURLDefaultWhenLLM(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("PROMPTCTL_ENHANCE", "llm")
	t.Setenv("PROMPTCTL_ENHANCE_URL", "") // unset => use default hosted URL

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg.EnhanceURL != DefaultEnhanceURL {
		t.Errorf("EnhanceURL = %q, want %q when mode=llm and URL unset", cfg.EnhanceURL, DefaultEnhanceURL)
	}
}

func TestDefaultEnhanceURL(t *testing.T) {
	if DefaultEnhanceURL == "" {
		t.Error("DefaultEnhanceURL must be non-empty")
	}
	if len(DefaultEnhanceURL) < 8 || DefaultEnhanceURL[:8] != "https://" {
		t.Errorf("DefaultEnhanceURL must be https: %q", DefaultEnhanceURL)
	}
}
