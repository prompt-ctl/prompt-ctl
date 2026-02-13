package prompt

import (
	"fmt"
	"strings"
)

// EnhanceConfig controls how the prompt is enhanced
type EnhanceConfig struct {
	// Intent is the raw user input describing what they need
	Intent string
	// OutputFormat controls the output style: "xml" (Claude-optimized), "markdown", "plain"
	OutputFormat string
	// SaveAs optionally saves as a reusable template
	SaveAs string
	// Persona overrides the default expert role
	Persona string
	// NoConstraints skips adding constraint section
	NoConstraints bool
	// ClientVersion optional; when set, sent as User-Agent promptctl/<version> for analytics
	ClientVersion string
}

// EnhanceResult holds the enhanced prompt and metadata
type EnhanceResult struct {
	Prompt   string
	Template string // YAML template version if SaveAs was set
}

// EnhanceWithFallback uses the LLM enhance API when enhanceMode is "llm" and enhanceURL is set;
// otherwise (or on API failure) falls back to rule-based Enhance.
func EnhanceWithFallback(cfg EnhanceConfig, enhanceURL, enhanceMode string) (*EnhanceResult, error) {
	if strings.TrimSpace(cfg.Intent) == "" {
		return nil, fmt.Errorf("intent cannot be empty")
	}
	if enhanceMode == "llm" && enhanceURL != "" {
		result, err := EnhanceViaAPI(enhanceURL, cfg)
		if err == nil {
			return result, nil
		}
		// Fall through to rule-based on any error
	}
	return Enhance(cfg)
}

// Enhance transforms raw intent into a well-structured prompt
// This is a deterministic, rule-based enhancement - no LLM needed.
// It applies structural best practices: context framing, task decomposition,
// output formatting, and constraint injection.
func Enhance(cfg EnhanceConfig) (*EnhanceResult, error) {
	if strings.TrimSpace(cfg.Intent) == "" {
		return nil, fmt.Errorf("intent cannot be empty")
	}

	if cfg.OutputFormat == "" {
		cfg.OutputFormat = "xml"
	}

	// Step 1: Analyze the intent to determine task type and components
	analysis := analyzeIntent(cfg.Intent)

	// Step 2: Build the structured prompt based on analysis
	var builder promptBuilder
	switch cfg.OutputFormat {
	case "xml":
		builder = &xmlPromptBuilder{}
	case "markdown":
		builder = &markdownPromptBuilder{}
	default:
		builder = &xmlPromptBuilder{}
	}

	prompt := builder.Build(analysis, cfg)

	result := &EnhanceResult{
		Prompt: prompt,
	}

	// Step 3: Optionally generate a reusable template
	if cfg.SaveAs != "" {
		result.Template = generateTemplateFromAnalysis(cfg.SaveAs, analysis, prompt)
	}

	return result, nil
}

// intentAnalysis holds the decomposed understanding of what the user wants
type intentAnalysis struct {
	// TaskType categorizes the request
	TaskType string
	// Subject is the core topic/object being discussed
	Subject string
	// Actions are the specific things the user wants done
	Actions []string
	// ImpliedNeeds are things the user probably wants but didn't explicitly ask for
	ImpliedNeeds []string
	// Tone captures how critical/supportive the user wants the response
	Tone string
	// OutputHints captures any output format clues from the intent
	OutputHints []string
	// RawIntent preserves the original text
	RawIntent string
}

// analyzeIntent performs rule-based decomposition of the user's raw intent
func analyzeIntent(intent string) *intentAnalysis {
	lower := strings.ToLower(intent)

	analysis := &intentAnalysis{
		RawIntent: intent,
	}

	// Detect task type from keywords
	analysis.TaskType = detectTaskType(lower)

	// Extract subject - the core thing being discussed
	analysis.Subject = extractSubject(intent)

	// Extract explicit actions from the intent
	analysis.Actions = extractActions(intent)

	// Infer implied needs based on task type
	analysis.ImpliedNeeds = inferNeeds(analysis.TaskType, lower)

	// Detect tone signals
	analysis.Tone = detectTone(lower)

	// Detect output format hints
	analysis.OutputHints = detectOutputHints(lower)

	return analysis
}

