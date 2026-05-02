#!/bin/sh
# apply_safe_fixes.sh — Apply auto-fixable findings via `archfit fix`.
#
# Shows a dry-run preview first, then applies if confirmed (or if -y is set).
#
# Usage:
#   apply_safe_fixes.sh [options] [path]
#
# Options:
#   -y       Skip confirmation, apply immediately
#   -n       Dry-run only, do not apply
#   -h       Help
#
# Requires: archfit
set -eu

AUTO_APPLY=false
DRY_ONLY=false
TARGET="."

while [ $# -gt 0 ]; do
  case "$1" in
    -y|--yes) AUTO_APPLY=true; shift ;;
    -n|--dry-run) DRY_ONLY=true; shift ;;
    -h|--help) sed -n '2,14p' "$0"; exit 0 ;;
    -*) echo "unknown flag: $1" >&2; exit 2 ;;
    *)  TARGET="$1"; shift ;;
  esac
done

if ! command -v archfit >/dev/null 2>&1; then
  echo "ERROR: archfit is required but not found" >&2
  exit 3
fi

echo "=== Dry-run: checking what archfit can auto-fix ==="
echo ""

archfit fix --all --dry-run "$TARGET" 2>&1
DRY_EXIT=$?

if [ "$DRY_EXIT" != "0" ]; then
  echo ""
  echo "No auto-fixable findings, or dry-run encountered an error."
  exit 0
fi

if [ "$DRY_ONLY" = true ]; then
  echo ""
  echo "(dry-run only — use without -n to apply)"
  exit 0
fi

if [ "$AUTO_APPLY" = false ]; then
  printf "\nApply these fixes? [y/N] "
  read -r REPLY
  case "$REPLY" in
    y|Y|yes|YES) ;;
    *) echo "Aborted."; exit 0 ;;
  esac
fi

echo ""
echo "=== Applying fixes ==="
archfit fix --all "$TARGET"
FIX_EXIT=$?

if [ "$FIX_EXIT" = "0" ]; then
  echo ""
  echo "Fixes applied. Run \`archfit scan\` to verify."
else
  echo ""
  echo "Some fixes failed (exit $FIX_EXIT). Check output above."
  exit "$FIX_EXIT"
fi
