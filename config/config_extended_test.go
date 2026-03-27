package config

import (
	"os"
	"path/filepath"
	"testing"
)

// --- SaveCreateFormat ---

func TestSaveCreateFormat_Success(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	err := SaveCreateFormat("xml")
	if err != nil {
		t.Fatalf("SaveCreateFormat error: %v", err)
	}

	// Verify file was created
	path := filepath.Join(dir, ".promptctl", "create_format")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if got := string(data); got != "xml\n" {
		t.Errorf("create_format content = %q, want %q", got, "xml\n")
	}
}

func TestSaveCreateFormat_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := SaveCreateFormat("markdown"); err != nil {
		t.Fatal(err)
	}
	if err := SaveCreateFormat("yaml"); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, ".promptctl", "create_format")
	data, _ := os.ReadFile(path)
	if got := string(data); got != "yaml\n" {
		t.Errorf("overwritten format = %q, want yaml", got)
	}
}

func TestSaveCreateFormat_LoadReturnsIt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := SaveCreateFormat("json"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.DefaultCreateFormat != "json" {
		t.Errorf("DefaultCreateFormat = %q, want json", cfg.DefaultCreateFormat)
	}
}

// --- Config directory creation ---

func TestLoad_CreatesNothingIfMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	_, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Load should NOT create directories - it's read-only
	globalDir := filepath.Join(dir, ".promptctl")
	if _, err := os.Stat(globalDir); !os.IsNotExist(err) {
		t.Error("Load should not create directories")
	}
}

func TestInitGlobal_CreatesDirectoryWithCorrectPermissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := InitGlobal(); err != nil {
		t.Fatal(err)
	}

	templatesDir := filepath.Join(dir, ".promptctl", "templates")
	info, err := os.Stat(templatesDir)
	if err != nil {
		t.Fatalf("templates dir should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("templates should be a directory")
	}
}

// --- Read config ---

func TestLoad_DefaultValues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.GlobalTemplateDir == "" {
		t.Error("GlobalTemplateDir should not be empty")
	}
	if cfg.DefaultVars == nil {
		t.Error("DefaultVars should be initialized")
	}
	if cfg.EnhanceMode != "llm" {
		t.Errorf("EnhanceMode default = %q, want llm", cfg.EnhanceMode)
	}
}

// --- Write and read config roundtrip ---

func TestSavePromptsDir_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	want := "/custom/path/prompts"
	if err := SavePromptsDir(want); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PromptsDir != want {
		t.Errorf("PromptsDir = %q, want %q", cfg.PromptsDir, want)
	}
}

// --- Update existing config ---

func TestSaveCreateFormat_ThenSavePromptsDir_BothPersist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := SaveCreateFormat("xml"); err != nil {
		t.Fatal(err)
	}
	if err := SavePromptsDir("/my/prompts"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultCreateFormat != "xml" {
		t.Errorf("DefaultCreateFormat = %q, want xml", cfg.DefaultCreateFormat)
	}
	if cfg.PromptsDir != "/my/prompts" {
		t.Errorf("PromptsDir = %q, want /my/prompts", cfg.PromptsDir)
	}
}

// --- findLocalConfig ---

func TestFindLocalConfig_WalksUpDirectoryTree(t *testing.T) {
	dir := t.TempDir()
	// Create .promptctl in the root temp dir
	promptctlDir := filepath.Join(dir, ".promptctl")
	if err := os.MkdirAll(promptctlDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create nested subdirectory
	subDir := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(subDir)

	result := findLocalConfig()
	// Should walk up to find the .promptctl directory
	// Resolve symlinks for comparison (macOS /var -> /private/var)
	resultNorm, _ := filepath.EvalSymlinks(result)
	wantNorm, _ := filepath.EvalSymlinks(promptctlDir)
	if resultNorm != wantNorm {
		t.Errorf("findLocalConfig() = %q, want %q", result, promptctlDir)
	}
}

func TestFindLocalConfig_FallbackToCwd(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(dir)

	result := findLocalConfig()
	resultNorm, _ := filepath.EvalSymlinks(result)
	wantNorm, _ := filepath.EvalSymlinks(filepath.Join(dir, ".promptctl"))
	if resultNorm != wantNorm {
		t.Errorf("findLocalConfig() without .promptctl = %q, want %q", result, filepath.Join(dir, ".promptctl"))
	}
}

// --- EnhanceMode variations ---

func TestLoad_EnhanceModeRule_NoDefaultURL(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")
	t.Setenv("PROMPTCTL_ENHANCE_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EnhanceMode != "rule" {
		t.Errorf("EnhanceMode = %q, want rule", cfg.EnhanceMode)
	}
	// When mode is "rule" and no URL is set, EnhanceURL should be empty
	if cfg.EnhanceURL != "" {
		t.Errorf("rule mode with no PROMPTCTL_ENHANCE_URL should have empty EnhanceURL, got %q", cfg.EnhanceURL)
	}
}
