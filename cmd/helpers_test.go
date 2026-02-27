package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- parseVars ---

func TestParseVars_EmptyArgs(t *testing.T) {
	vars := parseVars(nil)
	if len(vars) != 0 {
		t.Errorf("expected empty map, got %v", vars)
	}
}

func TestParseVars_SingleVar(t *testing.T) {
	vars := parseVars([]string{"--model=gpt-4"})
	if vars["model"] != "gpt-4" {
		t.Errorf("expected model=gpt-4, got %q", vars["model"])
	}
}

func TestParseVars_MultipleVars(t *testing.T) {
	vars := parseVars([]string{"--model=gpt-4", "--file=auth.ts", "--min-score=80"})
	if vars["model"] != "gpt-4" {
		t.Errorf("model = %q, want gpt-4", vars["model"])
	}
	if vars["file"] != "auth.ts" {
		t.Errorf("file = %q, want auth.ts", vars["file"])
	}
	if vars["min-score"] != "80" {
		t.Errorf("min-score = %q, want 80", vars["min-score"])
	}
}

func TestParseVars_IgnoresNonFlagArgs(t *testing.T) {
	vars := parseVars([]string{"review", "--file=auth.ts", "extra"})
	if _, ok := vars["review"]; ok {
		t.Error("non-flag arg 'review' should not be in vars")
	}
	if vars["file"] != "auth.ts" {
		t.Errorf("file = %q, want auth.ts", vars["file"])
	}
}

func TestParseVars_FlagWithoutEquals_StoresEmptyString(t *testing.T) {
	vars := parseVars([]string{"--record"})
	val, ok := vars["record"]
	if !ok {
		t.Error("--record flag should be stored in vars")
	}
	if val != "" {
		t.Errorf("--record value = %q, want empty string", val)
	}
}

func TestParseVars_ValueContainsEquals(t *testing.T) {
	vars := parseVars([]string{"--query=a=b=c"})
	if vars["query"] != "a=b=c" {
		t.Errorf("query = %q, want a=b=c", vars["query"])
	}
}

// --- formatNumSimple ---

func TestFormatNumSimple_Below1000(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
	}
	for _, tt := range tests {
		got := formatNumSimple(tt.in)
		if got != tt.want {
			t.Errorf("formatNumSimple(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatNumSimple_ThousandsComma(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{1000, "1,000"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1000000, "1,000,000"},
	}
	for _, tt := range tests {
		got := formatNumSimple(tt.in)
		if got != tt.want {
			t.Errorf("formatNumSimple(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// --- maskKey ---

func TestMaskKey_ShortKey(t *testing.T) {
	got := maskKey("abc")
	if got != "****" {
		t.Errorf("maskKey(short) = %q, want ****", got)
	}
}

func TestMaskKey_ExactlyEightChars(t *testing.T) {
	got := maskKey("12345678")
	if got != "1234...5678" {
		t.Errorf("maskKey(8 chars) = %q, want 1234...5678", got)
	}
}

func TestMaskKey_LongKey(t *testing.T) {
	got := maskKey("sk-1234567890abcdef")
	want := "sk-1...cdef"
	if got != want {
		t.Errorf("maskKey(long) = %q, want %q", got, want)
	}
}

// --- enrichFileVars ---

func TestEnrichFileVars_NoFileKey_DoesNothing(t *testing.T) {
	vars := map[string]string{"foo": "bar"}
	if err := enrichFileVars(vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := vars["file_content"]; ok {
		t.Error("file_content should not be set when no file key present")
	}
	if _, ok := vars["file_name"]; ok {
		t.Error("file_name should not be set when no file key present")
	}
	if _, ok := vars["file_ext"]; ok {
		t.Error("file_ext should not be set when no file key present")
	}
}

func TestEnrichFileVars_ValidFile_SetsVars(t *testing.T) {
	content := "package main\n"
	tmp, err := os.CreateTemp(".", "testfile*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	vars := map[string]string{"file": filepath.Base(tmp.Name())}
	if err := enrichFileVars(vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vars["file_content"] != content {
		t.Errorf("file_content = %q, want %q", vars["file_content"], content)
	}
	if vars["file_name"] != filepath.Base(tmp.Name()) {
		t.Errorf("file_name = %q, want %q", vars["file_name"], filepath.Base(tmp.Name()))
	}
	if vars["file_ext"] != "go" {
		t.Errorf("file_ext = %q, want %q", vars["file_ext"], "go")
	}
}

func TestEnrichFileVars_NoExtension_EmptyExt(t *testing.T) {
	tmp, err := os.CreateTemp(".", "testfile_noext")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.Close()

	vars := map[string]string{"file": filepath.Base(tmp.Name())}
	if err := enrichFileVars(vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["file_ext"] != "" {
		t.Errorf("file_ext = %q, want empty string for file without extension", vars["file_ext"])
	}
}

func TestEnrichFileVars_NonExistentFile_ReturnsError(t *testing.T) {
	vars := map[string]string{"file": "no_such_file_xyz.go"}
	if err := enrichFileVars(vars); err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestEnrichFileVars_PathTraversal_ReturnsError(t *testing.T) {
	vars := map[string]string{"file": "../../etc/passwd"}
	err := enrichFileVars(vars)
	if err == nil {
		t.Error("expected error for path traversal attempt, got nil")
	}
	if !strings.Contains(err.Error(), "file path must be under current directory") {
		t.Errorf("expected path traversal message, got: %v", err)
	}
}

func TestEnrichFileVars_PreservesOtherVars(t *testing.T) {
	tmp, err := os.CreateTemp(".", "testfile*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.Close()

	vars := map[string]string{
		"file":      filepath.Base(tmp.Name()),
		"other_key": "other_value",
	}
	if err := enrichFileVars(vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["other_key"] != "other_value" {
		t.Errorf("other_key = %q, want %q", vars["other_key"], "other_value")
	}
}

// --- runSpinner ---

func TestRunSpinner_ImmediateClose_DoesNotBlock(t *testing.T) {
	done := make(chan struct{})
	close(done)
	// In test, stderr is not a terminal, so runSpinner returns immediately.
	runSpinner(done, "gpt-4", 0)
}

func TestRunSpinner_WithModelAndTokens_DoesNotPanic(t *testing.T) {
	done := make(chan struct{})
	close(done)
	runSpinner(done, "claude-sonnet-4.5", 1500)
}

func TestRunSpinner_EmptyModel_DoesNotPanic(t *testing.T) {
	done := make(chan struct{})
	close(done)
	runSpinner(done, "", 0)
}
