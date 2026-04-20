package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldRunAutoUpdateNow_NoMarkerFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if !shouldRunAutoUpdateNow() {
		t.Fatal("shouldRunAutoUpdateNow() = false, want true when marker is missing")
	}
}

func TestShouldRunAutoUpdateNow_AfterMark(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	markAutoUpdateCheckNow()

	if shouldRunAutoUpdateNow() {
		t.Fatal("shouldRunAutoUpdateNow() = true, want false right after mark")
	}
}

func TestShouldRunAutoUpdateNow_StaleMarker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".promptctl", "last_auto_update_check")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	stale := time.Now().Add(-autoUpdateInterval - time.Minute).UTC().Format(time.RFC3339)
	if err := os.WriteFile(path, []byte(stale), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !shouldRunAutoUpdateNow() {
		t.Fatal("shouldRunAutoUpdateNow() = false, want true for stale marker")
	}
}

func TestShouldRunAutoUpdateNow_InvalidMarker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".promptctl", "last_auto_update_check")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("not-a-time"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !shouldRunAutoUpdateNow() {
		t.Fatal("shouldRunAutoUpdateNow() = false, want true for invalid marker")
	}
}
