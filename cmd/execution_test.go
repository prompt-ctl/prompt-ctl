package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/prompt-ctl/promptctl/config"
	"github.com/prompt-ctl/promptctl/llm"
)

// --- enrichFileVars is tested in helpers_test.go ---

// --- runSingleExecution ---

func TestRunSingleExecution_Success(t *testing.T) {
	client := mockClient{score: 80, cost: 0.01}
	result, err := runSingleExecution(client, "test prompt", "test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Content == "" {
		t.Error("content should not be empty")
	}
	if result.Cost != 0.01 {
		t.Errorf("cost = %f, want 0.01", result.Cost)
	}
	if result.LatencyMs <= 0 {
		// Latency could be 0 for very fast mock calls
		// Just verify it's non-negative
		if result.LatencyMs < 0 {
			t.Errorf("latency should be non-negative, got %d", result.LatencyMs)
		}
	}
}

func TestRunSingleExecution_ClientError(t *testing.T) {
	client := &mockClientSequence{
		responses: nil,
	}
	_, err := runSingleExecution(client, "test", "model")
	// When responses is nil, callCount >= len(responses) returns nil, nil
	// which should produce "nil completion result" error
	if err == nil {
		t.Fatal("expected error for nil result")
	}
}

// --- extractFieldFromYAML ---

func TestExtractFieldFromYAML_BasicField(t *testing.T) {
	content := "name: review\ndescription: A code review template\nbody: |\n  hello"
	if got := extractFieldFromYAML(content, "name"); got != "review" {
		t.Errorf("extractFieldFromYAML(name) = %q, want review", got)
	}
	if got := extractFieldFromYAML(content, "description"); got != "A code review template" {
		t.Errorf("extractFieldFromYAML(description) = %q", got)
	}
}

func TestExtractFieldFromYAML_QuotedField(t *testing.T) {
	content := `name: "quoted-name"`
	if got := extractFieldFromYAML(content, "name"); got != "quoted-name" {
		t.Errorf("extractFieldFromYAML(quoted) = %q, want quoted-name", got)
	}
}

func TestExtractFieldFromYAML_MissingField(t *testing.T) {
	content := "name: test"
	if got := extractFieldFromYAML(content, "missing"); got != "" {
		t.Errorf("extractFieldFromYAML(missing) = %q, want empty", got)
	}
}

// --- extractBodyFromYAML ---

func TestExtractBodyFromYAML_WithBody(t *testing.T) {
	content := "name: test\nbody: |\n  hello world\n  second line"
	body := extractBodyFromYAML(content)
	if body == "" {
		t.Fatal("body should not be empty")
	}
	if body != "hello world\nsecond line" {
		t.Errorf("body = %q", body)
	}
}

func TestExtractBodyFromYAML_NoBody(t *testing.T) {
	content := "name: test\ndescription: no body"
	body := extractBodyFromYAML(content)
	if body != "" {
		t.Errorf("expected empty body, got %q", body)
	}
}

// --- extractVariablesFromYAML ---

func TestExtractVariablesFromYAML_MultipleVars(t *testing.T) {
	content := `name: test
variables:
  - name: file
    description: Path to file
    required: true
  - name: focus
    description: Focus area
    default: general
body: |
  hello`
	vars := extractVariablesFromYAML(content)
	if len(vars) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(vars))
	}
	if vars[0].Name != "file" {
		t.Errorf("vars[0].Name = %q, want file", vars[0].Name)
	}
	if !vars[0].Required {
		t.Error("vars[0] should be required")
	}
	if vars[1].Name != "focus" {
		t.Errorf("vars[1].Name = %q, want focus", vars[1].Name)
	}
	if vars[1].Default != "general" {
		t.Errorf("vars[1].Default = %q, want general", vars[1].Default)
	}
}

func TestExtractVariablesFromYAML_NoVars(t *testing.T) {
	content := "name: test\nbody: |\n  hello"
	vars := extractVariablesFromYAML(content)
	if len(vars) != 0 {
		t.Errorf("expected 0 variables, got %d", len(vars))
	}
}

// --- dedentYAMLBlock ---

func TestDedentYAMLBlock_Indented(t *testing.T) {
	input := "\n  line one\n  line two\n"
	got := dedentYAMLBlock(input)
	if got != "line one\nline two" {
		t.Errorf("dedentYAMLBlock = %q", got)
	}
}

func TestDedentYAMLBlock_Empty(t *testing.T) {
	got := dedentYAMLBlock("\n\n\n")
	if got != "" {
		t.Errorf("dedentYAMLBlock(empty) = %q, want empty", got)
	}
}

