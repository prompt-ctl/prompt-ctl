package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/prompt-ctl/promptctl/config"
	"github.com/prompt-ctl/promptctl/internal/safepath"
	"github.com/prompt-ctl/promptctl/llm"
	"github.com/prompt-ctl/promptctl/prompt"
)

type ExecutionResult struct {
	Content   string
	Cost      float64
	LatencyMs int64
}

// enrichFileVars reads the file referenced by vars["file"] and injects
// file_content, file_name, and file_ext into vars. It is a no-op when
// vars["file"] is absent.
func enrichFileVars(vars map[string]string) error {
	filePath, ok := vars["file"]
	if !ok {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	safePath, err := safepath.SafeFilePath(cwd, filePath)
	if err != nil {
		if errors.Is(err, safepath.ErrPathOutsideBase) {
			return fmt.Errorf("file path must be under current directory: %s", filePath)
		}
		return fmt.Errorf("failed to read file '%s': %w", filePath, err)
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Errorf("failed to read file '%s': %w", filePath, err)
	}

	vars["file_content"] = string(content)
	vars["file_name"] = filepath.Base(safePath)
	vars["file_ext"] = strings.TrimPrefix(filepath.Ext(safePath), ".")
	return nil
}

func runSingleExecution(client llm.Client, prompt string, model string) (*ExecutionResult, error) {

	start := time.Now()

	res, err := client.CompleteWithOptions(prompt, model, nil)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("nil completion result")
	}

	latency := time.Since(start).Milliseconds()

	return &ExecutionResult{
		Content:   res.Content,
		Cost:      res.ActualCost,
		LatencyMs: latency,
	}, nil
}

// loadSpecificTemplateVersion loads a specific version of a template from versioned folders.
// Searches local first, then global. Returns clear error if version not found.
func loadSpecificTemplateVersion(name, version string, cfg *config.Config) (*prompt.Template, error) {
	// Search local first, then global
	for _, dir := range []string{cfg.LocalTemplateDir, cfg.GlobalTemplateDir} {
		path := filepath.Join(dir, name, version+".yaml")
		if _, err := os.Stat(path); err == nil {
			// File exists, parse it
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read baseline version '%s': %w", version, err)
			}

			content := string(data)
			tmpl := &prompt.Template{
				Name:        extractFieldFromYAML(content, "name"),
				Description: extractFieldFromYAML(content, "description"),
				Variables:   extractVariablesFromYAML(content),
				Body:        extractBodyFromYAML(content),
				Path:        path,
				IsLocal:     dir == cfg.LocalTemplateDir,
			}

			// Fallback name to filename if not in YAML
			if tmpl.Name == "" {
				tmpl.Name = strings.TrimSuffix(filepath.Base(path), ".yaml")
			}

			return tmpl, nil
		}
	}

	return nil, fmt.Errorf("baseline version '%s' not found for template '%s'", version, name)
}

// extractFieldFromYAML extracts simple "key: value" from YAML-like content
func extractFieldFromYAML(content, key string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		prefix := key + ":"
		if strings.HasPrefix(trimmed, prefix) {
			val := strings.TrimPrefix(trimmed, prefix)
			val = strings.TrimSpace(val)
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return ""
}

// extractBodyFromYAML extracts the template body after "body: |"
func extractBodyFromYAML(content string) string {
	bodyMarker := "body: |"
	idx := strings.Index(content, bodyMarker)
	if idx >= 0 {
		body := content[idx+len(bodyMarker):]
		return dedentYAMLBlock(body)
	}
	return ""
}

// extractVariablesFromYAML parses the variables section from template YAML
func extractVariablesFromYAML(content string) []prompt.Variable {
	var vars []prompt.Variable

	lines := strings.Split(content, "\n")
	inVariables := false
	var current *prompt.Variable

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
			current = &prompt.Variable{
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

// dedentYAMLBlock removes common leading whitespace from YAML block
func dedentYAMLBlock(text string) string {
	lines := strings.Split(text, "\n")

	// Skip initial empty lines
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	if start >= len(lines) {
		return ""
	}

	// Find minimum indentation
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

	// Remove common indent
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

// runAndAggregate executes a prompt against a model multiple times and returns aggregated scores.
// Returns error if all repetitions fail (to prevent silent data corruption from --record).
func runAndAggregate(
	client llm.Client,
	renderedPrompt string,
	model string,
	repeat int,
	profile EvaluationProfile,
) (avgScore int, avgCost float64, avgLatency int64, failures int, err error) {

	totalScore := 0
	totalCost := 0.0
	totalLatency := int64(0)
	failures = 0

	for i := 0; i < repeat; i++ {
		exec, err := runSingleExecution(client, renderedPrompt, model)
		if err != nil || exec == nil {
			failures++
			continue
		}

		score, jsonInvalid := Evaluate(exec.Content, exec.Cost, exec.LatencyMs, profile)

		if jsonInvalid && profile.StrictJSON {
			failures++
			continue
		}

		totalScore += score
		totalCost += exec.Cost
		totalLatency += exec.LatencyMs
	}

	// Fail if all runs errored to prevent silent data corruption
	if failures == repeat {
		return 0, 0, 0, failures, fmt.Errorf("all %d repetitions failed for model %s", repeat, model)
	}

	success := repeat - failures
	avgScore = totalScore / success
	avgCost = totalCost / float64(success)
	avgLatency = totalLatency / int64(success)

	return avgScore, avgCost, avgLatency, failures, nil
}

// updateBenchmark reads meta.json, updates or creates a benchmarks section with the model score,
// and writes it back with proper formatting. Preserves all existing fields.
func updateBenchmark(templateName string, model string, score int, cfg *config.Config) error {
	// Locate meta.json in template directory
	metaPath := filepath.Join(cfg.LocalTemplateDir, templateName, "meta.json")

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
