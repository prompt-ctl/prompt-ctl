package ui

import (
	"errors"

	"github.com/AlecAivazis/survey/v2"
)

var ErrNotInteractive = errors.New("not interactive")

func SelectOption(message string, options []string, result *string) error {
	if !Interactive() {
		*result = ""
		return ErrNotInteractive
	}
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}
	return survey.AskOne(prompt, result)
}

func Confirm(message string, defaultYes bool) (bool, error) {
	if !Interactive() {
		return defaultYes, nil
	}
	var v bool
	prompt := &survey.Confirm{
		Message: message,
		Default: defaultYes,
	}
	err := survey.AskOne(prompt, &v)
	return v, err
}

func Input(message string, result *string) error {
	if !Interactive() {
		return ErrNotInteractive
	}
	prompt := &survey.Input{
		Message: message,
	}
	return survey.AskOne(prompt, result)
}

// InputWithDefault is like Input but with a default value (user can press Enter to accept).
func InputWithDefault(message string, defaultVal string, result *string) error {
	if !Interactive() {
		*result = defaultVal
		return ErrNotInteractive
	}
	prompt := &survey.Input{
		Message: message,
		Default: defaultVal,
	}
	return survey.AskOne(prompt, result)
}

// Password prompts for input with masking (no echo). Use for API keys and secrets.
func Password(message string, result *string) error {
	if !Interactive() {
		return ErrNotInteractive
	}
	prompt := &survey.Password{
		Message: message,
	}
	return survey.AskOne(prompt, result)
}
