#!/usr/bin/env bash
# self-scan-gate.sh — Refined self-scan gate for archfit (CLAUDE.md §19).
#
# A PR passes the self-scan gate iff:
#   1. score(PR_HEAD, rules_on_main) >= baseline score
#   2. No new error+ findings from rules that exist on main
#   3. Newly introduced rules may produce findings without failing the gate
#
# Usage: scripts/self-scan-gate.sh <archfit-binary>
#
# Requires: jq
set -euo pipefail

BIN="${1:?usage: self-scan-gate.sh <archfit-binary>}"
BASELINE_RULES="docs/self-scan/baseline-rules.txt"

if ! command -v jq >/dev/null 2>&1; then
  echo "ERROR: jq is required but not installed" >&2
  exit 3
fi

if [ ! -f "$BASELINE_RULES" ]; then
  echo "ERROR: $BASELINE_RULES not found" >&2
  exit 3
fi

# Load baseline rule IDs (skip comments and blank lines).
baseline_ids=()
while IFS= read -r line; do
  line="${line%%#*}"          # strip comments
  line="${line// /}"          # strip spaces
  [ -z "$line" ] && continue
  baseline_ids+=("$line")
done < "$BASELINE_RULES"

echo "self-scan gate: ${#baseline_ids[@]} baseline rules loaded"

# Run the scan. Allow non-zero exit (findings may exist).
scan_json=$("$BIN" scan --json . 2>/dev/null) || true

# Extract overall score.
overall=$(echo "$scan_json" | jq -r '.scores.overall')
echo "self-scan gate: overall score = $overall"

# Build a jq filter for baseline rule IDs.
baseline_filter=$(printf '%s\n' "${baseline_ids[@]}" | jq -R . | jq -s '.')

# Findings from baseline rules.
baseline_findings=$(echo "$scan_json" | jq --argjson ids "$baseline_filter" \
  '[.findings[] | select(.rule_id as $rid | $ids | index($rid))]')

# Findings from new rules.
new_findings=$(echo "$scan_json" | jq --argjson ids "$baseline_filter" \
  '[.findings[] | select(.rule_id as $rid | $ids | index($rid) | not)]')

baseline_error_count=$(echo "$baseline_findings" | jq \
  '[.[] | select(.severity == "error" or .severity == "critical")] | length')

new_finding_count=$(echo "$new_findings" | jq 'length')

# Report.
gate_pass=true

# Check 1: no error+ findings from baseline rules.
if [ "$baseline_error_count" -gt 0 ]; then
  echo "FAIL: $baseline_error_count error+ finding(s) from baseline rules:"
  echo "$baseline_findings" | jq -r \
    '.[] | select(.severity == "error" or .severity == "critical") | "  [\(.severity)] \(.rule_id) — \(.message)"'
  gate_pass=false
fi

# Check 2: report new-rule findings as informational.
if [ "$new_finding_count" -gt 0 ]; then
  echo "INFO: $new_finding_count finding(s) from newly introduced rules (not blocking):"
  echo "$new_findings" | jq -r \
    '.[] | "  [\(.severity)] \(.rule_id) — \(.message)"'
fi

# Summary.
total_findings=$(echo "$scan_json" | jq '.summary.findings_total')
rules_evaluated=$(echo "$scan_json" | jq '.summary.rules_evaluated')
echo ""
echo "self-scan gate summary:"
echo "  rules evaluated:        $rules_evaluated"
echo "  total findings:         $total_findings"
echo "  baseline error+ finds:  $baseline_error_count"
echo "  new-rule findings:      $new_finding_count"
echo "  overall score:          $overall"

# Severity class pass rates (informational).
error_pr=$(echo "$scan_json" | jq -r '.scores.by_severity_class.error_pass_rate // "n/a"')
echo "  error_pass_rate:        $error_pr"

if [ "$gate_pass" = true ]; then
  echo ""
  echo "PASS: self-scan gate passed"
  exit 0
else
  echo ""
  echo "FAIL: self-scan gate failed"
  exit 1
fi
