# Promptctl: Integration Tests for Real Workflows

You are writing end-to-end integration tests for promptctl that test actual user workflows while naturally boosting code coverage to 55-60%.

## Context
- Current coverage: 47.0%
- Target coverage: 55-60% (via integration tests, not unit test grinding)
- Strategy: Test real user workflows with mocked LLM APIs
- Benefit: Tests what users actually do + covers cmd/ package naturally

## Key Insight
Integration tests are MORE valuable than unit tests for CLI tools because they:
- Test the full workflow (cmd → template → provider → output)
- Naturally cover cmd/ code without brittle flag-parsing tests
- Test realistic error scenarios
- Validate end-to-end behavior that users care about
- Are faster to write and easier to maintain

## Tasks

### 1. Create Integration Test Infrastructure
- [x] Create `integration_test.go` in repo root
- [x] Create `testdata/integration/` directory with:
  - [x] Sample files for testing (auth.ts, api.go, handler.py)
  - [x] Sample templates (review.yaml, debug.yaml, arch.yaml)
  - [x] Sample config files
- [x] Create test helper functions:
  - [x] `setupTestEnv()` - create temp directory with test files
  - [x] `cleanupTestEnv()` - cleanup temp directory
  - [x] `mockLLMServer()` - httptest server that mocks Anthropic/OpenAI
  - [x] `runPrompctl()` - execute promptctl CLI with args and capture output
  - [x] `assertOutput()` - verify CLI output contains expected text

### 2. Write Integration Tests for Key Workflows

**Test Set 1: Review Command**
- [ ] Test: `promptctl review --file=testdata/integration/auth.ts`
  - [ ] Verify: template loads correctly
  - [ ] Verify: file content injected into prompt
  - [ ] Verify: LLM called with correct prompt
  - [ ] Verify: response streamed to stdout
  - [ ] Error case: file not found
  - [ ] Error case: LLM API error
- [ ] Test: review with different focus areas
  - [ ] `--focus=security` modifies template
  - [ ] `--focus=performance` modifies template
  - [ ] `--focus=all` uses default

**Test Set 2: Score Command**
- [ ] Test: `promptctl score testdata/integration/review.yaml`
  - [ ] Verify: template parsed correctly
  - [ ] Verify: scoring logic applied
  - [ ] Verify: JSON output valid
  - [ ] Verify: score between 0-100
  - [ ] Error case: invalid YAML
  - [ ] Error case: missing template

**Test Set 3: Fix Command**
- [ ] Test: `promptctl fix testdata/integration/low-score-template.yaml`
  - [ ] Verify: LLM called to improve template
  - [ ] Verify: output is valid YAML
  - [ ] Verify: improved template is returned
  - [ ] Error case: LLM API failure
  - [ ] Error case: invalid input template

**Test Set 4: Debug Command**
- [ ] Test: `promptctl debug --file=testdata/integration/api.go --error="TypeError: Cannot read..."`
  - [ ] Verify: error context injected
  - [ ] Verify: file context provided
  - [ ] Verify: LLM response includes debugging suggestions
  - [ ] Error case: no error provided
  - [ ] Error case: file not readable

**Test Set 5: Architecture Command**
- [ ] Test: `promptctl arch --problem="Should we use event sourcing?"`
  - [ ] Verify: problem statement injected
  - [ ] Verify: LLM returns structured decision
  - [ ] Verify: includes pros/cons
  - [ ] Error case: LLM API failure

**Test Set 6: Commit Command**
- [ ] Test: `promptctl commit --changes="Added retry logic with exponential backoff"`
  - [ ] Verify: changelog template applied
  - [ ] Verify: changes injected correctly
  - [ ] Verify: output is formatted commit message
  - [ ] Verify: JSON output includes structured data
  - [ ] Error case: empty changes

**Test Set 7: Init Command**
- [ ] Test: `promptctl init` in temp directory
  - [ ] Verify: ~/.promptctl/templates/ created
  - [ ] Verify: starter templates copied
  - [ ] Verify: all 5+ templates present
  - [ ] Verify: templates are valid YAML
  - [ ] Error case: init in already-initialized dir

**Test Set 8: Variants Command**
- [ ] Test: `promptctl variants --template=review.yaml --count=3`
  - [ ] Verify: 3 variants generated
  - [ ] Verify: each variant is valid YAML
  - [ ] Verify: variants differ from each other
  - [ ] Verify: JSON ranking output valid
  - [ ] Error case: invalid template

**Test Set 9: Execute Command**
- [ ] Test: `promptctl execute --template=custom.yaml --var1=value1`
  - [ ] Verify: variables substituted correctly
  - [ ] Verify: prompt streamed to stdout
  - [ ] Verify: exit code 0 on success
  - [ ] Error case: missing required variable
  - [ ] Error case: template not found