func TestDedentYAMLBlock_MixedIndent(t *testing.T) {
	input := "\n    deep\n  shallow\n    deep again"
	got := dedentYAMLBlock(input)
	if got != "  deep\nshallow\n  deep again" {
		t.Errorf("dedentYAMLBlock(mixed) = %q", got)
	}
}

// --- runAndAggregate ---

func TestRunAndAggregate_AllSucceed(t *testing.T) {
	client := mockClient{score: 80, cost: 0.01}
	profile := DefaultProfile()
	avgScore, avgCost, avgLatency, failures, err := runAndAggregate(client, "test", "test-model", 3, profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failures != 0 {
		t.Errorf("expected 0 failures, got %d", failures)
	}
	if avgScore <= 0 {
		t.Errorf("avgScore should be positive, got %d", avgScore)
	}
	if avgCost <= 0 {
		t.Errorf("avgCost should be positive, got %f", avgCost)
	}
	if avgLatency < 0 {
		t.Errorf("avgLatency should be non-negative, got %d", avgLatency)
	}
}

func TestRunAndAggregate_AllFail(t *testing.T) {
	client := &mockClientSequence{responses: nil}
	profile := DefaultProfile()
	_, _, _, failures, err := runAndAggregate(client, "test", "model", 3, profile)
	if err == nil {
		t.Fatal("expected error when all repetitions fail")
	}
	if failures != 3 {
		t.Errorf("expected 3 failures, got %d", failures)
	}
}

// --- updateBenchmark ---

func TestUpdateBenchmark_NewBenchmark(t *testing.T) {
	dir := t.TempDir()
	templateDir := filepath.Join(dir, "templates", "test-tmpl")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	meta := map[string]interface{}{
		"current":  "v1",
		"versions": []string{"v1"},
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := filepath.Join(templateDir, "meta.json")
	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{LocalTemplateDir: filepath.Join(dir, "templates")}
	err := updateBenchmark("test-tmpl", "gpt-5", 85, cfg)
	if err != nil {
		t.Fatalf("updateBenchmark error: %v", err)
	}

	// Read back and verify
	readBack, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(readBack, &result); err != nil {
		t.Fatal(err)
	}
	benchmarks, ok := result["benchmarks"].(map[string]interface{})
	if !ok {
		t.Fatal("benchmarks not found in meta.json")
	}
	if benchmarks["gpt-5"] != float64(85) {
		t.Errorf("benchmark score = %v, want 85", benchmarks["gpt-5"])
	}
}

func TestUpdateBenchmark_ExistingBenchmark(t *testing.T) {
	dir := t.TempDir()
	templateDir := filepath.Join(dir, "templates", "test-tmpl")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	meta := map[string]interface{}{
		"current":    "v1",
		"versions":   []string{"v1"},
		"benchmarks": map[string]interface{}{"old-model": float64(70)},
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(templateDir, "meta.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{LocalTemplateDir: filepath.Join(dir, "templates")}
	err := updateBenchmark("test-tmpl", "new-model", 90, cfg)
	if err != nil {
		t.Fatalf("updateBenchmark error: %v", err)
	}

	readBack, _ := os.ReadFile(filepath.Join(templateDir, "meta.json"))
	var result map[string]interface{}
	json.Unmarshal(readBack, &result)
	benchmarks := result["benchmarks"].(map[string]interface{})
	if benchmarks["old-model"] != float64(70) {
		t.Error("existing benchmark should be preserved")
	}
	if benchmarks["new-model"] != float64(90) {
		t.Errorf("new benchmark = %v, want 90", benchmarks["new-model"])
	}
}

func TestUpdateBenchmark_MissingMetaJSON(t *testing.T) {
	cfg := &config.Config{LocalTemplateDir: "/nonexistent"}
	err := updateBenchmark("test-tmpl", "model", 80, cfg)
	if err == nil {
		t.Error("expected error for missing meta.json")
	}
}

// --- loadSpecificTemplateVersion ---
// Note: TestLoadSpecificTemplateVersion_Found and _NotFound are in experiment_baseline_test.go

func TestLoadSpecificTemplateVersion_NameFallbackToFilename(t *testing.T) {
	dir := t.TempDir()
	templateDir := filepath.Join(dir, "templates", "review")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Template without a name field - should fall back to filename
	content := "description: Code review\nbody: |\n  review content"
	if err := os.WriteFile(filepath.Join(templateDir, "v1.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		LocalTemplateDir:  filepath.Join(dir, "templates"),
		GlobalTemplateDir: filepath.Join(dir, "global"),
	}
	tmpl, err := loadSpecificTemplateVersion("review", "v1", cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if tmpl.Name != "v1" {
		t.Errorf("Name should fallback to filename, got %q", tmpl.Name)
	}
}

// --- version command ---

func TestExecute_VersionCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"version", []string{"promptctl", "version"}},
		{"-v", []string{"promptctl", "-v"}},
		{"--version", []string{"promptctl", "--version"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args
			err := Execute()
			if err != nil {
				t.Fatalf("Execute() err = %v", err)
			}
		})
	}
}

// --- saveNewVersion ---

func TestSaveNewVersion_Success(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Also need to chdir so findLocalConfig picks up the right .promptctl
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(dir)

	// Create local template dir structure that config.Load() will discover
	localTemplateDir := filepath.Join(dir, ".promptctl", "templates")
	templateDir := filepath.Join(localTemplateDir, "review")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}

	meta := TemplateMeta{
		Current:  "v1",
		Versions: []string{"v1"},
	}
	metaData, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(templateDir, "meta.json"), metaData, 0644); err != nil {
		t.Fatal(err)
	}

	err := saveNewVersion("review", "new body content")
	if err != nil {
		t.Fatalf("saveNewVersion error: %v", err)
	}

	// Verify new version file exists
	if _, err := os.Stat(filepath.Join(templateDir, "v2.yaml")); os.IsNotExist(err) {
		t.Error("v2.yaml should be created")
	}

	// Verify meta.json updated
	data, _ := os.ReadFile(filepath.Join(templateDir, "meta.json"))
	var updated TemplateMeta
	json.Unmarshal(data, &updated)
	if updated.Current != "v2" {
		t.Errorf("current = %q, want v2", updated.Current)
	}
	if len(updated.Versions) != 2 {
		t.Errorf("versions count = %d, want 2", len(updated.Versions))
	}
}

func TestSaveNewVersion_NotVersioned(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, ".promptctl", "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	err := saveNewVersion("nonexistent", "body")
	if err == nil {
		t.Error("expected error for non-versioned template")
	}
}

// --- runVersion ---

func TestRunVersion_NoArgs(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "version"}
	err := runVersion()
	if err == nil {
		t.Error("expected error with insufficient args")
	}
}

