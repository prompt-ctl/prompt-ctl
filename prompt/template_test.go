package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oleg-koval/promptctl/config"
)

func TestTemplate_VariableNames(t *testing.T) {
	tmpl := &Template{
		Variables: []Variable{
			{Name: "file", Required: true},
			{Name: "focus", Default: "general"},
		},
	}
	names := tmpl.VariableNames()
	if len(names) != 2 {
		t.Fatalf("len(VariableNames) = %d, want 2", len(names))
	}
	if names[0] != "file" || names[1] != "focus" {
		t.Errorf("VariableNames = %v", names)
	}
}

func TestTemplate_Render_RequiredMissing(t *testing.T) {
	tmpl := &Template{
		Body: "Hello {{.name}}",
		Variables: []Variable{
			{Name: "name", Required: true},
		},
	}
	_, err := tmpl.Render(map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing required var")
	}
}

func TestTemplate_Render_DefaultApplied(t *testing.T) {
	tmpl := &Template{
		Body: "Focus: {{.focus}}",
		Variables: []Variable{
			{Name: "focus", Default: "general"},
		},
	}
	out, err := tmpl.Render(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Focus: general" {
		t.Errorf("got %q", out)
	}
}

func TestTemplate_Render_Placeholders(t *testing.T) {
	tmpl := &Template{
		Body:      "File {{.file_name}} ext {{.file_ext}}",
		Variables: []Variable{},
	}
	out, err := tmpl.Render(map[string]string{"file_name": "main.go", "file_ext": "go"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "File main.go ext go" {
		t.Errorf("got %q", out)
	}
}

func TestTemplate_Render_IfEnd(t *testing.T) {
	tmpl := &Template{
		Body:      "A {{if .opt}}B {{.opt}}{{end}} C",
		Variables: []Variable{{Name: "opt", Default: ""}},
	}
	out1, _ := tmpl.Render(map[string]string{"opt": "yes"})
	if out1 != "A B yes C" {
		t.Errorf("with opt: got %q", out1)
	}
	out2, _ := tmpl.Render(map[string]string{})
	if out2 != "A  C" {
		t.Errorf("without opt: got %q", out2)
	}
}

func TestLoadTemplate_NotFound(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		GlobalTemplateDir: filepath.Join(dir, "templates"),
		LocalTemplateDir:  filepath.Join(dir, "local"),
	}
	_ = os.MkdirAll(cfg.GlobalTemplateDir, 0755)
	_, err := LoadTemplate("nonexistent", cfg)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestLoadTemplate_Found(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "templates")
	_ = os.MkdirAll(global, 0755)
	content := `name: test
description: A test template
variables:
  - name: x
    default: "1"
body: |
  hello {{.x}}
`
	_ = os.WriteFile(filepath.Join(global, "test.yaml"), []byte(content), 0644)
	cfg := &config.Config{
		GlobalTemplateDir: global,
		LocalTemplateDir:  filepath.Join(dir, "local"),
	}
	tmpl, err := LoadTemplate("test", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Name != "test" {
		t.Errorf("Name = %q", tmpl.Name)
	}
	out, _ := tmpl.Render(map[string]string{})
	if out != "hello 1" {
		t.Errorf("Render = %q", out)
	}
}

func TestFindTemplatePath_NotFound(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		GlobalTemplateDir: filepath.Join(dir, "templates"),
		LocalTemplateDir:  filepath.Join(dir, "local"),
	}
	_ = os.MkdirAll(cfg.GlobalTemplateDir, 0755)
	_, err := FindTemplatePath("missing", cfg)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindTemplatePath_Found(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "templates")
	_ = os.MkdirAll(global, 0755)
	path := filepath.Join(global, "foo.yaml")
	_ = os.WriteFile(path, []byte("name: foo\nbody: |\n  x"), 0644)
	cfg := &config.Config{
		GlobalTemplateDir: global,
		LocalTemplateDir:  filepath.Join(dir, "local"),
	}
	got, err := FindTemplatePath("foo", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Errorf("path = %q, want %q", got, path)
	}
}

func TestListTemplates_EmptyDirs(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		GlobalTemplateDir: filepath.Join(dir, "global"),
		LocalTemplateDir:  filepath.Join(dir, "local"),
	}
	_ = os.MkdirAll(cfg.GlobalTemplateDir, 0755)
	_ = os.MkdirAll(cfg.LocalTemplateDir, 0755)
	list, err := ListTemplates(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 templates, got %d", len(list))
	}
}

func TestListTemplates_WithFiles(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global")
	local := filepath.Join(dir, "local")
	_ = os.MkdirAll(global, 0755)
	_ = os.MkdirAll(local, 0755)
	_ = os.WriteFile(filepath.Join(global, "foo.yaml"), []byte("name: foo\ndescription: Foo template\nbody: |\n  hello"), 0644)
	cfg := &config.Config{GlobalTemplateDir: global, LocalTemplateDir: local}
	list, err := ListTemplates(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 template, got %d", len(list))
	}
	if list[0].Name != "foo" || list[0].Description != "Foo template" {
		t.Errorf("list[0] = %+v", list[0])
	}
}

func TestScaffoldTemplate(t *testing.T) {
	s := ScaffoldTemplate("my-template")
	if s == "" {
		t.Fatal("empty scaffold")
	}
	if !strings.Contains(s, "name: my-template") {
		t.Error("scaffold should contain name")
	}
	if !strings.Contains(s, "body: |") {
		t.Error("scaffold should contain body")
	}
}
