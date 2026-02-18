#!/usr/bin/env bash
# Emulated user testing: run 5-6 personas with isolated config, collect stdout/stderr and exit codes.
# Usage: from repo root, build first (go build -o promptctl .), then:
#   ./docs/scripts/emulated_user_test.sh
# Outputs: docs/emulated-runs/user-{1..6}/ and docs/emulated-runs/run.log

set -e

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO_ROOT"

BIN="${PROMPTCTL_BIN:-./promptctl}"
OUT="${EMULATED_OUT:-docs/emulated-runs}"
RUN_LOG="$OUT/run.log"

mkdir -p "$OUT"
: > "$RUN_LOG"

run_user() {
  local id="$1"
  local label="$2"
  shift 2
  local dir="$OUT/user-$id"
  mkdir -p "$dir"
  export PROMPTCTL_CONSENT_DIR="$dir/config"
  export PROMPTCTL_ONBOARDING_DIR="$dir/config"
  mkdir -p "$dir/config"
  echo "  user-$id: $label"
  echo "=== user-$id: $label ===" >> "$RUN_LOG"
  local idx=0
  while [[ $# -gt 0 ]]; do
    local cmd="$1"
    shift
    ((idx++)) || true
    local name="cmd_${idx}"
    local outfile="$dir/${name}.out"
    local errfile="$dir/${name}.err"
    local exitfile="$dir/${name}.exit"
    echo "  $cmd" >> "$RUN_LOG"
    set +e
    $BIN $cmd > "$outfile" 2> "$errfile"
    echo $? > "$exitfile"
    set -e
  done
  echo "" >> "$RUN_LOG"
}

if [[ ! -x "$BIN" ]]; then
  echo "Build the CLI first: go build -o promptctl ."
  exit 1
fi

echo "Running emulated users (bin: $BIN)..."

run_user 1 "Alex (first-time)" \
  "version" \
  "list" \
  "savings"

run_user 2 "Sam (daily)" \
  "list" \
  "vars review" \
  "cost review --file=cmd/root.go" \
  "cost --compare \"review this function\""

run_user 3 "Jordan (cost)" \
  "models" \
  "savings" \
  "savings --calls-per-day=100"

run_user 4 "Riley (templates)" \
  "show review" \
  "run review" \
  "run review --file=cmd/root.go"

run_user 5 "Morgan (onboarding)" \
  "init" \
  "list" \
  "memory list"

run_user 6 "Taylor (create)" \
  "create \"summarize this in 3 bullets\""

echo "Done. Emulated runs written to $OUT. Log: $RUN_LOG"
