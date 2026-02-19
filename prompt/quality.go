package prompt

import (
	"regexp"
	"strings"
)

// QualityScore holds per-file score and which rules triggered (penalties).
type QualityScore struct {
	Score int      // 0-100
	Rules []string // e.g. missing_constraints, overbroad_scope
}

const maxSentenceLen = 80

var (
	vagueWords = []string{"something", "things", "stuff", "somehow", "somewhere"}
	// overbroadWords is used with word-boundary match
	overbroadWordsRegex = regexp.MustCompile(`\b(everything|all|always)\b`)
)

func ScorePromptQuality(prompt string) QualityScore {
	var totalPenalty int
	var rules []string

	// Structure: role/context, subject, task, output, constraint (lowercase substring)
	structurePenalty := scoreQualityStructure(prompt)
	totalPenalty += structurePenalty
	if structurePenalty > 0 {
		rules = append(rules, "missing_structure")
	}

	// Clarity: vague words or very long sentence; cap 10
	clarityPenalty := scoreQualityClarity(prompt)
	totalPenalty += clarityPenalty
	if clarityPenalty > 0 {
		rules = append(rules, "clarity")
	}

	// Constraints: no "do not"/"only"/"max"/"constraint"; cap 20
	constraintsPenalty := scoreQualityConstraints(prompt)
	totalPenalty += constraintsPenalty
	if constraintsPenalty > 0 {
		rules = append(rules, "missing_constraints")
	}

	// Scope: "everything"/"all"/"always" without "only" or "specific"; cap 15
	scopePenalty := scoreQualityScope(prompt)
	totalPenalty += scopePenalty
	if scopePenalty > 0 {
		rules = append(rules, "overbroad_scope")
	}

	// Persona: no "you are"/"role"/"acting as"; cap 15
	personaPenalty := scoreQualityPersona(prompt)
	totalPenalty += personaPenalty
	if personaPenalty > 0 {
		rules = append(rules, "missing_persona")
	}

	score := 100 - totalPenalty
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return QualityScore{Score: score, Rules: rules}
}

// scoreQualityStructure returns penalty 0–25 for missing required sections.
func scoreQualityStructure(prompt string) int {
	lower := strings.ToLower(prompt)
	missing := 0
	if !strings.Contains(lower, "context") && !strings.Contains(lower, "role") {
		missing++
	}
	if !strings.Contains(lower, "subject") {
		missing++
	}
	if !strings.Contains(lower, "task") {
		missing++
	}
	if !strings.Contains(lower, "output") {
		missing++
	}
	if !strings.Contains(lower, "constraint") {
		missing++
	}
	penalty := missing * 5
	if penalty > 25 {
		penalty = 25
	}
	return penalty
}

// scoreQualityClarity returns penalty 0–10 for vague words or sentences over ~80 chars.
func scoreQualityClarity(prompt string) int {
	lower := strings.ToLower(prompt)
	penalty := 0
	for _, w := range vagueWords {
		if strings.Contains(lower, w) {
			penalty += 3
			if penalty >= 10 {
				return 10
			}
		}
	}
	// Sentences over ~80 chars
	for _, s := range splitSentences(prompt) {
		if len(strings.TrimSpace(s)) > maxSentenceLen {
			penalty += 5
			if penalty >= 10 {
				return 10
			}
		}
	}
	if penalty > 10 {
		return 10
	}
	return penalty
}

func splitSentences(prompt string) []string {
	// Split on ., !, ?
	re := regexp.MustCompile(`[.!?]+`)
	parts := re.Split(prompt, -1)
	return parts
}

// scoreQualityConstraints returns penalty 0–20 when prompt has no "do not", "only", "max", "constraint".
func scoreQualityConstraints(prompt string) int {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "do not") || strings.Contains(lower, "only") ||
		strings.Contains(lower, "max") || strings.Contains(lower, "constraint") {
		return 0
	}
	return 20
}

// scoreQualityScope returns penalty 0–15 for "everything"/"all"/"always" without "only" or "specific".
func scoreQualityScope(prompt string) int {
	lower := strings.ToLower(prompt)
	if !overbroadWordsRegex.MatchString(lower) {
		return 0
	}
	if strings.Contains(lower, "only") || strings.Contains(lower, "specific") {
		return 0
	}
	return 15
}

// scoreQualityPersona returns penalty 0–15 when prompt has no "you are", "role", "acting as".
func scoreQualityPersona(prompt string) int {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "you are") || strings.Contains(lower, "role") ||
		strings.Contains(lower, "acting as") {
		return 0
	}
	return 15
}
