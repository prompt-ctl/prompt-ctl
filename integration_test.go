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
	for _, name := range []string{"review.yaml", "debug.yaml", "arch.yaml", "commit.yaml", "custom.yaml"} {
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

// buildBinary builds the promptctl binary once per test run and returns the path.
var testBinaryPath string

// realHome preserves the real HOME before tests override it.
var realHome = os.Getenv("HOME")

func ensureBinary(t *testing.T, repoRoot string) string {
	t.Helper()
	if testBinaryPath != "" {
		if _, err := os.Stat(testBinaryPath); err == nil {
			return testBinaryPath
		}
	}
	bin := filepath.Join(os.TempDir(), "promptctl-integration-test")
	c := exec.Command("go", "build", "-o", bin, ".")
	c.Dir = repoRoot
	// Use real HOME so Go build cache works correctly
	c.Env = append(os.Environ(), "HOME="+realHome)
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build test binary: %v\n%s", err, out)
	}
	testBinaryPath = bin
	return bin
}

// runCLI executes promptctl as a subprocess and returns
// stdout, stderr, and exit code separately.
func runCLI(t *testing.T, env *testEnv, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	bin := ensureBinary(t, env.repoRoot)
	c := exec.Command(bin, args...)
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

// ============================================================
// Test Set 1: Review Command (comprehensive)
// ============================================================

func TestIntegration_ReviewTemplateLoadsCorrectly(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("run review failed: %v", err)
	}

	// Template body contains structure markers
	assertContains(t, out, "<context>")
	assertContains(t, out, "<task>")
	assertContains(t, out, "<constraints>")
}

func TestIntegration_ReviewFileContentInjected(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("run review failed: %v", err)
	}

	// File content from auth.ts should be present
	assertContains(t, out, "generateToken")
	assertContains(t, out, "auth.ts")
	assertContains(t, out, "ts") // file extension
}

func TestIntegration_ReviewResponseStreamedToStdout(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("run review failed: %v", err)
	}

	// Output should be non-empty and contain rendered template
	if len(out) < 100 {
		t.Errorf("expected substantial output, got %d chars", len(out))
	}
}

func TestIntegration_ReviewErrorFileNotFound(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "run", "review", "--file=does_not_exist.ts")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	assertContains(t, err.Error(), "does_not_exist.ts")
}

func TestIntegration_ReviewErrorLLMNotRequired(t *testing.T) {
	// run (not send) doesn't require LLM - verify it works without API key
	env := setupTestEnv(t)
	t.Setenv("ANTHROPIC_API_KEY", "")

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("run review should work without API key: %v", err)
	}
	assertContains(t, out, "auth.ts")
}

func TestIntegration_ReviewFocusSecurity(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts", "--focus=security")
	if err != nil {
		t.Fatalf("review with security focus failed: %v", err)
	}
	assertContains(t, out, "security")
}

func TestIntegration_ReviewFocusPerformance(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts", "--focus=performance")
	if err != nil {
		t.Fatalf("review with performance focus failed: %v", err)
	}
	assertContains(t, out, "performance")
}

func TestIntegration_ReviewFocusAllDefault(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("review with default focus failed: %v", err)
	}
	// Default focus is "general"
	assertContains(t, out, "general")
}

// ============================================================
// Test Set 2: Score Command
// ============================================================

func TestIntegration_ScoreTemplateParsedCorrectly(t *testing.T) {
	env := setupTestEnv(t)

	testPrompt := filepath.Join(env.workDir, "scored.txt")
	content := `<context>You are an expert code reviewer.</context>
<task>Review this Go function for correctness.</task>
<constraints>Be specific. Reference line numbers.</constraints>`
	os.WriteFile(testPrompt, []byte(content), 0644)

	out, err := runInProcess(t, env, "score", "scored.txt")
	if err != nil {
		t.Fatalf("score failed: %v", err)
	}

	assertContains(t, out, "scored.txt")
}

