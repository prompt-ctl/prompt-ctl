package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/prompt-ctl/promptctl/config"
)

type TemplateMeta struct {
	Current  string   `json:"current"`
	Versions []string `json:"versions"`
}

// Variable represents a template variable
type Variable struct {
	Name        string
	Description string
	Required    bool
	Default     string
}

// Template represents a prompt template
type Template struct {
	Name        string
	Description string
	Variables   []Variable
	Body        string
	Path        string
	IsLocal     bool
}

// TemplateInfo is a lightweight template listing entry
type TemplateInfo struct {
	Name        string
	Description string
	IsLocal     bool
}

// VariableNames returns just the names of all variables
func (t *Template) VariableNames() []string {
	names := make([]string, len(t.Variables))
	for i, v := range t.Variables {
		names[i] = v.Name
	}
	return names
}

// Render processes the template with given variables
func (t *Template) Render(vars map[string]string) (string, error) {
	// Apply defaults for missing variables
	for _, v := range t.Variables {
		if _, ok := vars[v.Name]; !ok {
			if v.Required && v.Default == "" {
				return "", fmt.Errorf("required variable '--%s' not provided", v.Name)
			}
			if v.Default != "" {
				vars[v.Name] = v.Default
			}
		}
	}

	result := t.Body

	// Handle {{if .var}} ... {{end}} conditionals
	ifRegex := regexp.MustCompile(`\{\{if \.(\w+)\}\}([\s\S]*?)\{\{end\}\}`)
	result = ifRegex.ReplaceAllStringFunc(result, func(match string) string {
		submatches := ifRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		varName := submatches[1]
		content := submatches[2]
		if val, ok := vars[varName]; ok && val != "" {
			// Process the inner content for variable substitution
			return content
		}
		return ""
	})

	// Replace {{.var}} placeholders
	varRegex := regexp.MustCompile(`\{\{\.(\w+)\}\}`)
	result = varRegex.ReplaceAllStringFunc(result, func(match string) string {
		submatches := varRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		varName := submatches[1]
		if val, ok := vars[varName]; ok {
			return val
		}
		return match // leave unreplaced if not found
	})

	return strings.TrimSpace(result), nil
}

// MinimalTemplate returns a valid YAML template with the given name and body.
// Used when saving to memory without an existing Template (e.g. rule-based enhance).
func MinimalTemplate(name, body string) string {
	lines := strings.Split(body, "\n")
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	return fmt.Sprintf(`name: %s
description: Saved from create

body: |
%s
`, name, strings.Join(lines, "\n"))
}

