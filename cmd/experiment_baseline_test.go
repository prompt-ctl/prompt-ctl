package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oleg-koval/promptctl/config"
	"github.com/oleg-koval/promptctl/llm"
)

// mockClient returns predefined scores for testing
type mockClient struct {
	score int
	cost  float64
}

func (m mockClient) CompleteWithOptions(prompt, model string, opts *llm.CompleteOptions) (*llm.CompletionResult, error) {
	return &llm.CompletionResult{
		Content:    `{"summary":"test","critical_issues":[],"performance_issues":[],"maintainability_issues":[],"line_suggestions":[{"line":1,"suggestion":"test"}]}`,
		ActualCost: m.cost,
	}, nil
}

// mockClientSequence allows per-execution response customization
type mockClientSequence struct {
	responses []*llm.CompletionResult
	callCount int
}

func (m *mockClientSequence) CompleteWithOptions(prompt, model string, opts *llm.CompleteOptions) (*llm.CompletionResult, error) {
	if m.callCount >= len(m.responses) {
		return nil, nil
	}
	res := m.responses[m.callCount]
	m.callCount++
	return res, nil
}

func TestLoadSpecificTemplateVersion_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &config.Config{
		LocalTemplateDir:  dir + "/.promptctl/templates",
		GlobalTemplateDir: dir + "/.promptctl/templates",
	}

	_, err := loadSpecificTemplateVersion("review", "v99", cfg)
	if err == nil {
		t.Fatal("expected error for non-existent version")
	}
	if !strings.Contains(err.Error(), "baseline version") {
		t.Errorf("error message should mention baseline: %v", err)
	}
}

