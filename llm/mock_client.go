package llm

// MockClient is used in tests to simulate LLM responses.
type MockClient struct {
	Response *CompletionResult
	Err      error
}

func (m MockClient) CompleteWithOptions(prompt string, model string, opts any) (*CompletionResult, error) {
	return m.Response, m.Err
}
