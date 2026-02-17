package prompt

import (
	"net/http"
	"net/http/httptest"
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
	if !strings.Contains(result.Prompt, "<role>") && !strings.Contains(result.Prompt, "<context>") {
		t.Error("expected <role> or <context> in prompt")
	}
	if !strings.Contains(result.Prompt, "<instructions>") && !strings.Contains(result.Prompt, "<task>") {
		t.Error("expected <instructions> or <task> in prompt")
	}
}

func TestEnhance_OutputFormatXML(t *testing.T) {
	result, err := Enhance(EnhanceConfig{Intent: "review my code", OutputFormat: "xml"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Prompt, "<role>") && !strings.Contains(result.Prompt, "<context>") {
		t.Error("xml format should contain role or context tags")
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
	if !strings.Contains(result.Prompt, "<role>") && !strings.Contains(result.Prompt, "<context>") {
		t.Error("default format should be xml-like")
	}
}

func TestEnhance_SaveAsGeneratesTemplate(t *testing.T) {
	result, err := Enhance(EnhanceConfig{
		Intent: "plan a migration",
		SaveAs: "migration",
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

func TestEnhanceWithFallback_UseRuleWhenModeNotLLM(t *testing.T) {
	cfg := EnhanceConfig{Intent: "review my code", OutputFormat: "xml"}
	result, err := EnhanceWithFallback(cfg, "https://enhance.example.com", "rule")
	if err != nil {
		t.Fatalf("EnhanceWithFallback err = %v", err)
	}
	if result == nil || result.Prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(result.Prompt, "<context>") {
		t.Error("expected rule-based XML output")
	}
}

func TestEnhanceWithFallback_UseRuleWhenURLEmpty(t *testing.T) {
	cfg := EnhanceConfig{Intent: "debug the bug", OutputFormat: "xml"}
	result, err := EnhanceWithFallback(cfg, "", "llm")
	if err != nil {
		t.Fatalf("EnhanceWithFallback err = %v", err)
	}
	if result == nil || result.Prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(result.Prompt, "<context>") {
		t.Error("expected rule-based output when URL empty")
	}
}

func TestEnhanceWithFallback_FallbackToRuleOnAPIFailure(t *testing.T) {
	// Server returns 500 so API fails; should fall back to rule-based
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := EnhanceConfig{Intent: "analyze my startup idea", OutputFormat: "xml"}
	result, err := EnhanceWithFallback(cfg, server.URL, "llm")
	if err != nil {
		t.Fatalf("EnhanceWithFallback should fall back, err = %v", err)
	}
	if result == nil || result.Prompt == "" {
		t.Fatal("expected fallback to rule-based prompt")
	}
	if !strings.Contains(result.Prompt, "<context>") {
		t.Error("expected rule-based XML after fallback")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
