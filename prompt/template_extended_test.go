package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prompt-ctl/promptctl/config"
)

// --- YAML parsing edge cases ---

func TestParseTemplateFile_AllFieldTypes(t *testing.T) {
	dir := t.TempDir()
	content := `# Comment line
name: full-test
description: A comprehensive test template
variables:
  - name: file
    description: Path to file
    required: true
  - name: focus
    description: Focus area
    default: "general"
  - name: extra
    description: Optional extra
body: |
  <context>
  You are reviewing {{.file}}.
  Focus: {{.focus}}
  </context>
  {{if .extra}}Extra: {{.extra}}{{end}}
`
	path := filepath.Join(dir, "full-test.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tmpl, err := parseTemplateFile(path, false)
	if err != nil {
		t.Fatalf("parseTemplateFile error: %v", err)
	}
	if tmpl.Name != "full-test" {
		t.Errorf("Name = %q, want full-test", tmpl.Name)
	}
	if tmpl.Description != "A comprehensive test template" {
		t.Errorf("Description = %q", tmpl.Description)
	}
	if len(tmpl.Variables) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(tmpl.Variables))
	}
	if !tmpl.Variables[0].Required {
		t.Error("file variable should be required")
	}
	if tmpl.Variables[1].Default != "general" {
		t.Errorf("focus default = %q, want general", tmpl.Variables[1].Default)
	}
	if tmpl.Body == "" {
		t.Error("Body should not be empty")
	}
}

// --- Variable substitution ---

