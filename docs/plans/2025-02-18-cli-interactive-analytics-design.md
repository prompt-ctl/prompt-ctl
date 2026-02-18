# CLI Interactive UX and Analytics — Design

**Status:** Approved  
**Date:** 2025-02-18

## Goal

Make the promptctl CLI more interactive (selections/dialogs, colored output), keep only essential typed inputs (create intent, tokens, etc.), add first-run onboarding with a reminder if skipped, and add optional anonymous analytics to GA4 (G-DQBN89S2FZ).

## Decisions

| Topic | Choice |
|-------|--------|
| Analytics identity | Optional anonymous id: random id stored in `~/.promptctl` when user opts in. |
| TUI | Survey library (AlecAivazis/survey) for list select, confirm, input. |
| When to use TUI | Only when stdin and stdout are TTY; piped/CI keeps current plain behavior. |
| Onboarding | Run when `promptctl config` or when config missing on first use; if user skips, show one-line reminder on next run. |
| Analytics consent | First-run prompt: "Send anonymous usage stats to improve promptctl? (Y/n)"; store choice, no flag for now. |

---

## Section 1: Architecture and TTY

- Add **github.com/AlecAivazis/survey**. Use it only when `interactive()` is true (stdin and stdout are TTY).
- **UI layer:** Small helper (`internal/ui` or `cmd/ui.go`) wrapping Survey: SelectOption, Confirm, Input, InputRequired. Each checks `interactive()`; if false, callers keep existing non-interactive branches.
- **Colors:** Optional ANSI in same layer (success, hint) when TTY; no extra dependency.
- **Backward compatibility:** Piped usage and flag-based config unchanged. Scripts remain valid.

---

## Section 2: Interactive Flows

**Use Survey (when TTY):**

- **Onboarding:** Select provider → Select model → Confirm keep key? → Input/password for API key.
- **Model picker** (`models --set`): Select from models (name, price, key status).
- **Rating** (after create): Select 1–5 or Skip.
- **Save to memory:** Confirm → Input folder (optional) → Input prompt name.
- **Free retry?:** Confirm.
- **Analytics consent:** One Confirm; store in `~/.promptctl/analytics.json`.

**Stay typed / CLI arg only:** Create intent (arg). API key via Survey Input in onboarding when TTY; non-TTY remains flag-only.

---

## Section 3: Onboarding (first run + reminder)

- **Explicit:** `promptctl config` (no flags) → full wizard.
- **Implicit:** Commands that need config (`create`, `send`, etc.) check for config; if missing and TTY → run onboarding. If not TTY → error: "Run `promptctl config` in a terminal."
- **Skipped:** If user quits/skips, store sentinel (e.g. `~/.promptctl/onboarding_skipped`). Next run (TTY): show one-line reminder then offer wizard again. No reminder after completion.

---

## Section 4: Analytics

- **Backend:** GA4 Measurement Protocol (HTTP). Measurement ID G-DQBN89S2FZ; API Secret from GA4 Admin (code or env e.g. `PROMPTCTL_GA4_SECRET`).
- **Identity:** On "Y" to consent, generate and store anonymous client_id in `~/.promptctl/analytics.json` with `enabled: true`. On "n", store `enabled: false`. Never send if disabled.
- **First-run prompt:** Before first event, if no stored choice and TTY, ask once; store result.
- **Events:** `onboarding_started`, `onboarding_completed` (model_id, provider), `onboarding_skipped`, `prompt_created`, `prompt_saved`, `prompt_rated` (rating, optional intent_length), `model_selected` (model_id, provider). Fire-and-forget HTTP; no user-facing errors.

---

## Section 5: Error Handling and Testing

- Survey/TTY errors: treat like today (exit/return, no panic). Skipped onboarding → send `onboarding_skipped` only if consent already given.
- Analytics failures: ignore; optional single retry, no user message.
- **Testing:** Unit tests for UI helpers with non-TTY; analytics payload and no-send when disabled. Integration: existing tests unchanged when not TTY. Manual smoke for onboarding, model picker, create→rate→save, consent.

**File layout:** `internal/ui` (Survey wrappers, ANSI); `internal/analytics` or `analytics/` (GA4 client, consent, events). Config: `~/.promptctl/analytics.json`, `~/.promptctl/onboarding_skipped` or sentinel. `cmd/root.go` calls UI and analytics; no change to `prompt/`, `llm/`, `config/` APIs beyond "config missing" and "onboarding skipped".
