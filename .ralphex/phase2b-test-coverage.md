# Promptctl: Boost Test Coverage to 80%+

You are improving test coverage for promptctl from 38.7% to 80%+ before the open-source launch.

## Context
- Current coverage: 38.7% (220 tests across 13 packages)
- Target: 80%+ (industry standard for production v1.0.0)
- Goal: Identify uncovered code paths and write tests for them
- Constraint: No mocking beyond what's essential - test real behavior

## Tasks

### 1. Generate Coverage Report
- [x] Run `go test -cover ./...` and capture per-package coverage
- [x] Run `go test -coverprofile=coverage.out ./...`
- [x] Run `go tool cover -html=coverage.out -o coverage.html`
- [x] Analyze which packages/functions are NOT covered
- [x] Create list of uncovered code paths by priority

### 2. Identify Coverage Gaps
Analyze coverage for each package. For uncovered code:
- [ ] `cmd/` - CLI commands
  - [ ] Each command's Execute() method
  - [ ] Error handling paths
  - [ ] Flag validation
  - [ ] Output formatting
- [ ] `internal/template` - Template parsing & rendering
  - [ ] YAML parsing edge cases
  - [ ] Variable substitution with special characters
  - [ ] Missing variables error handling
  - [ ] Template file not found errors
- [ ] `internal/config` - Config directory management
  - [ ] Directory creation
  - [ ] File permissions
  - [ ] Config read/write errors
  - [ ] Legacy config migration
- [ ] `internal/ui` - User interaction
  - [ ] Prompt user flow
  - [ ] Input validation
  - [ ] Error message formatting
- [ ] `internal/agent` - LLM agent orchestration
  - [ ] Agent initialization
  - [ ] State transitions
  - [ ] Error recovery
- [ ] `llm/` - Provider integrations
  - [ ] Anthropic provider: auth, streaming, error handling
  - [ ] OpenAI provider: auth, streaming, error handling
  - [ ] Token estimation
  - [ ] Cost calculation
- [ ] `prompt/` - Embedded templates
  - [ ] Template loading
  - [ ] Variable extraction from templates

### 3. Write Tests for Uncovered Paths
For EACH uncovered code path, write tests covering:
- [ ] Happy path (normal operation)
- [ ] Error paths (missing files, invalid input, API errors)
- [ ] Edge cases (empty input, special characters, large inputs)
- [ ] State management (before/after state is correct)

**High-impact test categories:**

#### cmd/ Tests
- [ ] review command: happy path, file not found, API error
- [ ] fix command: valid input, empty input, API error
- [ ] score command: high score, low score, parsing errors
- [ ] debug command: error context provided, no error provided
- [ ] arch command: complex prompt generation, provider selection
- [ ] commit command: changelog generation, formatting
- [ ] execute command: template execution, variable substitution
- [ ] variants command: variant generation, ranking

#### template/ Tests
- [ ] Parse YAML with all field types
- [ ] Variable substitution: single var, multiple vars, nested vars
- [ ] Missing required variable error
- [ ] Default variable values
- [ ] Template with special characters in variables (quotes, newlines, etc.)
- [ ] Template file not found
- [ ] Invalid YAML syntax
- [ ] Variable name validation

#### config/ Tests
- [ ] Create config directory with permissions
- [ ] Read config file
- [ ] Write config file
- [ ] Update existing config
- [ ] Permission errors
- [ ] Disk full errors
- [ ] Legacy config format migration

#### llm/Provider Tests
- [ ] Anthropic: send prompt, stream response, handle errors
- [ ] OpenAI: send prompt, stream response, handle errors
- [ ] Token estimation: accurate for different models
- [ ] Cost calculation: accurate for different models
- [ ] Auth errors (invalid key, expired key)
- [ ] Network errors (timeout, connection refused)
- [ ] Rate limit handling
- [ ] Invalid response format

### 4. Validate New Tests
- [ ] Run `go test ./...` - all tests pass
- [ ] Run `go test -cover ./...` - report new coverage per package
- [ ] Run `go vet ./...` - no lint issues
- [ ] Run coverage report again - verify 80%+ overall

### 5. Document Test Patterns
- [ ] Create docs/TESTING.md with:
  - [ ] How to run tests locally
  - [ ] How to generate coverage reports
  - [ ] Testing patterns used in the codebase
  - [ ] How to add tests for new features
  - [ ] Mock vs real API testing strategy

### 6. CI/CD Coverage Gating
- [ ] Update .github/workflows/ci.yml to:
  - [ ] Run coverage report
  - [ ] Fail if coverage drops below 80%
  - [ ] Report coverage badge
- [ ] Add coverage badge to README.md

## Success Criteria
- ✓ Coverage increased to 80%+ (overall)
- ✓ All packages have 75%+ coverage
- ✓ Critical paths (cmd/, llm/, template/) have 85%+ coverage
- ✓ All new tests pass
- ✓ go vet and go fmt pass
- ✓ CI/CD enforces minimum coverage
- ✓ Coverage badge in README
- ✓ docs/TESTING.md is comprehensive

## Testing Strategy Notes

**Use real implementations, not mocks:**
- Real YAML template parsing (not mocked)
- Real file I/O for config tests (temp directories)
- Real text rendering and formatting
- Mock ONLY external APIs (Anthropic, OpenAI) with httptest

**Test organization:**
- Unit tests: Small, focused tests for single functions
- Integration tests: Tests that verify multiple components work together
- Example: template loading + variable substitution + API call
- Command tests: Full end-to-end CLI flows with test fixtures

**Test fixtures:**
- Create testdata/ directory with sample:
  - Valid prompt templates
  - Invalid prompt templates
  - Sample files for review/debug commands
  - Sample error messages

**API mocking pattern:**
```go
// Use httptest.Server for LLM provider tests
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  // Mock Anthropic/OpenAI response
  w.Header().Set("Content-Type", "application/json")
  fmt.Fprint(w, `{"content": [...], "usage": {...}}`)
}))
defer server.Close()

// Test provider against mock server
provider := NewAnthropicProvider(server.URL)
result := provider.Send(ctx, prompt)
```

## Priority Order
1. **cmd/** - highest impact (all user-facing commands)
2. **llm/** - critical path (provider integrations)
3. **template/** - core functionality
4. **config/** - file I/O, error handling
5. **internal/agent** - orchestration logic
6. **Others** - supporting packages

## Estimate
- Identifying gaps: 30 mins
- Writing tests: 2-3 hours
- Validation & CI/CD setup: 30 mins
- **Total: 3-4 hours work → likely ~30-45 mins with AI assistance**

## Notes
- Don't skip hard-to-test code - refactor if needed to make it testable
- Aim for readability in tests - use table-driven tests where appropriate
- Include comments explaining WHY each test case is important
- Once at 80%, maintain or improve (don't let coverage regress)