// IsValidTemplateName returns true if name is safe (no path traversal).
// Allows only alphanumeric, underscore, and hyphen.
func IsValidTemplateName(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	for _, r := range name {
		if r != '-' && r != '_' && !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// LoadTemplate finds and parses a template by name
// Local templates take precedence over global ones
func LoadTemplate(name string, cfg *config.Config) (*Template, error) {
	if !IsValidTemplateName(name) {
		return nil, fmt.Errorf("invalid template name: %q", name)
	}

	// Check local first, then global
	dirs := []struct {
		dir     string
		isLocal bool
	}{
		{cfg.LocalTemplateDir, true},
		{cfg.GlobalTemplateDir, false},
	}

	for _, d := range dirs {

		// --- Versioned folder support ---
		folder := filepath.Join(d.dir, name)
		metaPath := filepath.Join(folder, "meta.json")

		if _, statErr := os.Stat(metaPath); statErr == nil {

			data, readErr := os.ReadFile(metaPath)
			if readErr != nil {
				return nil, fmt.Errorf("reading meta.json: %w", readErr)
			}

			var meta TemplateMeta
			if parseErr := json.Unmarshal(data, &meta); parseErr != nil {
				return nil, fmt.Errorf("parsing meta.json: %w", parseErr)
			}

			versionPath := filepath.Join(folder, meta.Current+".yaml")
			return parseTemplateFile(versionPath, d.isLocal)
		}

		// --- Legacy flat template fallback ---
		path := filepath.Join(d.dir, name+".yaml")
		if _, statErr := os.Stat(path); statErr == nil {
			return parseTemplateFile(path, d.isLocal)
		}
	}

	return nil, fmt.Errorf("template not found: %s", name)
}

// FindTemplatePath returns the filesystem path of a template
func FindTemplatePath(name string, cfg *config.Config) (string, error) {
	if !IsValidTemplateName(name) {
		return "", fmt.Errorf("invalid template name: %q", name)
	}
	dirs := []string{cfg.LocalTemplateDir, cfg.GlobalTemplateDir}
	for _, dir := range dirs {
		path := filepath.Join(dir, name+".yaml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("not found: %s", name)
}

// ListTemplates returns all available templates from both directories
func ListTemplates(cfg *config.Config) ([]TemplateInfo, error) {
	seen := make(map[string]bool)
	var templates []TemplateInfo

	// Local templates first (they override global)
	for _, d := range []struct {
		dir     string
		isLocal bool
	}{
		{cfg.LocalTemplateDir, true},
		{cfg.GlobalTemplateDir, false},
	} {
		entries, err := os.ReadDir(d.dir)
		if err != nil {
			continue // directory might not exist
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			name := strings.TrimSuffix(entry.Name(), ".yaml")
			if !IsValidTemplateName(name) {
				continue // skip path-traversal or invalid names
			}
			if seen[name] {
				continue // local already registered
			}
			seen[name] = true

			// Quick-parse to get description
			path := filepath.Join(d.dir, entry.Name())
			tmpl, err := parseTemplateFile(path, d.isLocal)
			if err != nil {
				templates = append(templates, TemplateInfo{Name: name, IsLocal: d.isLocal, Description: "(parse error)"})
				continue
			}

			templates = append(templates, TemplateInfo{
				Name:        name,
				Description: tmpl.Description,
				IsLocal:     d.isLocal,
			})
		}
	}

	return templates, nil
}

// ScaffoldTemplate generates a new template YAML scaffold
func ScaffoldTemplate(name string) string {
	return fmt.Sprintf(`# %s - Custom prompt template
name: %s
description: TODO - describe what this template does

# Define variables that can be passed via --name=value
variables:
  - name: file
    description: Path to the input file
    required: true
  - name: focus
    description: Specific area to focus on
    default: general

# The prompt body. Use {{.variable_name}} for substitution.
# Special auto-variables when --file is used:
#   {{.file_content}} - file contents
#   {{.file_name}}    - filename (basename)
#   {{.file_ext}}     - file extension (without dot)
#
# Conditionals: {{if .var}}content{{end}}
body: |
  <context>
  You are an expert assistant helping with %s tasks.
  </context>

  <task>
  Analyze the following file with focus on: {{.focus}}
  </task>

  <file name="{{.file_name}}" language="{{.file_ext}}">
  {{.file_content}}
  </file>

  <output_format>
  Return strictly valid JSON with this structure:

	{
		"summary": "string",
		"critical_issues": ["string"],
		"performance": ["string"],
		"maintainability": ["string"],
		"line_suggestions": [
			{
				"line": number,
				"issue": "string",
				"suggested_fix": "string"
			}
		]
	}

No explanations outside JSON.
  </output_format>

  <constraints>
  - Be concise and actionable
  - Prioritize by impact
  - Suggest concrete improvements
  </constraints>
`, name, name, name)
}

// parseTemplateFile reads and parses a YAML template file
// We do a lightweight custom parse to avoid importing a YAML library
func parseTemplateFile(path string, isLocal bool) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	tmpl := &Template{
		Path:    path,
		IsLocal: isLocal,
	}

	// Parse name
	tmpl.Name = extractField(content, "name")
	if tmpl.Name == "" {
		tmpl.Name = strings.TrimSuffix(filepath.Base(path), ".yaml")
	}

	// Parse description
	tmpl.Description = extractField(content, "description")

	// Parse variables section
	tmpl.Variables = extractVariables(content)

	// Parse body - everything after "body: |"
	bodyMarker := "body: |"
	idx := strings.Index(content, bodyMarker)
	if idx >= 0 {
		body := content[idx+len(bodyMarker):]
		// Dedent: find the common leading whitespace and remove it
		tmpl.Body = dedent(body)
	}

	return tmpl, nil
}

// extractField pulls a simple "key: value" from YAML-like content
func extractField(content, key string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		prefix := key + ":"
		if strings.HasPrefix(trimmed, prefix) {
			val := strings.TrimPrefix(trimmed, prefix)
			val = strings.TrimSpace(val)
			// Remove surrounding quotes if present
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return ""
}

// extractVariables parses the variables section from template YAML
func extractVariables(content string) []Variable {
	var vars []Variable

	lines := strings.Split(content, "\n")
	inVariables := false
	var current *Variable

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "variables:" {
			inVariables = true
			continue
		}

		if !inVariables {
			continue
		}

		// End of variables section
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" && !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "#") {
			break
		}

		if strings.HasPrefix(trimmed, "- name:") {
			if current != nil {
				vars = append(vars, *current)
			}
			current = &Variable{
				Name: strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")),
			}
			continue
		}

		if current != nil {
			if strings.HasPrefix(trimmed, "description:") {
				current.Description = strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			} else if strings.HasPrefix(trimmed, "required:") {
				current.Required = strings.TrimSpace(strings.TrimPrefix(trimmed, "required:")) == "true"
			} else if strings.HasPrefix(trimmed, "default:") {
				val := strings.TrimSpace(strings.TrimPrefix(trimmed, "default:"))
				current.Default = strings.Trim(val, "\"'")
			}
		}
	}

	if current != nil {
		vars = append(vars, *current)
	}

	return vars
}

// dedent removes common leading whitespace from a multi-line string
func dedent(text string) string {
	lines := strings.Split(text, "\n")

	// Skip initial empty lines
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	if start >= len(lines) {
		return ""
	}

	// Find minimum indentation (excluding empty lines)
	minIndent := -1
	for _, line := range lines[start:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return strings.Join(lines[start:], "\n")
	}

	// Remove the common indent
	result := make([]string, 0, len(lines)-start)
	for _, line := range lines[start:] {
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
		} else if len(line) >= minIndent {
			result = append(result, line[minIndent:])
		} else {
			result = append(result, line)
		}
	}

	// Trim trailing empty lines
	for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
		result = result[:len(result)-1]
	}

	return strings.Join(result, "\n")
}
