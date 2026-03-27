package config

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultEnhanceURL is empty in the open-source version. Cloud enhancement is not included.
// Set PROMPTCTL_ENHANCE_URL to point to your own backend, or use PROMPTCTL_ENHANCE=rule for offline mode.
const DefaultEnhanceURL = ""

// Config holds all promptctl configuration
type Config struct {
	GlobalTemplateDir string
	LocalTemplateDir  string
	GlobalConfigFile  string
	// PromptsDir is where saved prompts (memory) are stored. Defaults to GlobalTemplateDir.
	// Override with PROMPTCTL_PROMPTS_DIR or ~/.promptctl/prompts_dir file.
	PromptsDir  string
	DefaultVars map[string]string
	// EnhanceURL is an optional custom enhance API endpoint. Set with PROMPTCTL_ENHANCE_URL.
	EnhanceURL string
	// EnhanceMode is "llm" or "rule" (offline, default). Set with PROMPTCTL_ENHANCE.
	EnhanceMode string
	// DefaultCreateFormat is the saved output format for "promptctl create" when not overridden by --format. Stored in ~/.promptctl/create_format.
	DefaultCreateFormat string
}

// Load discovers and merges config from global and local sources.
// Global: ~/.promptctl/
// Local: .promptctl/ (in current directory or any parent)
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	globalDir := filepath.Join(home, ".promptctl")
	localDir := findLocalConfig()

	globalTemplates := filepath.Join(globalDir, "templates")
	cfg := &Config{
		GlobalTemplateDir: globalTemplates,
		LocalTemplateDir:  filepath.Join(localDir, "templates"),
		GlobalConfigFile:  filepath.Join(globalDir, "config.yaml"),
		PromptsDir:        globalTemplates, // default
		DefaultVars:       make(map[string]string),
		EnhanceURL:        os.Getenv("PROMPTCTL_ENHANCE_URL"),
		EnhanceMode:       os.Getenv("PROMPTCTL_ENHANCE"),
	}
	if d := os.Getenv("PROMPTCTL_PROMPTS_DIR"); d != "" {
		cfg.PromptsDir = strings.TrimSpace(d)
	} else if b, err := os.ReadFile(filepath.Join(globalDir, "prompts_dir")); err == nil {
		if p := strings.TrimSpace(string(b)); p != "" {
			cfg.PromptsDir = p
		}
	}
	if cfg.EnhanceMode == "" {
		cfg.EnhanceMode = "llm"
	}
	if cfg.EnhanceMode == "llm" && cfg.EnhanceURL == "" {
		cfg.EnhanceURL = DefaultEnhanceURL
	}
	if b, err := os.ReadFile(filepath.Join(globalDir, "create_format")); err == nil {
		if f := strings.TrimSpace(string(b)); f != "" {
			cfg.DefaultCreateFormat = f
		}
	}

	return cfg, nil
}

// SaveCreateFormat persists the default create output format to ~/.promptctl/create_format.
func SaveCreateFormat(format string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	globalDir := filepath.Join(home, ".promptctl")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(globalDir, "create_format"), []byte(format+"\n"), 0644)
}

// SavePromptsDir persists the prompts directory path to ~/.promptctl/prompts_dir.
func SavePromptsDir(dir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	globalDir := filepath.Join(home, ".promptctl")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(globalDir, "prompts_dir")
	return os.WriteFile(path, []byte(dir+"\n"), 0644)
}

