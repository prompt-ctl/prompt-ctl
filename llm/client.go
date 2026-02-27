package llm

// Client defines the LLM execution interface.
// Allows mocking in tests.
type Client interface {
	CompleteWithOptions(prompt string, model string, opts *CompleteOptions) (*CompletionResult, error)
}

// RealClient wraps existing CompleteWithOptions.
type RealClient struct{}

func (c RealClient) CompleteWithOptions(prompt string, model string, opts *CompleteOptions) (*CompletionResult, error) {
	return CompleteWithOptions(prompt, model, opts)
}
