package cmd

import (
	"strings"
	"testing"
)

func TestGenerateVariants_SingleVariant(t *testing.T) {
	variants := generateVariants("base prompt", 1)
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(variants))
	}
	if variants[0] != "base prompt" {
		t.Errorf("single variant should be the base prompt, got %q", variants[0])
	}
}

func TestGenerateVariants_ZeroOrNegative(t *testing.T) {
	variants := generateVariants("base", 0)
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant for n=0, got %d", len(variants))
	}

	variants = generateVariants("base", -1)
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant for n=-1, got %d", len(variants))
	}
}

func TestGenerateVariants_MultipleVariants(t *testing.T) {
	variants := generateVariants("base prompt", 5)
	if len(variants) != 5 {
		t.Fatalf("expected 5 variants, got %d", len(variants))
	}
	// First variant should be the base prompt unchanged
	if variants[0] != "base prompt" {
		t.Errorf("first variant should be base prompt, got %q", variants[0])
	}
	// All variants should contain the base prompt
	for i, v := range variants {
		if !strings.Contains(v, "base prompt") {
			t.Errorf("variant %d should contain base prompt: %q", i, v)
		}
	}
}

func TestGenerateVariants_MaxVariants(t *testing.T) {
	// Request more variants than available
	variants := generateVariants("base", 100)
	// Should return all available variants (8 total)
	if len(variants) > 100 {
		t.Errorf("should not exceed requested count, got %d", len(variants))
	}
	if len(variants) < 2 {
		t.Errorf("should have at least 2 variants, got %d", len(variants))
	}
}

func TestGenerateVariants_VariantsAreUnique(t *testing.T) {
	variants := generateVariants("test prompt", 8)
	seen := make(map[string]bool)
	for _, v := range variants {
		if seen[v] {
			t.Errorf("duplicate variant found: %q", v)
		}
		seen[v] = true
	}
}

func TestGenerateVariants_TruncatedToN(t *testing.T) {
	variants := generateVariants("base", 3)
	if len(variants) != 3 {
		t.Fatalf("expected exactly 3 variants, got %d", len(variants))
	}
}
