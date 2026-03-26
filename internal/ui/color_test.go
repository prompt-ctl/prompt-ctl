package ui

import (
	"strings"
	"testing"
)

// In test, Interactive() returns false, so color functions return plain text.

func TestSuccess_NonInteractive(t *testing.T) {
	got := Success("ok")
	if got != "ok" {
		t.Errorf("Success = %q, want ok (no ANSI in non-TTY)", got)
	}
}

func TestHint_NonInteractive(t *testing.T) {
	got := Hint("hint text")
	if got != "hint text" {
		t.Errorf("Hint = %q, want plain text", got)
	}
}

func TestBold_NonInteractive(t *testing.T) {
	got := Bold("bold text")
	if got != "bold text" {
		t.Errorf("Bold = %q, want plain text", got)
	}
}

func TestHeading_NonInteractive(t *testing.T) {
	got := Heading("heading")
	if got != "heading" {
		t.Errorf("Heading = %q, want plain text", got)
	}
}

func TestIsMarkdownHeader(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"# Title", true},
		{"## Subtitle", true},
		{"### Section", true},
		{"Not a header", false},
		{"#nospace", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isMarkdownHeader(tt.input)
		if got != tt.want {
			t.Errorf("isMarkdownHeader(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestXmlTagOnly(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"<context>", true},
		{"</context>", true},
		{"  <tag>  ", true},
		{"<tag>content</tag>", false},
		{"not a tag", false},
		{"", false},
	}
	for _, tt := range tests {
		got := xmlTagOnly.MatchString(tt.input)
		if got != tt.want {
			t.Errorf("xmlTagOnly(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestColorizeLine_NonInteractive(t *testing.T) {
	// When not interactive, should return line unmodified
	tests := []string{
		"# Header",
		"<context>",
		"plain text",
		"<tag> with content",
	}
	for _, line := range tests {
		got := colorizeLine(line)
		if got != line {
			t.Errorf("colorizeLine(%q) = %q, want same (non-interactive)", line, got)
		}
	}
}

func TestIsSectionBoundary(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"# Header", true},
		{"## Sub", true},
		{"<context>", true},
		{"</context>", true},
		{"<tag> text", true},
		{"plain text", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isSectionBoundary(tt.input)
		if got != tt.want {
			t.Errorf("isSectionBoundary(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNormalizePromptNewlines(t *testing.T) {
	input := "# Header\nsome text\n## Sub\nmore text"
	result := normalizePromptNewlines(input)
	// Should have blank line before ## Sub
	if !strings.Contains(result, "some text\n\n## Sub") {
		t.Errorf("should have blank line before section: %q", result)
	}
	// Should end with newline
	if !strings.HasSuffix(result, "\n") {
		t.Error("should end with newline")
	}
}

func TestNormalizePromptNewlines_AlreadyHasBlankLines(t *testing.T) {
	input := "# Header\n\nsome text\n\n## Sub\nmore text"
	result := normalizePromptNewlines(input)
	// Should not add extra blank lines
	if strings.Contains(result, "\n\n\n") {
		t.Errorf("should not have triple newlines: %q", result)
	}
}

func TestNormalizePromptNewlines_Empty(t *testing.T) {
	result := normalizePromptNewlines("")
	if result != "\n" {
		t.Errorf("empty input: got %q, want newline", result)
	}
}

func TestFormatPromptForTerminal_NonInteractive(t *testing.T) {
	input := "# Header\ntext\n<context>\ncontent\n</context>"
	result := FormatPromptForTerminal(input)
	// Non-interactive: no ANSI codes, just normalized newlines
	if strings.Contains(result, "\033[") {
		t.Error("should not contain ANSI codes in non-interactive mode")
	}
	if !strings.Contains(result, "# Header") {
		t.Error("should contain header")
	}
	if !strings.Contains(result, "<context>") {
		t.Error("should contain XML tag")
	}
}

func TestFormatPromptForTerminal_PreservesContent(t *testing.T) {
	input := "Hello world\nSecond line"
	result := FormatPromptForTerminal(input)
	if !strings.Contains(result, "Hello world") {
		t.Error("should preserve content")
	}
	if !strings.Contains(result, "Second line") {
		t.Error("should preserve second line")
	}
}
