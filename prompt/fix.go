package prompt

import (
	"strings"
)

// ApplyFormat normalizes prompt text: LF line endings, trim trailing space per line,
// collapse 3+ consecutive blank lines to 2.
func ApplyFormat(prompt string) string {
	// Normalize line endings: \r\n or \r -> \n
	s := strings.ReplaceAll(prompt, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}

	// Collapse 3+ consecutive blank lines to 1 (one blank line = two \n in a row)
	var out []string
	blankCount := 0
	for _, line := range lines {
		if line == "" {
			blankCount++
			if blankCount <= 1 {
				out = append(out, "")
			}
		} else {
			blankCount = 0
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}

// hasStructure reports whether the prompt already contains role/context/task structure
// (XML-like tags or markdown headings).
func hasStructure(prompt string) bool {
	lower := strings.ToLower(prompt)
	// XML-like sections
	if strings.Contains(lower, "<role>") || strings.Contains(lower, "<context>") || strings.Contains(lower, "<task>") {
		return true
	}
	// Markdown-style headings
	if strings.Contains(prompt, "## Role") || strings.Contains(prompt, "## Context") || strings.Contains(prompt, "## Task") {
		return true
	}
	return false
}

// ApplyStructure ensures the prompt has role/context/task structure. If it already has
// structure (e.g. <role>, <context>, <task> or ## Role, ## Context, ## Task), returns
// as-is. Otherwise wraps in default XML-like sections with empty role/context and
// the prompt text inside <task>. Does not reword user text.
func ApplyStructure(prompt string) string {
	if hasStructure(prompt) {
		return prompt
	}
	return "<role>\n\n</role>\n<context>\n\n</context>\n<task>\n" + prompt + "\n</task>"
}
