# Prompt scoring and fix — Design

**Status:** Approved  
**Date:** 2025-02-19

## Goal

Add `promptctl score` and `promptctl fix` so promptctl becomes "ESLint/Prettier for prompts": CI-ready scoring, deterministic formatting and structure fixes, and optional LLM suggestions. Success = `promptctl score` reliable in CI (exit code + threshold); fix is optional follow-up.

## Decisions

| Topic | Choice |
|-------|--------|
| Fix scope (v1) | Formatting + structure (deterministic) + optional LLM suggest for selected rules (scope, constraints). |
| Success bar | CI-ready: exit code + threshold; fix optional. |
| Input | Directory or explicit paths; discovery via include/ignore globs. |
| Config | Hybrid: optional `.promptctl/score.yaml`; flags/env override. |
| Score rules | Offline, heuristic (structure, clarity, constraints, scope, persona). |

---

## Section 1: Overview

**Scope:** New CLI commands `score` and `fix`. Input: directory (with discovery) or explicit paths. Config: optional repo config with flag/env overrides.

**Out of scope for v1:** Cost/telemetry, GitHub Action packaging, public spec, benchmark leaderboard.

**Existing reuse:** `prompt/score.go` today scores *enhance* results. We add a separate prompt-quality scorer for raw prompts; can reuse structure-detection ideas, not the same function.

---

## Section 2: Score — rules, output, exit code

**Rules (v1):** Each rule contributes to 0–100 (subtractive penalties, clamped).

| Rule | What it checks | Penalty / behavior |
|------|----------------|--------------------|
| Structure | Recognizable sections (role/context/task or XML-style); no duplicate headers | Missing: up to ~25; duplicates: extra |
| Clarity | Sentence length, vague words, unclear pronouns | Small per issue |
| Constraints | Explicit constraints or boundaries | Missing: up to ~20 |
| Scope | Overbroad phrasing without narrowing | Up to ~15; candidate for LLM suggest |
| Persona | Clear role/identity for model | Missing: up to ~15 |

All heuristic (regex/structure/word lists); scoring is offline and deterministic.

**Output:** Per-file line (e.g. `prompts/foo.txt  72  (missing constraints, overbroad scope)`). `--verbose` for rule breakdown. `--format=json` for CI: per-file score and triggered rules.

**Exit code:** 0 = all files ≥ threshold; 1 = at least one below or no files found; 2 = usage/config error.

**Threshold:** Config `min_score` or `--min-score` (default e.g. 80). If unset, exit 0 when scoring ran (no failure by score).

---

## Section 3: Discovery and config (hybrid)

**Discovery:** Accept file(s) or directory(ies). Directory: recursive find by **include** globs (e.g. `*.txt`, `*.md`), optional **ignore** patterns. Default when no config: include `*.txt`, `*.md`; ignore hidden and common dirs (e.g. `.git`, `node_modules`, `vendor`). Empty / no matches → exit 1, clear message.

**Config (optional):** `.promptctl/score.yaml` (or score section in unified config). Lookup from cwd upward. Contents: `dirs`/`paths`, `include`, `ignore`, `min_score`, optional `rules` (enable/disable by ID). CLI flags override: `--min-score`, `--dir` or positional paths; env e.g. `PROMPTCTL_MIN_SCORE`, `PROMPTCTL_SCORE_DIR`.

**Precedence:** CLI paths → config dirs; then include/ignore from config or defaults; min_score: flag → env → config → default.

---

## Section 4: Fix — formatting, structure, LLM suggest

**Formatting (deterministic):** Normalize line endings (LF), trim trailing whitespace, collapse excess blank lines. Optional `--wrap=N` for line length.

**Structure (deterministic):** Detect existing blocks; if expected sections missing, insert empty sections or move unstructured blob into e.g. `<task>`. No rewording.

**LLM suggest (optional):** `--suggest`. For rules that support it (v1: scope, constraints), call existing LLM config; return short suggestion. Output to stdout only in v1 (no write into file). If no API key: skip suggest, warn once; still do format + structure.

**Write behavior:** Default: write format + structure back. `--dry-run`: print diff/result to stdout, no writes. `--suggest`: add suggestions to stdout only.

**Edge cases:** Non-UTF-8 or empty-after-trim: skip and report; do not overwrite.

---

## Section 5: CI contract and error handling

**CI:** `promptctl score [paths...]` or `promptctl score` (config dirs). Exit 0 = all ≥ threshold; 1 = below or no files; 2 = invalid args/config/I/O. `--format=json`: single object with `files`, `min_score`, `ok`.

**Errors:** Missing path → exit 2. Config parse error → exit 2, no fallback. Non-UTF-8/unreadable: skip file, warn; if all skipped → exit 1. Empty file: skip, warn; do not score 0. Fix write failure → exit 2. Score is offline; fix with `--suggest` may call LLM; on failure warn and still apply format + structure.

---

## Section 6: Testing

- **Unit:** Per-rule score tests (fixed input → expected score/penalties). Format/structure fix steps: known before/after.
- **Integration:** `promptctl score <dir>` on fixture dir: exit 0 vs 1; `--format=json` structure and `ok`/scores.
- **Fix:** Fixture prompts; assert format + structure output (or file with `--dry-run`). `--suggest`: mock LLM or skip when no API key so CI stays offline.
