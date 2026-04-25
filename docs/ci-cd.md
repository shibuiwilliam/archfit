# CI/CD Integration

archfit produces structured output designed for CI consumption.

## SARIF for GitHub Code Scanning

```yaml
- name: Build archfit
  run: go install github.com/shibuiwilliam/archfit/cmd/archfit@latest

- name: Scan
  run: archfit scan --format=sarif . > archfit.sarif

- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: archfit.sarif
```

## PR Gate — Fail on New Findings Only

```yaml
- name: Baseline (main)
  run: |
    git stash
    git checkout origin/main
    archfit scan --json . > baseline.json
    git checkout -

- name: Current scan (PR)
  run: archfit scan --json . > current.json

- name: Diff
  run: archfit diff baseline.json current.json
  # exits 1 when new findings appear
```

## Auto-Fix in CI

```yaml
- name: Fix and commit
  run: |
    archfit fix --all .
    if ! git diff --quiet; then
      git commit -am "chore: archfit auto-fix"
      git push
    fi
```

## LLM-Enriched PR Comment

```yaml
- name: Enriched scan
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  run: archfit scan --with-llm --format=md . > report.md

- name: Comment on PR
  uses: marocchino/sticky-pull-request-comment@v2
  with:
    path: report.md
```

## Trend Tracking

Archive scans in CI and track fitness over time:

```yaml
- name: Archive scan
  run: |
    mkdir -p .archfit-history
    archfit scan --json . > .archfit-history/$(date +%Y-%m-%d)-$(git rev-parse --short HEAD).json

- name: Show trend
  run: archfit trend --history .archfit-history/
```

## Cross-Repo Comparison

```bash
archfit compare repo-a.json repo-b.json repo-c.json --format=md
```

## Organization Policy

```bash
archfit scan --policy policy.json .
```

Policy violations are reported to stderr (advisory, do not change exit code).

## Exit Codes

| Code | CI Interpretation |
|---|---|
| `0` | Pass |
| `1` | Findings at or above `--fail-on` threshold |
| `2` | Usage error |
| `3` | Runtime error |
| `4` | Configuration error |

Use `--fail-on=error` (default) so `warn` findings don't block PRs.

See [Exit Codes](exit-codes.md) for the full contract.