// detectTaskType categorizes the request based on keyword patterns
func detectTaskType(lower string) string {
	patterns := []struct {
		keywords []string
		taskType string
	}{
		{[]string{"business idea", "startup", "market", "validate", "viability", "product-market"}, "business_analysis"},
		{[]string{"review", "code review", "pull request", "pr review"}, "code_review"},
		{[]string{"debug", "fix", "error", "bug", "broken", "crash"}, "debugging"},
		{[]string{"architect", "design", "system design", "infrastructure"}, "architecture"},
		{[]string{"write", "draft", "compose", "blog", "article", "post"}, "writing"},
		{[]string{"explain", "teach", "how does", "what is", "understand"}, "explanation"},
		{[]string{"analyze", "analysis", "evaluate", "assess", "compare"}, "analysis"},
		{[]string{"plan", "roadmap", "strategy", "timeline", "milestones"}, "planning"},
		{[]string{"refactor", "improve", "optimize", "clean up"}, "refactoring"},
		{[]string{"test", "testing", "test cases", "unit test"}, "testing"},
		{[]string{"convert", "transform", "migrate", "translate"}, "transformation"},
	}

	for _, p := range patterns {
		for _, kw := range p.keywords {
			if strings.Contains(lower, kw) {
				return p.taskType
			}
		}
	}

	return "general"
}

// extractSubject tries to identify the core topic from the intent
func extractSubject(intent string) string {
	// Look for "about X" patterns
	lower := strings.ToLower(intent)
	aboutPatterns := []string{"about ", "regarding ", "for ", "on "}

	for _, p := range aboutPatterns {
		idx := strings.Index(lower, p)
		if idx >= 0 {
			rest := intent[idx+len(p):]
			// Take until the next sentence boundary or clause
			end := findClauseBoundary(rest)
			subject := strings.TrimSpace(rest[:end])
			if len(subject) > 0 && len(subject) < 200 {
				return subject
			}
		}
	}

	// Fallback: use the whole intent as subject context
	if len(intent) > 200 {
		return intent[:200]
	}
	return intent
}

// findClauseBoundary finds where a clause/phrase ends
func findClauseBoundary(text string) int {
	boundaries := []string{". ", "! ", "? ", ", also", ", and also", "; "}
	minIdx := len(text)

	for _, b := range boundaries {
		idx := strings.Index(strings.ToLower(text), b)
		if idx >= 0 && idx < minIdx {
			minIdx = idx
		}
	}

	return minIdx
}

// extractActions pulls out verb phrases that indicate what the user wants done
func extractActions(intent string) []string {
	var actions []string

	// Split on common separators
	parts := splitOnActionBoundaries(intent)

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if len(trimmed) > 5 { // skip trivial fragments
			actions = append(actions, trimmed)
		}
	}

	// Deduplicate
	if len(actions) == 0 {
		actions = []string{intent}
	}

	return actions
}

// splitOnActionBoundaries splits text into action phrases
func splitOnActionBoundaries(text string) []string {
	// Split on natural boundaries that often separate distinct requests
	separators := []string{
		". also ", ". Also ",
		". help ", ". Help ",
		". then ", ". Then ",
		", also ", ", Also ",
		"; ",
	}

	parts := []string{text}
	for _, sep := range separators {
		var newParts []string
		for _, p := range parts {
			split := strings.Split(p, sep)
			newParts = append(newParts, split...)
		}
		parts = newParts
	}

	return parts
}

// inferNeeds adds implied requirements based on task type
func inferNeeds(taskType, lower string) []string {
	needs := make([]string, 0)

	switch taskType {
	case "business_analysis":
		needs = append(needs,
			"Identify target market size and growth trajectory",
			"Map the competitive landscape with specific named competitors",
			"Assess unit economics and path to profitability",
			"Evaluate timing - why now and not 2 years ago",
			"Identify the riskiest assumptions to validate first",
		)
		if strings.Contains(lower, "critic") || strings.Contains(lower, "honest") || strings.Contains(lower, "waste") {
			needs = append(needs,
				"List specific reasons this idea could fail",
				"Identify the 'kill criteria' - signals to stop pursuing this",
			)
		}
	case "code_review":
		needs = append(needs,
			"Check for security vulnerabilities",
			"Assess performance implications",
			"Evaluate error handling completeness",
		)
	case "debugging":
		needs = append(needs,
			"Identify root cause, not just symptoms",
			"Suggest how to prevent recurrence",
		)
	case "architecture":
		needs = append(needs,
			"Consider operational complexity",
			"Evaluate failure modes",
			"Assess team capability to maintain",
		)
	case "analysis":
		needs = append(needs,
			"Present data and evidence for claims",
			"Consider counterarguments",
			"Quantify where possible",
		)
	case "planning":
		needs = append(needs,
			"Define concrete milestones with success criteria",
			"Identify dependencies and blockers",
			"Include risk mitigation for each phase",
		)
	case "writing":
		needs = append(needs,
			"Define target audience clearly",
			"Consider the key takeaway for the reader",
		)
	}

	return needs
}

// detectTone identifies the desired response tone
func detectTone(lower string) string {
	criticalSignals := []string{"critic", "honest", "brutal", "don't sugarcoat",
		"waste time", "waste money", "devil's advocate", "challenge", "push back",
		"no bs", "straightforward", "blunt"}
	for _, s := range criticalSignals {
		if strings.Contains(lower, s) {
			return "critical"
		}
	}

	supportiveSignals := []string{"help me", "guide", "encourage", "support"}
	for _, s := range supportiveSignals {
		if strings.Contains(lower, s) {
			return "supportive"
		}
	}

	return "balanced"
}

