package prompt

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/prompt-ctl/promptctl/llm"
)

const maxPromptChars = 500

// Suggest asks the LLM for one concrete improvement for a prompt under the given rule.
// rule should be "scope" or "constraints". Returns the suggestion text or an error if
// config has no default model, LoadConfig fails, or the LLM call fails (e.g. no API key).
func Suggest(promptText string, rule string) (string, error) {
	cfg, err := llm.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("LLM config: %w", err)
	}
	if cfg.DefaultModel == "" {
		return "", fmt.Errorf("no default model set; run 'promptctl config' to set a model")
	}

	userPrompt := promptText
	if utf8.RuneCountInString(userPrompt) > maxPromptChars {
		userPrompt = string([]rune(promptText)[:maxPromptChars])
	}

	systemInstruction := fmt.Sprintf("You suggest one concrete improvement for a prompt. Rule: %s. Reply with only the suggestion, one or two sentences.", rule)
	fullPrompt := systemInstruction + "\n\nPrompt to improve:\n" + userPrompt

	result, err := llm.Complete(fullPrompt, cfg.DefaultModel)
	if err != nil {
		return "", fmt.Errorf("LLM suggest: %w", err)
	}

	return strings.TrimSpace(result.Content), nil
}
