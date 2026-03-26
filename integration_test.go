package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oleg-koval/promptctl/cmd"
)

// --- Integration Test Infrastructure ---
//
// These tests exercise real user workflows end-to-end.
// They use two approaches:
// 1. Subprocess via `go run .` for full CLI behavior (slower)
// 2. In-process via cmd.Execute() for fast tests (shares process state)

// testEnv holds the temporary directory structure for an integration test.
type testEnv struct {
	t           *testing.T
	homeDir     string
	workDir     string
	templateDir string
	repoRoot    string
}

// setupTestEnv creates a temp directory structure with templates and
// test source files, mimicking a real user environment.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	home := t.TempDir()
	workDir := t.TempDir()
	repoRoot := findRepoRoot(t)

	// Create global template directory
	templateDir := filepath.Join(home, ".promptctl", "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}

	// Copy integration test templates from testdata
	srcDir := filepath.Join(repoRoot, "testdata", "integration")
	for _, name := range []string{"review.yaml", "debug.yaml", "arch.yaml"} {
		data, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			t.Fatalf("read template %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(templateDir, name), data, 0644); err != nil {
			t.Fatalf("write template %s: %v", name, err)
		}
	}

	// Copy test source files into workDir
	for _, name := range []string{"auth.ts", "api.go", "handler.py"} {
		data, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			t.Fatalf("read file %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workDir, name), data, 0644); err != nil {
			t.Fatalf("write file %s: %v", name, err)
		}
	}

	t.Setenv("HOME", home)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")

	return &testEnv{
		t:           t,
		homeDir:     home,
		workDir:     workDir,
		templateDir: templateDir,
		repoRoot:    repoRoot,
	}
}

// runCLI executes promptctl as a subprocess via `go run .` and returns
// stdout, stderr, and exit code separately.
func runCLI(t *testing.T, env *testEnv, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	goArgs := append([]string{"run", env.repoRoot}, args...)
	c := exec.Command("go", goArgs...)
	c.Dir = env.workDir
	c.Env = append(os.Environ(),
		"HOME="+env.homeDir,
		"PROMPTCTL_ENHANCE=rule",
	)

	var outBuf, errBuf bytes.Buffer
	c.Stdout = &outBuf
	c.Stderr = &errBuf

	err := c.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run promptctl: %v", err)
		}
	}

	return outBuf.String(), errBuf.String(), code
}

// runInProcess calls cmd.Execute() directly, manipulating os.Args.
// Faster than subprocess. Captures stdout via pipe.
func runInProcess(t *testing.T, env *testEnv, args ...string) (stdout string, err error) {
	t.Helper()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = append([]string{"promptctl"}, args...)

	// Change working directory for file-relative operations
	oldWd, _ := os.Getwd()
	if env.workDir != "" {
		if err := os.Chdir(env.workDir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		defer os.Chdir(oldWd)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("pipe: %v", pipeErr)
	}
	os.Stdout = w

	execErr := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	r.Close()

	return buf.String(), execErr
}

// assertContains checks that output contains all expected substrings.
func assertContains(t *testing.T, output string, expected ...string) {
	t.Helper()
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("output missing %q\nGot:\n%s", exp, truncate(output, 500))
		}
	}
}

// assertNotContains checks that output does NOT contain any of the given substrings.
func assertNotContains(t *testing.T, output string, unexpected ...string) {
	t.Helper()
	for _, u := range unexpected {
		if strings.Contains(output, u) {
			t.Errorf("output should not contain %q\nGot:\n%s", u, truncate(output, 500))
		}
	}
}

// findRepoRoot walks up from cwd to find go.mod.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

// truncate shortens strings for error display.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}

// --- Integration Tests ---

func TestIntegration_RunReviewTemplate(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("run review failed: %v", err)
	}

	// Template should render with file content injected
	assertContains(t, out, "auth.ts")       // file_name variable
	assertContains(t, out, "ts")            // file_ext variable
	assertContains(t, out, "generateToken") // file content from auth.ts
	assertContains(t, out, "code reviewer") // template body text
}

func TestIntegration_RunReviewWithFocus(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts", "--focus=security")
	if err != nil {
		t.Fatalf("run review with focus failed: %v", err)
	}

	assertContains(t, out, "security") // focus variable should appear
	assertContains(t, out, "auth.ts")
}

func TestIntegration_RunDebugTemplate(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "debug", "--file=api.go", "--error=TypeError: Cannot read property")
	if err != nil {
		t.Fatalf("run debug failed: %v", err)
	}

	assertContains(t, out, "TypeError: Cannot read property")
	assertContains(t, out, "api.go")
	assertContains(t, out, "HandleHealth") // from the api.go file content
}

func TestIntegration_RunArchTemplate(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "arch", "--problem=Should we use event sourcing?")
	if err != nil {
		t.Fatalf("run arch failed: %v", err)
	}

	assertContains(t, out, "event sourcing")
	assertContains(t, out, "software architect")
}

func TestIntegration_RunTemplateFileNotFound(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "run", "review", "--file=nonexistent.ts")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "nonexistent.ts") {
		t.Errorf("error should mention filename, got: %v", err)
	}
}

