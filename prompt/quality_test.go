package prompt

import "testing"

func TestQualityScore_ZeroPenaltyReturns100(t *testing.T) {
	s := ScorePromptQuality("valid prompt with role context subject task output constraints")
	if s.Score != 100 {
		t.Errorf("expected 100, got %d", s.Score)
	}
}

func TestQualityScore_StructureMissingSectionsDeducts(t *testing.T) {
	s := ScorePromptQuality("just a paragraph with no sections")
	if s.Score >= 100 {
		t.Errorf("expected penalty for missing structure, got score %d", s.Score)
	}
	found := false
	for _, r := range s.Rules {
		if r == "missing_structure" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_structure in Rules, got %v", s.Rules)
	}
}

func TestQualityScore_ClarityVagueOrLongSentence(t *testing.T) {
	// Vague words ("something", "things") or very long sentence → score < 100 and "clarity" in Rules
	s := ScorePromptQuality("You are a helper. Role: X. Context: Y. Subject: Z. Task: do something with things. Output: result. Constraint: only once.")
	if s.Score >= 100 {
		t.Errorf("expected penalty for clarity (vague words), got score %d", s.Score)
	}
	found := false
	for _, r := range s.Rules {
		if r == "clarity" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected clarity in Rules, got %v", s.Rules)
	}
}

func TestQualityScore_MissingConstraints(t *testing.T) {
	// No "do not"/"only"/"max"/"constraint" → "missing_constraints" in Rules and penalty
	s := ScorePromptQuality("You are a helper. Role: X. Context: Y. Subject: Z. Task: do it. Output: result.")
	found := false
	for _, r := range s.Rules {
		if r == "missing_constraints" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_constraints in Rules, got %v", s.Rules)
	}
	if s.Score >= 100 {
		t.Errorf("expected penalty for missing constraints, got score %d", s.Score)
	}
}

func TestQualityScore_OverbroadScope(t *testing.T) {
	// "everything" or "all" or "always" without narrowing → "overbroad_scope" in Rules
	s := ScorePromptQuality("You are a helper. Role: X. Do everything the user asks. Context subject task output constraint.")
	found := false
	for _, r := range s.Rules {
		if r == "overbroad_scope" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected overbroad_scope in Rules, got %v", s.Rules)
	}
	if s.Score >= 100 {
		t.Errorf("expected penalty for overbroad scope, got score %d", s.Score)
	}
}

func TestQualityScore_MissingPersona(t *testing.T) {
	// No "you are"/"role"/"acting as" → "missing_persona" in Rules
	s := ScorePromptQuality("Context: X. Subject: Y. Task: Z. Output: O. Constraint: only once.")
	found := false
	for _, r := range s.Rules {
		if r == "missing_persona" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_persona in Rules, got %v", s.Rules)
	}
	if s.Score >= 100 {
		t.Errorf("expected penalty for missing persona, got score %d", s.Score)
	}
}
