# Prompt score and fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `promptctl score` and `promptctl fix` with directory discovery, optional config, CI exit codes, and optional LLM suggest for fix.

**Architecture:** New prompt-quality scorer in `prompt` package (separate from existing EnhanceScore); discovery in `internal/discover` or under `cmd`; score/fix config in `.promptctl/score.yaml` with flag override; CLI branches in `cmd/root.go` and new handlers in `cmd/score.go` / `cmd/fix.go`.

**Tech Stack:** Go (existing promptctl CLI), YAML (gopkg.in/yaml.v3 or existing), existing `config` and `llm` packages for fix --suggest.

---

### Task 1: Score rules — types and structure

**Files:**
- Create: `promptctl/prompt/quality.go` (types and rule result)
- Modify: none
- Test: `promptctl/prompt/quality_test.go`

**Step 1: Define types and write failing test**

In `prompt/quality.go` add:

```go
package prompt

// QualityScore holds per-file score and which rules triggered (penalties).
type QualityScore struct {
	Score int      // 0-100
	Rules []string // e.g. "missing_constraints", "overbroad_scope"
}
```

In `prompt/quality_test.go`:

```go
package prompt

import "testing"

func TestQualityScore_ZeroPenaltyReturns100(t *testing.T) {
	s := ScorePromptQuality("valid prompt with role context task constraints")
	if s.Score != 100 {
		t.Errorf("expected 100, got %d", s.Score)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/olegkoval/projects/personal/active/promptctl && go test ./prompt -run TestQualityScore_ZeroPenaltyReturns100 -v`  
Expected: FAIL (undefined ScorePromptQuality)

**Step 3: Stub ScorePromptQuality**

In `prompt/quality.go` add:

```go
func ScorePromptQuality(prompt string) QualityScore {
	return QualityScore{Score: 100, Rules: nil}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./prompt -run TestQualityScore_ZeroPenaltyReturns100 -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add prompt/quality.go prompt/quality_test.go
git commit -m "feat(prompt): add QualityScore type and ScorePromptQuality stub"
```

---

### Task 2: Structure rule (prompt quality)

**Files:**
- Modify: `promptctl/prompt/quality.go`
- Test: `promptctl/prompt/quality_test.go`

**Step 1: Write failing test for structure penalty**

Add to `prompt/quality_test.go`:

```go
func TestQualityScore_StructureMissingSectionsDeducts(t *testing.T) {
	s := ScorePromptQuality("just a paragraph with no sections")
	if s.Score >= 100 {
		t.Errorf("expected penalty for missing structure, got score %d", s.Score)
	}
	found := false
	for _, r := range s.Rules {
		if r == "missing_structure" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_structure in Rules, got %v", s.Rules)
	}
}
```

**Step 2: Run test — expect fail**

Run: `go test ./prompt -run TestQualityScore_StructureMissingSectionsDeducts -v`  
Expected: FAIL (score still 100 or rule not set)

**Step 3: Implement structure check in ScorePromptQuality**

In `quality.go`: implement structure rule (look for role/context/task or XML-style blocks; deduct up to 25, append "missing_structure" to Rules). Reuse ideas from `prompt/score.go` scoreStructure; keep QualityScore calculation in one place (sum penalties from each rule, 100 - total, clamp 0-100).

**Step 4: Run tests**

Run: `go test ./prompt -run "TestQualityScore" -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add prompt/quality.go prompt/quality_test.go
git commit -m "feat(prompt): add structure rule to prompt quality score"
```

---

### Task 3: Remaining rules (clarity, constraints, scope, persona)

**Files:**
- Modify: `promptctl/prompt/quality.go`, `promptctl/prompt/quality_test.go`

**Step 1: Add failing tests per rule**

- Clarity: vague words or very long sentences reduce score, rule "clarity".
- Constraints: no "do not" / "only" / "max" / "constraint" → "missing_constraints".
- Scope: "everything"/"all"/"always" without narrowing → "overbroad_scope".
- Persona: no "you are" / "role" / "acting as" → "missing_persona".

One test per rule (or two: one that triggers, one that does not).

**Step 2: Run tests — expect failures**

Run: `go test ./prompt -run "TestQualityScore" -v`

