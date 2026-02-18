package ui

const (
	ansiGreen = "\033[32m"
	ansiDim   = "\033[2m"
	ansiReset = "\033[0m"
)

// Success returns s with green ANSI when TTY, else s unchanged.
func Success(s string) string {
	if !Interactive() {
		return s
	}
	return ansiGreen + s + ansiReset
}

// Hint returns s with dim ANSI when TTY, else s unchanged.
func Hint(s string) string {
	if !Interactive() {
		return s
	}
	return ansiDim + s + ansiReset
}
