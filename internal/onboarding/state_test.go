package onboarding

import (
	"testing"
)

func TestOnboardingSkipped_WhenFileMissing_ReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PROMPTCTL_ONBOARDING_DIR", dir)
	if OnboardingSkipped() {
		t.Error("expected false when file missing")
	}
}

func TestMarkOnboardingSkipped_ThenOnboardingSkipped_ReturnsTrue(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PROMPTCTL_ONBOARDING_DIR", dir)
	if err := MarkOnboardingSkipped(); err != nil {
		t.Fatalf("MarkOnboardingSkipped: %v", err)
	}
	if !OnboardingSkipped() {
		t.Error("expected true after MarkOnboardingSkipped")
	}
}

func TestClearOnboardingSkipped_ThenOnboardingSkipped_ReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PROMPTCTL_ONBOARDING_DIR", dir)
	_ = MarkOnboardingSkipped()
	if err := ClearOnboardingSkipped(); err != nil {
		t.Fatalf("ClearOnboardingSkipped: %v", err)
	}
	if OnboardingSkipped() {
		t.Error("expected false after ClearOnboardingSkipped")
	}
}
