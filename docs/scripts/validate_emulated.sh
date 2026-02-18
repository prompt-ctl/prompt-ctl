#!/usr/bin/env bash
# Validate emulated user run outputs: expected exit codes and key strings.
# Usage: ./docs/scripts/validate_emulated.sh [docs/emulated-runs]
# Exit 0 if all checks pass; else 1 and print failures.

OUT="${1:-docs/emulated-runs}"
FAIL=0

check() {
  local user="$1"
  local cmd_idx="$2"
  local want_exit="${3:-0}"
  local want_in_out="$4"
  local want_in_err="$5"
  local dir="$OUT/user-$user"
  local exitfile="$dir/cmd_${cmd_idx}.exit"
  local outfile="$dir/cmd_${cmd_idx}.out"
  local errfile="$dir/cmd_${cmd_idx}.err"
  if [[ ! -f "$exitfile" ]]; then
    echo "FAIL user-$user cmd_$cmd_idx: missing $exitfile"
    FAIL=1
    return
  fi
  local got_exit
  got_exit=$(cat "$exitfile")
  if [[ "$got_exit" != "$want_exit" ]]; then
    echo "FAIL user-$user cmd_$cmd_idx: exit code $got_exit (want $want_exit)"
    FAIL=1
  fi
  if [[ -n "$want_in_out" && -f "$outfile" ]]; then
    if ! grep -q "$want_in_out" "$outfile"; then
      echo "FAIL user-$user cmd_$cmd_idx: stdout missing '$want_in_out'"
      FAIL=1
    fi
  fi
  if [[ -n "$want_in_err" && -f "$errfile" ]]; then
    if ! grep -q "$want_in_err" "$errfile"; then
      echo "FAIL user-$user cmd_$cmd_idx: stderr missing '$want_in_err'"
      FAIL=1
    fi
  fi
}

if [[ ! -d "$OUT" ]]; then
  echo "Run docs/scripts/emulated_user_test.sh first. Missing: $OUT"
  exit 1
fi

check 1 1 0 "promptctl"
check 1 2 0 ""
check 1 3 0 "calls/day\|saves"

check 2 1 0 ""
check 2 2 0 "file"
check 2 3 0 ""
check 2 4 0 ""

check 3 1 0 ""
check 3 2 0 ""
check 3 3 0 "100"

check 4 1 0 "review"
check 4 2 1 ""
check 4 3 0 ""

check 5 1 0 "Initialized"
check 5 2 0 ""
check 5 3 0 ""

check 6 1 0 ""
if [[ -f "$OUT/user-6/cmd_1.out" ]]; then
  if [[ ! -s "$OUT/user-6/cmd_1.out" ]]; then
    echo "FAIL user-6 cmd_1: create stdout empty"
    FAIL=1
  fi
fi

if [[ $FAIL -eq 0 ]]; then
  echo "All emulated user checks passed."
else
  echo "Some checks failed."
  exit 1
fi
