# Emulated User Test Report (CLI v0.7.5+)

**Date:** 2026-02-18  
**Scope:** promptctl CLI with interactive TUI (Survey), onboarding, and analytics.  
**Method:** Non-interactive emulation of 6 personas; collected stdout/stderr and exit codes; automated validation.

---

## 1. How to run and validate

```bash
# From repo root
go build -o promptctl .
./docs/scripts/emulated_user_test.sh
./docs/scripts/validate_emulated.sh
```

- **Outputs:** `docs/emulated-runs/user-{1..6}/` (per-command `.out`, `.err`, `.exit`) and `docs/emulated-runs/run.log`.
- **Validation:** Checks expected exit codes and key substrings (e.g. version prints "promptctl", create produces non-empty stdout, `run review` without vars exits non-zero).

---

## 2. Personas and commands

| User | Persona           | Commands |
|------|-------------------|----------|
| 1    | Alex (first-time) | version, list, savings |
| 2    | Sam (daily)       | list, vars review, cost review --file=cmd/root.go, cost --compare "review this function" |
| 3    | Jordan (cost)     | models, savings, savings --calls-per-day=100 |
| 4    | Riley (templates) | show review, run review (expect fail), run review --file=cmd/root.go |
| 5    | Morgan (onboarding) | init, list, memory list |
| 6    | Taylor (create)   | create "summarize this in 3 bullets" |

All runs use isolated config dirs (`PROMPTCTL_CONSENT_DIR`, `PROMPTCTL_ONBOARDING_DIR`) so no TTY and no shared state.

---

## 3. Quality validation (automated)

The validator verifies:

- **User 1:** version stdout contains "promptctl"; list and savings exit 0; savings refers to calls/day or saves.
- **User 2:** all four commands exit 0; vars review stdout mentions "file".
- **User 3:** all exit 0; third command (savings --calls-per-day=100) stdout contains "100".
- **User 4:** show review exit 0 and stdout contains "review"; run review (no vars) exit 1; run review --file=... exit 0.
- **User 5:** init exit 0 and stdout contains "Initialized"; list and memory list exit 0.
- **User 6:** create exit 0 and stdout non-empty (prompt body).

**Result:** All emulated user checks passed (run `./docs/scripts/validate_emulated.sh` to reproduce).

---

## 4. Limitations

- **Non-interactive only:** Survey flows (config wizard, model picker, rating, save-to-memory) are not exercised; only non-TTY code paths run.
- **No network:** `create` uses the rule-based enhancer when the enhance URL is unreachable. Commands that call the enhance API or LLMs (`send`, or `cost`/`create` with the hosted enhancer) are not run here to avoid external calls.
- **Full E2E:** To test interactive TTY flows and analytics, run the same flows manually in a terminal or use an expect/pexpect-style driver.

---

## 5. Fix / improve based on simulations

From the earlier [Simulated User Testing Report](SIMULATED_USER_TESTING_REPORT.md) and emulated runs, these are still open:

| Priority   | Item | Source | Where to change |
|-----------|------|--------|------------------|
| Nice-to-have | **`savings --model=MODEL`** — Project annual savings for a specific model without changing default | Jordan | `cmd/root.go` `showSavings()` |
| Nice-to-have | **Main help:** Add one line: “New? Run `promptctl savings` to see your potential annual savings.” | Alex | `cmd/root.go` `printUsage()` |
| Nice-to-have | **`run` with missing vars:** When failing for required variable, append: “Run 'promptctl vars <name>' for required variables.” | Riley | `cmd/root.go` run path (template render error) |
| Nice-to-have | **MEMORY in help:** Under MEMORY section, add: “Prompts you saved from create.” | Morgan | `cmd/root.go` `printUsage()` |

**Already done:** Cost comparison shows N/A (not NaN%) when model cost is $0 (Atlas/hosted).

---

## 6. Skills (find-skills)

Relevant skills from the ecosystem:

- **quality-validation** — [qodex-ai/ai-agent-skills@quality-validation](https://skills.sh/qodex-ai/ai-agent-skills/quality-validation) — quality checks on outputs.
- **e2e-testing-automation** — [aj-geddes/useful-ai-prompts@e2e-testing-automation](https://skills.sh/aj-geddes/useful-ai-prompts/e2e-testing-automation) — E2E test design.
- **ui-testing** — [alinaqi/claude-bootstrap@ui-testing](https://skills.sh/alinaqi/claude-bootstrap/ui-testing) — UI test patterns.

Install example: `npx skills add qodex-ai/ai-agent-skills@quality-validation -g -y`

---

*For full persona feedback and observed behavior, see [Simulated User Testing Report](SIMULATED_USER_TESTING_REPORT.md).*
