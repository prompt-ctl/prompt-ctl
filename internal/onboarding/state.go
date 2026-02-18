package onboarding

import (
	"os"
	"path/filepath"
)

const skippedFilename = "onboarding_skipped"

func baseDir() string {
	if d := os.Getenv("PROMPTCTL_ONBOARDING_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".promptctl")
}

func skippedPath() string {
	return filepath.Join(baseDir(), skippedFilename)
}

// OnboardingSkipped returns true if the user previously skipped onboarding.
func OnboardingSkipped() bool {
	_, err := os.Stat(skippedPath())
	return err == nil
}

// MarkOnboardingSkipped records that the user skipped onboarding.
func MarkOnboardingSkipped() error {
	dir := baseDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(skippedPath(), []byte("true"), 0644)
}

// ClearOnboardingSkipped removes the skipped marker (e.g. after successful onboarding).
func ClearOnboardingSkipped() error {
	err := os.Remove(skippedPath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ReminderMessage returns the one-line reminder to run promptctl config.
func ReminderMessage() string {
	return "Run `promptctl config` to set up your LLM."
}
