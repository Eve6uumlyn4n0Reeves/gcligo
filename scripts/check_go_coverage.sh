#!/usr/bin/env bash
set -euo pipefail

PROFILE=${1:-coverage.out}
THRESHOLD=${GO_COVERAGE_MIN:-13}
GO_BIN=${GO_BIN:-go}

if [[ ! -f "$PROFILE" ]]; then
  echo "coverage profile $PROFILE not found" >&2
  exit 1
fi

total=$($GO_BIN tool cover -func="$PROFILE" | awk '/^total:/ {print substr($3, 1, length($3)-1)}')
if [[ -z "$total" ]]; then
  echo "failed to parse coverage percentage" >&2
  exit 1
fi

awk -v total="$total" -v threshold="$THRESHOLD" 'BEGIN {
  if ((total + 0.0) < (threshold + 0.0)) {
    printf "Go coverage %.2f%% is below required %.2f%%\n", total, threshold > "/dev/stderr"
    exit 1
  }
}'

printf "Go coverage %.2f%% >= %.2f%% (pass)\n" "$total" "$THRESHOLD"
