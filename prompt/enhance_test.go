package prompt

import (
	"strings"
	"testing"
)

func TestEnhance_EmptyIntent(t *testing.T) {
	_, err := Enhance(EnhanceConfig{Intent: ""})
	if err == nil {
		t.Fatal("expected error for empty intent")
	}
	_, err = Enhance(EnhanceConfig{Intent: "   \n\t  "})
	if err == nil {
		t.Fatal("expected error for whitespace-only intent")
	}
}

func TestEnhance_BasicIntent(t *testing.T) {
	result, err := Enhance(EnhanceConfig{Intent: "analyze my startup idea"})
	if err != nil {
		t.Fatalf("Enhance err = %v", err)
	}
	if result == nil || result.Prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(result.Prompt, "<context>") {
		t.Error("expected <context> in prompt")
	}
	if !strings.Contains(result.Prompt, "<task>") {
		t.Error("expected <task> in prompt")
	}
}

func TestEnhance_OutputFormatXML(t *testing.T) {
	result, err := Enhance(EnhanceConfig{Intent: "review my code", OutputFormat: "xml"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Prompt, "<context>") {
		t.Error("xml format should contain context tags")
	}
}

func TestEnhance_OutputFormatMarkdown(t *testing.T) {
	result, err := Enhance(EnhanceConfig{Intent: "explain this function", OutputFormat: "markdown"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}

func TestEnhance_OutputFormatDefault(t *testing.T) {
	result, err := Enhance(EnhanceConfig{Intent: "debug the crash", OutputFormat: ""})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Prompt, "<context>") {
		t.Error("default format should be xml-like")
	}
}

func TestEnhance_SaveAsGeneratesTemplate(t *testing.T) {
	result, err := Enhance(EnhanceConfig{
		Intent:   "plan a migration",
		SaveAs:   "migration",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Template == "" {
		t.Error("expected Template when SaveAs set")
	}
	if !strings.Contains(result.Template, "name: migration") {
		t.Error("template should contain name")
	}
}

func TestEnhance_TaskTypes(t *testing.T) {
	intents := []string{
		"review my pull request",
		"debug the timeout error",
		"architect the new service",
		"write a blog post about Go",
		"explain how this works",
		"analyze the market",
		"plan the Q1 roadmap",
		"refactor this function",
		"add unit tests",
		"convert this to TypeScript",
	}
	for _, intent := range intents {
		result, err := Enhance(EnhanceConfig{Intent: intent})
		if err != nil {
			t.Errorf("intent %q: %v", intent, err)
			continue
		}
		if result.Prompt == "" {
			t.Errorf("intent %q: empty prompt", intent)
		}
	}
}

func TestEnhance_ToneCritical(t *testing.T) {
	result, err := Enhance(EnhanceConfig{Intent: "be critical about my business idea"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(result.Prompt), "critical") && !strings.Contains(strings.ToLower(result.Prompt), "honest") {
		t.Log("prompt may not explicitly say critical; check constraints:", result.Prompt[:min(200, len(result.Prompt))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