**Test Set 10: Config-Related Workflows**
- [ ] Test: project-local template overrides global
  - [ ] Create `.promptctl/templates/review.yaml` in temp dir
  - [ ] Run `promptctl review --file=test.ts`
  - [ ] Verify: project template used (not global)
  - [ ] Verify: output differs from global template
- [ ] Test: `promptctl config` command reads/writes config
  - [ ] Verify: config location correct
  - [ ] Verify: provider settings persisted
  - [ ] Verify: custom template paths saved

### 3. Mock LLM Provider Responses
- [ ] Create `testdata/responses/` directory
- [ ] Mock successful responses for:
  - [ ] Anthropic Claude streaming response
  - [ ] OpenAI GPT streaming response
  - [ ] Token counts and cost estimates
- [ ] Mock error responses:
  - [ ] 401 Unauthorized (auth failure)
  - [ ] 429 Rate Limited
  - [ ] 500 Server Error
  - [ ] Connection timeout
  - [ ] Invalid JSON response

### 4. Test Error Scenarios
- [ ] File not found errors
- [ ] Permission denied errors
- [ ] Invalid template YAML
- [ ] Missing required variables
- [ ] LLM API failures (auth, rate limit, timeout)
- [ ] Invalid LLM responses
- [ ] Corrupted config files
- [ ] Disk full errors (if feasible)

### 5. Test Real-World Scenarios
- [ ] Multi-line file content injection
- [ ] Special characters in variables (quotes, newlines, etc.)
- [ ] Large files (1MB+)
- [ ] Very long error messages
- [ ] Unicode/emoji in prompts
- [ ] Project-local template overrides
- [ ] Custom provider settings
- [ ] Cost estimation accuracy
- [ ] Token counting accuracy

### 6. Verify Coverage Improvement
- [ ] Run `go test -cover ./...` with integration tests
- [ ] Verify cmd/ coverage increased (at least 40%)
- [ ] Verify overall coverage at 55-60%
- [ ] Verify all tests pass
- [ ] Verify go vet clean

### 7. Document Integration Test Approach
- [ ] Update docs/TESTING.md to explain:
  - [ ] Why integration tests matter for CLI tools
  - [ ] How to run integration tests
  - [ ] How to add new integration tests
  - [ ] Mock LLM provider pattern
  - [ ] Test fixtures and testdata/ structure

## Test Implementation Pattern

**Setup Mock LLM Server:**
```go
func setupMockLLM() (*httptest.Server, func()) {
  server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Mock Anthropic streaming response
    w.Header().Set("Content-Type", "text/event-stream")
    fmt.Fprint(w, `event: content_block_delta\ndata: {"type":"content_block_delta","delta":{"type":"text_delta","text":"test response"}}\n\n`)
  }))
  return server, func() { server.Close() }
}
```

**Run Promptctl CLI:**
```go
func runPrompctl(t *testing.T, args ...string) string {
  cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
  cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY=test-key")
  output, err := cmd.CombinedOutput()
  return string(output)
}
```

**Example Integration Test:**
```go
func TestReviewWorkflow(t *testing.T) {
  // Setup
  tmpDir := t.TempDir()
  testFile := filepath.Join(tmpDir, "auth.ts")
  os.WriteFile(testFile, []byte("export function auth() {}"), 0644)
  
  server, cleanup := setupMockLLM()
  defer cleanup()
  
  // Execute
  output := runPrompctl(t, "review", "--file="+testFile, "--api-base-url="+server.URL)
  
  // Verify
  if !strings.Contains(output, "auth") {
    t.Fatalf("output missing file name: %s", output)
  }
}
```

## Success Criteria
- ✓ 10+ integration tests covering key workflows
- ✓ All error scenarios tested (file not found, API errors, invalid input)
- ✓ Coverage: 55-60% overall (cmd/ at 40%+)
- ✓ All tests pass (`go test ./...`)
- ✓ go vet clean
- ✓ Tests document real user workflows
- ✓ docs/TESTING.md updated with integration test guide

## Priority Order
1. Review command (most used)
2. Score + Fix (core functionality)
3. Debug + Architecture (common workflows)
4. Init + Execute (setup + advanced)
5. Variants + Commit (less critical)
6. Config + overrides (important for teams)

## Estimate
- Test infrastructure: 30 mins
- Writing 10+ integration tests: 1.5-2 hours
- Validation and docs: 30 mins
- **Total: 2.5-3 hours → likely ~45-60 mins with AI assistance**

## Notes
- Don't test LLM logic (it's mocked) - test CLI behavior
- Focus on end-to-end workflows, not function isolation
- Use testdata/ for sample files and templates
- Keep tests independent (no shared state between tests)
- Integration tests are more maintainable than brittle unit tests for CLI code