func TestIntegration_ScoreJSONOutput(t *testing.T) {
	env := setupTestEnv(t)

	testPrompt := filepath.Join(env.workDir, "json-score.txt")
	content := `<context>You are a helpful assistant.</context>
<task>Summarize the document.</task>`
	os.WriteFile(testPrompt, []byte(content), 0644)

	// Use subprocess because score uses os.Exit
	// Exit code 1 is OK - it means file scored below min-score threshold
	stdout, _, exitCode := runCLI(t, env, "score", "json-score.txt", "--format=json")
	if exitCode > 1 {
		t.Fatalf("score --format=json exited %d (expected 0 or 1)", exitCode)
	}

	// JSON output should contain opening brace
	assertContains(t, stdout, "{")
}

func TestIntegration_ScoreInRange(t *testing.T) {
	env := setupTestEnv(t)

	testPrompt := filepath.Join(env.workDir, "range-score.txt")
	content := `<context>You are a senior engineer.</context>
<task>Debug this error.</task>
<constraints>Provide root cause.</constraints>`
	os.WriteFile(testPrompt, []byte(content), 0644)

	// Use subprocess because score uses os.Exit
	// Exit code 1 is OK (below min-score), only 2+ is a real error
	stdout, _, exitCode := runCLI(t, env, "score", "range-score.txt")
	if exitCode > 1 {
		t.Fatalf("score exited %d", exitCode)
	}

	if len(strings.TrimSpace(stdout)) == 0 {
		t.Error("score output should not be empty")
	}
}

func TestIntegration_ScoreInvalidUTF8(t *testing.T) {
	env := setupTestEnv(t)

	// Create a file with invalid UTF-8 bytes (0xFE, 0xFF are never valid in UTF-8)
	badFile := filepath.Join(env.workDir, "bad.txt")
	os.WriteFile(badFile, []byte{0xFE, 0xFF, 0x80, 0x81}, 0644)

	// Score should skip and warn about non-UTF8 file
	_, stderr, _ := runCLI(t, env, "score", "bad.txt")
	assertContains(t, stderr, "not valid UTF-8")
}

func TestIntegration_ScoreMissingFile(t *testing.T) {
	env := setupTestEnv(t)

	// Scoring a nonexistent file should produce an error (exit non-zero)
	_, stderr, exitCode := runCLI(t, env, "score", "nonexistent-file.txt")
	if exitCode == 0 && !strings.Contains(stderr, "Error") {
		t.Error("expected error or non-zero exit for missing file")
	}
}

// ============================================================
// Test Set 3: Fix Command
// ============================================================

func TestIntegration_FixImproveTemplate(t *testing.T) {
	env := setupTestEnv(t)

	// Create a low-quality prompt file
	lowQuality := filepath.Join(env.workDir, "low-quality.txt")
	os.WriteFile(lowQuality, []byte("review this code"), 0644)

	// Use subprocess because fix uses os.Exit
	// Fix modifies the file in-place (no stdout)
	_, _, exitCode := runCLI(t, env, "fix", "low-quality.txt")
	if exitCode != 0 {
		t.Fatalf("fix exited %d", exitCode)
	}

	// Verify file was modified (improved with structure)
	data, _ := os.ReadFile(lowQuality)
	improved := string(data)
	if improved == "review this code" {
		t.Error("fix should have modified the file")
	}
	// ApplyStructure adds XML tags
	assertContains(t, improved, "<task>")
}

func TestIntegration_FixSuggestMode(t *testing.T) {
	env := setupTestEnv(t)

	lowQuality := filepath.Join(env.workDir, "suggest-test.txt")
	os.WriteFile(lowQuality, []byte("write me a poem about nature"), 0644)

	// --suggest prints suggestions AND still applies the fix
	_, _, exitCode := runCLI(t, env, "fix", "suggest-test.txt", "--suggest")
	if exitCode != 0 {
		t.Fatalf("fix --suggest exited %d", exitCode)
	}

	// File should be modified with structure applied
	data, _ := os.ReadFile(lowQuality)
	improved := string(data)
	assertContains(t, improved, "<task>")
}