**Step 3: Implement each rule**

Implement heuristic checks; append rule ID to QualityScore.Rules when triggered; sum penalties (cap per rule as in design), 100 - total, clamp 0-100.

**Step 4: Run all prompt tests**

Run: `go test ./prompt/... -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add prompt/quality.go prompt/quality_test.go
git commit -m "feat(prompt): add clarity, constraints, scope, persona rules to quality score"
```

---

### Task 4: Discovery — find prompt files in directory

**Files:**
- Create: `promptctl/internal/discover/discover.go`
- Test: `promptctl/internal/discover/discover_test.go`

**Step 1: Write failing test**

In `internal/discover/discover_test.go`: test that Discover("testdata", []string{"*.txt"}, nil) returns files under testdata matching *.txt, and that ignore pattern excludes matches.

**Step 2: Run test — expect fail**

Run: `go test ./internal/discover -v`

**Step 3: Implement Discover**

Discover(dir string, include []string, ignore []string) ([]string, error). Use filepath.WalkDir; match include globs; skip paths matching ignore; skip hidden and common dirs (.git, node_modules, vendor). Return sorted paths.

**Step 4: Run test**

Run: `go test ./internal/discover -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/discover/discover.go internal/discover/discover_test.go
git commit -m "feat: add discovery for prompt files by include/ignore globs"
```

---

### Task 5: Score config — load .promptctl/score.yaml

**Files:**
- Create: `promptctl/internal/scoreconfig/config.go`
- Test: `promptctl/internal/scoreconfig/config_test.go`

**Step 1: Define config struct and write failing test**

Struct: Dirs []string, Include []string, Ignore []string, MinScore int, Rules []string. Test: Load from a test YAML file (or in-memory); assert fields.

**Step 2: Run test — expect fail**

Run: `go test ./internal/scoreconfig -v`

**Step 3: Implement Load**

Find .promptctl/score.yaml from cwd upward (reuse pattern from config or safepath). Parse YAML into struct. Return default (e.g. Include: ["*.txt","*.md"], MinScore: 80) when file missing.

**Step 4: Run test**

Run: `go test ./internal/scoreconfig -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/scoreconfig/config.go internal/scoreconfig/config_test.go
git commit -m "feat: add score config load from .promptctl/score.yaml"
```

---

### Task 6: CLI score command — wiring

**Files:**
- Create: `promptctl/cmd/score.go`
- Modify: `promptctl/cmd/root.go` (add case "score")

**Step 1: Add case in root.go**

In `cmd/root.go` switch, add: `case "score": return runScore()`

**Step 2: Implement runScore() stub**

In `cmd/score.go`: parse args (positional paths or use config dirs); if no paths and no config dirs, default current dir. Load score config; resolve paths (expand dirs with discover). For each file, read content, call prompt.ScorePromptQuality, aggregate. Apply min_score from config or --min-score (flag). Output: per-file line to stdout; exit 0 if all >= threshold, 1 otherwise. Exit 2 on usage/config error.

**Step 3: Manual smoke test**

Run: `go build -o promptctl . && ./promptctl score` (from repo root with a .txt file)  
Expected: prints score line, exit 0 or 1

**Step 4: Add --format=json**

When --format=json, output JSON object: {"files":[{"path":"...","score":N,"rules":[]}],"min_score":80,"ok":bool}. Exit code unchanged.

**Step 5: Commit**

```bash
git add cmd/root.go cmd/score.go
git commit -m "feat(cli): add promptctl score with discovery and config"
```

---

### Task 7: Fix — formatting (deterministic)

**Files:**
- Create: `promptctl/prompt/fix.go`
- Test: `promptctl/prompt/fix_test.go`

**Step 1: Failing test**

ApplyFormat("  foo  \n\n\n  bar  \r\n") returns normalized string (LF, trim, collapse blanks).

**Step 2: Implement ApplyFormat**

Normalize line endings to LF, trim trailing space per line, collapse 3+ blank lines to 2 (or 1). Return string.

**Step 3: Run test**

Run: `go test ./prompt -run ApplyFormat -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add prompt/fix.go prompt/fix_test.go
git commit -m "feat(prompt): add deterministic formatting for fix"
```

---

### Task 8: Fix — structure (deterministic)

