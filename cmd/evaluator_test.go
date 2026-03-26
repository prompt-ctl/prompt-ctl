package cmd

import (
	"testing"
)

func TestEvaluate_JSONMode_AllKeysPresent(t *testing.T) {
	content := `{"summary":"good","critical_issues":[],"performance_issues":[],"maintainability_issues":[],"line_suggestions":[{"line":1,"suggestion":"test"}]}`
	profile := DefaultProfile()
	score, jsonInvalid := Evaluate(content, 0.01, 2000, profile)
	if jsonInvalid {
		t.Error("valid JSON should not be marked invalid")
	}
	if score <= 0 {
		t.Errorf("score should be positive, got %d", score)
	}
}

func TestEvaluate_JSONMode_MissingKeys(t *testing.T) {
	content := `{"summary":"only summary"}`
	profile := DefaultProfile()
	score, jsonInvalid := Evaluate(content, 0.01, 2000, profile)
	if jsonInvalid {
		t.Error("valid JSON with missing keys should not be marked invalid")
	}
	// Score should be lower than when all keys are present
	allKeys := `{"summary":"good","critical_issues":[],"performance_issues":[],"maintainability_issues":[],"line_suggestions":[{"line":1,"suggestion":"test"}]}`
	fullScore, _ := Evaluate(allKeys, 0.01, 2000, profile)
	if score >= fullScore {
		t.Errorf("partial keys score (%d) should be less than full keys score (%d)", score, fullScore)
	}
}

func TestEvaluate_InvalidJSON_StrictMode(t *testing.T) {
	content := "not json at all"
	profile := DefaultProfile()
	profile.StrictJSON = true
	_, jsonInvalid := Evaluate(content, 0.01, 2000, profile)
	if !jsonInvalid {
		t.Error("invalid JSON should be marked invalid in strict mode")
	}
}

func TestEvaluate_JSONInFencedBlock(t *testing.T) {
	content := "Here is the result:\n```json\n{\"summary\":\"test\",\"critical_issues\":[],\"performance_issues\":[],\"maintainability_issues\":[],\"line_suggestions\":[]}\n```\nDone."
	profile := DefaultProfile()
	score, jsonInvalid := Evaluate(content, 0.01, 2000, profile)
	if jsonInvalid {
		t.Error("JSON in fenced block should be parseable")
	}
	if score <= 0 {
		t.Errorf("score should be positive for fenced JSON, got %d", score)
	}
}

func TestEvaluate_DepthScore_LongContent(t *testing.T) {
	// Content longer than MinLength should get depth score
	longContent := make([]byte, 600)
	for i := range longContent {
		longContent[i] = 'a'
	}
	profile := EvaluationProfile{
		RequireJSON:     false,
		MinLength:       500,
		MaxCost:         0.05,
		WeightDepth:     20,
		WeightEfficiency: 20,
	}
	score, _ := Evaluate(string(longContent), 0.01, 2000, profile)
	if score < 20 {
		t.Errorf("long content should get depth score, got %d", score)
	}
}

func TestEvaluate_DepthScore_ShortContent(t *testing.T) {
	profile := EvaluationProfile{
		RequireJSON:     false,
		MinLength:       500,
		MaxCost:         0.05,
		WeightDepth:     20,
		WeightEfficiency: 20,
	}
	score, _ := Evaluate("short", 0.01, 2000, profile)
	// Should not get depth score but should get efficiency score
	if score > 20 {
		t.Errorf("short content should not get depth score, got %d", score)
	}
}

func TestEvaluate_EfficiencyScore_LowCostFastLatency(t *testing.T) {
	profile := EvaluationProfile{
		RequireJSON:      false,
		MinLength:        0,
		MaxCost:          0.05,
		WeightEfficiency: 20,
	}
	score, _ := Evaluate("x", 0.01, 2000, profile)
	if score != 20 {
		t.Errorf("low cost + fast latency should get full efficiency, got %d", score)
	}
}

func TestEvaluate_EfficiencyScore_HighCostSlowLatency(t *testing.T) {
	profile := EvaluationProfile{
		RequireJSON:      false,
		MinLength:        0,
		MaxCost:          0.01,
		WeightEfficiency: 20,
	}
	score, _ := Evaluate("x", 0.10, 5000, profile)
	if score != 0 {
		t.Errorf("high cost + slow latency should get no efficiency, got %d", score)
	}
}

func TestEvaluate_ScoreCappedAt100(t *testing.T) {
	profile := EvaluationProfile{
		RequireJSON:       false,
		MinLength:         0,
		MaxCost:           1.0,
		WeightStructure:   50,
		WeightSpecificity: 50,
		WeightDepth:       50,
		WeightEfficiency:  50,
	}
	score, _ := Evaluate("x", 0.001, 1000, profile)
	if score > 100 {
		t.Errorf("score should be capped at 100, got %d", score)
	}
}

func TestDefaultProfile_HasRequiredFields(t *testing.T) {
	p := DefaultProfile()
	if !p.RequireJSON {
		t.Error("default profile should require JSON")
	}
	if len(p.RequiredJSONKeys) == 0 {
		t.Error("default profile should have required JSON keys")
	}
	if !p.StrictJSON {
		t.Error("default profile should have strict JSON")
	}
	if p.MinLength <= 0 {
		t.Errorf("default MinLength should be positive, got %d", p.MinLength)
	}
	total := p.WeightStructure + p.WeightSpecificity + p.WeightDepth + p.WeightEfficiency
	if total != 100 {
		t.Errorf("weights should sum to 100, got %d", total)
	}
}

func TestExtractJSON_PlainJSON(t *testing.T) {
	content := `{"key": "value"}`
	parsed, err := extractJSON(content)
	if err != nil {
		t.Fatalf("extractJSON error: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("expected key=value, got %v", parsed["key"])
	}
}

func TestExtractJSON_FencedBlock(t *testing.T) {
	content := "Some text\n```json\n{\"foo\": \"bar\"}\n```\nMore text"
	parsed, err := extractJSON(content)
	if err != nil {
		t.Fatalf("extractJSON error: %v", err)
	}
	if parsed["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", parsed["foo"])
	}
}

func TestExtractJSON_EmbeddedInText(t *testing.T) {
	content := "Here is the result: {\"a\": 1} and more text"
	parsed, err := extractJSON(content)
	if err != nil {
		t.Fatalf("extractJSON error: %v", err)
	}
	if parsed["a"] != float64(1) {
		t.Errorf("expected a=1, got %v", parsed["a"])
	}
}

func TestExtractJSON_InvalidJSON(t *testing.T) {
	_, err := extractJSON("no json here")
	if err == nil {
		t.Error("expected error for non-JSON content")
	}
}

func TestExtractJSON_FencedBlockWithoutLanguageTag(t *testing.T) {
	content := "```\n{\"x\": 1}\n```"
	parsed, err := extractJSON(content)
	if err != nil {
		t.Fatalf("extractJSON error: %v", err)
	}
	if parsed["x"] != float64(1) {
		t.Errorf("expected x=1, got %v", parsed["x"])
	}
}
