# Persistent Benchmark Recording Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add persistent benchmark storage to promptctl's experiment command, enabling regression tracking systems to record model performance over time.

**Architecture:** The `--record` flag stores successful experiment scores in template `meta.json` under a `benchmarks` object keyed by model name. Recording only happens on successful runs in CI mode (pass == true) and always in non-CI mode. The update logic is isolated in a single `updateBenchmark()` function.

**Tech Stack:** Go, JSON unmarshaling/marshaling, file I/O

---

## Task 1: Test Infrastructure and Flag Parsing Tests

**Files:**
- Modify: `cmd/experiment_baseline_test.go` - Add test helpers and test cases for --record
- Test: Verify flag parsing works correctly

**Step 1: Write test for --record flag validation with single model**

Add to `cmd/experiment_baseline_test.go`:

```go
func TestRunExperiment_Record_SingleModelSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Create versioned template
	tplDir := filepath.Join(dir, ".promptctl", "templates")
	versionDir := filepath.Join(tplDir, "review", "v1")
	_ = os.MkdirAll(versionDir, 0755)
	_ = os.WriteFile(filepath.Join(versionDir, "meta.json"),
		[]byte(`{"current":"v1","versions":["v1"]}`), 0644)
	_ = os.WriteFile(filepath.Join(versionDir, "v1.yaml"),
		[]byte(`name: review\ndescription: test\nbody: |\n  test body`), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "experiment", "review",
		"--models=claude-sonnet-4-5", "--record"}

	client := &mockClient{score: 75, cost: 0.002, latency: 1500}
	err := runExperimentWithClient(client)
	if err != nil {
		t.Fatalf("runExperimentWithClient with --record should not error: %v", err)
	}

	// Verify meta.json was updated with benchmark
	metaPath := filepath.Join(versionDir, "meta.json")
	data, _ := os.ReadFile(metaPath)
	var meta map[string]interface{}
	json.Unmarshal(data, &meta)

	benchmarks, ok := meta["benchmarks"].(map[string]interface{})
	if !ok {
		t.Fatal("benchmarks field should exist in meta.json")
	}

	score, ok := benchmarks["claude-sonnet-4-5"].(float64)
	if !ok || int(score) != 75 {
		t.Errorf("benchmark score should be 75, got %v", score)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd -run TestRunExperiment_Record_SingleModelSuccess -v`
Expected: FAIL - "updateBenchmark not defined"

**Step 3: Write test for --record with multiple models (should fail)**

Add to `cmd/experiment_baseline_test.go`:

```go
func TestRunExperiment_Record_MultiModel_Error(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	tplDir := filepath.Join(dir, ".promptctl", "templates")
	versionDir := filepath.Join(tplDir, "review", "v1")
	_ = os.MkdirAll(versionDir, 0755)
	_ = os.WriteFile(filepath.Join(versionDir, "meta.json"),
		[]byte(`{"current":"v1","versions":["v1"]}`), 0644)
	_ = os.WriteFile(filepath.Join(versionDir, "v1.yaml"),
		[]byte(`name: review\ndescription: test\nbody: |\n  test body`), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "experiment", "review",
		"--models=claude-sonnet-4-5,claude-haiku-4-5", "--record"}

	client := &mockClient{score: 75, cost: 0.002, latency: 1500}
	err := runExperimentWithClient(client)
	if err == nil {
		t.Fatal("--record with multiple models should error")
	}
	if !strings.Contains(err.Error(), "--record requires exactly one model") {
		t.Errorf("error should mention --record requires one model, got: %v", err)
	}
}
```

**Step 4: Run tests to verify they fail**

Run: `go test ./cmd -run TestRunExperiment_Record -v`
Expected: FAIL - "updateBenchmark not defined", "functions not implemented"

**Step 5: Commit**

```bash
git add cmd/experiment_baseline_test.go
git commit -m "test: add --record flag tests for single and multi-model validation"
```

---

## Task 2: Add --record Flag Parsing and Validation

**Files:**
- Modify: `cmd/experiment.go:runExperimentWithClient()` - Add flag parsing after baseline parsing

**Step 1: Read current experiment.go to understand flag parsing pattern**

Read: `cmd/experiment.go` around where baseline flag is parsed

**Step 2: Add --record flag parsing**