func TestRunVersion_TemplateNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(dir)
	if err := os.MkdirAll(filepath.Join(dir, ".promptctl", "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "version", "nonexistent"}
	err := runVersion()
	if err == nil {
		t.Error("expected error for missing template")
	}
}

func TestRunVersion_InitializeVersioning(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(dir)
	tplDir := filepath.Join(dir, ".promptctl", "templates")
	if err := os.MkdirAll(tplDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a flat template
	if err := os.WriteFile(filepath.Join(tplDir, "mytest.yaml"), []byte("name: mytest\nbody: |\n  hello"), 0644); err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "version", "mytest"}
	err := runVersion()
	if err != nil {
		t.Fatalf("runVersion error: %v", err)
	}

	// Verify versioned structure created
	versionDir := filepath.Join(tplDir, "mytest")
	if _, err := os.Stat(filepath.Join(versionDir, "v1.yaml")); os.IsNotExist(err) {
		t.Error("v1.yaml should be created")
	}
	if _, err := os.Stat(filepath.Join(versionDir, "meta.json")); os.IsNotExist(err) {
		t.Error("meta.json should be created")
	}

	// Verify original was moved
	if _, err := os.Stat(filepath.Join(tplDir, "mytest.yaml")); !os.IsNotExist(err) {
		t.Error("original yaml should have been moved")
	}
}

func TestRunVersion_AlreadyVersioned(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(dir)
	versionDir := filepath.Join(dir, ".promptctl", "templates", "mytest")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatal(err)
	}
	meta := TemplateMeta{Current: "v1", Versions: []string{"v1"}}
	data, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(versionDir, "meta.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "version", "mytest"}
	err := runVersion()
	if err != nil {
		t.Fatalf("already versioned should not error: %v", err)
	}
}

// --- printOptimizeHelp ---

func TestPrintOptimizeHelp_DoesNotPanic(t *testing.T) {
	// Capture stdout to avoid noise
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout; r.Close() }()

	printOptimizeHelp()
	w.Close()
}

// --- experiment ---

func TestRunExperimentWithClient_HelpFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"promptctl", "experiment", "--help"}
	// This should print help and return nil
	err := runExperimentWithClient(mockClient{score: 80, cost: 0.01})
	if err != nil {
		t.Fatalf("experiment --help should not error: %v", err)
	}
}

// Use a mock for the mock client
var _ llm.Client = mockClient{}
var _ llm.Client = &mockClientSequence{}

func init() {
	// Verify interface compatibility at init time
	_ = fmt.Sprintf("%T implements llm.Client", mockClient{})
}
