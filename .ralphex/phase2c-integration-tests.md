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
- [x] Test: `promptctl review --file=testdata/integration/auth.ts`
  - [x] Verify: template loads correctly
  - [x] Verify: file content injected into prompt
  - [x] Verify: LLM called with correct prompt
  - [x] Verify: response streamed to stdout
  - [x] Error case: file not found
  - [x] Error case: LLM API error
- [x] Test: review with different focus areas
  - [x] `--focus=security` modifies template
  - [x] `--focus=performance` modifies template
  - [x] `--focus=all` uses default

**Test Set 2: Score Command**
- [x] Test: `promptctl score testdata/integration/review.yaml`
  - [x] Verify: template parsed correctly
  - [x] Verify: scoring logic applied
  - [x] Verify: JSON output valid
  - [x] Verify: score between 0-100
  - [x] Error case: invalid YAML
  - [x] Error case: missing template

**Test Set 3: Fix Command**
- [x] Test: `promptctl fix testdata/integration/low-score-template.yaml`
  - [x] Verify: LLM called to improve template
  - [x] Verify: output is valid YAML
  - [x] Verify: improved template is returned
  - [x] Error case: LLM API failure
  - [x] Error case: invalid input template

**Test Set 4: Debug Command**
- [x] Test: `promptctl debug --file=testdata/integration/api.go --error="TypeError: Cannot read..."`
  - [x] Verify: error context injected
  - [x] Verify: file context provided
  - [x] Verify: LLM response includes debugging suggestions
  - [x] Error case: no error provided
  - [x] Error case: file not readable

**Test Set 5: Architecture Command**
- [x] Test: `promptctl arch --problem="Should we use event sourcing?"`
  - [x] Verify: problem statement injected
  - [x] Verify: LLM returns structured decision
  - [x] Verify: includes pros/cons
  - [x] Error case: LLM API failure

**Test Set 6: Commit Command**
- [x] Test: `promptctl commit --changes="Added retry logic with exponential backoff"`
  - [x] Verify: changelog template applied
  - [x] Verify: changes injected correctly
  - [x] Verify: output is formatted commit message
  - [x] Verify: JSON output includes structured data
  - [x] Error case: empty changes

**Test Set 7: Init Command**
- [x] Test: `promptctl init` in temp directory
  - [x] Verify: ~/.promptctl/templates/ created
  - [x] Verify: starter templates copied
  - [x] Verify: all 5+ templates present
  - [x] Verify: templates are valid YAML
  - [x] Error case: init in already-initialized dir

**Test Set 8: Variants Command**
- [x] Test: `promptctl variants --template=review.yaml --count=3`
  - [x] Verify: 3 variants generated
  - [x] Verify: each variant is valid YAML
  - [x] Verify: variants differ from each other
  - [x] Verify: JSON ranking output valid
  - [x] Error case: invalid template

**Test Set 9: Execute Command**
- [x] Test: `promptctl execute --template=custom.yaml --var1=value1`
  - [x] Verify: variables substituted correctly
  - [x] Verify: prompt streamed to stdout
  - [x] Verify: exit code 0 on success
  - [x] Error case: missing required variable
  - [x] Error case: template not found

**Test Set 10: Config-Related Workflows**
- [x] Test: project-local template overrides global
  - [x] Create `.promptctl/templates/review.yaml` in temp dir
  - [x] Run `promptctl review --file=test.ts`
  - [x] Verify: project template used (not global)
  - [x] Verify: output differs from global template
- [x] Test: `promptctl config` command reads/writes config
  - [x] Verify: config location correct
  - [x] Verify: provider settings persisted
  - [x] Verify: custom template paths saved

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