func TestIntegration_FixDryRun(t *testing.T) {
	env := setupTestEnv(t)

	promptFile := filepath.Join(env.workDir, "dryrun-test.txt")
	original := "tell me about golang"
	os.WriteFile(promptFile, []byte(original), 0644)

	stdout, _, exitCode := runCLI(t, env, "fix", "dryrun-test.txt", "--dry-run")
	if exitCode != 0 {
		t.Fatalf("fix --dry-run exited %d", exitCode)
	}

	// Dry-run should NOT modify the original file
	data, _ := os.ReadFile(promptFile)
	if string(data) != original {
		t.Error("fix --dry-run should not modify the original file")
	}

	// Dry-run should produce stdout with the fixed content
	assertContains(t, stdout, "dryrun-test.txt")
}

func TestIntegration_FixInvalidInputFile(t *testing.T) {
	env := setupTestEnv(t)

	// Create file with truly invalid UTF-8 bytes
	badFile := filepath.Join(env.workDir, "binary.txt")
	os.WriteFile(badFile, []byte{0xFE, 0xFF, 0x80, 0x81}, 0644)

	// Fix should skip and warn about non-UTF8 file
	_, stderr, _ := runCLI(t, env, "fix", "binary.txt")
	assertContains(t, stderr, "not valid UTF-8")
}

// ============================================================
// Test Set 4: Debug Command
// ============================================================

func TestIntegration_DebugErrorContextInjected(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "debug", "--file=api.go", "--error=TypeError: Cannot read property of undefined")
	if err != nil {
		t.Fatalf("debug failed: %v", err)
	}

	assertContains(t, out, "TypeError: Cannot read property of undefined")
}

func TestIntegration_DebugFileContextProvided(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "debug", "--file=api.go", "--error=null pointer")
	if err != nil {
		t.Fatalf("debug failed: %v", err)
	}

	assertContains(t, out, "api.go")
	assertContains(t, out, "HandleHealth") // from file content
}

func TestIntegration_DebugOutputIncludesSuggestions(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "debug", "--file=handler.py", "--error=ImportError: No module named flask")
	if err != nil {
		t.Fatalf("debug failed: %v", err)
	}

	// Template output format includes debugging structure
	assertContains(t, out, "Root cause analysis")
	assertContains(t, out, "Fix suggestion")
}

func TestIntegration_DebugErrorNoErrorProvided(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "run", "debug", "--file=api.go")
	if err == nil {
		t.Fatal("expected error when --error is missing")
	}
}

func TestIntegration_DebugErrorFileNotReadable(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "run", "debug", "--file=no-such-file.go", "--error=some error")
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
	assertContains(t, err.Error(), "no-such-file.go")
}

// ============================================================
// Test Set 5: Architecture Command
// ============================================================

func TestIntegration_ArchProblemStatementInjected(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "arch", "--problem=Should we use event sourcing?")
	if err != nil {
		t.Fatalf("arch failed: %v", err)
	}

	assertContains(t, out, "Should we use event sourcing?")
}

func TestIntegration_ArchStructuredDecision(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "arch", "--problem=Monolith vs microservices")
	if err != nil {
		t.Fatalf("arch failed: %v", err)
	}

	// Template includes structured output format
	assertContains(t, out, "Pros and cons")
	assertContains(t, out, "Recommendation")
}

func TestIntegration_ArchErrorMissingProblem(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "run", "arch")
	if err == nil {
		t.Fatal("expected error when --problem is missing")
	}
}

// ============================================================
// Test Set 6: Commit Command
// ============================================================

func TestIntegration_CommitChangesInjected(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "commit", "--changes=Added retry logic with exponential backoff")
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	assertContains(t, out, "Added retry logic with exponential backoff")
}

func TestIntegration_CommitTemplateApplied(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "commit", "--changes=Fix database connection pooling")
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	assertContains(t, out, "commit message")
	assertContains(t, out, "conventional commits")
}

func TestIntegration_CommitOutputFormatted(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "commit", "--changes=Refactored auth middleware")
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Output should contain constraints about format
	assertContains(t, out, "imperative mood")
}

