package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prompt-ctl/promptctl/llm"
)

type testCommandMockClient struct {
	cost float64
}

func (m testCommandMockClient) CompleteWithOptions(prompt, model string, opts *llm.CompleteOptions) (*llm.CompletionResult, error) {
	return &llm.CompletionResult{
		Content:    `{"summary":"ok","critical_issues":[],"performance_issues":[],"maintainability_issues":[],"line_suggestions":[{"line":1,"suggestion":"ok"}]}`,
		ActualCost: m.cost,
	}, nil
}

func TestRunTest_Help(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "test", "--help"}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout; _ = w.Close() }()

	err := runTestWithClient(testCommandMockClient{cost: 0.01})
	_ = w.Close()
	if err != nil {
		t.Fatalf("runTestWithClient() err = %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()
	if !strings.Contains(out, "promptctl test <template>") {
		t.Fatalf("help output missing usage, got:\n%s", out)
	}
}

func TestRunTest_MissingTemplate(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "test"}

	err := runTestWithClient(testCommandMockClient{cost: 0.01})
	if err == nil {
		t.Fatal("expected error for missing template")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Fatalf("expected usage error, got: %v", err)
	}
}

func TestRunTest_ModelShorthandRunsExperiment(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	tmplDir := filepath.Join(dir, ".promptctl", "templates")
	if err := os.MkdirAll(tmplDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `name: review
description: Review template
variables:
  - name: file
    required: false
body: |
  Review this file:
  {{.file_content}}`
	if err := os.WriteFile(filepath.Join(tmplDir, "review.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	testFile := "main.go"
	if err := os.WriteFile(filepath.Join(dir, testFile), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "test", "review", "--model=claude-sonnet-4-5", "--file=" + testFile}

	err = runTestWithClient(testCommandMockClient{cost: 0.01})
	if err != nil {
		t.Fatalf("runTestWithClient() err = %v", err)
	}
}

func TestExecute_TestCommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	tmplDir := filepath.Join(dir, ".promptctl", "templates")
	if err := os.MkdirAll(tmplDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `name: review
description: Review template
variables:
  - name: file
    required: false
body: |
  Review output.`
	if err := os.WriteFile(filepath.Join(tmplDir, "review.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "test", "review", "--models=claude-sonnet-4-5"}

	// run real dispatch path, but with deterministic network-free mode
	// by replacing the command to help and just validating dispatch accepts "test".
	os.Args = []string{"promptctl", "test", "--help"}
	if err := Execute(); err != nil {
		t.Fatalf("Execute() test --help err = %v", err)
	}
}
