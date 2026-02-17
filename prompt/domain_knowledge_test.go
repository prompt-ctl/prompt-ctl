package prompt

import (
	"strings"
	"testing"
)

func TestDetectDomain(t *testing.T) {
	tests := []struct {
		intent         string
		expectedDomain string
	}{
		{"Build football virtual manager", "gaming"},
		{"Create a mobile game with multiplayer features", "gaming"},
		{"payment processing system for e-commerce", "fintech"},
		{"build a trading platform with real-time data", "fintech"},
		{"create an online shop with product catalog", "ecommerce"},
		{"marketplace for digital products", "ecommerce"},
		{"SaaS dashboard with multi-tenant architecture", "saas"},
		{"B2B analytics platform with subscription tiers", "saas"},
		{"iOS app with push notifications", "mobile"},
		{"Android application with offline support", "mobile"},
		{"train a neural network for image classification", "ai_ml"},
		{"ML model for predicting customer churn", "ai_ml"},
		{"setup CI/CD pipeline with Docker and Kubernetes", "devops"},
		{"deploy infrastructure using Terraform", "devops"},
		{"online course platform with quizzes", "education"},
		{"LMS for student management", "education"},
		{"patient management system with HIPAA compliance", "healthcare"},
		{"medical records platform", "healthcare"},
		{"social network with feeds and profiles", "social"},
		{"community platform with messaging", "social"},
		{"analyze my codebase", "general"},
		{"explain how this algorithm works", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			domain := detectDomain(strings.ToLower(tt.intent))
			if domain != tt.expectedDomain {
				t.Errorf("detectDomain(%q) = %q, want %q", tt.intent, domain, tt.expectedDomain)
			}
		})
	}
}

func TestDomainKnowledgeCompleteness(t *testing.T) {
	// Verify all domains have complete knowledge
	requiredDomains := []string{"gaming", "fintech", "ecommerce", "saas", "mobile", "ai_ml", "devops", "education", "healthcare", "social", "general"}

	for _, domain := range requiredDomains {
		dk, exists := domainKnowledgeMap[domain]
		if !exists {
			t.Errorf("Missing domain knowledge for %q", domain)
			continue
		}

		if dk.ExpertRole == "" {
			t.Errorf("Domain %q missing ExpertRole", domain)
		}

		if len(dk.KeyConcerns) == 0 {
			t.Errorf("Domain %q missing KeyConcerns", domain)
		}

		if len(dk.OutputSections) == 0 {
			t.Errorf("Domain %q missing OutputSections", domain)
		}

		if len(dk.Constraints) == 0 {
			t.Errorf("Domain %q missing Constraints", domain)
		}
	}
}

func TestDomainKnowledgeGaming(t *testing.T) {
	dk := domainKnowledgeMap["gaming"]

	if !strings.Contains(dk.ExpertRole, "game") {
		t.Error("Gaming expert role should mention games")
	}

	// Check for critical gaming concerns
	concerns := strings.Join(dk.KeyConcerns, " ")
	if !strings.Contains(strings.ToLower(concerns), "game loop") {
		t.Error("Gaming should mention game loop in concerns")
	}
	if !strings.Contains(strings.ToLower(concerns), "monetiz") {
		t.Error("Gaming should mention monetization in concerns")
	}

	// Check for gaming-specific output sections
	sections := strings.Join(dk.OutputSections, " ")
	if !strings.Contains(sections, "GAME") {
		t.Error("Gaming should have GAME-related output sections")
	}
}

func TestDomainKnowledgeFintech(t *testing.T) {
	dk := domainKnowledgeMap["fintech"]

	if !strings.Contains(strings.ToLower(dk.ExpertRole), "fintech") && !strings.Contains(strings.ToLower(dk.ExpertRole), "payment") {
		t.Error("Fintech expert role should mention fintech or payments")
	}

	// Check for regulatory compliance
	concerns := strings.Join(dk.KeyConcerns, " ")
	if !strings.Contains(concerns, "compliance") && !strings.Contains(concerns, "regulatory") {
		t.Error("Fintech should mention compliance in concerns")
	}

	// Check for money handling constraints
	constraints := strings.Join(dk.Constraints, " ")
	if !strings.Contains(constraints, "decimal") || !strings.Contains(constraints, "BigDecimal") {
		t.Error("Fintech should warn against floating point for money")
	}
}

