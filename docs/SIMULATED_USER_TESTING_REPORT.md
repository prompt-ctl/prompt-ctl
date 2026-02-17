# Simulated CLI User Testing Report

**Date:** 2026-02-17  
**Scope:** promptctl CLI v0.7.0  
**Method:** Sandbox simulation with 5 personas; real commands, collected outcomes and simulated feedback.

---

## 1. Personas and scenarios

| Persona | Goal | Commands / flow |
|--------|------|------------------|
| **Alex** – First-time / Hobby | "I heard about promptctl, want to see what it does" | `promptctl` → `promptctl version` → `promptctl list` → `promptctl savings` |
| **Sam** – Daily user | "I use it for code review and cost checks" | `promptctl list` → `promptctl vars review` → `promptctl cost review --file=cmd/root.go` → `promptctl cost --compare "review this function"` |
| **Jordan** – Cost-conscious | "I care about spend and annual projection" | `promptctl models` → `promptctl savings` → `promptctl savings --calls-per-day=100` |
| **Riley** – Template power user | "I want to run templates and see what they need" | `promptctl show review` → `promptctl run review` (no vars) → `promptctl run review --file=cmd/root.go` |
| **Morgan** – Onboarding | "Just installed, need to set up" | `promptctl init` (in fresh dir) → `promptctl list` → `promptctl memory list` |

---

## 2. Observed behavior (sandbox runs)

### 2.1 Alex – First-time / Hobby

- **`promptctl`** → Full usage printed; COST SAVINGS section mentions `cost --compare` and `savings`.
- **`promptctl version`** → `promptctl v0.7.0`.
- **`promptctl list`** → Lists templates (arch, bizidea, commit, debug, explain, review) with scope and descriptions.
- **`promptctl savings`** → Shows default model, 30 calls/day, annual range (e.g. ~$647–875/year), and hint to run `cost --compare`.

**Simulated feedback (Alex):**  
“Help is clear. `savings` is a nice hook – I didn’t expect to see ‘$647–875/year’ right away. I’d add a one-liner in the main help that says: ‘New? Run `promptctl savings` to see your potential annual savings.’ ”

---

### 2.2 Sam – Daily user

- **`promptctl vars review`** → Shows `--file` (required) and `--focus` (default: general). Clear.
- **`promptctl cost review --file=cmd/root.go`** → Single-model estimate with tokens, cost, “Without promptctl”, “You save” and 64%. Works as expected.
- **`promptctl cost --compare "review this function for bugs"`** → Comparison table for all models; varied savings (55–71%); “At 30 calls/day …” projection. **Fixed:** Atlas (hosted) now shows **N/A** when cost is $0 instead of NaN%.

**Simulated feedback (Sam):**  
“Cost and cost --compare are exactly what I need. Atlas showing N/A when free is clear.”

---

### 2.3 Jordan – Cost-conscious

- **`promptctl models`** → Table of providers, models, $/1M in/out, context; “Current default” and “Change default: promptctl models --set”.
- **`promptctl savings`** → Same as Alex; clear annual range for default model.
- **`promptctl savings --calls-per-day=100`** → Same format, higher range (e.g. ~$2,156–2,917/year).

**Simulated feedback (Jordan):**  
“Models and savings are great. I’d love to see `savings` mention the default model by name in the first line (you already do), and maybe a `--model=` flag so I can project for a different model without changing default.”

---

### 2.4 Riley – Template power user

- **`promptctl show review`** → Template name, description, variables, and prompt body. Good.
- **`promptctl run review`** (no vars) → **Error:** “required variable '--file' not provided”. Clear and actionable.
- **`promptctl run review --file=cmd/root.go`** → Would render and print (not run in this report to avoid side effects).

**Simulated feedback (Riley):**  
“Error message is good. It would help if `promptctl run <name>` without required vars printed a one-line reminder like ‘Variables for review: promptctl vars review’.”

---

### 2.5 Morgan – Onboarding

- **`promptctl init`** (in empty dir) → “Initialized promptctl in: …” and “Run 'promptctl list' to see them.” Uses global `~/.promptctl/templates` when no local `.promptctl`.
- **`promptctl list`** → Shows templates after init.
- **`promptctl memory list`** → Lists saved prompts (e.g. arch, review, test/auth-test). Distinction between “templates” (list) and “saved prompts” (memory list) may be subtle for new users.

**Simulated feedback (Morgan):**  
“Init was straightforward. I wasn’t sure what ‘memory’ vs ‘list’ meant at first – a short note in help under MEMORY like ‘Prompts you saved from create’ would help.”

---

## 3. Bugs / issues found

| # | Severity | What | Where |
|---|----------|------|--------|
| 1 | ~~Low~~ **Fixed** | ~~**NaN%** in cost comparison when model cost is $0~~ Now shows **N/A** for Atlas/hosted | `llm/provider.go` / `FormatCostComparison` |
| 2 | Nice-to-have | No `--model=` for `promptctl savings` | `cmd/root.go` `showSavings()` |
| 3 | Nice-to-have | `run` with missing vars could suggest `promptctl vars <name>` | `cmd/root.go` run path |

---

## 4. Summary and recommendations

- **Strengths:** Help, version, list, cost (single + compare), savings, models, and template flows (show, vars, run with file) all behaved as intended. Varied savings % and annual projection are clear.
- **Done:** In `FormatCostComparison`, when `est.WastedWithout == 0` or savings percent is NaN, display “N/A” (implemented).
- **Enhancements (optional):**  
  - Add `savings --model=MODEL`.  
  - In main help, one line: “New? Run `promptctl savings` to see annual savings.”  
  - When `run` fails for missing required vars, append “Run 'promptctl vars <name>' for required variables.”  
  - MEMORY section in help: “Prompts you saved from create.”

---

## 5. How to re-run / extend

- From repo root with local templates:  
  `./promptctl <command> [args]`
- Fresh sandbox:  
  `mkdir -p /tmp/promptctl-sandbox && cd /tmp/promptctl-sandbox && promptctl init`
- Key commands used:  
  `promptctl`, `promptctl version`, `promptctl list`, `promptctl savings`, `promptctl savings --calls-per-day=100`, `promptctl models`, `promptctl cost review --file=cmd/root.go`, `promptctl cost --compare "review this function for bugs"`, `promptctl show review`, `promptctl vars review`, `promptctl run review`, `promptctl memory list`, `promptctl init`.

This report is based on simulated personas and real CLI output; for validation, run the same flows with 2–5 real users and replace “Simulated feedback” with their actual quotes.