func TestIntegration_RunTemplateMissing(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "run", "nonexistent-template")
	if err == nil {
		t.Fatal("expected error for missing template")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestIntegration_RunMissingRequiredVar(t *testing.T) {
	env := setupTestEnv(t)

	// debug template requires both --file and --error
	_, err := runInProcess(t, env, "run", "debug", "--file=auth.ts")
	if err == nil {
		t.Fatal("expected error for missing required variable")
	}
	if !strings.Contains(err.Error(), "error") {
		t.Errorf("error should mention missing 'error' variable, got: %v", err)
	}
}

func TestIntegration_ListTemplates(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	assertContains(t, out, "review")
	assertContains(t, out, "debug")
	assertContains(t, out, "arch")
}

func TestIntegration_ShowVars(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "vars", "review")
	if err != nil {
		t.Fatalf("vars failed: %v", err)
	}

	assertContains(t, out, "file")
}

func TestIntegration_HelpCommand(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "help")
	if err != nil {
		t.Fatalf("help failed: %v", err)
	}

	assertContains(t, out, "promptctl")
	assertContains(t, out, "USAGE")
}

func TestIntegration_VersionCommand(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}

	assertContains(t, out, "promptctl v")
}

func TestIntegration_InitGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")

	env := &testEnv{
		t:        t,
		homeDir:  home,
		workDir:  t.TempDir(),
		repoRoot: findRepoRoot(t),
	}

	out, err := runInProcess(t, env, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	assertContains(t, out, "Initialized")

	// Verify templates were created
	templateDir := filepath.Join(home, ".promptctl", "templates")
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		t.Fatalf("read template dir: %v", err)
	}

	if len(entries) < 3 {
		t.Errorf("expected at least 3 starter templates, got %d", len(entries))
	}

	// Verify templates are valid (non-empty)
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(templateDir, entry.Name()))
		if err != nil {
			t.Errorf("read template %s: %v", entry.Name(), err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("template %s is empty", entry.Name())
		}
	}
}

func TestIntegration_InitLocal(t *testing.T) {
	home := t.TempDir()
	workDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")

	env := &testEnv{
		t:        t,
		homeDir:  home,
		workDir:  workDir,
		repoRoot: findRepoRoot(t),
	}

	out, err := runInProcess(t, env, "init", "--local")
	if err != nil {
		t.Fatalf("init --local failed: %v", err)
	}

	assertContains(t, out, "local")

	// Verify local .promptctl/templates/ was created
	localDir := filepath.Join(workDir, ".promptctl", "templates")
	if _, err := os.Stat(localDir); os.IsNotExist(err) {
		t.Error("local template directory should exist after init --local")
	}
}

func TestIntegration_CreateWithRuleEnhance(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "create", "analyze my SaaS pricing strategy")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Rule-based enhance should produce structured output
	if len(out) < 50 {
		t.Errorf("create output too short (%d chars), expected structured prompt", len(out))
	}
}

func TestIntegration_ScoreCommand(t *testing.T) {
	env := setupTestEnv(t)

	// Create a test file to score in workDir
	testPrompt := filepath.Join(env.workDir, "test-prompt.txt")
	content := `<context>You are a code reviewer.</context>
<task>Review the code for security issues.</task>
<constraints>Be specific. Reference line numbers.</constraints>`
	os.WriteFile(testPrompt, []byte(content), 0644)

	out, err := runInProcess(t, env, "score", "test-prompt.txt")
	if err != nil {
		t.Fatalf("score failed: %v", err)
	}

	// Score output should contain the filename and a numeric score
	assertContains(t, out, "test-prompt.txt")
	// Score is a number (e.g. "90") - verify output is non-empty
	if len(strings.TrimSpace(out)) == 0 {
		t.Error("score output should not be empty")
	}
}

func TestIntegration_CostEstimate(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "cost", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("cost failed: %v", err)
	}

	assertContains(t, out, "$") // should show cost
}

func TestIntegration_CostCompare(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "cost", "--compare", "analyze my startup")
	if err != nil {
		t.Fatalf("cost --compare failed: %v", err)
	}

	assertContains(t, out, "$") // cost comparison output
}

func TestIntegration_ModelsCommand(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "models")
	if err != nil {
		t.Fatalf("models failed: %v", err)
	}

	// Should list at least some models
	assertContains(t, out, "claude")
}

func TestIntegration_ShowTemplate(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "show", "review")
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}

	assertContains(t, out, "review")
}

func TestIntegration_NoArgsShowsUsage(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env)
	if err != nil {
		t.Fatalf("no args failed: %v", err)
	}

	assertContains(t, out, "USAGE")
}

func TestIntegration_RunTemplateByShortcut(t *testing.T) {
	env := setupTestEnv(t)

	// Unknown commands are treated as template names
	out, err := runInProcess(t, env, "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("review shortcut failed: %v", err)
	}

	// Should render the review template
	assertContains(t, out, "auth.ts")
	assertContains(t, out, "generateToken")
}

func TestIntegration_PythonFileReview(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=handler.py")
	if err != nil {
		t.Fatalf("run review python file failed: %v", err)
	}

	assertContains(t, out, "handler.py")
	assertContains(t, out, "py")  // file_ext
	assertContains(t, out, "app") // from handler.py content
}

func TestIntegration_DefaultFocusApplied(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("run review default focus failed: %v", err)
	}

	// Default focus is "general" per the template
	assertContains(t, out, "general")
}