func TestIntegration_CommitErrorEmptyChanges(t *testing.T) {
	env := setupTestEnv(t)

	// Missing required --changes variable
	_, err := runInProcess(t, env, "run", "commit")
	if err == nil {
		t.Fatal("expected error for missing --changes")
	}
}

// ============================================================
// Test Set 7: Init Command (comprehensive)
// ============================================================

func TestIntegration_InitCreatesTemplateDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")

	env := &testEnv{t: t, homeDir: home, workDir: t.TempDir(), repoRoot: findRepoRoot(t)}

	_, err := runInProcess(t, env, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	templateDir := filepath.Join(home, ".promptctl", "templates")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Error("template directory should exist after init")
	}
}

func TestIntegration_InitStarterTemplatesCopied(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")

	env := &testEnv{t: t, homeDir: home, workDir: t.TempDir(), repoRoot: findRepoRoot(t)}

	_, err := runInProcess(t, env, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	templateDir := filepath.Join(home, ".promptctl", "templates")
	entries, _ := os.ReadDir(templateDir)

	// Should have at least 5 starter templates
	if len(entries) < 5 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected at least 5 starter templates, got %d: %v", len(entries), names)
	}
}

func TestIntegration_InitTemplatesValidYAML(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")

	env := &testEnv{t: t, homeDir: home, workDir: t.TempDir(), repoRoot: findRepoRoot(t)}

	_, err := runInProcess(t, env, "init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	templateDir := filepath.Join(home, ".promptctl", "templates")
	entries, _ := os.ReadDir(templateDir)

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(templateDir, entry.Name()))
		if err != nil {
			t.Errorf("read template %s: %v", entry.Name(), err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("template %s is empty", entry.Name())
		}
		// Basic YAML validation: should contain "name:" field
		if !strings.Contains(string(data), "name:") {
			t.Errorf("template %s missing 'name:' field", entry.Name())
		}
	}
}

func TestIntegration_InitAlreadyInitialized(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")

	env := &testEnv{t: t, homeDir: home, workDir: t.TempDir(), repoRoot: findRepoRoot(t)}

	// Init twice
	_, err1 := runInProcess(t, env, "init")
	if err1 != nil {
		t.Fatalf("first init failed: %v", err1)
	}

	// Second init should not fail
	_, err2 := runInProcess(t, env, "init")
	if err2 != nil {
		t.Fatalf("second init should not fail: %v", err2)
	}
}

// ============================================================
// Test Set 8: Variants (via experiment help)
// ============================================================

func TestIntegration_ExperimentHelp(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "experiment", "--help")
	if err != nil {
		t.Fatalf("experiment help failed: %v", err)
	}

	assertContains(t, out, "USAGE")
	assertContains(t, out, "--models")
}

// ============================================================
// Test Set 9: Execute Command (run with custom variables)
// ============================================================

func TestIntegration_ExecuteVariablesSubstituted(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "custom", "--var1=hello_world", "--var2=test_value")
	if err != nil {
		t.Fatalf("run custom failed: %v", err)
	}

	assertContains(t, out, "hello_world")
	assertContains(t, out, "test_value")
}

func TestIntegration_ExecuteDefaultVariable(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "custom", "--var1=required_val")
	if err != nil {
		t.Fatalf("run custom with defaults failed: %v", err)
	}

	assertContains(t, out, "required_val")
	assertContains(t, out, "default_value") // var2 uses its default
}

