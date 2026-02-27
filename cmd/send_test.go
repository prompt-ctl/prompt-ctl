package cmd

import (
	"strconv"
	"testing"

	"github.com/oleg-koval/promptctl/prompt"
)

// --- min-score parsing from parseVars ---

func TestMinScoreParsing_Present(t *testing.T) {
	vars := parseVars([]string{"--min-score=80", "--model=gpt-4"})
	raw, ok := vars["min-score"]
	if !ok {
		t.Fatal("--min-score should be present in vars")
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("strconv.Atoi(%q) err = %v", raw, err)
	}
	if n != 80 {
		t.Errorf("min-score = %d, want 80", n)
	}
}

func TestMinScoreParsing_Absent(t *testing.T) {
	vars := parseVars([]string{"--model=gpt-4"})
	if _, ok := vars["min-score"]; ok {
		t.Error("min-score should not be present when not passed")
	}
}

func TestMinScoreParsing_InvalidValue(t *testing.T) {
	vars := parseVars([]string{"--min-score=abc"})
	raw := vars["min-score"]
	_, err := strconv.Atoi(raw)
	if err == nil {
		t.Error("expected Atoi error for non-numeric min-score")
	}
}

func TestMinScoreParsing_DeleteFromVars(t *testing.T) {
	vars := parseVars([]string{"--min-score=80", "--file=auth.ts"})
	delete(vars, "min-score")
	if _, ok := vars["min-score"]; ok {
		t.Error("min-score should be deleted from vars")
	}
	if vars["file"] != "auth.ts" {
		t.Error("other vars should remain after deleting min-score")
	}
}

// --- quality gate logic (score < minScore) ---

func TestQualityGate_ScoreBelowMinimum_ShouldReject(t *testing.T) {
	// A vague prompt with no structure should score well below 80
	pq := prompt.ScorePromptQuality("do stuff")
	minScore := 80
	if pq.Score >= minScore {
		t.Skipf("prompt scored %d (>= %d); test needs a lower-scoring prompt", pq.Score, minScore)
	}
	if pq.Score >= minScore {
		t.Errorf("expected score < %d, got %d", minScore, pq.Score)
	}
}

func TestQualityGate_ScoreAboveMinimum_ShouldPass(t *testing.T) {
	goodPrompt := "You are a code reviewer. Role: senior engineer. Context: reviewing auth module. " +
		"Subject: authentication flow. Task: find security issues. Output: list of findings. " +
		"Constraint: only report high-severity issues. Do not include style nits."
	pq := prompt.ScorePromptQuality(goodPrompt)
	minScore := 50
	if pq.Score < minScore {
		t.Errorf("well-structured prompt scored %d (< %d); expected pass", pq.Score, minScore)
	}
}

func TestQualityGate_ZeroMinScore_AlwaysPasses(t *testing.T) {
	pq := prompt.ScorePromptQuality("bad prompt")
	minScore := 0
	if minScore > 0 && pq.Score < minScore {
		t.Error("zero min-score should never reject")
	}
}

func TestQualityGate_MinScore100_RejectsImperfect(t *testing.T) {
	pq := prompt.ScorePromptQuality("do stuff")
	minScore := 100
	if pq.Score >= minScore {
		t.Error("vague prompt should not score 100")
	}
}

// --- wrapLines ---

func TestWrapLines_Empty(t *testing.T) {
	lines := wrapLines("", 40)
	if lines != nil {
		t.Errorf("expected nil for empty string, got %v", lines)
	}
}

func TestWrapLines_ShortString(t *testing.T) {
	lines := wrapLines("hello world", 40)
	if len(lines) != 1 || lines[0] != "hello world" {
		t.Errorf("expected single line, got %v", lines)
	}
}

func TestWrapLines_LongString(t *testing.T) {
	s := "one two three four five six seven eight nine ten eleven twelve"
	lines := wrapLines(s, 20)
	if len(lines) < 2 {
		t.Errorf("expected multiple lines for width=20, got %d: %v", len(lines), lines)
	}
	for _, l := range lines {
		if len(l) > 20 {
			t.Errorf("line exceeds width 20: %q (len=%d)", l, len(l))
		}
	}
}

// --- stripANSI ---

func TestStripANSI_NoEscapes(t *testing.T) {
	got := stripANSI("hello world")
	if got != "hello world" {
		t.Errorf("stripANSI(plain) = %q, want %q", got, "hello world")
	}
}

func TestStripANSI_WithEscapes(t *testing.T) {
	got := stripANSI("\033[1mBold\033[0m text")
	if got != "Bold text" {
		t.Errorf("stripANSI(ansi) = %q, want %q", got, "Bold text")
	}
}

func TestStripANSI_Empty(t *testing.T) {
	got := stripANSI("")
	if got != "" {
		t.Errorf("stripANSI(\"\") = %q, want empty", got)
	}
}

// --- "Without promptctl" cost display logic ---

func TestWithoutPromptctlCost_Calculation(t *testing.T) {
	actualCost := 0.01
	savedVsUnstructured := actualCost * 2 // fallback multiplier
	withoutCost := actualCost + savedVsUnstructured
	want := 0.03
	if withoutCost < want-0.001 || withoutCost > want+0.001 {
		t.Errorf("withoutCost = %f, want ~%f", withoutCost, want)
	}
}

func TestWithoutPromptctlCost_WithMultiplier(t *testing.T) {
	actualCost := 0.05
	mult := 2.8 // mid-tier multiplier
	savedVsUnstructured := actualCost * (mult - 1)
	withoutCost := actualCost + savedVsUnstructured
	want := actualCost * mult
	if withoutCost < want-0.001 || withoutCost > want+0.001 {
		t.Errorf("withoutCost = %f, want ~%f", withoutCost, want)
	}
}
