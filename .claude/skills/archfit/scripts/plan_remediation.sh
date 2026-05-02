#!/bin/sh
# plan_remediation.sh — Propose a prioritized fix order for archfit findings.
#
# Priority: error/critical first, then auto-fixable, then by severity desc.
# For each finding, shows the remediation summary and whether archfit can
# auto-fix it.
#
# Usage:
#   plan_remediation.sh [path]
#
# Requires: archfit, jq
set -eu

TARGET="${1:-.}"

for cmd in archfit jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "ERROR: $cmd is required but not found" >&2
    exit 3
  fi
done

SCAN=$(archfit scan --json "$TARGET" 2>/dev/null) || true
TOTAL=$(echo "$SCAN" | jq '.summary.findings_total')

if [ "$TOTAL" = "0" ]; then
  echo "No findings — nothing to remediate."
  exit 0
fi

# Get the fix plan for auto-fixable findings.
FIX_PLAN=$(archfit fix --all --plan --json "$TARGET" 2>/dev/null) || true
FIXABLE_IDS=$(echo "$FIX_PLAN" | jq -r '[.plan.fixes[]?.rule_id] | unique | .[]' 2>/dev/null) || true

echo "archfit remediation plan — $TOTAL finding(s)"
echo ""

# Priority 1: error + critical.
BLOCKERS=$(echo "$SCAN" | jq '[.findings[] | select(.severity == "error" or .severity == "critical")]')
BLOCKER_COUNT=$(echo "$BLOCKERS" | jq 'length')

if [ "$BLOCKER_COUNT" != "0" ]; then
  echo "=== PRIORITY 1: Blocking (error/critical) — fix these first ==="
  echo ""
  echo "$BLOCKERS" | jq -r '.[] |
    "  [\(.severity)] \(.rule_id) \(if .path == "" then "(repo)" else .path end)\n    → \(.remediation.summary)\n"'
fi

# Priority 2: auto-fixable warn/info.
AUTO_FINDINGS=$(echo "$SCAN" | jq --argjson fixable "$(echo "$FIXABLE_IDS" | jq -R . | jq -s '.')" \
  '[.findings[] | select(.severity != "error" and .severity != "critical") | select(.rule_id as $rid | $fixable | index($rid))]')
AUTO_COUNT=$(echo "$AUTO_FINDINGS" | jq 'length')

if [ "$AUTO_COUNT" != "0" ]; then
  echo "=== PRIORITY 2: Auto-fixable (run \`archfit fix --all\`) ==="
  echo ""
  echo "$AUTO_FINDINGS" | jq -r '.[] |
    "  [\(.severity)] \(.rule_id) \(if .path == "" then "(repo)" else .path end)\n    → \(.remediation.summary)\n"'
fi

# Priority 3: remaining manual fixes.
MANUAL=$(echo "$SCAN" | jq --argjson fixable "$(echo "$FIXABLE_IDS" | jq -R . | jq -s '.')" \
  '[.findings[] | select(.severity != "error" and .severity != "critical") | select(.rule_id as $rid | $fixable | index($rid) | not)]')
MANUAL_COUNT=$(echo "$MANUAL" | jq 'length')

if [ "$MANUAL_COUNT" != "0" ]; then
  echo "=== PRIORITY 3: Manual remediation ==="
  echo ""
  echo "$MANUAL" | jq -r '.[] |
    "  [\(.severity)] \(.rule_id) \(if .path == "" then "(repo)" else .path end)\n    → \(.remediation.summary)\n    guide: \(.remediation.guide_ref // "n/a")\n"'
fi

echo "---"
echo "Summary: $BLOCKER_COUNT blocking, $AUTO_COUNT auto-fixable, $MANUAL_COUNT manual"