func TestDomainKnowledgeSaaS(t *testing.T) {
	dk := domainKnowledgeMap["saas"]

	// Check for multi-tenancy
	concerns := strings.Join(dk.KeyConcerns, " ")
	if !strings.Contains(concerns, "multi-tenant") && !strings.Contains(concerns, "Multi-tenancy") {
		t.Error("SaaS should mention multi-tenancy in concerns")
	}

	// Check for billing
	if !strings.Contains(concerns, "billing") && !strings.Contains(concerns, "Billing") {
		t.Error("SaaS should mention billing in concerns")
	}

	// Check for pricing tier constraint
	constraints := strings.Join(dk.Constraints, " ")
	if !strings.Contains(constraints, "pricing tier") {
		t.Error("SaaS should mention pricing tiers in constraints")
	}
}

func TestDomainKnowledgeHealthcare(t *testing.T) {
	dk := domainKnowledgeMap["healthcare"]

	// Check for HIPAA
	concerns := strings.Join(dk.KeyConcerns, " ")
	if !strings.Contains(concerns, "HIPAA") {
		t.Error("Healthcare should mention HIPAA in concerns")
	}

	// Check for PHI encryption
	constraints := strings.Join(dk.Constraints, " ")
	if !strings.Contains(constraints, "HIPAA") {
		t.Error("Healthcare should mention HIPAA compliance in constraints")
	}
}

func TestEnhanceWithDomainKnowledge(t *testing.T) {
	tests := []struct {
		name           string
		intent         string
		expectedDomain string
		shouldContain  []string
	}{
		{
			name:           "Football manager game",
			intent:         "Build football virtual manager",
			expectedDomain: "gaming",
			shouldContain:  []string{"game loop", "GAME CONCEPT", "MVP DEFINITION"},
		},
		{
			name:           "Payment system",
			intent:         "Create a payment processing platform",
			expectedDomain: "fintech",
			shouldContain:  []string{"compliance", "decimal", "SECURITY"},
		},
		{
			name:           "SaaS dashboard",
			intent:         "Build a B2B analytics dashboard with subscriptions",
			expectedDomain: "saas",
			shouldContain:  []string{"multi-tenant", "pricing tier", "BILLING"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Enhance(EnhanceConfig{
				Intent:       tt.intent,
				OutputFormat: "xml",
			})

			if err != nil {
				t.Fatalf("Enhance failed: %v", err)
			}

			if result.Prompt == "" {
				t.Fatal("Expected non-empty prompt")
			}

			// Verify domain-specific content appears
			for _, expected := range tt.shouldContain {
				if !strings.Contains(strings.ToLower(result.Prompt), strings.ToLower(expected)) {
					t.Errorf("Prompt should contain %q but doesn't.\nPrompt:\n%s", expected, result.Prompt)
				}
			}
		})
	}
}

func TestEnhanceFootballManagerExample(t *testing.T) {
	// This is the exact example from the issue
	result, err := Enhance(EnhanceConfig{
		Intent:       "Build football virtual manager",
		OutputFormat: "xml",
	})

	if err != nil {
		t.Fatalf("Enhance failed: %v", err)
	}

	prompt := result.Prompt

	// Should NOT be the old generic output
	if strings.Contains(prompt, "Virtual assistant") {
		t.Error("Prompt should not use generic 'Virtual assistant' role")
	}

	// Should have gaming-specific expert role
	if !strings.Contains(prompt, "game") {
		t.Error("Prompt should mention games in the role")
	}

	// Should have domain-specific concerns
	requiredConcerns := []string{
		"game loop",
		"data model",
		"engagement",
	}

	for _, concern := range requiredConcerns {
		if !strings.Contains(strings.ToLower(prompt), concern) {
			t.Errorf("Prompt should mention %q but doesn't", concern)
		}
	}

	// Should have gaming-specific output sections
	if !strings.Contains(prompt, "GAME CONCEPT") {
		t.Error("Prompt should have GAME CONCEPT section")
	}

	if !strings.Contains(prompt, "TECH STACK") {
		t.Error("Prompt should have TECH STACK section")
	}

	if !strings.Contains(prompt, "START HERE") {
		t.Error("Prompt should have START HERE section for gaming domain")
	}

	// Should be substantial (not a 5-line skeleton)
	lines := strings.Split(prompt, "\n")
	if len(lines) < 30 {
		t.Errorf("Prompt should be substantial (at least 30 lines), got %d lines", len(lines))
	}
}
