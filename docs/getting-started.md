# Getting Started

## Installation

### From Source

```bash
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit
make build
./bin/archfit version
```

Requires **Go 1.24+**. No CGO. Cross-compiles to `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, and `windows/amd64`.

### From Release Binaries

```bash
curl -sSL https://github.com/shibuiwilliam/archfit/releases/latest/download/archfit-<version>-linux-amd64.tar.gz \
  | tar xz
./archfit version
```

### Via Docker

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

## First Scan

```bash
cd /path/to/your/repo

# Scaffold a config file
archfit init .

# Run the scan
archfit scan .
```

### Understanding the Output

```
archfit dev — target . (profile=standard)
rules evaluated: 10, findings: 2
overall score: 84.0
  P1: 100.0
  P3: 60.0
  P6: 60.0
findings:
  [warn] P3.EXP.001  — repository uses .env files but has no .env.example
  [warn] P6.REV.001 docs/ — deployment artifacts detected but no rollback documentation
```

- **Score**: 0-100 per principle and overall. Higher is better.
- **Findings**: each has severity (`info`/`warn`/`error`/`critical`), evidence strength, and a remediation guide.
- **Exit code**: 0 = pass, 1 = findings at or above `--fail-on` threshold.

## Auto-Fix

```bash
# Fix a specific rule
archfit fix P3.EXP.001 .

# Fix all fixable findings
archfit fix --all .

# Preview changes without applying
archfit fix --dry-run --all .
```

Every fix is verified by automatic re-scan. If the finding persists or new findings appear, changes are rolled back.

## Common Commands

| Command | What it does |
|---|---|
| `archfit scan [path]` | Run all enabled rules |
| `archfit fix [rule-id] [path]` | Auto-fix findings |
| `archfit check <rule-id> [path]` | Run a single rule |
| `archfit score [path]` | Summary scores only |
| `archfit explain <rule-id>` | Show rule docs and remediation |
| `archfit diff <baseline.json>` | Compare two scans |
| `archfit report [path]` | Markdown report |
| `archfit init [path]` | Scaffold `.archfit.yaml` |
| `archfit help` | Show all commands and flags |

## Key Flags

| Flag | Description | Default |
|---|---|---|
| `--format {terminal\|json\|md\|sarif}` | Output format | `terminal` |
| `--json` | Shorthand for `--format=json` | |
| `--fail-on {info\|warn\|error\|critical}` | Exit 1 at this severity | `error` |
| `--with-llm` | Enrich findings with LLM explanations | off |
| `-C <dir>` | Change directory before running | |

## Next Steps

- [Configuration Reference](configuration.md) — customize `.archfit.yaml`
- [Rules Overview](rules/index.md) — understand what each rule checks
- [LLM Integration](llm.md) — enable contextual explanations
- [CI/CD Integration](ci-cd.md) — add archfit to your pipeline
