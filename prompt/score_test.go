package prompt

import (
	"strings"
	"testing"
)

func TestScoreEnhanceResult_Fidelity(t *testing.T) {
	intent := "Add photo frame emulation and zoom slider like artsy.com, close button and escape to close."
	prompt := "Role: Expert. Subject: Photo gallery. Task: 1. Add frame emulation 2. Zoom slider 3. Close button and escape."
	s := ScoreEnhanceResult(intent, prompt)
	if s.Score < 50 {
		t.Errorf("expected reasonable score when key terms preserved; got %d", s.Score)
	}
}

func TestScoreEnhanceResult_DuplicateSections(t *testing.T) {
	intent := "Review my code"
	prompt := `## Role
Expert
## Subject
Code
## Design Deliverables
A
### Design Deliverables
B
### Design Deliverables
C`
	s := ScoreEnhanceResult(intent, prompt)
	if s.Score > 85 {
		t.Errorf("expected penalty for duplicate Design Deliverables; got score %d", s.Score)
	}
	var hasDupHint bool
	for _, h := range s.Hints {
		if len(h) > 0 && (contains(h, "Duplicate") || contains(h, "duplicate")) {
			hasDupHint = true
			break
		}
	}
	if !hasDupHint {
		t.Errorf("expected hint about duplicate sections; got %q", s.Hints)
	}
}

func contains(a, b string) bool {
	return strings.Contains(a, b)
}

func TestScoreEnhanceResult_Structure(t *testing.T) {
	intent := "Analyze X"
	prompt := "some text with no structure"
	s := ScoreEnhanceResult(intent, prompt)
	if s.Score > 90 {
		t.Errorf("expected penalty for missing structure; got %d", s.Score)
	}
}