// detectOutputHints picks up on format preferences
func detectOutputHints(lower string) []string {
	var hints []string

	formatSignals := map[string]string{
		"step by step":    "step_by_step",
		"table":           "table",
		"compare":         "comparison",
		"pros and cons":   "pros_cons",
		"list":            "list",
		"summary":         "summary",
		"detailed":        "detailed",
		"brief":           "brief",
		"actionable":      "actionable",
		"framework":       "framework",
		"swot":            "swot",
	}

	for signal, hint := range formatSignals {
		if strings.Contains(lower, signal) {
			hints = append(hints, hint)
		}
	}

	return hints
}

// promptBuilder interface for different output formats
type promptBuilder interface {
	Build(analysis *intentAnalysis, cfg EnhanceConfig) string
}

// xmlPromptBuilder creates Claude-optimized XML-structured prompts
type xmlPromptBuilder struct{}

func (b *xmlPromptBuilder) Build(a *intentAnalysis, cfg EnhanceConfig) string {
	var sb strings.Builder

	// Context section - who should the AI be
	persona := cfg.Persona
	if persona == "" {
		persona = defaultPersona(a.TaskType)
	}

	sb.WriteString("<context>\n")
	sb.WriteString(persona)
	sb.WriteString("\n</context>\n\n")

	// Subject/topic section
	sb.WriteString("<subject>\n")
	sb.WriteString(a.Subject)
	sb.WriteString("\n</subject>\n\n")

	// Task section - what exactly to do
	sb.WriteString("<task>\n")
	sb.WriteString("Analyze the above and deliver the following:\n\n")

	for i, action := range a.Actions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, capitalizeFirst(strings.TrimSpace(action))))
	}

	if len(a.ImpliedNeeds) > 0 {
		sb.WriteString("\nAdditionally, address these critical dimensions:\n")
		for i, need := range a.ImpliedNeeds {
			sb.WriteString(fmt.Sprintf("%d. %s\n", len(a.Actions)+i+1, need))
		}
	}
	sb.WriteString("</task>\n\n")

	// Output format section
	sb.WriteString("<output_format>\n")
	sb.WriteString(buildOutputFormat(a))
	sb.WriteString("</output_format>\n\n")

	// Constraints section
	if !cfg.NoConstraints {
		sb.WriteString("<constraints>\n")
		sb.WriteString(buildConstraints(a))
		sb.WriteString("</constraints>\n")
	}

	return sb.String()
}

// markdownPromptBuilder creates markdown-structured prompts
type markdownPromptBuilder struct{}

func (b *markdownPromptBuilder) Build(a *intentAnalysis, cfg EnhanceConfig) string {
	var sb strings.Builder

	persona := cfg.Persona
	if persona == "" {
		persona = defaultPersona(a.TaskType)
	}

	sb.WriteString("## Role\n\n")
	sb.WriteString(persona)
	sb.WriteString("\n\n")

	sb.WriteString("## Subject\n\n")
	sb.WriteString(a.Subject)
	sb.WriteString("\n\n")

	sb.WriteString("## Task\n\n")
	for i, action := range a.Actions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, capitalizeFirst(strings.TrimSpace(action))))
	}

	if len(a.ImpliedNeeds) > 0 {
		sb.WriteString("\nAlso address:\n")
		for i, need := range a.ImpliedNeeds {
			sb.WriteString(fmt.Sprintf("%d. %s\n", len(a.Actions)+i+1, need))
		}
	}

	sb.WriteString("\n## Output Format\n\n")
	sb.WriteString(buildOutputFormat(a))

	sb.WriteString("\n## Constraints\n\n")
	sb.WriteString(buildConstraints(a))

	return sb.String()
}

