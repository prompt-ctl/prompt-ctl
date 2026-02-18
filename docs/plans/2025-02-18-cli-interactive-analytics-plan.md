# CLI Interactive UX and Analytics — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Survey-based TUI (selections, confirm, input) when TTY, first-run onboarding with reminder, and GA4 analytics (anonymous, consent-first) without changing piped/script behavior.

**Architecture:** UI layer in `internal/ui` wraps Survey and only runs when `interactive()`; analytics in `internal/analytics` handles consent file, client_id, and Measurement Protocol HTTP. `cmd/root.go` calls these where onboarding, model picker, rating, save-to-memory, and consent are needed.

**Tech Stack:** Go 1.22, github.com/AlecAivazis/survey, GA4 Measurement Protocol (net/http, JSON), existing config/llm/prompt packages.

**Design reference:** `docs/plans/2025-02-18-cli-interactive-analytics-design.md`

---

## Task 1: Add Survey dependency and internal/ui stub

**Files:**
- Modify: `go.mod`
- Create: `internal/ui/ui.go`

**Step 1: Add dependency**

Run: `cd /Users/olegkoval/projects/personal/active/promptctl && go get github.com/AlecAivazis/survey/v2`
Expected: dependency added to go.mod and go.sum.

**Step 2: Create UI package with interactive check**

In `internal/ui/ui.go`:

```go
package ui

import (
	"os"
)

// Interactive returns true when stdin and stdout are TTY (same as cmd.interactive).
func Interactive() bool {
	stdin, _ := os.Stdin.Stat()
	stdout, _ := os.Stdout.Stat()
	return (stdin.Mode()&os.ModeCharDevice) != 0 && (stdout.Mode()&os.ModeCharDevice) != 0
}
```

**Step 3: Build**

Run: `go build ./...`
Expected: build succeeds.

**Step 4: Commit**

```bash
git add go.mod go.sum internal/ui/ui.go
git commit -m "chore: add survey dependency and internal/ui Interactive helper"
```

---

## Task 2: UI helpers (Select, Confirm, Input) that run only when Interactive()

**Files:**
- Create: `internal/ui/prompt.go`
- Modify: `internal/ui/ui.go` (export nothing else; keep Interactive)

**Step 1: Write failing test**

Create `internal/ui/prompt_test.go`:

```go
package ui

import (
	"testing"
)

func TestSelectOptionNonInteractive(t *testing.T) {
	// When not TTY, SelectOption should not run survey; we test by not hanging.
	// In CI stdin/stdout are not TTY, so this just ensures we don't panic.
	opts := []string{"a", "b"}
	var out string
	err := SelectOption("Choose", opts, &out)
	// Non-interactive: expect we return without setting out, or get a safe default/error
	if err != nil && out != "" {
		t.Log("non-interactive: err or empty out is acceptable")
	}
}
```

Run: `go test ./internal/ui/... -v`
Expected: FAIL (SelectOption undefined) or test compiles and we adjust so test verifies non-interactive path.

**Step 2: Implement SelectOption, Confirm, Input in internal/ui/prompt.go**

- `SelectOption(prompt string, options []string, result *string) error`: if !Interactive() return nil or ErrNotInteractive; else survey.Select with options, write to result.
- `Confirm(prompt string, defaultYes bool) (bool, error)`: if !Interactive() return defaultYes, nil; else survey.Confirm.
- `Input(prompt string, result *string) error`: if !Interactive() return ErrNotInteractive; else survey.Input.
- Define `var ErrNotInteractive = errors.New("not interactive")`.

Use `survey.AskOne` with appropriate survey.* types. Set survey stdio to os.Stdin, os.Stdout, os.Stderr.

**Step 3: Run tests**

Run: `go test ./internal/ui/... -v`
Expected: PASS (non-TTY: SelectOption returns without hanging; Confirm returns default).

**Step 4: Commit**

```bash
git add internal/ui/
git commit -m "feat(ui): add SelectOption, Confirm, Input with TTY check"
```

---

## Task 3: Analytics package — consent file and client_id

**Files:**
- Create: `internal/analytics/consent.go`
- Create: `internal/analytics/consent_test.go`

**Step 1: Write failing test**

In `internal/analytics/consent_test.go`: test that ReadConsent returns enabled: false and no client_id when file does not exist; test that after WriteConsent(enabled: true) ReadConsent returns enabled and a non-empty client_id; test that WriteConsent(enabled: false) then ReadConsent returns enabled false (no client_id needed).

**Step 2: Implement consent read/write**

- Consent file: `~/.promptctl/analytics.json`. Struct: `{ "enabled": bool, "client_id": "uuid" }`. When enabled is false, client_id may be omitted.
- ReadConsent() (*Consent, error): read file; if missing return enabled: false, client_id: "".
- WriteConsent(enabled bool) error: if enabled, generate UUID (e.g. google/uuid or crypto/rand), write JSON; if !enabled, write {"enabled":false}.
- Use os.UserHomeDir(), filepath.Join for path.

