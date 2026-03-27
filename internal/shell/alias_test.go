package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfilePath_Zsh(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	p, err := ProfilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(p, ".zshrc") {
		t.Errorf("expected .zshrc suffix, got %s", p)
	}
}

func TestProfilePath_Bash(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	p, err := ProfilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(p, ".bashrc") {
		t.Errorf("expected .bashrc suffix, got %s", p)
	}
}

func TestProfilePath_DefaultIsBashrc(t *testing.T) {
	t.Setenv("SHELL", "/bin/fish")
	p, err := ProfilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(p, ".bashrc") {
		t.Errorf("expected .bashrc for non-zsh shell, got %s", p)
	}
}

func TestHasPromptctlAliasBlock_NoFile(t *testing.T) {
	got, err := HasPromptctlAliasBlock("/nonexistent/path/profile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false for nonexistent file")
	}
}

func TestHasPromptctlAliasBlock_WithBlock(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	content := "# some stuff\n" + promptctlBlockStart + "\nalias prompt='promptctl'\n" + promptctlBlockEnd + "\n"
	if err := os.WriteFile(profile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := HasPromptctlAliasBlock(profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected true when block is present")
	}
}

func TestHasPromptctlAliasBlock_WithoutBlock(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(profile, []byte("# just some config\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := HasPromptctlAliasBlock(profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false when block is absent")
	}
}

func TestAliasExists_NoFile(t *testing.T) {
	got, err := AliasExists("/nonexistent/path/profile", "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false for nonexistent file")
	}
}

func TestAliasExists_AliasPresent(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	content := "alias prompt='promptctl'\n"
	if err := os.WriteFile(profile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := AliasExists(profile, "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected true when alias is present")
	}
}

func TestAliasExists_AliasAbsent(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(profile, []byte("# empty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := AliasExists(profile, "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false when alias is absent")
	}
}

func TestAddAliases_NewFile(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	if err := AddAliases(profile, "prompt", "p"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !strings.Contains(s, "alias prompt='promptctl'") {
		t.Error("expected long alias in profile")
	}
	if !strings.Contains(s, "alias p='promptctl'") {
		t.Error("expected short alias in profile")
	}
	if !strings.Contains(s, promptctlBlockStart) {
		t.Error("expected block start marker")
	}
	if !strings.Contains(s, promptctlBlockEnd) {
		t.Error("expected block end marker")
	}
}

func TestAddAliases_SameAlias_SkipsShort(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	if err := AddAliases(profile, "prompt", "prompt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	count := strings.Count(string(content), "alias prompt='promptctl'")
	if count != 1 {
		t.Errorf("expected exactly 1 alias line when long==short, got %d", count)
	}
}

func TestAddAliases_EmptyShort(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	if err := AddAliases(profile, "prompt", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(content), "alias ") != 1 {
		t.Error("expected only one alias line when short is empty")
	}
}

func TestAddAliases_ExistingContent(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	existing := "export PATH=/usr/bin\n"
	if err := os.WriteFile(profile, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	if err := AddAliases(profile, "prompt", "p"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !strings.Contains(s, "export PATH=/usr/bin") {
		t.Error("existing content should be preserved")
	}
	if !strings.Contains(s, "alias prompt='promptctl'") {
		t.Error("expected alias to be added")
	}
}

func TestRemovePromptctlBlock_NoBlock(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	original := "export PATH=/usr/bin\n"
	if err := os.WriteFile(profile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RemovePromptctlBlock(profile); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "export PATH=/usr/bin") {
		t.Error("content should be unchanged when no block present")
	}
}

func TestRemovePromptctlBlock_WithBlock(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	content := "export PATH=/usr/bin\n" + promptctlBlockStart + "\nalias prompt='promptctl'\n" + promptctlBlockEnd + "\nmore stuff\n"
	if err := os.WriteFile(profile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RemovePromptctlBlock(profile); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(result)
	if strings.Contains(s, promptctlBlockStart) {
		t.Error("block start should be removed")
	}
	if strings.Contains(s, "alias prompt") {
		t.Error("alias line should be removed")
	}
}

func TestRemovePromptctlBlock_NonexistentFile(t *testing.T) {
	err := RemovePromptctlBlock("/nonexistent/path/profile")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestAddAndRemoveRoundtrip(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, ".zshrc")
	original := "export FOO=bar\n"
	if err := os.WriteFile(profile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	if err := AddAliases(profile, "prompt", "p"); err != nil {
		t.Fatal(err)
	}
	has, _ := HasPromptctlAliasBlock(profile)
	if !has {
		t.Fatal("block should exist after AddAliases")
	}
	if err := RemovePromptctlBlock(profile); err != nil {
		t.Fatal(err)
	}
	has, _ = HasPromptctlAliasBlock(profile)
	if has {
		t.Error("block should be removed after RemovePromptctlBlock")
	}
}
