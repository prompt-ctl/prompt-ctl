package prompt

import (
	"regexp"
	"strings"
)

// EnhanceScore holds a 0–100 quality score and optional improvement hints.
type EnhanceScore struct {
	Score int      // 0–100
	Hints []string // e.g. "Low fidelity: add specifics from intent"
}

// ScoreEnhanceResult scores an enhanced prompt against the original intent.
// Uses heuristics: fidelity (intent terms in output), no duplicate sections, structure present.
func ScoreEnhanceResult(intent, prompt string) EnhanceScore {
	var hints []string
	score := 100
	lowerIntent := strings.ToLower(intent)
	lowerPrompt := strings.ToLower(prompt)

	// 1) Fidelity: extract meaningful words from intent (skip common/HTML), check they appear in output
	fidelityScore, fidelityHint := scoreFidelity(lowerIntent, lowerPrompt, intent)
	score -= (100 - fidelityScore) / 2 // cap impact so other factors matter
	if fidelityHint != "" {
		hints = append(hints, fidelityHint)
	}

	// 2) Duplicate sections (e.g. "Design Deliverables" three times)
	dupPenalty, dupHint := scoreNoDuplicateSections(prompt)
	score -= dupPenalty
	if dupHint != "" {
		hints = append(hints, dupHint)
	}

	// 3) Structure: has role/context, subject, task, output format
	structPenalty := scoreStructure(prompt)
	score -= structPenalty

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return EnhanceScore{Score: score, Hints: hints}
}

// scoreFidelity: 0–100. Intent terms (excluding HTML/common) should appear in prompt.
func scoreFidelity(lowerIntent, lowerPrompt, intent string) (int, string) {
	// Strip URLs and markdown images so we don't score on those
	stripURLs := regexp.MustCompile(`https?://[^\s]+|!?\[[^\]]*\]\([^)]+\)`)
	cleanIntent := stripURLs.ReplaceAllString(lowerIntent, " ")
	// Tokenize into words (letters/numbers, min 3 chars to skip "the", "and")
	wordRx := regexp.MustCompile(`[a-z0-9]{3,}`)
	words := wordRx.FindAllString(cleanIntent, -1)
	seen := make(map[string]bool)
	var unique []string
	for _, w := range words {
		if seen[w] {
			continue
		}
		seen[w] = true
		// Skip very generic
		if isGenericWord(w) {
			continue
		}
		unique = append(unique, w)
	}
	if len(unique) == 0 {
		return 100, ""
	}
	var found int
	for _, w := range unique {
		if strings.Contains(lowerPrompt, w) {
			found++
		}
	}
	pct := (found * 100) / len(unique)
	hint := ""
	if pct < 40 {
		hint = "Low fidelity: preserve specific terms from your intent in the prompt (e.g. frame, zoom, close button, artsy)."
	} else if pct < 70 {
		hint = "Add more specifics from your intent so the enhanced prompt reflects your requirements."
	}
	return pct, hint
}

func isGenericWord(w string) bool {
	generic := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "had": true,
		"her": true, "was": true, "one": true, "our": true, "out": true,
		"have": true, "has": true, "this": true, "that": true, "with": true,
		"from": true, "when": true, "which": true, "there": true, "their": true,
		"about": true, "would": true, "could": true, "should": true, "into": true,
		"some": true, "more": true, "other": true, "only": true, "same": true,
		"than": true, "then": true, "them": true, "they": true, "what": true,
	}
	return generic[w]
}

// scoreNoDuplicateSections: penalty 0–30, hint if duplicate section headers.
func scoreNoDuplicateSections(prompt string) (penalty int, hint string) {
	// Match ## or ### Section (markdown)
	sectionRx := regexp.MustCompile(`(?m)^#{2,3}\s+([^\n]+)`)
	matches := sectionRx.FindAllStringSubmatch(prompt, -1)
	norm := func(s string) string {
		s = strings.TrimSpace(strings.ToLower(s))
		s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
		return s
	}
	counts := make(map[string]int)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		key := norm(m[1])
		if key != "" {
			counts[key]++
		}
	}
	for _, c := range counts {
		if c > 1 {
			penalty += 10 * (c - 1)
		}
	}
	if penalty > 30 {
		penalty = 30
	}
	if penalty > 0 {
		hint = "Duplicate sections detected; use each section (Role, Subject, Task, Output Format, Constraints) only once."
	}
	return penalty, hint
}

// scoreStructure: penalty 0–20 if required sections are missing.
func scoreStructure(prompt string) int {
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
	if penalty > 20 {
		penalty = 20
	}
	return penalty
}