// defaultPersona returns a role description based on task type
func defaultPersona(taskType string) string {
	personas := map[string]string{
		"business_analysis": "You are a seasoned venture analyst and strategy consultant with experience evaluating 500+ business ideas across multiple industries. You combine market data rigor with founder-level pragmatism. Your job is to give an honest, evidence-based assessment - not encouragement.",
		"code_review":       "You are a principal software engineer with 15+ years of production experience. You prioritize security, correctness, and maintainability over cleverness.",
		"debugging":         "You are an expert debugger who thinks systematically. You trace root causes, not symptoms, and always consider edge cases.",
		"architecture":      "You are a principal architect who has designed and operated systems at scale. You evaluate decisions through the lens of operational reality, not theoretical elegance.",
		"analysis":          "You are a senior analyst who combines quantitative rigor with strategic thinking. Every claim must be backed by evidence or clearly marked as assumption.",
		"planning":          "You are an experienced program manager who breaks complex initiatives into executable phases with clear milestones, dependencies, and risk mitigations.",
		"writing":           "You are a professional writer and editor who crafts clear, compelling content tailored to a specific audience and purpose.",
		"explanation":       "You are an expert educator who builds understanding from first principles, using concrete examples and analogies.",
		"refactoring":       "You are a senior engineer focused on code quality. You improve readability, reduce complexity, and eliminate duplication while preserving behavior.",
		"testing":           "You are a QA engineer who thinks in edge cases, boundary conditions, and failure modes.",
		"transformation":    "You are a technical specialist in data and code transformation, ensuring accuracy and completeness.",
		"general":           "You are a senior expert in the relevant domain. Be thorough, specific, and actionable in your analysis.",
	}

	if p, ok := personas[taskType]; ok {
		return p
	}
	return personas["general"]
}

// buildOutputFormat generates format instructions based on analysis
func buildOutputFormat(a *intentAnalysis) string {
	switch a.TaskType {
	case "business_analysis":
		return `Structure your response as:

1. VERDICT (2 sentences - your overall assessment and confidence level)
2. MARKET ANALYSIS (target market, size, trends, timing)
3. COMPETITIVE LANDSCAPE (named competitors, their strengths, your differentiation)
4. BUSINESS MODEL ASSESSMENT (unit economics, revenue model, path to profitability)
5. CRITICAL RISKS (ranked by likelihood x impact, with mitigations)
6. VALIDATION ROADMAP (3-5 concrete experiments to run, ordered by cost and learning value)
7. KILL CRITERIA (specific signals that should make you abandon this idea)
`
	case "code_review":
		return `Structure your response as:
1. Summary (2-3 sentences)
2. Critical Issues (security, correctness)
3. Performance
4. Maintainability
5. Line-by-line suggestions
`
	case "architecture":
		return `Structure your response as:
1. Problem restatement
2. Options (2-4 viable approaches)
3. Trade-off matrix
4. Recommendation with justification
5. Implementation steps
6. Risks and mitigations
`
	case "planning":
		return `Structure your response as:
1. Goal statement
2. Phases with milestones
3. Dependencies and critical path
4. Resource requirements
5. Risk register
6. Success criteria
`
	default:
		return `Structure your response with clear sections.
Lead with the most important finding or recommendation.
Be specific and reference concrete examples.
End with actionable next steps.
`
	}
}

// buildConstraints generates quality constraints based on analysis
func buildConstraints(a *intentAnalysis) string {
	var constraints []string

	// Universal constraints
	constraints = append(constraints,
		"Be specific - no vague advice. Every recommendation must be actionable.",
		"Quantify where possible (market sizes, time estimates, costs).",
	)

	// Tone-based constraints
	switch a.Tone {
	case "critical":
		constraints = append(constraints,
			"Be ruthlessly honest. If the idea is bad, say so directly and explain why.",
			"Challenge every assumption. Default to skepticism.",
			"Don't soften bad news. Time is the most expensive resource.",
		)
	case "supportive":
		constraints = append(constraints,
			"Be constructive - pair every criticism with a concrete suggestion.",
			"Acknowledge what's strong before diving into weaknesses.",
		)
	default:
		constraints = append(constraints,
			"Balance honesty with constructiveness. Flag real problems, suggest fixes.",
		)
	}

	// Task-specific constraints
	switch a.TaskType {
	case "business_analysis":
		constraints = append(constraints,
			"Name specific competitors, don't say 'existing players'.",
			"Distinguish between validated facts and assumptions.",
			"If you don't have data for a claim, say so explicitly.",
		)
	case "code_review":
		constraints = append(constraints,
			"Reference line numbers.",
			"Suggest concrete fixes, not vague improvements.",
		)
	case "architecture":
		constraints = append(constraints,
			"Include rough effort estimates.",
			"Consider the team's ability to operate this long-term.",
		)
	}

	var sb strings.Builder
	for _, c := range constraints {
		sb.WriteString("- ")
		sb.WriteString(c)
		sb.WriteString("\n")
	}
	return sb.String()
}

// generateTemplateFromAnalysis creates a reusable YAML template from the analysis
func generateTemplateFromAnalysis(name string, a *intentAnalysis, prompt string) string {
	return fmt.Sprintf(`# Auto-generated template from: %s
name: %s
description: %s
variables:
  - name: subject
    description: The specific topic or idea to analyze
    required: true

body: |
%s
`, name, name, "Generated from: "+truncate(a.RawIntent, 80),
		indentText(strings.ReplaceAll(prompt, a.Subject, "{{.subject}}"), "  "))
}

// capitalizeFirst uppercases the first character
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// truncate shortens a string to max length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// indentText adds a prefix to each line
func indentText(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
