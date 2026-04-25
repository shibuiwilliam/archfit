# CI/CD Integration

## Output Formats for CI

| Format | Flag | Use Case |
|---|---|---|
| Terminal | `--format=terminal` (default) | Human review |
| JSON | `--format=json` or `--json` | Machine consumption, `archfit diff`, trend tracking |
| Markdown | `--format=md` | PR comments, reports |
| SARIF 2.1.0 | `--format=sarif` | GitHub Code Scanning |

## GitHub Workflows

### Basic SARIF Integration

```yaml
name: archfit
on: [push, pull_request]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build archfit
        run: go install github.com/shibuiwilliam/archfit/cmd/archfit@latest

      - name: Scan
        run: archfit scan --format=sarif . > archfit.sarif

      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: archfit.sarif
```

### PR Gate (Fail on New Findings Only)

```yaml
name: archfit-gate
on: pull_request

jobs:
  gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # needed for baseline checkout

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build archfit
        run: go install github.com/shibuiwilliam/archfit/cmd/archfit@latest

      - name: Baseline scan (main)
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

### Auto-Fix in CI

```yaml
- name: Auto-fix
  run: |
    archfit fix --all .
    if ! git diff --quiet; then
      git config user.name "archfit-bot"
      git config user.email "archfit@noreply"
      git commit -am "chore: archfit auto-fix"
      git push
    fi
```

### LLM-Enriched PR Comment

```yaml
- name: Enriched scan
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  run: |
    archfit scan --with-llm --format=md . > report.md

- name: Comment on PR
  uses: marocchino/sticky-pull-request-comment@v2
  with:
    path: report.md
```

## GitHub Action (`.github/archfit-action/`)

The project includes a scaffolded composite GitHub Action. When complete, usage will be:

```yaml
- uses: shibuiwilliam/archfit-action@v1
  with:
    fail-on: error
    baseline-branch: main
    comment: true
    sarif: true
    version: latest
```

### Action Behavior

1. Install archfit binary
2. Scan PR branch (`archfit scan --json .`)
3. Scan base branch (cached)
4. Run `archfit diff` between them
5. Post PR comment with score delta, new findings, fixed findings
6. Optionally upload SARIF for Code Scanning
7. Exit 1 if new findings at or above `--fail-on` threshold

### Edge Cases

- **No baseline**: first scan, skip diff, show full results
- **No changes**: skip comment
- **Fork PRs**: no comment permission — log warning, skip comment

## Trend Tracking

### Archiving Scans

```bash
# In CI, after each scan:
mkdir -p .archfit-history
archfit scan --json . > .archfit-history/$(date +%Y-%m-%d)-$(git rev-parse --short HEAD).json
```

### Viewing Trends

```bash
archfit trend --history .archfit-history/
archfit trend --history .archfit-history/ --since 2026-01-01
archfit trend --history .archfit-history/ --format=csv
archfit trend --history .archfit-history/ --format=json
```

### Exporting to Dashboards

```bash
# Grafana / Datadog via CSV
archfit trend --format=csv > /tmp/archfit-trend.csv
# Import into your observability tool
```

## Cross-Repo Comparison

```bash
# Scan multiple repos
archfit scan --json ./repo-a > repo-a.json
archfit scan --json ./repo-b > repo-b.json
archfit scan --json ./repo-c > repo-c.json

# Compare
archfit compare repo-a.json repo-b.json repo-c.json
archfit compare --format=md repo-a.json repo-b.json repo-c.json
archfit compare --sort=name repo-a.json repo-b.json repo-c.json
```

## Organization Policy Enforcement

```bash
archfit scan --policy policy.json .
```

Policy file defines minimum scores, required packs, required rules, and per-repo exemptions. Policy violations are reported to stderr but do not change the exit code (advisory).

See `development/api-reference.md` for the `Policy` type definition.

## Exit Codes in CI

| Code | CI Interpretation |
|---|---|
| `0` | Pass (or all findings below threshold) |
| `1` | Fail: findings at or above `--fail-on` threshold |
| `2` | Build/config issue: bad flags or missing args |
| `3` | Infrastructure issue: scan couldn't complete |
| `4` | Config issue: `.archfit.yaml` invalid |

**Best practice**: use `--fail-on=error` (default) so `warn` findings don't block PRs.
