//go:build darwin

package llm

import (
	"os/exec"
	"strings"
)

const keychainService = "promptctl"
const keychainSentinel = "__keychain"

func keychainGet(provider string) (string, bool) {
	out, err := exec.Command("security", "find-generic-password", "-s", keychainService, "-a", provider, "-w").Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func keychainStore(provider, key string) error {
	cmd := exec.Command("security", "add-generic-password", "-s", keychainService, "-a", provider, "-w", key, "-U")
	return cmd.Run()
}