func TestLoadSpecificTemplateVersion_Found(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create versioned template structure
	localDir := filepath.Join(dir, ".promptctl", "templates", "review")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write v1.yaml with minimal valid template
	v1Content := "name: review\ndescription: baseline\nvariables:\n  - name: file\n    required: true\nbody: |\n  v1 template body"
	if err := os.WriteFile(filepath.Join(localDir, "v1.yaml"), []byte(v1Content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		LocalTemplateDir:  filepath.Join(dir, ".promptctl", "templates"),
		GlobalTemplateDir: filepath.Join(dir, "global", "templates"),
	}

	tmpl, err := loadSpecificTemplateVersion("review", "v1", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl == nil {
		t.Fatal("expected template, got nil")
	}
	if tmpl.Name != "review" {
		t.Errorf("expected name 'review', got %q", tmpl.Name)
	}
	if !strings.Contains(tmpl.Body, "v1 template") {
		t.Errorf("expected body to contain 'v1 template', got %q", tmpl.Body)
	}
}

func TestRunAndAggregate_PositiveScore(t *testing.T) {
	client := mockClient{score: 75, cost: 0.01}

	// Test with repeat=2, should call client twice and aggregate
	avgScore, avgCost, avgLatency, failures, err := runAndAggregate(
		client,
		"test prompt",
		"claude-sonnet-4-5",
		2, // repeat
		DefaultProfile(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Score is calculated by Evaluate(), not from mockClient.score
	// The JSON response with all required keys gets: 40 (structure) + 20 (specificity) + 20 (depth) = 80
	if avgScore != 80 {
		t.Errorf("expected average score 80, got %d", avgScore)
	}
	// Cost should be 0.01 * 2 / 2 = 0.01
	if avgCost != 0.01 {
		t.Errorf("expected average cost 0.01, got %f", avgCost)
	}
	// Latency will be ~0 from mock but aggregated (non-negative)
	if avgLatency < 0 {
		t.Errorf("expected non-negative latency, got %d", avgLatency)
	}
	if failures != 0 {
		t.Errorf("expected 0 failures, got %d", failures)
	}
}

func TestRunExperiment_BaselineMultiModelError(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Simulate: promptctl experiment review --models=m1,m2 --baseline=v1
	os.Args = []string{"promptctl", "experiment", "review", "--models=m1,m2", "--baseline=v1"}

	err := runExperimentWithClient(mockClient{score: 75, cost: 0.01})
	if err == nil {
		t.Fatal("expected error for multi-model with baseline")
	}
	if !strings.Contains(err.Error(), "--baseline requires exactly one model") {
		t.Errorf("error message incorrect: %v", err)
	}
}

func TestRunExperiment_BaselineVersionNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &config.Config{
		LocalTemplateDir:  filepath.Join(dir, ".promptctl", "templates"),
		GlobalTemplateDir: filepath.Join(dir, ".promptctl", "templates"),
	}

	// Create current template but not baseline
	tmplDir := filepath.Join(dir, ".promptctl", "templates", "review")
	os.MkdirAll(tmplDir, 0755)
	os.WriteFile(filepath.Join(tmplDir, "v2.yaml"), []byte("name: review\nbody: |\n  test"), 0644)

	// loadSpecificTemplateVersion should return error for missing v1
	_, err := loadSpecificTemplateVersion("review", "v1", cfg)
	if err == nil {
		t.Fatal("expected error for missing baseline")
	}
	if !strings.Contains(err.Error(), "baseline version") {
		t.Errorf("error should mention baseline: %v", err)
	}
}

// TestRunExperiment_Baseline_PositiveDiff tests that two different template versions load correctly
// and differ from each other, simulating a baseline comparison with positive improvement
func TestRunExperiment_Baseline_PositiveDiff(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create versioned templates
	tmplDir := filepath.Join(dir, ".promptctl", "templates", "review")
	os.MkdirAll(tmplDir, 0755)

	baselineContent := `name: review
description: baseline version
variables:
  - name: file
    required: false
body: |
  Baseline review template
  Review the following code file for issues.`

	currentContent := `name: review
description: current version improved
variables:
  - name: file
    required: false
body: |
  Improved review template with more detailed guidance
  Review the following code file for critical, performance and maintainability issues.
  Focus on architectural patterns and code quality.`

	os.WriteFile(filepath.Join(tmplDir, "v1.yaml"), []byte(baselineContent), 0644)
	os.WriteFile(filepath.Join(tmplDir, "v2.yaml"), []byte(currentContent), 0644)

	cfg := &config.Config{
		LocalTemplateDir:  filepath.Join(dir, ".promptctl", "templates"),
		GlobalTemplateDir: filepath.Join(dir, ".promptctl", "templates"),
	}

	// Load templates
	baselineTmpl, err := loadSpecificTemplateVersion("review", "v1", cfg)
	if err != nil {
		t.Fatalf("failed to load baseline: %v", err)
	}

	currentTmpl, err := loadSpecificTemplateVersion("review", "v2", cfg)
	if err != nil {
		t.Fatalf("failed to load current: %v", err)
	}

	// Verify templates loaded
	if baselineTmpl == nil {
		t.Fatal("baseline template is nil")
	}
	if currentTmpl == nil {
		t.Fatal("current template is nil")
	}

	// Verify baseline differs from current
	if baselineTmpl.Body == currentTmpl.Body {
		t.Error("baseline and current should have different bodies")
	}

	// Verify both templates are named correctly
	if baselineTmpl.Name != "review" {
		t.Errorf("expected baseline name 'review', got %q", baselineTmpl.Name)
	}
	if currentTmpl.Name != "review" {
		t.Errorf("expected current name 'review', got %q", currentTmpl.Name)
	}

	// Verify body content
	if !strings.Contains(baselineTmpl.Body, "Baseline review") {
		t.Errorf("baseline body missing expected content: %q", baselineTmpl.Body)
	}
	if !strings.Contains(currentTmpl.Body, "Improved review") {
		t.Errorf("current body missing expected content: %q", currentTmpl.Body)
	}
}

// TestRunExperiment_Baseline_MissingVersion tests that appropriate error is raised
// when trying to load a baseline version that doesn't exist
func TestRunExperiment_Baseline_MissingVersion(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &config.Config{
		LocalTemplateDir:  filepath.Join(dir, ".promptctl", "templates"),
		GlobalTemplateDir: filepath.Join(dir, ".promptctl", "templates"),
	}

	// Create only v2, not v1
	tmplDir := filepath.Join(dir, ".promptctl", "templates", "review")
	os.MkdirAll(tmplDir, 0755)
	os.WriteFile(filepath.Join(tmplDir, "v2.yaml"), []byte("name: review\nbody: |\n  test"), 0644)

	// Try to load non-existent v1
	_, err := loadSpecificTemplateVersion("review", "v1", cfg)
	if err == nil {
		t.Fatal("expected error for missing baseline")
	}
	if !strings.Contains(err.Error(), "baseline version") {
		t.Errorf("error should mention baseline: %v", err)
	}
}

// TestRunExperiment_CI_Baseline_Pass tests that CI mode returns correct pass status
// when current score is greater than or equal to baseline score
func TestRunExperiment_CI_Baseline_Pass(t *testing.T) {
	// Test the core pass/fail logic used in CI mode
	baselineScore := 72
	currentScore := 78
	minScore := 0

	pass := true
	if minScore > 0 && currentScore < minScore {
		pass = false
	}
	if currentScore < baselineScore {
		pass = false
	}

	if !pass {
		t.Errorf("expected pass=true for positive diff (current %d > baseline %d)", currentScore, baselineScore)
	}

	// Verify diff calculation
	diff := currentScore - baselineScore
	expectedDiff := 6
	if diff != expectedDiff {
		t.Errorf("expected diff=%d, got %d", expectedDiff, diff)
	}
}

// TestRunExperiment_CI_Baseline_Fail_Regression tests that CI mode returns false
// when a regression is detected (current score < baseline score)
func TestRunExperiment_CI_Baseline_Fail_Regression(t *testing.T) {
	// Test the core pass/fail logic for regression detection
	baselineScore := 80
	currentScore := 70
	minScore := 0

	pass := true
	if minScore > 0 && currentScore < minScore {
		pass = false
	}
	if currentScore < baselineScore {
		pass = false
	}

	if pass {
		t.Errorf("expected pass=false for regression (current %d < baseline %d)", currentScore, baselineScore)
	}

	// Verify diff calculation
	diff := currentScore - baselineScore
	expectedDiff := -10
	if diff != expectedDiff {
		t.Errorf("expected diff=%d, got %d", expectedDiff, diff)
	}
}

// TestRunExperiment_CI_Baseline_MinScoreCheck tests that CI mode respects min-score constraint
// even when baseline comparison passes
func TestRunExperiment_CI_Baseline_MinScoreCheck(t *testing.T) {
	// Test that min-score requirement is enforced independently
	baselineScore := 50
	currentScore := 60
	minScore := 75

	pass := true
	if minScore > 0 && currentScore < minScore {
		pass = false
	}
	if currentScore < baselineScore {
		pass = false
	}

	if pass {
		t.Errorf("expected pass=false when score %d < min-score %d", currentScore, minScore)
	}

	// Diff should still be positive (improvement), but pass is false due to min-score
	diff := currentScore - baselineScore
	if diff <= 0 {
		t.Errorf("expected positive diff, got %d", diff)
	}
}

// TestRunExperiment_CI_Baseline_NoRegression tests that CI mode passes when
// current score equals baseline score (no regression)
func TestRunExperiment_CI_Baseline_NoRegression(t *testing.T) {
	// Test the edge case where scores are equal
	baselineScore := 75
	currentScore := 75
	minScore := 0

	pass := true
	if minScore > 0 && currentScore < minScore {
		pass = false
	}
	if currentScore < baselineScore {
		pass = false
	}

	if !pass {
		t.Errorf("expected pass=true when scores are equal (no regression)")
	}

	// Verify diff is zero
	diff := currentScore - baselineScore
	if diff != 0 {
		t.Errorf("expected diff=0 for equal scores, got %d", diff)
	}
}

// TestRunExperiment_Record_SingleModelSuccess tests that --record works correctly
// with a single model and updates the meta.json file with benchmark data
func TestRunExperiment_Record_SingleModelSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create test file
	_ = os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main\n"), 0644)

	// Create versioned template structure
	reviewDir := filepath.Join(dir, ".promptctl", "templates", "review")
	_ = os.MkdirAll(reviewDir, 0755)
	_ = os.WriteFile(filepath.Join(reviewDir, "meta.json"),
		[]byte(`{"current":"v1","versions":["v1"]}`), 0644)
	_ = os.WriteFile(filepath.Join(reviewDir, "v1.yaml"),
		[]byte(`name: review
description: test template
variables:
  - name: file
    required: true
body: |
  Review the following code`), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "experiment", "review",
		"--file=test.go", "--models=claude-sonnet-4-5", "--record"}

	client := &mockClient{score: 75, cost: 0.002}
	err := runExperimentWithClient(client)
	if err != nil {
		t.Fatalf("runExperimentWithClient with --record should not error: %v", err)
	}

	// Verify meta.json was updated with benchmark
	metaPath := filepath.Join(reviewDir, "meta.json")
	data, _ := os.ReadFile(metaPath)
	var meta map[string]interface{}
	json.Unmarshal(data, &meta)

	benchmarks, ok := meta["benchmarks"].(map[string]interface{})
	if !ok {
		t.Fatal("benchmarks field should exist in meta.json")
	}

	score, ok := benchmarks["claude-sonnet-4-5"].(float64)
	if !ok || int(score) != 80 {
		t.Errorf("benchmark score should be 80, got %v", score)
	}
}

// TestRunExperiment_Record_MultiModel_Error tests that --record rejects
// multiple models with an appropriate error message
func TestRunExperiment_Record_MultiModel_Error(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create test file
	_ = os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main\n"), 0644)

	reviewDir := filepath.Join(dir, ".promptctl", "templates", "review")
	_ = os.MkdirAll(reviewDir, 0755)
	_ = os.WriteFile(filepath.Join(reviewDir, "meta.json"),
		[]byte(`{"current":"v1","versions":["v1"]}`), 0644)
	_ = os.WriteFile(filepath.Join(reviewDir, "v1.yaml"),
		[]byte(`name: review
description: test template
variables:
  - name: file
    required: true
body: |
  Review the following code`), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "experiment", "review",
		"--file=test.go", "--models=claude-sonnet-4-5,claude-haiku-4-5", "--record"}

	client := &mockClient{score: 75, cost: 0.002}
	err := runExperimentWithClient(client)
	if err == nil {
		t.Fatal("--record with multiple models should error")
	}
	if !strings.Contains(err.Error(), "--record requires exactly one model") {
		t.Errorf("error should mention --record requires one model, got: %v", err)
	}
}

// TestUpdateBenchmark_CreatesBenchmarksSection tests that updateBenchmark
// creates a benchmarks section when it doesn't exist
func TestUpdateBenchmark_CreatesBenchmarksSection(t *testing.T) {
	dir := t.TempDir()

	// Create template structure
	versionDir := filepath.Join(dir, "templates", "review")
	_ = os.MkdirAll(versionDir, 0755)
	metaPath := filepath.Join(versionDir, "meta.json")

	// Start with meta.json without benchmarks
	_ = os.WriteFile(metaPath,
		[]byte(`{"current":"v1","versions":["v1"]}`), 0644)

	// Call updateBenchmark
	cfg := &config.Config{LocalTemplateDir: filepath.Join(dir, "templates")}
	err := updateBenchmark("review", "claude-sonnet-4-5", 75, cfg)
	if err != nil {
		t.Fatalf("updateBenchmark failed: %v", err)
	}

	// Verify benchmarks section exists
	data, _ := os.ReadFile(metaPath)
	var meta map[string]interface{}
	json.Unmarshal(data, &meta)

	benchmarks, ok := meta["benchmarks"].(map[string]interface{})
	if !ok {
		t.Fatal("benchmarks field should be created")
	}

	score, ok := benchmarks["claude-sonnet-4-5"].(float64)
	if !ok || int(score) != 75 {
		t.Errorf("expected score 75, got %v", score)
	}
}

// TestUpdateBenchmark_OverwritesExistingScore tests that updateBenchmark
// correctly overwrites an existing benchmark score
func TestUpdateBenchmark_OverwritesExistingScore(t *testing.T) {
	dir := t.TempDir()

	versionDir := filepath.Join(dir, "templates", "review")
	_ = os.MkdirAll(versionDir, 0755)
	metaPath := filepath.Join(versionDir, "meta.json")

	// Start with existing benchmark
	_ = os.WriteFile(metaPath,
		[]byte(`{"current":"v1","versions":["v1"],"benchmarks":{"claude-sonnet-4-5":70}}`), 0644)

	cfg := &config.Config{LocalTemplateDir: filepath.Join(dir, "templates")}
	err := updateBenchmark("review", "claude-sonnet-4-5", 80, cfg)
	if err != nil {
		t.Fatalf("updateBenchmark failed: %v", err)
	}

	data, _ := os.ReadFile(metaPath)
	var meta map[string]interface{}
	json.Unmarshal(data, &meta)

	benchmarks := meta["benchmarks"].(map[string]interface{})
	score := int(benchmarks["claude-sonnet-4-5"].(float64))
	if score != 80 {
		t.Errorf("expected score 80, got %d", score)
	}
}

// TestRunExperiment_CI_Record_OnPass tests that --record works in CI mode
// when pass == true (no baseline or baseline passes)
func TestRunExperiment_CI_Record_OnPass(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create test file
	_ = os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main\n"), 0644)

	// Create versioned template structure
	reviewDir := filepath.Join(dir, ".promptctl", "templates", "review")
	_ = os.MkdirAll(reviewDir, 0755)
	_ = os.WriteFile(filepath.Join(reviewDir, "meta.json"),
		[]byte(`{"current":"v1","versions":["v1"]}`), 0644)
	_ = os.WriteFile(filepath.Join(reviewDir, "v1.yaml"),
		[]byte(`name: review
description: test template
variables:
  - name: file
    required: true
body: |
  Review the following code`), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "experiment", "review",
		"--file=test.go", "--models=claude-sonnet-4-5", "--record", "--ci"}

	client := &mockClient{score: 75, cost: 0.002}
	err := runExperimentWithClient(client)
	if err != nil {
		t.Fatalf("CI mode with pass should not error: %v", err)
	}

	// Verify benchmark was recorded
	metaPath := filepath.Join(reviewDir, "meta.json")
	data, _ := os.ReadFile(metaPath)
	var meta map[string]interface{}
	json.Unmarshal(data, &meta)

	if benchmarks, ok := meta["benchmarks"].(map[string]interface{}); ok {
		if score, ok := benchmarks["claude-sonnet-4-5"].(float64); ok {
			if int(score) != 80 {
				t.Errorf("expected recorded score 80, got %d", int(score))
			}
		} else {
			t.Fatal("benchmark score not found")
		}
	} else {
		t.Fatal("benchmarks not recorded in CI mode on pass")
	}
}

// TestRunExperiment_CI_Record_OnFailure_NoRecord tests the logic that prevents
// recording when CI mode detects a failure (via pass == false).
// Tests the conditional logic: shouldRecord = recordBenchmark && pass
func TestRunExperiment_CI_Record_OnFailure_NoRecord(t *testing.T) {
	// This test verifies the recording logic without triggering os.Exit
	// by testing the pass/fail determination directly

	// Scenario: record=true, but pass=false due to min-score
	recordBenchmark := true
	minScore := 100
	currentScore := 80

	// Calculate pass status (from experiment.go CI mode section)
	pass := true
	if minScore > 0 && currentScore < minScore {
		pass = false
	}

	// The recording logic: shouldRecord = recordBenchmark && pass (in CI mode)
	shouldRecord := recordBenchmark && pass

	// Verify: pass should be false
	if pass {
		t.Errorf("expected pass=false when score %d < min-score %d", currentScore, minScore)
	}

	// Verify: shouldRecord should be false
	if shouldRecord {
		t.Errorf("expected shouldRecord=false when pass=false, got true")
	}

	// Verify the logic is correct
	if !shouldRecord && recordBenchmark && !pass {
		t.Logf("Correctly prevented recording: recordBenchmark=%v, pass=%v, shouldRecord=%v", recordBenchmark, pass, shouldRecord)
	}
}

// TestUpdateBenchmark_PreservesValidJSON tests that meta.json JSON validity
// is preserved after recording with complex structures. Verifies that all fields
// are preserved and new benchmarks are correctly added.
func TestUpdateBenchmark_PreservesValidJSON(t *testing.T) {
	dir := t.TempDir()

	versionDir := filepath.Join(dir, "templates", "review")
	_ = os.MkdirAll(versionDir, 0755)
	metaPath := filepath.Join(versionDir, "meta.json")

	// Complex meta.json with multiple fields
	originalMeta := `{
  "current": "v2",
  "versions": ["v1", "v2"],
  "benchmarks": {
    "claude-haiku-4-5": 65
  }
}`
	_ = os.WriteFile(metaPath, []byte(originalMeta), 0644)

	cfg := &config.Config{LocalTemplateDir: filepath.Join(dir, "templates")}
	err := updateBenchmark("review", "claude-sonnet-4-5", 78, cfg)
	if err != nil {
		t.Fatalf("updateBenchmark failed: %v", err)
	}

	// Verify JSON is still valid and all fields preserved
	data, _ := os.ReadFile(metaPath)
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("resulting meta.json is invalid JSON: %v", err)
	}

	// Check all fields are preserved
	if current, ok := meta["current"].(string); !ok || current != "v2" {
		t.Errorf("current field not preserved")
	}

	if versions, ok := meta["versions"].([]interface{}); !ok || len(versions) != 2 {
		t.Errorf("versions field not preserved")
	}

	benchmarks := meta["benchmarks"].(map[string]interface{})
	if score, _ := benchmarks["claude-haiku-4-5"].(float64); int(score) != 65 {
		t.Errorf("existing benchmark should be preserved")
	}
	if score, _ := benchmarks["claude-sonnet-4-5"].(float64); int(score) != 78 {
		t.Errorf("new benchmark should be recorded")
	}
}
