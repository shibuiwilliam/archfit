#!/bin/sh
# verify_loop.sh — Fix → re-scan → diff loop. Stops on regression or convergence.
#
# Each iteration:
#   1. Scan to get a baseline.
#   2. Apply safe fixes (archfit fix --all).
#   3. Re-scan.
#   4. Diff: if new findings appeared (regression), stop and report.
#   5. If no findings remain or no progress was made, stop.
#
# Usage:
#   verify_loop.sh [options] [path]
#
# Options:
#   -m NUM   Max iterations (default: 5)
#   -h       Help
#
# Requires: archfit, jq
set -eu

MAX_ITER=5
TARGET="."

while [ $# -gt 0 ]; do
  case "$1" in
    -m) MAX_ITER="$2"; shift 2 ;;
    -h|--help) sed -n '2,17p' "$0"; exit 0 ;;
    -*) echo "unknown flag: $1" >&2; exit 2 ;;
    *)  TARGET="$1"; shift ;;
  esac
done

for cmd in archfit jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "ERROR: $cmd is required but not found" >&2
    exit 3
  fi
done

ITER=0
PREV_COUNT=-1

while [ "$ITER" -lt "$MAX_ITER" ]; do
  ITER=$((ITER + 1))
  echo "=== Iteration $ITER/$MAX_ITER ==="

  # 1. Baseline scan.
  BEFORE=$(archfit scan --json "$TARGET" 2>/dev/null) || true
  BEFORE_COUNT=$(echo "$BEFORE" | jq '.summary.findings_total')
  BEFORE_SCORE=$(echo "$BEFORE" | jq '.scores.overall')
  echo "  before: $BEFORE_COUNT findings, score $BEFORE_SCORE"

  if [ "$BEFORE_COUNT" = "0" ]; then
    echo ""
    echo "No findings remain. Done."
    exit 0
  fi

  # Check for convergence (no progress from last iteration).
  if [ "$BEFORE_COUNT" = "$PREV_COUNT" ]; then
    echo ""
    echo "No progress since last iteration ($BEFORE_COUNT findings remain). Stopping."
    echo "Remaining findings are not auto-fixable — see \`plan_remediation.sh\` for manual steps."
    exit 0
  fi
  PREV_COUNT="$BEFORE_COUNT"

  # 2. Apply safe fixes.
  echo "  applying auto-fixes..."
  archfit fix --all "$TARGET" >/dev/null 2>&1 || true

  # 3. Re-scan.
  AFTER=$(archfit scan --json "$TARGET" 2>/dev/null) || true
  AFTER_COUNT=$(echo "$AFTER" | jq '.summary.findings_total')
  AFTER_SCORE=$(echo "$AFTER" | jq '.scores.overall')
  echo "  after:  $AFTER_COUNT findings, score $AFTER_SCORE"

  FIXED=$((BEFORE_COUNT - AFTER_COUNT))
  if [ "$FIXED" -lt 0 ]; then
    echo ""
    echo "REGRESSION: $((AFTER_COUNT - BEFORE_COUNT)) new finding(s) appeared after fix!"
    echo "Dumping new findings:"
    # Quick diff via jq.
    echo "$AFTER" | jq -r '.findings[] | "  [\(.severity)] \(.rule_id) — \(.message)"'
    exit 1
  fi

  echo "  fixed $FIXED finding(s) this iteration"
  echo ""
done

FINAL=$(archfit scan --json "$TARGET" 2>/dev/null) || true
FINAL_COUNT=$(echo "$FINAL" | jq '.summary.findings_total')
echo "Max iterations reached. $FINAL_COUNT finding(s) remain."
echo "Run \`plan_remediation.sh\` for manual remediation steps."