**Files:**
- Modify: `promptctl/prompt/fix.go`, `promptctl/prompt/fix_test.go`

**Step 1: Failing test**

ApplyStructure("some text") wraps in default sections (e.g. <role>, <context>, <task>); or moves single blob into <task>. No rewording.

**Step 2: Implement ApplyStructure**

Detect existing XML/markdown sections; if expected set missing, insert empty sections in order or move full text into <task>. Return string.

**Step 3: Run test**

Run: `go test ./prompt -run "ApplyStructure|ApplyFormat" -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add prompt/fix.go prompt/fix_test.go
git commit -m "feat(prompt): add structure normalization for fix"
```

---

### Task 9: Fix — LLM suggest (optional)

**Files:**
- Modify: `promptctl/prompt/fix.go` or new `promptctl/prompt/suggest.go`
- Test: `promptctl/prompt/suggest_test.go` (mock or skip when no key)

**Step 1: Failing test**

SuggestScope(prompt, client) returns suggestion string or error. Test with mock client.

**Step 2: Implement suggest**

Call existing LLM config (via llm package); send short system + user prompt asking for one suggestion for scope or constraints. Parse reply; return suggestion. If no API key, return error so CLI can warn and skip.

**Step 3: Run test**

Mock provider in test; run tests. In CI, test can skip if no key.

**Step 5: Commit**

```bash
git add prompt/suggest.go prompt/suggest_test.go  # or fix.go
git commit -m "feat(prompt): add optional LLM suggest for scope/constraints"
```

---

### Task 10: CLI fix command

**Files:**
- Create: `promptctl/cmd/fix.go`
- Modify: `promptctl/cmd/root.go` (add case "fix")

**Step 1: Add case in root.go**

`case "fix": return runFix()`

**Step 2: Implement runFix()**

Parse args (paths or config dirs); load score config for discovery; for each file: read, ApplyFormat, ApplyStructure; if --suggest, call suggest and append to stdout only. If not --dry-run, write back. --dry-run: print diff or result to stdout. Exit 2 on write error.

**Step 3: Smoke test**

Run: `./promptctl fix --dry-run prompts/`  
Expected: prints transformed content, no write

**Step 5: Commit**

```bash
git add cmd/root.go cmd/fix.go
git commit -m "feat(cli): add promptctl fix with format, structure, --suggest, --dry-run"
```

---

### Task 11: Integration test and CI exit codes

**Files:**
- Create or extend: `promptctl/cmd/score_test.go` or `promptctl/tdd_tests/score_integration_test.go`

**Step 1: Integration test**

Create fixture dir with two prompt files (one high score, one low). Run `promptctl score <dir> --min-score=80`; assert exit 1. Run with threshold below both; assert exit 0. Assert --format=json contains "ok" and "files".

**Step 2: Run test**

Run: `go test ./cmd -run Score -v` (or path to integration test)  
Expected: PASS

**Step 3: Document exit codes in README or FEATURE_CONFIG**

Add to docs: exit 0/1/2 behavior and --min-score, --format=json.

**Step 5: Commit**

```bash
git add cmd/score_test.go docs/...
git commit -m "test: integration test for score exit codes and JSON output"
```

---

### Task 12: Docs and changelog

**Files:**
- Modify: `promptctl/README.md`, `promptctl/CHANGELOG.md`, `promptctl/docs/FEATURE_CONFIG.md` (if score config lives there)

**Step 1: Add score and fix to README**

Commands table: score, fix. Short description and example (promptctl score prompts/, promptctl fix --dry-run).

**Step 2: Add config section**

Document .promptctl/score.yaml: dirs, include, ignore, min_score, rules. Flag overrides.

**Step 3: Changelog**

Entry for new commands and config.

**Step 5: Commit**

```bash
git add README.md CHANGELOG.md docs/FEATURE_CONFIG.md
git commit -m "docs: document score, fix, and score config"
```

---

## Execution handoff

Plan is saved to `docs/plans/2025-02-19-prompt-score-fix.md`.

**Two execution options:**

1. **Subagent-driven (this session)** — Dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Parallel session (separate)** — Open a new session with executing-plans and run through the plan with checkpoints.

Which approach do you want?
