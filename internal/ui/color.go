package ui

import "strings"

const (
	ansiGreen = "\033[32m"
	ansiDim   = "\033[2m"
	ansiCyan  = "\033[36m"
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

// Heading returns s with cyan ANSI when TTY, else s unchanged.
func Heading(s string) string {
	if !Interactive() {
		return s
	}
	return ansiCyan + s + ansiReset
}

// FormatPromptForTerminal returns the prompt with ## lines colorized when TTY; otherwise unchanged.
func FormatPromptForTerminal(prompt string) string {
	if !Interactive() {
		return prompt
	}
	var b strings.Builder
	lines := strings.Split(prompt, "\n")
	for i, line := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		if strings.HasPrefix(line, "## ") {
			b.WriteString(Heading(line))
		} else {
			b.WriteString(line)
		}
	}
	return b.String()
}
