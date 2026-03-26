package examples_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oleg-koval/promptctl/config"
	"github.com/oleg-koval/promptctl/prompt"
)

// TestExampleTemplatesParse verifies that all YAML templates in examples/ are
// valid and can be loaded by the template engine.
func TestExampleTemplatesParse(t *testing.T) {
	examplesDir := "."

	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("reading examples dir: %v", err)
	}

	yamlCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		yamlCount++

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		t.Run(name, func(t *testing.T) {
			// Set up a config that points to the examples dir as global templates
			cfg := &config.Config{
				GlobalTemplateDir: examplesDir,
				LocalTemplateDir:  filepath.Join(t.TempDir(), "local"),
			}

			tmpl, err := prompt.LoadTemplate(name, cfg)
			if err != nil {
				t.Fatalf("LoadTemplate(%q): %v", name, err)
			}

			// Every template must have a name
			if tmpl.Name == "" {
				t.Error("template has empty name")
			}

			// Every template must have a description
			if tmpl.Description == "" {
				t.Error("template has empty description")
			}

			// Every template must have a body
			if tmpl.Body == "" {
				t.Error("template has empty body")
			}

			// Every template must have at least one variable
			if len(tmpl.Variables) == 0 {
				t.Error("template has no variables defined")
			}

			// Check that required variables are properly marked
			hasRequired := false
			for _, v := range tmpl.Variables {
				if v.Name == "" {
					t.Error("found variable with empty name")
				}
				if v.Required {
					hasRequired = true
				}
			}
			if !hasRequired {
				t.Error("template has no required variables")
			}
		})
	}

	if yamlCount == 0 {
		t.Fatal("no YAML templates found in examples/")
	}

	t.Logf("validated %d example templates", yamlCount)
}

// TestExampleTemplatesRender verifies that templates can render with sample data.
func TestExampleTemplatesRender(t *testing.T) {
	// Sample variables to test rendering
	sampleVars := map[string]map[string]string{
		"review-security": {
			"file":         "src/auth.ts",
			"file_name":    "auth.ts",
			"file_ext":     "ts",
			"file_content": "function login(user, pass) { return db.query('SELECT * FROM users WHERE name=' + user); }",
			"focus":        "injection",
		},
		"debug-error-context": {
			"file":          "src/db.go",
			"file_name":     "db.go",
			"file_ext":      "go",
			"file_content":  "func getUser(id int) *User { return users[id] }",
			"error_message": "nil pointer dereference",
			"stack_trace":   "goroutine 1 [running]: main.getUser(0x0)\n\tdb.go:3",
		},
		"architecture-decision": {
			"title":   "Use PostgreSQL over MongoDB",
			"context": "We need a database for our user service with ACID requirements.",
			"options": "PostgreSQL, MongoDB, CockroachDB",
		},
		"commit-changelog": {
			"diff":   "+ func newFeature() {}\n- func oldFeature() {}",
			"ticket": "PROJ-123",
		},
		"api-review": {
			"file":         "api/users.go",
			"file_name":    "users.go",
			"file_ext":     "go",
			"file_content": "func HandleGetUser(w http.ResponseWriter, r *http.Request) {}",
			"api_style":    "rest",
		},
	}

	examplesDir := "."
	cfg := &config.Config{
		GlobalTemplateDir: examplesDir,
		LocalTemplateDir:  filepath.Join(t.TempDir(), "local"),
	}

	for name, vars := range sampleVars {
		t.Run(name, func(t *testing.T) {
			tmpl, err := prompt.LoadTemplate(name, cfg)
			if err != nil {
				t.Fatalf("LoadTemplate(%q): %v", name, err)
			}

			rendered, err := tmpl.Render(vars)
			if err != nil {
				t.Fatalf("Render(%q): %v", name, err)
			}

			if rendered == "" {
				t.Error("rendered template is empty")
			}

			// Check that no unresolved placeholders remain for provided vars
			for varName := range vars {
				placeholder := "{{." + varName + "}}"
				if strings.Contains(rendered, placeholder) {
					t.Errorf("unresolved placeholder %s in rendered output", placeholder)
				}
			}
		})
	}
}
