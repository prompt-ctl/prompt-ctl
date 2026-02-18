package ui

import (
	"testing"
)

func TestInteractive_WhenNotTTY_ReturnsFalse(t *testing.T) {
	// In go test, stdin/stdout are typically not TTY.
	if Interactive() {
		t.Error("Interactive() should be false when not in a terminal (e.g. go test)")
	}
}