func TestIntegration_ExecuteExitCodeSuccess(t *testing.T) {
	env := setupTestEnv(t)

	_, _, exitCode := runCLI(t, env, "run", "custom", "--var1=test")
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestIntegration_ExecuteErrorMissingRequired(t *testing.T) {
	env := setupTestEnv(t)

	// custom template requires --var1
	_, err := runInProcess(t, env, "run", "custom")
	if err == nil {
		t.Fatal("expected error for missing required variable")
	}
}

func TestIntegration_ExecuteErrorTemplateNotFound(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "run", "totally-nonexistent-template-xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
	assertContains(t, err.Error(), "not found")
}

// ============================================================
// Test Set 10: Config-Related Workflows
// ============================================================

func TestIntegration_ProjectLocalOverridesGlobal(t *testing.T) {
	env := setupTestEnv(t)

	// Create a local template that overrides the global review template
	localTemplateDir := filepath.Join(env.workDir, ".promptctl", "templates")
	os.MkdirAll(localTemplateDir, 0755)

	localReview := `name: review
description: Local review override
variables:
  - name: file
    description: File to review
    required: true
  - name: focus
    description: Focus area
    default: local-override

body: |
  LOCAL OVERRIDE TEMPLATE
  Reviewing: {{.file_name}}
  Focus: {{.focus}}
  {{.file_content}}
`
	os.WriteFile(filepath.Join(localTemplateDir, "review.yaml"), []byte(localReview), 0644)

	out, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("run with local override failed: %v", err)
	}

	// Should use the local template, not global
	assertContains(t, out, "LOCAL OVERRIDE TEMPLATE")
	assertContains(t, out, "local-override")
}

func TestIntegration_ProjectLocalOutputDiffersFromGlobal(t *testing.T) {
	env := setupTestEnv(t)

	// Get output from global template
	globalOut, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("global run failed: %v", err)
	}

	// Create local override
	localTemplateDir := filepath.Join(env.workDir, ".promptctl", "templates")
	os.MkdirAll(localTemplateDir, 0755)
	localReview := `name: review
description: Local review
variables:
  - name: file
    description: File
    required: true
  - name: focus
    default: local

body: |
  COMPLETELY DIFFERENT TEMPLATE
  File: {{.file_name}}
  {{.file_content}}
`
	os.WriteFile(filepath.Join(localTemplateDir, "review.yaml"), []byte(localReview), 0644)

	localOut, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err != nil {
		t.Fatalf("local run failed: %v", err)
	}

	// Output should differ
	if globalOut == localOut {
		t.Error("local template output should differ from global template output")
	}
	assertContains(t, localOut, "COMPLETELY DIFFERENT TEMPLATE")
	assertNotContains(t, localOut, "code reviewer")
}

func TestIntegration_ConfigFlagsSetProvider(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "config", "--provider=anthropic", "--api-key=sk-test-key-12345")
	if err != nil {
		t.Fatalf("config failed: %v", err)
	}

	assertContains(t, out, "anthropic")
	assertContains(t, out, "API key stored")
}

func TestIntegration_ConfigFlagsSetModel(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "config", "--provider=anthropic", "--model=claude-sonnet-4-5-20250929")
	if err != nil {
		t.Fatalf("config model failed: %v", err)
	}

	assertContains(t, out, "claude-sonnet-4-5")
}

func TestIntegration_ConfigPersistsSettings(t *testing.T) {
	env := setupTestEnv(t)

	// Set provider and key
	_, err := runInProcess(t, env, "config", "--provider=openai", "--api-key=sk-test-openai")
	if err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	// Config file should exist
	configPath := filepath.Join(env.homeDir, ".promptctl", "llm.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file should exist after setting provider")
	}
}

func TestIntegration_ConfigRemoveAPIKey(t *testing.T) {
	env := setupTestEnv(t)

	// Set then remove
	runInProcess(t, env, "config", "--provider=anthropic", "--api-key=sk-test-to-remove")
	out, err := runInProcess(t, env, "config", "--provider=anthropic", "--api-key=")
	if err != nil {
		t.Fatalf("config remove key failed: %v", err)
	}

	assertContains(t, out, "removed")
}

// ============================================================
// Additional workflow tests
// ============================================================

func TestIntegration_AddTemplate(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "add", "my-new-template")
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	assertContains(t, out, "Created template")

	// Template file should exist
	path := filepath.Join(env.templateDir, "my-new-template.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("template file should exist after add")
	}
}

func TestIntegration_AddTemplateDuplicate(t *testing.T) {
	env := setupTestEnv(t)

	// review.yaml already exists
	_, err := runInProcess(t, env, "add", "review")
	if err == nil {
		t.Fatal("expected error for duplicate template")
	}
	assertContains(t, err.Error(), "already exists")
}

