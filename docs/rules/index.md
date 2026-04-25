# Rules Overview

archfit evaluates repositories against seven architectural principles. Each principle expands into concrete rules with deterministic detection logic.

## Rule ID Format

```
P<principle>.<dimension>.<number>
```

- **Principle**: P1-P7 (one of the seven principles)
- **Dimension**: 3 uppercase letters (LOC, SPC, EXP, VER, AGG, REV, MRD)
- **Number**: 3 digits, sequential

## Current Rules

### `core` pack — applies to every repository

| Rule | Principle | Severity | What it checks |
|---|---|---|---|
| [P1.LOC.001](P1.LOC.001.md) | P1 Locality | warn | `CLAUDE.md` or `AGENTS.md` exists at repo root |
| [P1.LOC.002](P1.LOC.002.md) | P1 Locality | warn | Vertical-slice directories carry `AGENTS.md` |
| [P3.EXP.001](P3.EXP.001.md) | P3 Explicitness | warn | `.env` files → `.env.example` must exist |
| [P4.VER.001](P4.VER.001.md) | P4 Verifiability | warn | Fast verification entrypoint exists |
| [P5.AGG.001](P5.AGG.001.md) | P5 Aggregation | warn | Security-sensitive files concentrated |
| [P6.REV.001](P6.REV.001.md) | P6 Reversibility | warn | Deployment artifacts → rollback docs exist |
| [P7.MRD.001](P7.MRD.001.md) | P7 Machine-readability | warn | CLI repos document exit codes |

### `agent-tool` pack — opt-in, for agent-consumed tools

| Rule | Principle | Severity | What it checks |
|---|---|---|---|
| [P2.SPC.010](P2.SPC.010.md) | P2 Spec-first | warn | Versioned JSON output schema with `$id` |
| [P7.MRD.002](P7.MRD.002.md) | P7 Machine-readability | warn | `CHANGELOG.md` at repo root |
| [P7.MRD.003](P7.MRD.003.md) | P7 Machine-readability | warn | `cmd/` repos have `docs/adr/` directory |

## Rule Qualities

Every finding carries:

- **Severity**: `info` / `warn` / `error` / `critical` — how bad is it if true?
- **Evidence strength**: `strong` / `medium` / `weak` / `sampled` — how reliable is the detection?
- **Confidence**: 0.0-1.0 numeric score
- **Remediation**: summary + guide reference + auto-fixable flag

All current rules have `strong` evidence and `experimental` stability.

## Constraint

`error` severity requires `strong` evidence. This is enforced at the type level — a rule with `weak` evidence and `error` severity will fail validation. **False positives are treated as bugs.**

## Suppressing Rules

Add to `.archfit.yaml`:

```json
{
  "ignore": [
    {
      "rule": "P1.LOC.002",
      "paths": ["packs/legacy-*"],
      "reason": "Legacy slices on a documented deletion path",
      "expires": "2026-12-31"
    }
  ]
}
```

Every `ignore` requires a `reason` and `expires` date. Expired suppressions surface as warnings.

## Planned Rules

Additional packs (`web-saas`, `iac`, `mobile`, `data-event`) are planned for future releases. See [Implementation Plan](../development/implementation-plan.md).
