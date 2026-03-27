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
	if cfg.PromptsDir == "" {
		t.Error("PromptsDir empty")
	}
	if cfg.PromptsDir != cfg.GlobalTemplateDir {
		t.Errorf("PromptsDir should default to GlobalTemplateDir; got %q", cfg.PromptsDir)
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
	// In the open-source version, DefaultEnhanceURL is empty.
	// Users can set PROMPTCTL_ENHANCE_URL to point to their own backend.
	if DefaultEnhanceURL != "" {
		t.Errorf("DefaultEnhanceURL should be empty in OSS build, got %q", DefaultEnhanceURL)
	}
}

func TestLoad_PromptsDirFromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("PROMPTCTL_PROMPTS_DIR", "/custom/prompts")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg.PromptsDir != "/custom/prompts" {
		t.Errorf("PromptsDir = %q, want /custom/prompts", cfg.PromptsDir)
	}
}

func TestLoad_PromptsDirFromFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	globalDir := filepath.Join(dir, ".promptctl")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	promptsDirPath := filepath.Join(globalDir, "prompts_dir")
	if err := os.WriteFile(promptsDirPath, []byte("/saved/prompts/dir\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg.PromptsDir != "/saved/prompts/dir" {
		t.Errorf("PromptsDir = %q, want /saved/prompts/dir", cfg.PromptsDir)
	}
}

func TestSavePromptsDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	want := filepath.Join(dir, "my-prompts")

	err := SavePromptsDir(want)
	if err != nil {
		t.Fatalf("SavePromptsDir() err = %v", err)
	}
	path := filepath.Join(dir, ".promptctl", "prompts_dir")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(prompts_dir) err = %v", err)
	}
	if got := string(data); got != want+"\n" {
		t.Errorf("prompts_dir content = %q, want %q", got, want+"\n")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() after SavePromptsDir err = %v", err)
	}
	if cfg.PromptsDir != want {
		t.Errorf("Load() PromptsDir = %q, want %q", cfg.PromptsDir, want)
	}
}