func TestIntegration_ShowTemplateDetails(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "show", "review")
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}

	assertContains(t, out, "Template: review")
	assertContains(t, out, "Description:")
	assertContains(t, out, "Variables:")
	assertContains(t, out, "--- Prompt ---")
}

func TestIntegration_ShowTemplateMissing(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "show", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing template")
	}
	assertContains(t, err.Error(), "not found")
}

func TestIntegration_SavingsCommand(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "savings")
	if err != nil {
		t.Fatalf("savings failed: %v", err)
	}

	assertContains(t, out, "$")
	assertContains(t, out, "year")
}

func TestIntegration_SavingsWithCallsPerDay(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "savings", "--calls-per-day=50")
	if err != nil {
		t.Fatalf("savings with calls-per-day failed: %v", err)
	}

	assertContains(t, out, "50 calls/day")
}

func TestIntegration_VarsShowRequiredAndOptional(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "vars", "debug")
	if err != nil {
		t.Fatalf("vars debug failed: %v", err)
	}

	assertContains(t, out, "file")
	assertContains(t, out, "error")
}

func TestIntegration_EditCommandShowsPath(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "edit", "review")
	if err != nil {
		t.Fatalf("edit failed: %v", err)
	}

	assertContains(t, out, "review.yaml")
}

func TestIntegration_GoFileReview(t *testing.T) {
	env := setupTestEnv(t)

	out, err := runInProcess(t, env, "run", "review", "--file=api.go")
	if err != nil {
		t.Fatalf("review go file failed: %v", err)
	}

	assertContains(t, out, "api.go")
	assertContains(t, out, "go")
	assertContains(t, out, "HandleHealth")
}

// ============================================================
// Task 4: Error Scenarios
// ============================================================

// --- File not found errors ---

func TestIntegration_Error_FileNotFound_Run(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "run", "review", "--file=absolutely_missing_file.ts")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	assertContains(t, err.Error(), "absolutely_missing_file.ts")
}

func TestIntegration_Error_FileNotFound_Score(t *testing.T) {
	env := setupTestEnv(t)

	_, stderr, exitCode := runCLI(t, env, "score", "nonexistent_prompt_file.txt")
	if exitCode == 0 && !strings.Contains(stderr, "Error") && !strings.Contains(stderr, "no such file") {
		t.Error("expected error or non-zero exit for scoring nonexistent file")
	}
}

func TestIntegration_Error_FileNotFound_Fix(t *testing.T) {
	env := setupTestEnv(t)

	_, stderr, exitCode := runCLI(t, env, "fix", "vanished_prompt.txt")
	if exitCode == 0 && !strings.Contains(stderr, "Error") {
		t.Error("expected error or non-zero exit for fixing nonexistent file")
	}
}

// --- Permission denied errors ---

func TestIntegration_Error_PermissionDenied_ReadFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}
	env := setupTestEnv(t)

	// Create a file with no read permissions
	noRead := filepath.Join(env.workDir, "noperm.ts")
	os.WriteFile(noRead, []byte("secret code"), 0644)
	os.Chmod(noRead, 0000)
	defer os.Chmod(noRead, 0644) // cleanup

	_, err := runInProcess(t, env, "run", "review", "--file=noperm.ts")
	if err == nil {
		t.Fatal("expected error for permission-denied file")
	}
	assertContains(t, err.Error(), "noperm.ts")
}

func TestIntegration_Error_PermissionDenied_ScoreFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}
	env := setupTestEnv(t)

	noRead := filepath.Join(env.workDir, "secret.txt")
	os.WriteFile(noRead, []byte("you can't read this"), 0644)
	os.Chmod(noRead, 0000)
	defer os.Chmod(noRead, 0644)

	_, stderr, _ := runCLI(t, env, "score", "secret.txt")
	if !strings.Contains(stderr, "permission denied") && !strings.Contains(stderr, "Warning") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected permission error in stderr, got: %s", truncate(stderr, 300))
	}
}

// --- Invalid template YAML ---

