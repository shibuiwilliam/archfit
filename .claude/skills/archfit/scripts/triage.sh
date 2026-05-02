#!/bin/sh
# triage.sh — Show top-N most severe findings from an archfit scan.
#
# Filters to error + critical by default. Use -a/--all for all severities.
# Output is JSON (machine-readable) or terminal (human-readable).
#
# Usage:
#   triage.sh [options] [path]
#
# Options:
#   -n NUM   Number of findings to show (default: 5)
#   -a       Show all severities, not just error+critical
#   -j       JSON output
#   -h       Help
#
# Requires: archfit, jq
set -eu

N=5
ALL_SEV=false
JSON=false
TARGET="."

while [ $# -gt 0 ]; do
  case "$1" in
    -n) N="$2"; shift 2 ;;
    -a|--all) ALL_SEV=true; shift ;;
    -j|--json) JSON=true; shift ;;
    -h|--help) sed -n '2,16p' "$0"; exit 0 ;;
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

SCAN=$(archfit scan --json "$TARGET" 2>/dev/null) || true

if [ "$ALL_SEV" = true ]; then
  FILTER='.'
else
  FILTER='select(.severity == "error" or .severity == "critical")'
fi

RESULTS=$(echo "$SCAN" | jq --arg n "$N" \
  "[.findings[] | $FILTER] | .[:(\$n | tonumber)]")

COUNT=$(echo "$RESULTS" | jq 'length')
OVERALL=$(echo "$SCAN" | jq -r '.scores.overall')
ERROR_PR=$(echo "$SCAN" | jq -r '.scores.by_severity_class.error_pass_rate // "n/a"')

if [ "$JSON" = true ]; then
  echo "$SCAN" | jq --arg n "$N" --argjson filtered "$RESULTS" '{
    overall_score: .scores.overall,
    error_pass_rate: (.scores.by_severity_class.error_pass_rate // null),
    total_findings: .summary.findings_total,
    triage: $filtered
  }'
else
  echo "archfit triage — score: $OVERALL, error_pass_rate: $ERROR_PR"
  echo ""
  if [ "$COUNT" = "0" ]; then
    if [ "$ALL_SEV" = true ]; then
      echo "No findings."
    else
      echo "No error/critical findings. Use -a to see all severities."
    fi
  else
    echo "$RESULTS" | jq -r '.[] |
      "  [\(.severity)] \(.rule_id) \(if .path == "" then "(repo)" else .path end)\n    \(.message)\n    fix: \(.remediation.summary)\n"'
  fi
fi
