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

### `core` pack — applies to every repository (11 rules)

| Rule | Principle | Severity | Evidence | What it checks |
|---|---|---|---|---|
| [P1.LOC.001](P1.LOC.001.md) | P1 Locality | warn | strong | `CLAUDE.md` or `AGENTS.md` exists at repo root |
| [P1.LOC.002](P1.LOC.002.md) | P1 Locality | warn | strong | Vertical-slice directories carry `AGENTS.md` |
| [P1.LOC.003](P1.LOC.003.md) | P1 Locality | info | medium | Dependency coupling is bounded (max reach ≤10) |
| [P1.LOC.004](P1.LOC.004.md) | P1 Locality | info | sampled | Commits touch a bounded number of files (≤8) |
| [P3.EXP.001](P3.EXP.001.md) | P3 Explicitness | warn | strong | Config documented (.env, Spring profiles, tfvars, Rails) |
| [P4.VER.001](P4.VER.001.md) | P4 Verifiability | warn | strong | Verification entrypoint exists (Makefile, pom.xml, etc.) |
| [P4.VER.002](P4.VER.002.md) | P4 Verifiability | info | medium | ≥70% of source directories have test files |
| [P4.VER.003](P4.VER.003.md) | P4 Verifiability | info | strong | CI configuration present (GitHub Actions, GitLab, etc.) |
| [P5.AGG.001](P5.AGG.001.md) | P5 Aggregation | warn | strong | Security-sensitive files concentrated (≤2 top-level dirs) |
| [P6.REV.001](P6.REV.001.md) | P6 Reversibility | warn | strong | Deployment artifacts → rollback docs exist |
| [P7.MRD.001](P7.MRD.001.md) | P7 Machine-readability | warn | strong | CLI repos document exit codes |

### `agent-tool` pack — opt-in, for agent-consumed tools (3 rules)

| Rule | Principle | Severity | Evidence | What it checks |
|---|---|---|---|---|
| [P2.SPC.010](P2.SPC.010.md) | P2 Spec-first | warn | strong | Spec-first artifacts exist (JSON Schema, OpenAPI, Protobuf, GraphQL) |
| [P7.MRD.002](P7.MRD.002.md) | P7 Machine-readability | warn | strong | `CHANGELOG.md` at repo root |
| [P7.MRD.003](P7.MRD.003.md) | P7 Machine-readability | warn | strong | CLI repos have `docs/adr/` directory |

## Rule Qualities

Every finding carries:

- **Severity**: `info` / `warn` / `error` / `critical` — how bad is it if true?
- **Evidence strength**: `strong` / `medium` / `weak` / `sampled` — how reliable is the detection?
- **Confidence**: 0.0-1.0 numeric score
- **Remediation**: summary + guide reference + auto-fixable flag

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