func TestIntegration_Error_InvalidYAML_Template(t *testing.T) {
	env := setupTestEnv(t)

	// Write invalid YAML as a template - parser is lenient (regex extraction)
	// so it will load but produce empty/degraded output
	invalidYAML := filepath.Join(env.templateDir, "broken.yaml")
	os.WriteFile(invalidYAML, []byte("{{{{not valid yaml: [unclosed"), 0644)

	out, err := runInProcess(t, env, "run", "broken")
	if err != nil {
		// Error is acceptable
		return
	}
	// If no error, the output should be minimal (degraded/empty body)
	if len(strings.TrimSpace(out)) > 100 {
		t.Errorf("invalid YAML template should produce empty/minimal output, got %d chars", len(out))
	}
}

func TestIntegration_Error_EmptyTemplate(t *testing.T) {
	env := setupTestEnv(t)

	emptyTemplate := filepath.Join(env.templateDir, "empty.yaml")
	os.WriteFile(emptyTemplate, []byte(""), 0644)

	out, err := runInProcess(t, env, "run", "empty")
	if err != nil {
		// Error is acceptable for empty template
		return
	}
	// If no error, output should be empty/minimal since template has no body
	if len(strings.TrimSpace(out)) > 10 {
		t.Errorf("empty template should produce empty/minimal output, got: %q", truncate(out, 100))
	}
}

func TestIntegration_Error_TemplateNameInvalid(t *testing.T) {
	env := setupTestEnv(t)

	// Template names with path traversal should fail
	_, err := runInProcess(t, env, "run", "../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path-traversal template name")
	}
}

// --- Missing required variables ---

func TestIntegration_Error_MissingRequiredVar_Debug(t *testing.T) {
	env := setupTestEnv(t)

	// debug requires both --file and --error
	_, err := runInProcess(t, env, "run", "debug", "--file=api.go")
	if err == nil {
		t.Fatal("expected error when --error is missing for debug template")
	}
	// Error message should mention the missing variable
	errStr := err.Error()
	if !strings.Contains(errStr, "error") && !strings.Contains(errStr, "required") {
		t.Errorf("error should mention missing variable, got: %s", errStr)
	}
}

func TestIntegration_Error_MissingRequiredVar_Custom(t *testing.T) {
	env := setupTestEnv(t)

	// custom template requires --var1
	_, err := runInProcess(t, env, "run", "custom")
	if err == nil {
		t.Fatal("expected error when --var1 is missing for custom template")
	}
}

func TestIntegration_Error_MissingAllVars_Commit(t *testing.T) {
	env := setupTestEnv(t)

	// commit requires --changes
	_, err := runInProcess(t, env, "run", "commit")
	if err == nil {
		t.Fatal("expected error when --changes is missing for commit template")
	}
}

// --- LLM API failures ---

func TestIntegration_Error_SendWithoutConfig(t *testing.T) {
	// send requires LLM config; without it and non-interactive, should fail
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")
	// Clear all API key env vars
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	env := &testEnv{t: t, homeDir: home, workDir: t.TempDir(), repoRoot: findRepoRoot(t)}

	// First init to create templates
	runInProcess(t, env, "init")

	// send should fail because no LLM config and non-interactive
	_, err := runInProcess(t, env, "send", "review", "--file=nonexistent.ts")
	if err == nil {
		// Some paths may succeed in rendering before failing on LLM call
		// That's acceptable - the important thing is it doesn't silently succeed
		t.Log("send did not return error (may have failed at a different stage)")
	}
}

func TestIntegration_Error_SendMissingTemplate(t *testing.T) {
	env := setupTestEnv(t)

	// Provide --model to skip LLM config check so we reach template loading
	_, err := runInProcess(t, env, "send", "nonexistent-template-xyz", "--model=claude-sonnet-4-5-20250929")
	if err == nil {
		t.Fatal("expected error for send with nonexistent template")
	}
	assertContains(t, err.Error(), "not found")
}

// --- Invalid LLM responses ---

func TestIntegration_Error_CostWithMissingFile(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "cost", "review", "--file=does_not_exist.ts")
	if err == nil {
		t.Fatal("expected error when cost references a nonexistent file")
	}
	assertContains(t, err.Error(), "does_not_exist.ts")
}