func TestTemplate_Render_SingleVar(t *testing.T) {
	tmpl := &Template{
		Body:      "Hello {{.name}}!",
		Variables: []Variable{{Name: "name"}},
	}
	out, err := tmpl.Render(map[string]string{"name": "World"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello World!" {
		t.Errorf("got %q, want %q", out, "Hello World!")
	}
}

func TestTemplate_Render_MultipleVars(t *testing.T) {
	tmpl := &Template{
		Body:      "File: {{.file}}, Focus: {{.focus}}, Lang: {{.lang}}",
		Variables: []Variable{{Name: "file"}, {Name: "focus"}, {Name: "lang"}},
	}
	out, err := tmpl.Render(map[string]string{
		"file":  "main.go",
		"focus": "security",
		"lang":  "go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "File: main.go, Focus: security, Lang: go" {
		t.Errorf("got %q", out)
	}
}

func TestTemplate_Render_NestedConditionals(t *testing.T) {
	tmpl := &Template{
		Body: "A {{if .x}}X={{.x}}{{end}} B {{if .y}}Y={{.y}}{{end}} C",
		Variables: []Variable{
			{Name: "x", Default: ""},
			{Name: "y", Default: ""},
		},
	}
	// Both set
	out, _ := tmpl.Render(map[string]string{"x": "1", "y": "2"})
	if !strings.Contains(out, "X=1") || !strings.Contains(out, "Y=2") {
		t.Errorf("both set: got %q", out)
	}

	// Only x set
	out, _ = tmpl.Render(map[string]string{"x": "1"})
	if !strings.Contains(out, "X=1") {
		t.Errorf("x only: got %q", out)
	}
	if strings.Contains(out, "Y=") {
		t.Errorf("y should not be rendered: got %q", out)
	}
}

// --- Missing required variable error ---

func TestTemplate_Render_MissingRequiredVar_Error(t *testing.T) {
	tmpl := &Template{
		Body: "{{.required_var}}",
		Variables: []Variable{
			{Name: "required_var", Required: true},
		},
	}
	_, err := tmpl.Render(map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing required variable")
	}
	if !strings.Contains(err.Error(), "required_var") {
		t.Errorf("error should mention the variable name: %v", err)
	}
}

func TestTemplate_Render_RequiredVarWithDefault_NoError(t *testing.T) {
	tmpl := &Template{
		Body: "{{.var}}",
		Variables: []Variable{
			{Name: "var", Required: true, Default: "fallback"},
		},
	}
	out, err := tmpl.Render(map[string]string{})
	if err != nil {
		t.Fatalf("required var with default should not error: %v", err)
	}
	if out != "fallback" {
		t.Errorf("got %q, want fallback", out)
	}
}

// --- Default variable values ---

func TestTemplate_Render_DefaultValues(t *testing.T) {
	tmpl := &Template{
		Body: "{{.a}} {{.b}} {{.c}}",
		Variables: []Variable{
			{Name: "a", Default: "alpha"},
			{Name: "b", Default: "beta"},
			{Name: "c"},
		},
	}
	out, err := tmpl.Render(map[string]string{"c": "gamma"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "alpha beta gamma" {
		t.Errorf("got %q, want 'alpha beta gamma'", out)
	}
}

// --- Special characters in variables ---

func TestTemplate_Render_SpecialCharsInValues(t *testing.T) {
	tmpl := &Template{
		Body:      "Content: {{.text}}",
		Variables: []Variable{{Name: "text"}},
	}
	tests := []struct {
		name  string
		value string
	}{
		{"quotes", `He said "hello"`},
		{"newlines", "line1\nline2\nline3"},
		{"tabs", "col1\tcol2\tcol3"},
		{"unicode", "Hello 世界 🌍"},
		{"backslashes", `path\to\file`},
		{"angle brackets", "<tag>content</tag>"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tmpl.Render(map[string]string{"text": tt.value})
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if !strings.Contains(out, tt.value) {
				t.Errorf("output should contain %q, got %q", tt.value, out)
			}
		})
	}
}

// --- Template file not found ---

func TestParseTemplateFile_NotFound(t *testing.T) {
	_, err := parseTemplateFile("/nonexistent/path/template.yaml", false)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// --- Invalid YAML syntax ---

func TestParseTemplateFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	tmpl, err := parseTemplateFile(path, false)
	if err != nil {
		t.Fatalf("empty file should parse without error: %v", err)
	}
	// Name should fall back to filename
	if tmpl.Name != "empty" {
		t.Errorf("Name = %q, want empty (from filename)", tmpl.Name)
	}
}

func TestParseTemplateFile_NoBodySection(t *testing.T) {
	dir := t.TempDir()
	content := "name: no-body\ndescription: has no body section"
	path := filepath.Join(dir, "no-body.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	tmpl, err := parseTemplateFile(path, false)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if tmpl.Body != "" {
		t.Errorf("Body should be empty when no body section, got %q", tmpl.Body)
	}
}

// --- Variable name validation ---

func TestIsValidTemplateName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"review", true},
		{"code-review", true},
		{"my_template", true},
		{"Template123", true},
		{"", false},
		{"../etc/passwd", false},
		{"has space", false},
		{"has.dot", false},
		{strings.Repeat("a", 129), false},
		{strings.Repeat("a", 128), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidTemplateName(tt.name)
			if got != tt.want {
				t.Errorf("IsValidTemplateName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// --- MinimalTemplate ---

func TestMinimalTemplate(t *testing.T) {
	result := MinimalTemplate("test-name", "line one\nline two")
	if !strings.Contains(result, "name: test-name") {
		t.Error("should contain name")
	}
	if !strings.Contains(result, "body: |") {
		t.Error("should contain body marker")
	}
	if !strings.Contains(result, "  line one") {
		t.Error("body lines should be indented")
	}
}

// --- LoadTemplate with invalid name ---

func TestLoadTemplate_InvalidName(t *testing.T) {
	cfg := &config.Config{
		GlobalTemplateDir: "/tmp",
		LocalTemplateDir:  "/tmp",
	}
	_, err := LoadTemplate("../hack", cfg)
	if err == nil {
		t.Error("expected error for path traversal name")
	}
}

// --- LoadTemplate versioned folder ---

func TestLoadTemplate_VersionedFolder(t *testing.T) {
	dir := t.TempDir()
	versionDir := filepath.Join(dir, "global", "review")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create meta.json
	metaContent := `{"current":"v2","versions":["v1","v2"]}`
	if err := os.WriteFile(filepath.Join(versionDir, "meta.json"), []byte(metaContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Create v2.yaml
	if err := os.WriteFile(filepath.Join(versionDir, "v2.yaml"), []byte("name: review\ndescription: v2\nbody: |\n  v2 body"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GlobalTemplateDir: filepath.Join(dir, "global"),
		LocalTemplateDir:  filepath.Join(dir, "local"),
	}
	tmpl, err := LoadTemplate("review", cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(tmpl.Body, "v2 body") {
		t.Errorf("should load v2 body, got %q", tmpl.Body)
	}
}

// --- ListTemplates with local override ---

func TestListTemplates_LocalOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global")
	local := filepath.Join(dir, "local")
	os.MkdirAll(global, 0755)
	os.MkdirAll(local, 0755)
	os.WriteFile(filepath.Join(global, "review.yaml"), []byte("name: review\ndescription: global\nbody: |\n  x"), 0644)
	os.WriteFile(filepath.Join(local, "review.yaml"), []byte("name: review\ndescription: local\nbody: |\n  x"), 0644)
	cfg := &config.Config{GlobalTemplateDir: global, LocalTemplateDir: local}
	list, err := ListTemplates(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 template (local overrides), got %d", len(list))
	}
	if list[0].Description != "local" {
		t.Errorf("should use local template, got description=%q", list[0].Description)
	}
	if !list[0].IsLocal {
		t.Error("should be marked as local")
	}
}

// --- dedent edge cases ---

func TestDedent_AllEmptyLines(t *testing.T) {
	result := dedent("\n\n\n")
	if result != "" {
		t.Errorf("all empty lines should return empty, got %q", result)
	}
}

func TestDedent_NoIndent(t *testing.T) {
	result := dedent("\nno indent\nsecond line")
	if result != "no indent\nsecond line" {
		t.Errorf("no indent: got %q", result)
	}
}

func TestDedent_WithBlankLines(t *testing.T) {
	result := dedent("\n  first\n\n  third")
	if !strings.Contains(result, "first") || !strings.Contains(result, "third") {
		t.Errorf("should preserve content across blank lines: got %q", result)
	}
}

// --- extractField edge cases ---

func TestExtractField_KeyAtEnd(t *testing.T) {
	content := "name: test\ndescription: last field"
	val := extractField(content, "description")
	if val != "last field" {
		t.Errorf("extractField at end = %q", val)
	}
}

func TestExtractField_EmptyValue(t *testing.T) {
	content := "name:\ndescription: x"
	val := extractField(content, "name")
	if val != "" {
		t.Errorf("empty value: got %q", val)
	}
}

// --- extractVariables edge cases ---

func TestExtractVariables_WithComments(t *testing.T) {
	content := `name: test
variables:
  # This is a comment
  - name: var1
    description: First var
  - name: var2
body: |
  hello`
	vars := extractVariables(content)
	if len(vars) != 2 {
		t.Errorf("expected 2 variables with comments, got %d", len(vars))
	}
}

func TestExtractVariables_EmptySection(t *testing.T) {
	content := "name: test\nvariables:\nbody: |\n  hello"
	vars := extractVariables(content)
	if len(vars) != 0 {
		t.Errorf("expected 0 variables for empty section, got %d", len(vars))
	}
}

func TestExtractVariables_QuotedDefault(t *testing.T) {
	content := `variables:
  - name: style
    default: "concise"
body: |
  x`
	vars := extractVariables(content)
	if len(vars) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(vars))
	}
	if vars[0].Default != "concise" {
		t.Errorf("quoted default = %q, want concise", vars[0].Default)
	}
}

// --- FindTemplatePath ---

func TestFindTemplatePath_InvalidName(t *testing.T) {
	cfg := &config.Config{
		GlobalTemplateDir: "/tmp",
		LocalTemplateDir:  "/tmp",
	}
	_, err := FindTemplatePath("../hack", cfg)
	if err == nil {
		t.Error("expected error for invalid name")
	}
}