**Step 3: Run tests**

Run: `go test ./internal/analytics/... -v`
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/analytics/
git commit -m "feat(analytics): consent file read/write and client_id"
```

---

## Task 4: Analytics — GA4 Measurement Protocol client

**Files:**
- Create: `internal/analytics/ga4.go`
- Create: `internal/analytics/ga4_test.go`

**Step 1: Write test**

Test that when consent is disabled, SendEvent or EnqueueEvent does not send HTTP. Test payload shape (measurement_id, client_id, events[].name, events[].params) when sending (mock HTTP client or use httptest).

**Step 2: Implement**

- Measurement ID: G-DQBN89S2FZ. API secret: from env `PROMPTCTL_GA4_SECRET` (or constant for dev; document in design).
- SendEvent(ctx, clientID string, eventName string, params map[string]interface{}) error: POST to `https://www.google-analytics.com/mp/collect?measurement_id=G-DQBN89S2FZ&api_secret=<secret>`, body JSON: client_id, events: [{ name, params }]. Fire-and-forget: run in goroutine or non-blocking; on failure no user message.
- Only call SendEvent when ReadConsent().Enabled is true and client_id non-empty.

**Step 3: Run tests**

Run: `go test ./internal/analytics/... -v`
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/analytics/
git commit -m "feat(analytics): GA4 Measurement Protocol client"
```

---

## Task 5: Ensure first-run analytics consent prompt (Survey Confirm)

**Files:**
- Modify: `internal/ui/prompt.go` (already has Confirm)
- Create: `internal/analytics/consent_prompt.go` or integrate into cmd

**Step 1: Logic**

- Function: EnsureAnalyticsConsent() error. If !ui.Interactive() return nil (no prompt). ReadConsent(); if already has enabled true or false, return nil. Else run ui.Confirm("Send anonymous usage stats to improve promptctl? (Y/n)", true). If yes, WriteConsent(true); if no, WriteConsent(false). Return nil.

**Step 2: Call site**

- Call EnsureAnalyticsConsent() only from places that are about to send an event (e.g. before sending onboarding_started or prompt_created). So: from the analytics package when we’re about to send the first event, check consent; if not set, and Interactive(), prompt then persist. Alternatively expose EnsureAnalyticsConsent from analytics and call it at start of onboarding and once before create flow that may send prompt_created.

**Step 3: Test**

Unit test: when consent file exists with enabled true/false, EnsureAnalyticsConsent does not prompt (mock or no TTY). When file missing and non-TTY, does not prompt.

**Step 4: Commit**

```bash
git add internal/analytics/ internal/ui/
git commit -m "feat(analytics): first-run consent prompt when TTY"
```

---

## Task 6: Onboarding skipped state and reminder

**Files:**
- Create: `internal/onboarding/state.go` or use config dir
- Modify: `cmd/root.go` (where we decide to run onboarding)

**Step 1: State file**

- File: `~/.promptctl/onboarding_skipped` (empty file or one line "true"). Write when user quits/skips onboarding. Delete or clear when user completes onboarding.

**Step 2: Reminder message**

- When we’re about to run onboarding because config is missing, check for onboarding_skipped. If present, print to stderr: "Run `promptctl config` to set up your LLM." (or one-line from design). Then start wizard. After successful completion, remove onboarding_skipped.

**Step 3: Wire in cmd/root.go**

- In createPrompt, sendPrompt, etc., when config is needed: if no llm config, if TTY run onboarding (and if skipped file exists show reminder first); if not TTY return error "No LLM config. Run `promptctl config` in a terminal to set up."

**Step 4: Commit**

```bash
git add cmd/root.go internal/onboarding/ or config/
git commit -m "feat: onboarding skipped state and one-line reminder"
```

---

## Task 7: Replace config wizard with Survey (provider, model, API key)

**Files:**
- Modify: `cmd/root.go` (configOnboarding, configLLM)

**Step 1: Provider selection**

- Use ui.SelectOption with provider names (from llm.ProviderKeys() / llm.Providers). Map selected index to provider key. Only when ui.Interactive().

**Step 2: Model selection**

- Use ui.SelectOption with model names (and price) for selected provider. Map to model ID.

**Step 3: API key**

- Keep existing “keep existing key?” with ui.Confirm. For new key: ui.Input or password-style input (survey.Password). Open browser link when user presses Enter on first prompt (existing openBrowser).

**Step 4: Non-TTY**

- When !ui.Interactive(), keep current behavior: do not run wizard; config only via flags.

**Step 5: Test**

Run: `promptctl config` in terminal → see Survey selects. Run: `echo n | promptctl config` (or piped) → no Survey, or flag-based only.

**Step 6: Commit**

```bash
git add cmd/root.go
git commit -m "feat(config): use Survey for onboarding wizard when TTY"
```

---

## Task 8: Wire analytics events (onboarding_started, onboarding_completed, onboarding_skipped)

**Files:**
- Modify: `cmd/root.go` (configOnboarding)
- Modify: `internal/analytics/ga4.go` (ensure SendEvent is used)

**Step 1: onboarding_started**

- At start of configOnboarding (when we’re in wizard), call EnsureAnalyticsConsent(); then if consent enabled, SendEvent("onboarding_started", nil).

**Step 2: onboarding_completed**

- After saving config (provider + model), SendEvent("onboarding_completed", map[string]interface{}{"model_id": selectedModel.ID, "provider": selectedProviderKey}).

**Step 3: onboarding_skipped**

- When user quits or skips (no config saved), if consent enabled SendEvent("onboarding_skipped", nil). Write onboarding_skipped state file.

**Step 4: Commit**

```bash
git add cmd/root.go internal/analytics/
git commit -m "feat(analytics): onboarding_started, completed, skipped events"
```

---

## Task 9: Model picker (models --set) with Survey Select

**Files:**
- Modify: `cmd/root.go` (interactiveModelSwitch)

**Step 1: Replace number input with ui.SelectOption**

- Build options slice: each option string e.g. "[provider] modelName  $x in $y out  ✓/✗ key". Map selected index to model. Only when ui.Interactive(); else keep current "Enter number" or return error.

**Step 2: Send model_selected event**

- After saving default model, if analytics consent enabled SendEvent("model_selected", model_id, provider).

**Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "feat(models): Survey select for model picker, analytics event"
```

