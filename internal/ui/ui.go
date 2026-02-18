package ui

import (
	"os"
)

// Interactive returns true when stdin and stdout are TTY (same as cmd.interactive).
func Interactive() bool {
	stdin, _ := os.Stdin.Stat()
	stdout, _ := os.Stdout.Stat()
	return (stdin.Mode()&os.ModeCharDevice) != 0 && (stdout.Mode()&os.ModeCharDevice) != 0
}
