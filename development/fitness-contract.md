# Fitness Contract as Code

> Element 2 of the three strategic elements. Implemented first due to highest immediate value.

## Overview

A fitness contract (`.archfit-contract.yaml`) is a machine-executable declaration of what fitness means for a specific repository. It is consumed by CI, the CLI, and coding agents — not as a report to read, but as a constraint to satisfy.

The contract shifts archfit from "scan and report" to "declare, enforce, and adapt."

## Key Concepts

### Hard Constraints

Requirements that **must** be satisfied. CI exits 1 when any hard constraint is violated.

```json
{
  "principle": "P1",
  "min_score": 80,
  "scope": "**",
  "rationale": "All code must have strong locality"
}
```

Or finding-count-based:

```json
{
  "rule": "P5.AGG.001",
  "max_findings": 0,
  "scope": "**",
  "rationale": "Zero tolerance for scattered auth code"
}
```

### Soft Targets

Aspirational goals the team is working toward. Not enforced, but tracked.

```json
{
  "principle": "P1",
  "target_score": 95,
  "deadline": "2026-Q3"
}
```

### Area Budgets

SRE-style finding budgets per path. Like error budgets: when you hit zero, new violations block the PR.

```json
{
  "path": "services/billing/**",
  "max_findings": 2,
  "max_new_findings_per_pr": 0,
  "principles": ["P5", "P6"],
  "owner": "billing-team"
}
```

### Agent Directives

Machine-readable instructions for coding agents:

```json
{
  "when": "finding.severity >= error",
  "action": "stop and ask the user before proceeding"
}
```

## Package Structure

```
internal/contract/
├── contract.go         # Contract type, loading, validation
├── contract_test.go    # ≥8 table-test scenarios
├── check.go            # Evaluate scan results against contract
└── (future) budget.go  # PR-level budget tracking
schemas/
└── contract.schema.json
```

## Implementation Status

| Step | Description | Status |
|------|-------------|--------|
| 2.1 | Contract schema and types | Done |
| 2.2 | Contract checking logic | Done |
| 2.3 | CLI wiring (`archfit contract check/status/init`) | Not started |
| 2.4 | Agent directive support in skill | Not started |

## Architecture Decisions

- Contract types live in `internal/contract/`, NOT in `internal/model/`. No ADR required.
- `Check()` is a pure function: receives pre-computed `score.Scores` and `[]model.Finding`. No I/O.
- Loading follows the same JSON-in-YAML pattern as `internal/config/`.
- Scope matching uses `filepath.Match` from stdlib (no new dependencies).
- The contract package does NOT import from `internal/adapter/` or `internal/collector/`.

## CLI Commands (Step 2.3)

```bash
archfit contract check [path]    # check against contract (exit 0/1/5)
archfit contract status [path]   # show contract dashboard
archfit contract init [path]     # scaffold .archfit-contract.yaml
```

New exit code `5`: soft target missed, no hard violation. Requires ADR `docs/adr/0008-contract-exit-codes.md`.

## Agent Skill Integration (Step 2.4)

When `.archfit-contract.yaml` exists, the Claude Code skill:

1. Loads the contract before starting work
2. Checks which area budgets are affected by planned changes
3. Follows agent directives (e.g., "stop and ask" on severity >= error)
4. Verifies no hard constraint is violated after changes
5. Reports contract delta in commit message

## Adding a New Constraint Type

1. Add a new field to the `Constraint` struct in `contract.go`
2. Add validation logic in `Validate()`
3. Add checking logic in `checkHardConstraint()` in `check.go`
4. Add table test cases
5. Update `schemas/contract.schema.json`

## Related Files

- `internal/policy/` — organization-level policy enforcement (complementary, not overlapping)
- `schemas/contract.schema.json` — JSON Schema for the contract format
- `development/implementation-plan.md` — overall implementation sequencing
