package shell

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const promptctlBlockStart = "# promptctl aliases (added by promptctl init)"
const promptctlBlockEnd = "# end promptctl aliases"

// ProfilePath returns the path to the user's shell profile (~/.zshrc or ~/.bashrc).
func ProfilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return filepath.Join(home, ".zshrc"), nil
	}
	return filepath.Join(home, ".bashrc"), nil
}

// HasPromptctlAliasBlock reports whether the profile already contains the promptctl alias block.
func HasPromptctlAliasBlock(profilePath string) (bool, error) {
	b, err := os.ReadFile(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return strings.Contains(string(b), promptctlBlockStart), nil
}

// AliasExists reports whether the profile file already defines an alias with the given name.
func AliasExists(profilePath, aliasName string) (bool, error) {
	b, err := os.ReadFile(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	re := regexp.MustCompile(`(?m)^\s*alias\s+` + regexp.QuoteMeta(aliasName) + `\s*=`)
	return re.Match(b), nil
}

// AddAliases appends promptctl alias lines to the profile. longAlias and shortAlias are the names (e.g. "prompt", "p").
func AddAliases(profilePath, longAlias, shortAlias string) error {
	content, err := os.ReadFile(profilePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	body := string(content)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	block := "\n" + promptctlBlockStart + "\n"
	block += "alias " + longAlias + "='promptctl'\n"
	if shortAlias != "" && shortAlias != longAlias {
		block += "alias " + shortAlias + "='promptctl'\n"
	}
	block += promptctlBlockEnd + "\n"
	return os.WriteFile(profilePath, []byte(body+block), 0644)
}

// RemovePromptctlBlock removes the promptctl alias block from the profile if present (for re-run / repair).
func RemovePromptctlBlock(profilePath string) error {
	b, err := os.ReadFile(profilePath)
	if err != nil {
		return err
	}
	start := strings.Index(string(b), promptctlBlockStart)
	if start == -1 {
		return nil
	}
	end := strings.Index(string(b), promptctlBlockEnd)
	if end == -1 {
		return nil
	}
	end += len(promptctlBlockEnd)
	// Include trailing newline after block if present
	if end < len(b) && b[end] == '\n' {
		end++
	}
	newContent := strings.TrimRight(string(b[:start])+string(b[end:]), " \t\n")
	if newContent != "" && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	return os.WriteFile(profilePath, []byte(newContent), 0644)
}
