# Fix Engine Internals

## Overview

The fix engine (`internal/fix/`) implements `archfit fix`, which closes the scan-fix-verify loop. It is the core of Pillar 1 (Agent-in-the-Loop Remediation).

## Package Layout

```
internal/fix/
├── engine.go        # Engine: orchestrates plan → apply → verify → rollback
├── engine_test.go
├── fixer.go         # Fixer interface and Change type
├── plan.go          # Plan and PlannedFix types
├── plan_test.go
├── log.go           # Fix audit logging
├── log_test.go
├── static/          # Static (deterministic) fixers
│   ├── templates/   # Embedded Go templates
│   ├── loc_p1_001.go
│   ├── loc_p1_002.go
│   ├── ver_p4_001.go
│   ├── mrd_p7_001.go
│   ├── mrd_p7_002.go
│   ├── mrd_p7_003.go
│   └── spc_p2_010.go
└── llmfix/          # LLM-assisted fixers
    ├── fixer.go
    └── prompts.go
```

## Two Classes of Fixers

### Static Fixers

Deterministic file creation/modification. Safe for `--all` without confirmation.

| Rule | Fixer | What it creates |
|---|---|---|
| P1.LOC.001 | `NewLocP1LOC001()` | `CLAUDE.md` from template |
| P1.LOC.002 | `NewLocP1LOC002()` | `AGENTS.md` in each slice missing one |
| P4.VER.001 | `NewVerP4VER001()` | `Makefile` with `test` target |
| P7.MRD.001 | `NewMrdP7MRD001()` | `docs/exit-codes.md` from template |
| P7.MRD.002 | `NewMrdP7MRD002()` | `CHANGELOG.md` from template |
| P7.MRD.003 | `NewMrdP7MRD003()` | `docs/adr/` directory with template ADR |
| P2.SPC.010 | `NewSpcP2SPC010()` | `schemas/output.schema.json` skeleton |

Templates live in `internal/fix/static/templates/` as embedded files (`//go:embed`). They use `text/template` with variables: project name, date, detected language.

### LLM-Assisted Fixers

Context-dependent content generation. Require `--with-llm`. Default to plan mode.

LLM fixers wrap a static fixer and enrich its output:
1. Get the static plan as baseline
2. Call LLM to enrich content (e.g., write a contextual CLAUDE.md based on repo structure)
3. Return enriched changes

**Safety**: if LLM call fails, fall back to static template. Never fail the fix.

## Engine Flow

```
Fix(ctx, FixInput) → FixResult
  │
  ├── 1. Filter findings to those with registered fixers
  ├── 2. For each finding:
  │      └── fixer.Plan(ctx, finding, facts) → []Change
  ├── 3. Build Plan (aggregate all changes)
  │
  ├── if DryRun or PlanOnly: return Plan, stop
  │
  ├── 4. Snapshot original file contents
  ├── 5. Apply changes to disk (write files)
  ├── 6. Re-scan via injected Scanner function
  ├── 7. Compare findings:
  │      ├── Finding gone + no new findings → Verified = true
  │      └── Finding persists OR new findings → rollback, Verified = false
  └── 8. Log to .archfit-fix-log.json
```

## Registration

All fixers are registered explicitly in `cmd/archfit/main.go`:

```go
func buildFixEngine() *fix.Engine {
    e := fix.NewEngine()
    e.Register(static.NewLocP1LOC001())
    e.Register(static.NewLocP1LOC002())
    // ... etc
    return e
}
```

No reflection, no auto-discovery.

## Adding a New Fixer

1. Create `internal/fix/static/<rule_id>.go` implementing `fix.Fixer`
2. Create template in `internal/fix/static/templates/` if needed
3. Register in `buildFixEngine()` in `main.go`
4. Add unit test:
   - Build a FactStore that triggers the finding
   - Call `Plan()`, assert proposed changes
   - Apply changes, re-run resolver, assert finding gone
5. Update `reference/remediation/<rule-id>.md` to mention `archfit fix <rule-id>`

## CLI Flags

```
archfit fix [rule-id] [path]
  --all          fix all fixable findings
  --dry-run      show what would change without applying
  --plan         show fix plan and exit
  --json         emit fix result as JSON
  --with-llm     enrich fix content with LLM
  --llm-backend  LLM provider
  --llm-budget   max LLM calls
  -C <dir>       change directory
```

## Safety Model

- Static fixers for `strong`-evidence rules: auto-apply with `--dry-run` option
- LLM-assisted fixers: always show plan first, require explicit `--with-llm`
- All fixes are atomic: rolled back if re-scan shows regressions
- Fix history logged to `.archfit-fix-log.json` for audit
- The `Scanner` function is injected — fix engine never imports from `internal/core/`
