package scoreconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad_FromDirWithConfig_ReturnsParsedFields(t *testing.T) {
	dir := t.TempDir()
	promptctlDir := filepath.Join(dir, ".promptctl")
	if err := os.MkdirAll(promptctlDir, 0755); err != nil {
		t.Fatal(err)
	}
	scoreYAML := []byte(`dirs:
  - prompts
  - docs
include:
  - "*.txt"
  - "*.md"
  - "*.prompt"
ignore:
  - "vendor/*"
  - ".git/*"
min_score: 70
rules:
  - structure
  - clarity
`)
	scorePath := filepath.Join(promptctlDir, "score.yaml")
	if err := os.WriteFile(scorePath, scoreYAML, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Load(dir)

	wantDirs := []string{"prompts", "docs"}
	if !reflect.DeepEqual(cfg.Dirs, wantDirs) {
		t.Errorf("Dirs = %v, want %v", cfg.Dirs, wantDirs)
	}
	wantInclude := []string{"*.txt", "*.md", "*.prompt"}
	if !reflect.DeepEqual(cfg.Include, wantInclude) {
		t.Errorf("Include = %v, want %v", cfg.Include, wantInclude)
	}
	wantIgnore := []string{"vendor/*", ".git/*"}
	if !reflect.DeepEqual(cfg.Ignore, wantIgnore) {
		t.Errorf("Ignore = %v, want %v", cfg.Ignore, wantIgnore)
	}
	if cfg.MinScore != 70 {
		t.Errorf("MinScore = %d, want 70", cfg.MinScore)
	}
	wantRules := []string{"structure", "clarity"}
	if !reflect.DeepEqual(cfg.Rules, wantRules) {
		t.Errorf("Rules = %v, want %v", cfg.Rules, wantRules)
	}
}

func TestLoad_WhenFileMissing_ReturnsDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	// No .promptctl or score.yaml

	cfg := Load(dir)

	// Default: Include ["*.txt","*.md"], MinScore 80, others empty/nil
	wantInclude := []string{"*.txt", "*.md"}
	if !reflect.DeepEqual(cfg.Include, wantInclude) {
		t.Errorf("Include = %v, want %v (default)", cfg.Include, wantInclude)
	}
	if cfg.MinScore != 80 {
		t.Errorf("MinScore = %d, want 80 (default)", cfg.MinScore)
	}
	if cfg.Dirs != nil && len(cfg.Dirs) != 0 {
		t.Errorf("Dirs = %v, want nil or empty (default)", cfg.Dirs)
	}
	if cfg.Ignore != nil && len(cfg.Ignore) != 0 {
		t.Errorf("Ignore = %v, want nil or empty (default)", cfg.Ignore)
	}
	if cfg.Rules != nil && len(cfg.Rules) != 0 {
		t.Errorf("Rules = %v, want nil or empty (default)", cfg.Rules)
	}
}

func TestLoad_WhenPromptctlDirExistsButNoScoreYAML_ReturnsDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	promptctlDir := filepath.Join(dir, ".promptctl")
	if err := os.MkdirAll(promptctlDir, 0755); err != nil {
		t.Fatal(err)
	}
	// No score.yaml

	cfg := Load(dir)

	wantInclude := []string{"*.txt", "*.md"}
	if !reflect.DeepEqual(cfg.Include, wantInclude) {
		t.Errorf("Include = %v, want %v (default)", cfg.Include, wantInclude)
	}
	if cfg.MinScore != 80 {
		t.Errorf("MinScore = %d, want 80 (default)", cfg.MinScore)
	}
}
