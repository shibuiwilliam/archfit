# Fitness Contract

A fitness contract (`.archfit-contract.yaml`) declares what architectural fitness means for your repository. It is consumed by CI, the CLI, and coding agents.

## Quick Start

Create `.archfit-contract.yaml` at your repo root:

```json
{
  "version": 1,
  "hard_constraints": [
    {
      "principle": "P4",
      "min_score": 90,
      "scope": "**",
      "rationale": "All code must have fast verification"
    },
    {
      "rule": "P5.AGG.001",
      "max_findings": 0,
      "scope": "**",
      "rationale": "Zero tolerance for scattered auth code"
    }
  ],
  "soft_targets": [
    {
      "principle": "P1",
      "target_score": 95,
      "deadline": "2026-Q3"
    }
  ],
  "area_budgets": [
    {
      "path": "services/billing/**",
      "max_findings": 2,
      "max_new_findings_per_pr": 0,
      "principles": ["P5", "P6"],
      "owner": "billing-team"
    }
  ],
  "agent_directives": [
    {
      "when": "finding.severity >= error",
      "action": "stop and ask the user before proceeding"
    },
    {
      "when": "area_budget.remaining == 0",
      "action": "do not introduce new findings in this area"
    }
  ]
}
```

## Concepts

### Hard Constraints

Requirements that **must** be satisfied. `archfit contract check` exits 1 when any hard constraint is violated.

Two constraint types:

- **Score-based**: `min_score` on a principle or `overall`
- **Finding-based**: `max_findings` for a specific rule within a scope

### Soft Targets

Aspirational goals. Not enforced, but tracked. `archfit contract check` exits 5 (advisory) when soft targets are missed but all hard constraints are met.

### Area Budgets

SRE-style finding budgets per path area. Like error budgets: a fixed number of findings is tolerable; exceeding the budget signals that the area needs attention.

Fields:

- `path`: glob pattern (e.g., `services/billing/**`)
- `max_findings`: maximum tolerable findings
- `principles`: optional filter (only count findings for these principles)
- `owner`: team responsible for this area

### Agent Directives

Machine-readable instructions that coding agents follow when working on the repository. The agent reads the contract before starting work and adjusts its behavior accordingly.

## CLI Commands

```bash
archfit contract check [path]    # exit 0 (pass), 1 (hard violation), 5 (soft miss)
archfit contract status [path]   # dashboard view
archfit contract init [path]     # scaffold from current scan results
```

## CI Integration

```yaml
- name: Contract check
  run: archfit contract check .
  # Exits 1 on hard constraint violation (blocks merge)
  # Exits 5 on soft target miss (advisory only)
```

## Schema

The contract format is defined by [`schemas/contract.schema.json`](https://github.com/shibuiwilliam/archfit/blob/main/schemas/contract.schema.json).

## Relationship to Config and Policy

| File | Scope | Enforced by |
|------|-------|-------------|
| `.archfit.yaml` | Which rules and packs to run | CLI scan pipeline |
| `.archfit-contract.yaml` | What fitness level the repo must maintain | `archfit contract check` |
| `.archfit-policy.yaml` | Organization-level governance | `archfit scan --policy` |

## See Also

- [Configuration](configuration.md) — `.archfit.yaml` reference
- [CI/CD Integration](ci-cd.md) — workflows using contract check
- [Getting Started](getting-started.md) — initial setup
