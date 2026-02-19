package ui

import (
	"regexp"
	"strings"
)

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

// isMarkdownHeader returns true if the line is a markdown section header (# , ## , ### ).
func isMarkdownHeader(line string) bool {
	return strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ")
}

// xmlTagOnly matches a line that is only an XML open or close tag, e.g. <role> or </context>.
var xmlTagOnly = regexp.MustCompile(`^\s*</?[a-zA-Z0-9_-]+>\s*$`)

// xmlTagStart matches optional leading space and an XML tag at the start of a line, capturing tag and rest.
var xmlTagStart = regexp.MustCompile(`^\s*(</?[a-zA-Z0-9_-]+>)\s*(.*)$`)

// colorizeLine returns the line with section headers/tags colorized when TTY.
func colorizeLine(line string) string {
	if !Interactive() {
		return line
	}
	if isMarkdownHeader(line) {
		return Heading(line)
	}
	if xmlTagOnly.MatchString(line) {
		return Heading(strings.TrimSpace(line))
	}
	if sub := xmlTagStart.FindStringSubmatch(line); len(sub) == 3 {
		tag, rest := sub[1], sub[2]
		if rest == "" {
			return Heading(tag)
		}
		return Heading(tag) + " " + rest
	}
	return line
}

// isSectionBoundary returns true if the line starts a new logical section (markdown header or XML tag line).
func isSectionBoundary(line string) bool {
	return isMarkdownHeader(line) || xmlTagOnly.MatchString(line) || xmlTagStart.MatchString(line)
}

// normalizePromptNewlines ensures double newline between sections and a trailing newline.
func normalizePromptNewlines(prompt string) string {
	lines := strings.Split(strings.TrimRight(prompt, "\n"), "\n")
	var out []string
	for i, line := range lines {
		if i > 0 && isSectionBoundary(line) {
			// Ensure blank line before this section
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
		}
		out = append(out, line)
	}
	if len(out) > 0 && out[len(out)-1] != "" {
		out = append(out, "")
	}
	return strings.Join(out, "\n") + "\n"
}

// FormatPromptForTerminal returns the prompt with section headers colorized and normalized newlines when TTY; otherwise only newline normalization.
func FormatPromptForTerminal(prompt string) string {
	normalized := normalizePromptNewlines(prompt)
	if !Interactive() {
		return normalized
	}
	lines := strings.Split(strings.TrimSuffix(normalized, "\n"), "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(colorizeLine(line))
	}
	b.WriteString("\n")
	return b.String()
}
