#!/usr/bin/env bash
set -euo pipefail

# entrypoint.sh — composite action helper for archfit.
# Two subcommands: install and scan.

cmd_install() {
  local version="${1:-latest}"

  if command -v archfit &>/dev/null; then
    echo "archfit already on PATH: $(archfit version)"
    return 0
  fi

  echo "Installing archfit@${version} ..."
  if [ "$version" = "latest" ]; then
    go install github.com/shibuiwilliam/archfit/cmd/archfit@latest
  else
    go install "github.com/shibuiwilliam/archfit/cmd/archfit@${version}"
  fi
  echo "Installed: $(archfit version)"
}

cmd_scan() {
  local fail_on="${1:-error}"
  local baseline_branch="${2:-main}"
  local comment="${3:-true}"
  local sarif="${4:-false}"

  # --- 1. Run current scan ---
  echo "Running archfit scan ..."
  archfit scan --json . > /tmp/archfit-current.json || true

  # --- 2. Baseline diff (PR only) ---
  local is_pr="false"
  local pr_number=""
  if [ "${GITHUB_EVENT_NAME:-}" = "pull_request" ]; then
    is_pr="true"
    pr_number=$(jq -r '.pull_request.number // empty' "${GITHUB_EVENT_PATH:-/dev/null}" 2>/dev/null || true)
  fi

  if [ "$is_pr" = "true" ] && [ -n "$baseline_branch" ]; then
    echo "Generating baseline from ${baseline_branch} ..."
    local stashed="false"
    if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null; then
      git stash --include-untracked 2>/dev/null && stashed="true"
    fi

    if git rev-parse --verify "origin/${baseline_branch}" &>/dev/null; then
      git checkout "origin/${baseline_branch}" --detach 2>/dev/null
      archfit scan --json . > /tmp/archfit-baseline.json 2>/dev/null || true
      git checkout - 2>/dev/null
      if [ "$stashed" = "true" ]; then
        git stash pop 2>/dev/null || true
      fi

      echo "Computing diff ..."
      archfit diff /tmp/archfit-baseline.json /tmp/archfit-current.json --json > /tmp/archfit-diff.json 2>/dev/null || true
    else
      echo "Warning: baseline branch origin/${baseline_branch} not found; skipping diff."
      if [ "$stashed" = "true" ]; then
        git stash pop 2>/dev/null || true
      fi
    fi
  fi

  # --- 3. PR comment ---
  if [ "$comment" = "true" ] && [ "$is_pr" = "true" ] && [ -n "$pr_number" ]; then
    if ! command -v gh &>/dev/null; then
      echo "Warning: gh CLI not found; skipping PR comment."
    elif [ -z "${GITHUB_TOKEN:-}" ]; then
      echo "Warning: GITHUB_TOKEN not set; skipping PR comment."
    else
      local body=""
      if [ -f /tmp/archfit-diff.json ]; then
        local new_count fixed_count overall
        new_count=$(jq -r '.summary.new // 0' /tmp/archfit-diff.json 2>/dev/null || echo "0")
        fixed_count=$(jq -r '.summary.fixed // 0' /tmp/archfit-diff.json 2>/dev/null || echo "0")
        overall=$(jq -r '.scores.overall // "N/A"' /tmp/archfit-current.json 2>/dev/null || echo "N/A")
        body="## archfit scan results

**Overall score:** ${overall}

| Metric | Count |
|--------|-------|
| New findings | ${new_count} |
| Fixed findings | ${fixed_count} |

<details>
<summary>Full scan output</summary>

\`\`\`json
$(cat /tmp/archfit-current.json)
\`\`\`
</details>"
      else
        local overall
        overall=$(jq -r '.scores.overall // "N/A"' /tmp/archfit-current.json 2>/dev/null || echo "N/A")
        body="## archfit scan results

**Overall score:** ${overall}

<details>
<summary>Full scan output</summary>

\`\`\`json
$(cat /tmp/archfit-current.json)
\`\`\`
</details>"
      fi
      echo "$body" | gh pr comment "$pr_number" --body-file - 2>/dev/null || echo "Warning: failed to post PR comment (fork PR or insufficient permissions)."
    fi
  fi

  # --- 4. SARIF output ---
  if [ "$sarif" = "true" ]; then
    echo "Generating SARIF output ..."
    archfit scan --format=sarif . > archfit.sarif || true
  fi

  # --- 5. Threshold check ---
  echo "Checking threshold (fail-on=${fail_on}) ..."
  archfit scan --fail-on="$fail_on" . > /dev/null 2>&1
  local rc=$?
  if [ $rc -eq 1 ]; then
    echo "Findings at or above --fail-on=${fail_on} detected."
    exit 1
  elif [ $rc -ne 0 ]; then
    echo "Warning: archfit exited with code ${rc}."
    exit $rc
  fi

  echo "archfit scan passed."
}

# --- Dispatch ---
case "${1:-}" in
  install)
    shift
    cmd_install "$@"
    ;;
  scan)
    shift
    cmd_scan "$@"
    ;;
  *)
    echo "Usage: entrypoint.sh {install|scan} [args...]" >&2
    exit 2
    ;;
esac