// InitGlobal creates the global promptctl directory with starter templates
func InitGlobal() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	globalDir := filepath.Join(home, ".promptctl", "templates")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return err
	}

	// Create starter templates
	starters := map[string]string{
		"review": `# Code Review Template
name: review
description: Thorough code review with security and performance focus
variables:
  - name: file
    description: Path to the file to review
    required: true
  - name: focus
    description: Review focus area
    default: general

body: |
  <context>
  You are an expert code reviewer with deep knowledge of software engineering
  best practices, security vulnerabilities, and performance optimization.
  </context>

  <task>
  Review the following code file thoroughly.
  Focus area: {{.focus}}
  </task>

  <file name="{{.file_name}}" language="{{.file_ext}}">
  {{.file_content}}
  </file>

  <output_format>
  Structure your review as:
  1. Summary (2-3 sentences - overall assessment)
  2. Critical Issues (security, bugs, data loss risks)
  3. Performance (bottlenecks, unnecessary allocations, N+1 queries)
  4. Maintainability (naming, structure, complexity)
  5. Specific line-by-line suggestions (reference line numbers)
  </output_format>

  <constraints>
  - Be specific: reference line numbers and variable names
  - Prioritize issues by severity
  - Suggest concrete fixes, not vague advice
  - If the code is good, say so briefly and move on
  </constraints>
`,
		"debug": `# Debug Assistant Template
name: debug
description: Analyze and fix bugs with systematic debugging approach
variables:
  - name: file
    description: Path to the file with the bug
    required: true
  - name: error
    description: Error message or description of the bug
    required: true

body: |
  <context>
  You are a senior debugger. Analyze the issue systematically.
  </context>

  <bug_report>
  Error/Issue: {{.error}}
  File: {{.file_name}}
  </bug_report>

  <code language="{{.file_ext}}">
  {{.file_content}}
  </code>

  <task>
  1. Identify the root cause (not symptoms)
  2. Explain WHY the bug occurs
  3. Provide the minimal fix
  4. Suggest how to prevent similar bugs (types, tests, linting)
  </task>

  <constraints>
  - Show the exact lines that need to change
  - Use diff format for fixes
  - Don't refactor unrelated code
  </constraints>
`,
		"arch": `# Architecture Decision Template
name: arch
description: Evaluate architectural decisions with trade-off analysis
variables:
  - name: problem
    description: The architectural problem or decision to evaluate
    required: true
  - name: context
    description: System context and constraints
    default: ""

body: |
  <context>
  You are a principal engineer evaluating an architectural decision.
  Be rigorous, quantitative where possible, and opinionated.
  </context>

  <problem>
  {{.problem}}
  </problem>

  {{if .context}}
  <system_context>
  {{.context}}
  </system_context>
  {{end}}

  <task>
  Analyze this decision using the following framework:
  1. Problem Statement - restate clearly in one paragraph
  2. Options - list 2-4 viable approaches
  3. Trade-off Matrix - compare on: complexity, scalability, cost, time-to-implement, team skill match
  4. Recommendation - pick one and defend it
  5. Migration Path - concrete first 3 steps to implement
  6. Risks - what could go wrong and mitigations
  </task>

  <constraints>
  - No hand-waving. Every claim needs justification.
  - Include rough time/cost estimates where relevant.
  - Consider the team's ability to maintain this long-term.
  </constraints>
`,
		"commit": `# Commit Message Template
name: commit
description: Generate conventional commit message from diff or description
variables:
  - name: changes
    description: Description of changes or diff output
    required: true
  - name: scope
    description: Component or module scope
    default: ""

body: |
  <task>
  Generate a conventional commit message for these changes.
  </task>

  <changes>
  {{.changes}}
  </changes>

  {{if .scope}}
  <scope>{{.scope}}</scope>
  {{end}}

  <format>
  Use conventional commits format:
    type(scope): subject

    body (if needed - explain WHY, not WHAT)

  Types: feat, fix, refactor, perf, test, docs, chore, ci
  Subject: imperative mood, lowercase, no period, under 72 chars
  Body: wrap at 72 chars
  </format>

  <constraints>
  - One commit message only
  - If changes are trivial, skip the body
  - Be specific about what changed, not generic
  </constraints>
`,
		"explain": `# Code Explanation Template
name: explain
description: Explain complex code for understanding and knowledge transfer
variables:
  - name: file
    description: Path to the file to explain
    required: true
  - name: depth
    description: Explanation depth (overview, detailed, deep-dive)
    default: detailed

body: |
  <context>
  You are explaining this code to a competent developer who is new to this codebase.
  Depth level: {{.depth}}
  </context>

  <code name="{{.file_name}}" language="{{.file_ext}}">
  {{.file_content}}
  </code>

  <task>
  Explain this code at the "{{.depth}}" level:
  - overview: What does it do? (2-3 paragraphs max)
  - detailed: What, how, and why for each major section
  - deep-dive: Line-by-line with design decisions and edge cases
  </task>

  <constraints>
  - Start with the big picture before diving into details
  - Highlight non-obvious design decisions
  - Note potential gotchas or tricky parts
  - If there are patterns (factory, observer, etc.), name them
  </constraints>
`,
	}

	for name, content := range starters {
		path := filepath.Join(globalDir, name+".yaml")
		if _, err := os.Stat(path); err == nil {
			continue // don't overwrite existing
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// findLocalConfig walks up from cwd to find a .promptctl directory
func findLocalConfig() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ".promptctl"
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, ".promptctl")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}

	// Default to cwd
	return filepath.Join(cwd, ".promptctl")
}
