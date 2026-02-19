package onboarding

import (
	"os"
	"path/filepath"
)

const skippedFilename = "onboarding_skipped"
const firstRunDoneFilename = "first_run_done"

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

// FirstRunDone returns true if the user has already completed the first-time onboarding.
func FirstRunDone() bool {
	_, err := os.Stat(filepath.Join(baseDir(), firstRunDoneFilename))
	return err == nil
}

// MarkFirstRunDone records that first-time onboarding was shown (completed or skipped).
func MarkFirstRunDone() error {
	dir := baseDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, firstRunDoneFilename), []byte("true"), 0644)
}