In `runExperimentWithClient()`, after baseline flag parsing (around line 85-95):

```go
	recordBenchmark := false
	if _, ok := vars["record"]; ok {
		recordBenchmark = true
		delete(vars, "record")
	}
```

**Step 3: Add validation: fail fast if multiple models and --record**

After modelIDs are split (around line 100-110), add:

```go
	if recordBenchmark {
		if len(modelIDs) > 1 {
			return fmt.Errorf("--record requires exactly one model")
		}
	}
```

**Step 4: Run tests to verify validation works**

Run: `go test ./cmd -run TestRunExperiment_Record_MultiModel_Error -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/experiment.go
git commit -m "feat: add --record flag parsing and multi-model validation"
```

---

## Task 3: Implement updateBenchmark() Function

**Files:**
- Modify: `cmd/execution.go` - Add updateBenchmark function

**Step 1: Write failing test for updateBenchmark (creates benchmarks section)**

Add to `cmd/experiment_baseline_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd -run TestUpdateBenchmark_CreatesBenchmarksSection -v`
Expected: FAIL - "updateBenchmark not defined"

**Step 3: Write updateBenchmark function in execution.go**

Add to `cmd/execution.go`:

```go
func updateBenchmark(templateName string, model string, score int, cfg *config.Config) error {
	// Locate meta.json in versioned template directory
	versionDir := filepath.Join(cfg.LocalTemplateDir, templateName)
	metaPath := filepath.Join(versionDir, "meta.json")

	// Read existing meta.json
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read meta.json: %w", err)
	}

	// Unmarshal into generic map to preserve structure
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to parse meta.json: %w", err)
	}

	// Create or get benchmarks map
	var benchmarks map[string]interface{}
	if b, exists := meta["benchmarks"]; exists {
		if bm, ok := b.(map[string]interface{}); ok {
			benchmarks = bm
		} else {
			return fmt.Errorf("benchmarks field is not a valid object")
		}
	} else {
		benchmarks = make(map[string]interface{})
		meta["benchmarks"] = benchmarks
	}

	// Update model score
	benchmarks[model] = score

	// Write back to file
	output, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal meta.json: %w", err)
	}

	if err := os.WriteFile(metaPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write meta.json: %w", err)
	}

	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./cmd -run TestUpdateBenchmark -v`
Expected: PASS

**Step 5: Write test for overwriting existing model score**

Add to `cmd/experiment_baseline_test.go`:

```go
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
```

**Step 6: Run test to verify it passes**

Run: `go test ./cmd -run TestUpdateBenchmark_OverwritesExistingScore -v`
Expected: PASS

**Step 7: Commit**

```bash
git add cmd/execution.go cmd/experiment_baseline_test.go
git commit -m "feat: implement updateBenchmark function for persistent score storage"
```

---

## Task 4: Integrate --record into Experiment Execution Flow

**Files:**
- Modify: `cmd/experiment.go:runExperimentWithClient()` - Call updateBenchmark after success

**Step 1: Read current experiment.go to find where results are finalized**

Locate the section where experiments complete successfully (before returning nil)

**Step 2: Add benchmark recording logic**

After results are aggregated and before non-CI output (around the baseline comparison output section), add:

```go
	// Record benchmark if requested and execution was successful
	if recordBenchmark && len(results) > 0 {
		modelID := modelIDs[0]
		score := results[0].AvgScore
		if err := updateBenchmark(name, modelID, score, cfg); err != nil {
			return fmt.Errorf("failed to record benchmark: %w", err)
		}

		if !ciMode {
			fmt.Printf("Benchmark recorded for %s: %d\n", modelID, score)
		}
	}
```

**Step 3: Run tests to verify integration works**

