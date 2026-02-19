package prompt

import (
	"strings"
	"testing"
)

func TestApplyFormat(t *testing.T) {
	input := "  foo  \n\n\n  bar  \r\n"
	got := ApplyFormat(input)

	// Only LF line endings (no \r)
	if strings.Contains(got, "\r") {
		t.Errorf("ApplyFormat: result must have only LF line endings, got \\r in %q", got)
	}

	// Trailing space trimmed per line: "  foo  " -> "  foo", "  bar  " -> "  bar"
	lines := strings.Split(got, "\n")
	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if line != trimmed {
			t.Errorf("ApplyFormat: line %d has trailing space: %q", i, line)
		}
	}

	// Multiple blank lines collapsed (3+ to 1 or 2): no run of 3+ newlines between content
	// Input has \n\n\n between "foo" and "bar" -> at most 2 consecutive newlines (one blank line)
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("ApplyFormat: 3+ consecutive blank lines must be collapsed; got \\n\\n\\n in %q", got)
	}

	// Normalized result: "  foo" + "\n\n" + "  bar" + "\n"
	want := "  foo\n\n  bar\n"
	if got != want {
		t.Errorf("ApplyFormat(%q)\n  got  %q\n  want %q", input, got, want)
	}
}

func TestApplyStructure(t *testing.T) {
	// When prompt has no XML/markdown sections, ApplyStructure wraps so expected sections exist.
	input := "some text"
	got := ApplyStructure(input)

	// Output must contain the three sections (XML-like tags).
	for _, tag := range []string{"<role>", "<context>", "<task>"} {
		if !strings.Contains(got, tag) {
			t.Errorf("ApplyStructure(%q): output must contain %q, got %q", input, tag, got)
		}
	}
	// Original text must appear in output (e.g. inside <task>).
	if !strings.Contains(got, "some text") {
		t.Errorf("ApplyStructure(%q): output must contain original text %q, got %q", input, input, got)
	}
}
