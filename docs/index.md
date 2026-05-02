# archfit

> **Architecture fitness evaluator for the coding-agent era.**

archfit scans a repository and evaluates how well it is shaped for coding agents to work on safely and quickly. It measures seven architectural properties that determine whether an agent can change the system without breaking it.

## The Seven Principles

| | Principle | The question it asks |
|---|---|---|
| **P1** | Locality | Can a change be understood from a narrow slice of the repo? |
| **P2** | Spec-first | Are contracts schemas and types, not prose? |
| **P3** | Shallow explicitness | Is behavior visible without chasing reflection or deep indirection? |
| **P4** | Verifiability | Can correctness be proven locally in seconds? |
| **P5** | Aggregation of danger | Are auth, secrets, and migrations concentrated and guarded? |
| **P6** | Reversibility | Can every change be rolled back cheaply? |
| **P7** | Machine-readability | Are errors, ADRs, and CLIs readable by machines? |

## Quick Start

```bash
# Build from source (Go 1.24+)
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit && make build

# Scan a repository
./bin/archfit scan /path/to/your/repo

# Auto-fix findings
./bin/archfit fix --all /path/to/your/repo
```

See [Getting Started](getting-started.md) for detailed setup instructions.

## What archfit produces

- A **score per principle** (P1-P7) and an overall score (0-100)
- **Findings** with severity, evidence strength, confidence, and remediation
- **Metrics**: context span, verification latency, blast radius, and more
- **Auto-fixes** for rules with deterministic remediation
- **SARIF output** for GitHub Code Scanning integration

## Current Rule Set

27 rules across 2 packs. Most rules are `stability: stable` (frozen per ADR 0012). Three rules remain `experimental` per ADR 0014: P1.LOC.003, P1.LOC.004, P5.AGG.001. Output schema version: `1.1.0`.

- **`core` pack** (24 rules): locality (P1.LOC.001-004), spec-first (P2.SPC.001), explicitness (P3.EXP.001), verifiability (P4.VER.001-003), aggregation (P5.AGG.001-004), reversibility (P6.REV.001-002), machine-readability (P7.MRD.001), and additional rules across all seven principles. P5.AGG.004 is the first rule with `severity: error`.
- **`agent-tool` pack** (3 rules): spec-first (P2.SPC.010), machine-readability (P7.MRD.002, P7.MRD.003)

See [Rules Overview](rules/index.md) for the full catalog.

## Key Features

- **Deterministic**: base scan makes zero network calls, produces byte-identical output
- **LLM-enriched** (opt-in): `--with-llm` adds contextual explanations via Claude, OpenAI, or Gemini
- **Auto-fix**: `archfit fix` closes the scan-fix-verify loop with automatic rollback on failure
- **Fitness contracts**: declare hard constraints, area budgets, and agent directives in `.archfit-contract.yaml`
- **CI-ready**: SARIF, JSON, and Markdown output; `archfit diff` for PR gates
- **Self-consistent**: archfit passes its own scan at score 100.0

## Documentation

| Page | Description |
|------|-------------|
| [Installation](installation.md) | Install from source, binaries, `go install`, or Docker |
| [Getting Started](getting-started.md) | First scan, common commands |
| [Configuration](configuration.md) | `.archfit.yaml` reference |
| [Fitness Contract](contract.md) | `.archfit-contract.yaml` — hard constraints, area budgets, agent directives |
| [Rules](rules/index.md) | Rule catalog with detection patterns and remediation |
| [Agent Skill](agent-skill.md) | Claude Code skill — scan, remediate, and verify with progressive disclosure |
| [Auto-Fix](fix.md) | `archfit fix` — scan-fix-verify loop |
| [LLM Integration](llm.md) | `--with-llm` — Claude, OpenAI, Gemini enrichment |
| [CI/CD Integration](ci-cd.md) | SARIF, PR gates, trend tracking |
| [Exit Codes](exit-codes.md) | CLI exit code contract |
| [Dependencies](dependencies.md) | Runtime and build dependencies |
| [Deployment](deployment.md) | Release process and distribution |
