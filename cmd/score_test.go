package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// scoreIntegrationSkip skips if the test env has no HOME/GOCACHE (e.g. some CI or IDE test runners strip env so "go run" fails).
func scoreIntegrationSkip(t *testing.T) {
	if os.Getenv("HOME") == "" || os.Getenv("GOCACHE") == "" {
		t.Skip("score integration tests require HOME and GOCACHE (run with: go test ./cmd -run TestScoreIntegration)")
	}
}

// runScoreCmd runs "go run . score ..." with Dir=repo root. Call scoreIntegrationSkip(t) first to skip when env is stripped.
func runScoreCmd(t *testing.T, args ...string) *exec.Cmd {
	scoreIntegrationSkip(t)
	root := repoRoot(t)
	cmd := exec.Command("go", append([]string{"run", ".", "score"}, args...)...)
	cmd.Dir = root
	return cmd
}

func TestScoreIntegration_MinScore80_Exit1(t *testing.T) {
	root := repoRoot(t)
	fixtureDir := filepath.Join(root, "cmd", "testdata", "score_fixture")
	// At least one file in fixture scores below 80 → exit 1
	cmd := runScoreCmd(t, fixtureDir, "--min-score=80")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	var exit *exec.ExitError
	if err == nil {
		t.Fatal("expected exit code 1 (at least one file below threshold), got 0")
	}
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got err: %v", err)
	}
}

func TestScoreIntegration_MinScore30_Exit0(t *testing.T) {
	root := repoRoot(t)
	fixtureDir := filepath.Join(root, "cmd", "testdata", "score_fixture")
	// Threshold 30 is below both files → exit 0
	cmd := runScoreCmd(t, fixtureDir, "--min-score=30")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit code 0, got: %v\noutput: %s", err, out)
	}
}

func TestScoreIntegration_FormatJSON_HasFilesAndOk(t *testing.T) {
	root := repoRoot(t)
	fixtureDir := filepath.Join(root, "cmd", "testdata", "score_fixture")
	cmd := runScoreCmd(t, fixtureDir, "--format=json")
	out, err := cmd.Output()
	// Exit 1 is OK (one file below default threshold); we only need valid JSON on stdout
	if len(out) == 0 {
		t.Fatalf("score --format=json produced no stdout (err: %v)", err)
	}
	var v struct {
		Files    []interface{} `json:"files"`
		MinScore int           `json:"min_score"`
		OK       *bool         `json:"ok"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if v.Files == nil {
		t.Error("JSON must have \"files\" array")
	}
	if v.OK == nil {
		t.Error("JSON must have \"ok\" field")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	cmdDir := filepath.Dir(filename)
	root := filepath.Clean(filepath.Join(cmdDir, ".."))
	return root
}