Run: `go test ./cmd -run TestRunExperiment_Record_SingleModelSuccess -v`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/experiment.go
git commit -m "feat: integrate updateBenchmark into experiment execution flow"
```

---

## Task 5: CI Interaction - Only Record on Pass

**Files:**
- Modify: `cmd/experiment.go:runExperimentWithClient()` - Conditional recording in CI mode

**Step 1: Write test for CI mode with baseline pass**

Add to `cmd/experiment_baseline_test.go`:

```go
func TestRunExperiment_CI_Record_OnPass(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	tplDir := filepath.Join(dir, ".promptctl", "templates")
	versionDir := filepath.Join(tplDir, "review", "v1")
	_ = os.MkdirAll(versionDir, 0755)
	_ = os.WriteFile(filepath.Join(versionDir, "meta.json"),
		[]byte(`{"current":"v1","versions":["v1"]}`), 0644)
	_ = os.WriteFile(filepath.Join(versionDir, "v1.yaml"),
		[]byte(`name: review\ndescription: test\nbody: |\n  test body`), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "experiment", "review",
		"--models=claude-sonnet-4-5", "--record", "--ci"}

	client := &mockClient{score: 75, cost: 0.002, latency: 1500}
	err := runExperimentWithClient(client)
	if err != nil {
		t.Fatalf("CI mode with pass should not error: %v", err)
	}

	// Verify benchmark was recorded
	metaPath := filepath.Join(versionDir, "meta.json")
	data, _ := os.ReadFile(metaPath)
	var meta map[string]interface{}
	json.Unmarshal(data, &meta)

	if benchmarks, ok := meta["benchmarks"].(map[string]interface{}); ok {
		if score, ok := benchmarks["claude-sonnet-4-5"].(float64); ok {
			if int(score) != 75 {
				t.Errorf("expected recorded score 75, got %d", int(score))
			}
		} else {
			t.Fatal("benchmark score not found")
		}
	} else {
		t.Fatal("benchmarks not recorded in CI mode on pass")
	}
}
```

**Step 2: Run test to verify it fails initially**

Run: `go test ./cmd -run TestRunExperiment_CI_Record_OnPass -v`
Expected: FAIL - Logic not implemented yet

**Step 3: Update recording logic to check CI mode and pass status**

Modify the recording section in `runExperimentWithClient()`:

```go
	// Record benchmark if requested
	// In CI mode: only record if pass == true
	// In non-CI mode: always record on success
	shouldRecord := recordBenchmark
	if recordBenchmark && ciMode {
		// In CI mode, compute pass status
		pass := true
		if baselineVersion != "" && len(results) > 0 {
			pass = results[0].Diff >= 0
		}
		if minScore > 0 && len(results) > 0 {
			pass = pass && (results[0].AvgScore >= minScore)
		}
		shouldRecord = pass
	}

	if shouldRecord && len(results) > 0 {
		modelID := modelIDs[0]
		score := results[0].AvgScore
		if err := updateBenchmark(name, modelID, score, cfg); err != nil {
			return fmt.Errorf("failed to record benchmark: %w", err)
		}

		if !ciMode {
			fmt.Printf("Benchmark recorded for %s: %d\n", modelID, score)
		}
	}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd -run TestRunExperiment_CI_Record_OnPass -v`
Expected: PASS

**Step 5: Write test for CI mode with baseline fail (should NOT record)**

Add to `cmd/experiment_baseline_test.go`:

```go
func TestRunExperiment_CI_Record_OnFailure_NoRecord(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	tplDir := filepath.Join(dir, ".promptctl", "templates")
	versionDir := filepath.Join(tplDir, "review", "v1")
	baselineVersionDir := filepath.Join(tplDir, "review", "v0")
	_ = os.MkdirAll(versionDir, 0755)
	_ = os.MkdirAll(baselineVersionDir, 0755)

	_ = os.WriteFile(filepath.Join(versionDir, "meta.json"),
		[]byte(`{"current":"v1","versions":["v0","v1"]}`), 0644)
	_ = os.WriteFile(filepath.Join(versionDir, "v1.yaml"),
		[]byte(`name: review\ndescription: test\nbody: |\n  current body`), 0644)
	_ = os.WriteFile(filepath.Join(baselineVersionDir, "v0.yaml"),
		[]byte(`name: review\ndescription: test\nbody: |\n  baseline body`), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "experiment", "review",
		"--models=claude-sonnet-4-5", "--record", "--baseline=v0", "--ci"}

	// Score lower than baseline = regression
	client := &mockClientSequence{
		responses: []llm.ExecutionResult{
			{Content: "", Cost: 0.002, LatencyMs: 1500},
			{Content: "", Cost: 0.002, LatencyMs: 1500},
		},
		scores: []int{70, 80}, // current: 70, baseline: 80 -> regression
	}

	err := runExperimentWithClient(client)
	if err == nil {
		t.Fatal("regression should cause error in CI mode")
	}

	// Verify benchmark was NOT recorded
	metaPath := filepath.Join(versionDir, "meta.json")
	data, _ := os.ReadFile(metaPath)
	var meta map[string]interface{}
	json.Unmarshal(data, &meta)

	if benchmarks, ok := meta["benchmarks"].(map[string]interface{}); ok {
		if _, exists := benchmarks["claude-sonnet-4-5"]; exists {
			t.Fatal("benchmark should not be recorded on regression")
		}
	}
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./cmd -run TestRunExperiment_CI_Record_OnFailure -v`
Expected: PASS

**Step 7: Commit**

```bash
git add cmd/experiment.go cmd/experiment_baseline_test.go
git commit -m "feat: implement conditional recording - only record on pass in CI mode"
```

---

## Task 6: Comprehensive Integration Tests

**Files:**
- Modify: `cmd/experiment_baseline_test.go` - Add final integration tests

**Step 1: Write test for meta.json JSON validity after recording**

Add to `cmd/experiment_baseline_test.go`:

```go
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
```

**Step 2: Run test to verify it passes**

Run: `go test ./cmd -run TestUpdateBenchmark_PreservesValidJSON -v`
Expected: PASS

**Step 3: Run all --record tests to verify suite passes**

Run: `go test ./cmd -run TestRunExperiment_Record -v`
Expected: All PASS

**Step 4: Run full test suite to verify no regressions**

Run: `go test ./cmd -v 2>&1 | tail -5`
Expected: All tests pass, count increases from 67

**Step 5: Commit**

```bash
git add cmd/experiment_baseline_test.go
git commit -m "test: add comprehensive integration tests for --record feature"
```

---

## Task 7: Help Text Update and Final Validation

**Files:**
- Modify: `cmd/experiment.go:printExperimentHelp()` - Add --record documentation

**Step 1: Read current printExperimentHelp function**

Locate and read `cmd/experiment.go` printExperimentHelp function

**Step 2: Update help text to document --record flag**

Update the help text to include:

```
  --record               Record current score to benchmarks in meta.json
                        (requires single model, non-CI or pass only)
```

Add example showing --record usage:

```
EXAMPLE WITH RECORDING:
  promptctl experiment review --file=main.go --models=claude-sonnet-4-5 --record

  Outputs: Benchmark recorded for claude-sonnet-4-5: 78
```

Update the OPTIONS section to include --record documentation and its constraints.

**Step 3: Build to verify no compilation errors**

Run: `go build -o ./promptctl ./main.go`
Expected: Build successful

**Step 4: Run all tests one final time**

Run: `go test ./cmd -v 2>&1 | tail -10`
Expected: All tests pass, no errors

**Step 5: Commit**

```bash
git add cmd/experiment.go
git commit -m "docs: update experiment help text for --record benchmark feature"
```

---

## Verification Checklist

Before considering this complete:

- [ ] All 10+ new tests pass
- [ ] Original 67 tests still pass
- [ ] Build completes without errors
- [ ] No linter errors (style warnings OK)
- [ ] `updateBenchmark()` properly handles missing benchmarks section
- [ ] `updateBenchmark()` preserves all existing meta.json fields
- [ ] Recording only happens on successful runs
- [ ] CI mode respects pass/fail logic
- [ ] Non-CI mode always records on success
- [ ] Multi-model validation catches --record with multiple models
- [ ] Confirmation message prints in non-CI, silent in CI
- [ ] meta.json remains valid JSON after updates

---

## Architecture Summary

**The complete regression tracking system now includes:**

1. **Versioned Templates** - Templates stored in `templates/<name>/v1/`, `v2/` directories with `meta.json`
2. **Baseline Regression Comparison** - `--baseline=<version>` compares current against any previous version
3. **Persistent Benchmark Storage** - `--record` stores successful scores in `meta.json` for future baseline comparisons

**Key Functions Added:**
- `loadSpecificTemplateVersion()` - Loads any template version
- `runAndAggregate()` - Executes and aggregates scores
- `updateBenchmark()` - Persists scores to meta.json

**Constraints Maintained:**
- No modifications to prompt/ package
- Version logic stays in cmd/ layer
- No code duplication
- Clean JSON handling
- Proper error messages
