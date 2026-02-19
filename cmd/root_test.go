package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oleg-koval/promptctl/config"
)

func TestExecute_Version(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "version"}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestExecute_Help(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "help"}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestExecute_Create(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "create", "analyze my startup idea"}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestExecute_CreateRule_StdoutContainsContext(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")
	defer func() {
		os.Unsetenv("HOME")
		os.Unsetenv("PROMPTCTL_ENHANCE")
	}()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "create", "review my code"}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout; w.Close() }()

	err = Execute()
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()
	if !strings.Contains(out, "<context>") && !strings.Contains(out, "## Context") {
		t.Errorf("stdout should contain <context> or ## Context when using rule enhance; got:\n%s", out)
	}
}

func TestExecute_NoArgs(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl"}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() with no args should not error: %v", err)
	}
}

func TestExecute_List(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	globalDir := filepath.Join(dir, ".promptctl", "templates")
	_ = os.MkdirAll(globalDir, 0755)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "list"}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() list err = %v", err)
	}
}

func TestExecute_CostCompare(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "cost", "--compare", "analyze my startup"}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() cost --compare err = %v", err)
	}
}

func TestExecute_Models(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "models"}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() models err = %v", err)
	}
}

func TestExecute_Vars(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	// Need at least one template
	tplDir := filepath.Join(dir, ".promptctl", "templates")
	_ = os.MkdirAll(tplDir, 0755)
	_ = os.WriteFile(filepath.Join(tplDir, "review.yaml"), []byte("name: review\ndescription: x\nvariables:\n  - name: file\nbody: |\n  x"), 0644)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "vars", "review"}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() vars err = %v", err)
	}
}

func TestExecute_Memory_NoSubcommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "memory"}
	err := Execute()
	if err == nil {
		t.Fatal("Execute() memory with no subcommand should error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error should contain usage: %v", err)
	}
}

func TestExecute_MemoryList_Empty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	_ = os.MkdirAll(filepath.Join(dir, ".promptctl", "templates"), 0755)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "memory", "list"}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() memory list err = %v", err)
	}
}

func TestExecute_MemoryList_WithPrompts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	promptsDir := filepath.Join(dir, ".promptctl", "templates")
	_ = os.MkdirAll(promptsDir, 0755)
	_ = os.WriteFile(filepath.Join(promptsDir, "saved.yaml"), []byte("name: saved\ndescription: x\nbody: |\n  x"), 0644)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "memory", "list"}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() memory list err = %v", err)
	}
}

func TestExecute_MemorySetDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	customDir := filepath.Join(dir, "my-prompts")
	_ = os.MkdirAll(filepath.Join(dir, ".promptctl"), 0755)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "memory", "set-dir", customDir}
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() memory set-dir err = %v", err)
	}
	// Verify persisted: Load config and check PromptsDir
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() after set-dir err = %v", err)
	}
	abs, _ := filepath.Abs(customDir)
	if cfg.PromptsDir != abs {
		t.Errorf("PromptsDir = %q, want %q", cfg.PromptsDir, abs)
	}
}

func TestExecute_Savings_WithModel_UsesSpecifiedModel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	_ = os.MkdirAll(filepath.Join(dir, ".promptctl"), 0755)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "savings", "--model=claude-haiku-4-5-20251001"}
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout; w.Close() }()
	err := Execute()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()
	if err != nil {
		t.Fatalf("Execute() savings --model err = %v", err)
	}
	if !strings.Contains(out, "Haiku") && !strings.Contains(out, "claude-haiku") {
		t.Errorf("savings --model=claude-haiku should show model name; got:\n%s", out)
	}
}

func TestExecute_Help_ContainsNewUserSavingsHint(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "help"}
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout; w.Close() }()
	_ = Execute()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()
	if !strings.Contains(out, "New?") || !strings.Contains(out, "savings") {
		t.Errorf("help should contain New? and savings hint; got:\n%s", out)
	}
}

func TestExecute_Run_RequiredVarError_SuggestsVars(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	tplDir := filepath.Join(dir, ".promptctl", "templates")
	_ = os.MkdirAll(tplDir, 0755)
	_ = os.WriteFile(filepath.Join(tplDir, "review.yaml"), []byte("name: review\ndescription: x\nvariables:\n  - name: file\n    required: true\nbody: |\n  x"), 0644)
	defer os.Unsetenv("HOME")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "run", "review"}
	err := Execute()
	if err == nil {
		t.Fatal("Execute() run review without --file should error")
	}
	if !strings.Contains(err.Error(), "promptctl vars") {
		t.Errorf("error should suggest promptctl vars; got: %v", err)
	}
}

func TestExecute_Help_ContainsMemorySavedFromCreate(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "help"}
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout; w.Close() }()
	_ = Execute()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()
	if !strings.Contains(out, "saved from create") && !strings.Contains(out, "Prompts you saved") {
		t.Errorf("help MEMORY section should mention saved from create; got:\n%s", out)
	}
}