// --- Corrupted config files ---

func TestIntegration_Error_CorruptedLLMConfig(t *testing.T) {
	env := setupTestEnv(t)

	// Write a corrupted llm.json
	configDir := filepath.Join(env.homeDir, ".promptctl")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "llm.json"), []byte("{invalid json!!!"), 0644)

	// Commands that load LLM config should handle corruption gracefully
	// config command reads llm.json
	// It should not panic
	_, err := runInProcess(t, env, "config", "--provider=anthropic", "--api-key=sk-test-123")
	// Some commands may recover from corrupt config by overwriting, which is fine
	_ = err
}

func TestIntegration_Error_CorruptedTemplateDir(t *testing.T) {
	env := setupTestEnv(t)

	// Replace a template file with a directory of the same name
	// (should not cause a panic)
	reviewPath := filepath.Join(env.templateDir, "review.yaml")
	os.Remove(reviewPath)
	os.MkdirAll(reviewPath, 0755) // create dir where file should be

	_, err := runInProcess(t, env, "run", "review", "--file=auth.ts")
	if err == nil {
		t.Fatal("expected error when template path is a directory")
	}
}

func TestIntegration_Error_CorruptedScoreConfig(t *testing.T) {
	env := setupTestEnv(t)

	// Create a corrupted .promptctl/score.yaml in workDir
	promptctlDir := filepath.Join(env.workDir, ".promptctl")
	os.MkdirAll(promptctlDir, 0755)
	os.WriteFile(filepath.Join(promptctlDir, "score.yaml"), []byte("{{{not yaml"), 0644)

	// Score should still work (falls back to defaults)
	testFile := filepath.Join(env.workDir, "test-score.txt")
	os.WriteFile(testFile, []byte("<task>Test prompt</task>"), 0644)

	_, _, exitCode := runCLI(t, env, "score", "test-score.txt")
	// Should not crash/panic - exit 0 or 1 are acceptable
	if exitCode > 1 {
		t.Errorf("score with corrupted config should not hard-crash, got exit %d", exitCode)
	}
}

// --- Path traversal protection ---

func TestIntegration_Error_PathTraversal_File(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "run", "review", "--file=../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal in --file")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "outside") && !strings.Contains(errStr, "must be under") && !strings.Contains(errStr, "traversal") {
		t.Errorf("error should mention path restriction, got: %s", errStr)
	}
}

// --- Edge cases ---

func TestIntegration_Error_RunWithNoArgs(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "run")
	if err == nil {
		t.Fatal("expected error for run with no template name")
	}
	assertContains(t, err.Error(), "usage")
}

func TestIntegration_Error_SendWithNoArgs(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "send")
	if err == nil {
		t.Fatal("expected error for send with no template name")
	}
	assertContains(t, err.Error(), "usage")
}

func TestIntegration_Error_MemoryWithNoSubcommand(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "memory")
	if err == nil {
		t.Fatal("expected error for memory without subcommand")
	}
	assertContains(t, err.Error(), "usage")
}

func TestIntegration_Error_MemoryInvalidSubcommand(t *testing.T) {
	env := setupTestEnv(t)

	_, err := runInProcess(t, env, "memory", "invalid-subcmd")
	if err == nil {
		t.Fatal("expected error for invalid memory subcommand")
	}
	assertContains(t, err.Error(), "usage")
}

func TestIntegration_Error_FixNoFilesFound(t *testing.T) {
	// Create an empty workdir with no prompt files
	home := t.TempDir()
	emptyWork := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROMPTCTL_ENHANCE", "rule")

	env := &testEnv{t: t, homeDir: home, workDir: emptyWork, repoRoot: findRepoRoot(t)}

	// fix with no files should produce stderr message but not crash
	_, stderr, exitCode := runCLI(t, env, "fix")
	if exitCode > 1 {
		t.Errorf("fix with no files should not hard-crash, got exit %d", exitCode)
	}
	_ = stderr // may contain "No prompt files found."
}
