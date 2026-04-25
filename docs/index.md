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

10 rules across 2 packs, all with `strong` evidence and `experimental` stability.

- **`core` pack** (7 rules): P1.LOC.001, P1.LOC.002, P3.EXP.001, P4.VER.001, P5.AGG.001, P6.REV.001, P7.MRD.001
- **`agent-tool` pack** (3 rules): P2.SPC.010, P7.MRD.002, P7.MRD.003

See [Rules Overview](rules/index.md) for details on each rule.

## Key Features

- **Deterministic**: base scan makes zero network calls, produces byte-identical output
- **LLM-enriched** (opt-in): `--with-llm` adds contextual explanations via Claude, OpenAI, or Gemini
- **Auto-fix**: `archfit fix` closes the scan-fix-verify loop with automatic rollback on failure
- **CI-ready**: SARIF, JSON, and Markdown output; `archfit diff` for PR gates
- **Self-consistent**: archfit passes its own scan at score 100.0