---

## Task 10: Rating and save-to-memory with Survey

**Files:**
- Modify: `cmd/root.go` (askUserRating, askSaveToMemory)

**Step 1: Rating**

- When interactive(), use ui.SelectOption with options ["1 - Poor", "2", "3", "4", "5 - Great", "Skip"]. Map to 1–5 or 0 for skip. Then persistRating as today; if consent enabled SendEvent("prompt_rated", rating, optional intent_length).

**Step 2: Save to memory**

- Use ui.Confirm("Save to memory?", true). If yes, ui.Input("Folder (optional)", &folder), ui.Input("Prompt name", &name). Validate and save as today. If saved and consent enabled SendEvent("prompt_saved", nil).

**Step 3: Free retry**

- Use ui.Confirm("Try again for free? (once per day)", false).

**Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "feat(create): Survey for rating, save-to-memory, free retry"
```

---

## Task 11: prompt_created and prompt_saved events

**Files:**
- Modify: `cmd/root.go` (createPrompt)

**Step 1: prompt_created**

- After successful EnhanceWithFallback, EnsureAnalyticsConsent(); if enabled SendEvent("prompt_created", nil). Fire-and-forget.

**Step 2: prompt_saved**

- Already added in Task 10 when user saves to memory.

**Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "feat(analytics): prompt_created and prompt_saved events"
```

---

## Task 12: Optional ANSI color for stderr messages

**Files:**
- Create: `internal/ui/color.go` or add to prompt.go
- Modify: `cmd/root.go` (selected success/hint lines)

**Step 1: Helpers**

- When Interactive(), wrap success messages with green, hints with dim (ANSI codes). Functions e.g. Success(s string) string, Hint(s string) string. When !Interactive() return s unchanged.

**Step 2: Use in cmd**

- e.g. "✓ Default model set to" → Success("✓ ..."). "Run promptctl config" reminder → Hint(...). Minimal surface.

**Step 3: Commit**

```bash
git add internal/ui/ cmd/root.go
git commit -m "feat(ui): optional ANSI color for success/hint when TTY"
```

---

## Task 13: Documentation and GA4 secret

**Files:**
- Modify: `docs/plans/2025-02-18-cli-interactive-analytics-design.md` or README
- Add: env var PROMPTCTL_GA4_SECRET in design doc or README

**Step 1: Document**

- README or docs: mention interactive TUI (Survey) when in terminal; analytics optional, first-run prompt; opt-out by answering n. GA4 API secret: set PROMPTCTL_GA4_SECRET for analytics to work (or document that without it events are no-op).

**Step 2: Commit**

```bash
git add README.md or docs/
git commit -m "docs: CLI TUI and analytics (GA4 secret, consent)"
```

---

## Execution Handoff

Plan complete and saved to `docs/plans/2025-02-18-cli-interactive-analytics-plan.md`.

**Two execution options:**

1. **Subagent-Driven (this session)** — Dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Parallel Session (separate)** — Open a new session with executing-plans and run in batches with checkpoints.

Which approach do you want?
