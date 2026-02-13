package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if !strings.Contains(out, "<context>") {
		t.Errorf("stdout should contain <context> when using rule enhance; got:\n%s", out)
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
