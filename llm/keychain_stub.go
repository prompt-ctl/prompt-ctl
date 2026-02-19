//go:build !darwin

package llm

const keychainSentinel = ""

func keychainGet(provider string) (string, bool) {
	return "", false
}

func keychainStore(provider, key string) error {
	return nil
}
