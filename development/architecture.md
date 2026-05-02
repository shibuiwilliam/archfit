# Architecture Deep-Dive

This document describes archfit's internal architecture in detail sufficient for Claude Code or a human contributor to make informed changes.

## System Overview

archfit is a single Go binary that scans a repository and evaluates it against seven architectural principles. The architecture enforces a strict separation between data gathering (collectors) and evaluation (resolvers).

```
                    cmd/archfit/main.go
                    (explicit wiring)
                          │
              ┌───────────┼───────────┐
              │           │           │
         Collectors    Rule Engine   Renderers
         (read-only)   (pure logic)  (output)
              │           │           │
              └─────┬─────┘           │
                    │                 │
                FactStore         ScanResult
              (read-only view)   (findings, scores, metrics)
```

## Package Dependency Graph

```
cmd/archfit/main.go
├── internal/core/scheduler.go          ← orchestrator
│   ├── internal/collector/fs/          ← filesystem facts
│   ├── internal/collector/git/         ← git history facts
│   ├── internal/collector/schema/      ← JSON Schema detection
│   ├── internal/collector/depgraph/    ← import graph (Go)
│   ├── internal/collector/ast/         ← AST analysis (Go, via go/parser)
│   ├── internal/collector/command/     ← timed command execution
│   ├── internal/rule/                  ← engine + registry
│   └── internal/score/                 ← scoring + metrics
├── internal/adapter/exec/              ← subprocess runner
├── internal/adapter/llm/               ← LLM boundary (3 backends)
├── internal/fix/                       ← remediation engine
│   ├── internal/fix/static/            ← deterministic fixers
│   └── internal/fix/llmfix/            ← LLM-assisted fixers
├── internal/config/                    ← .archfit.yaml
├── internal/policy/                    ← org policy enforcement
├── internal/packman/                   ← pack validation
├── internal/report/                    ← renderers (terminal, json, md, sarif)
├── packs/core/                         ← 24 rules
└── packs/agent-tool/                   ← 3 rules
```

## Critical Boundary: Packs Cannot Import Adapters

This is the most important architectural invariant. It is enforced by `.go-arch-lint.yaml`.

```
packs/*  ──may import──>  internal/model, internal/rule
packs/*  ──MUST NOT──>    internal/adapter/*, internal/collector/*
```

If a resolver needs a new kind of data, the solution is always: add a Collector, expose facts through FactStore, consume in the resolver.

## Data Flow

### Scan Flow

```
1. main.go parses flags, builds Registry, loads Config
2. main.go calls core.Scan(ctx, ScanInput{Root, Rules, Runner, Depth})
3. scheduler.go runs collectors:
   a. fs.Collect(root)         → RepoFacts      (always)
   b. git.Collect(ctx, runner) → GitFacts        (when Runner != nil)
   c. schema.Collect(repo)     → SchemaFacts     (always)
   d. depgraph.Collect(repo)   → DepGraphFacts   (when Go source exists)
   e. ast.Collect(repo)        → ASTFacts        (Go files; standard + deep modes)
   f. command.Collect(...)     → CommandFacts     (only --depth=deep)
4. scheduler.go builds factStore from collector outputs
5. rule.Engine.Evaluate(ctx, rules, facts) → EvalResult
   - For each rule: call resolver(ctx, facts) → (findings, metrics, error)
   - Sort findings deterministically
6. scheduler.go computes metrics from facts
7. score.Compute(rules, findings) → Scores
8. Return ScanResult{Findings, Metrics, Scores, ...}
9. main.go renders via report.Render() or report.RenderSARIF()
10. main.go checks --fail-on threshold → exit code
```

### Fix Flow

```
1. main.go runs initial scan (same as above)
2. buildFixEngine() registers all Fixers
3. engine.Fix(ctx, FixInput{Root, RuleIDs, DryRun, Facts, Findings, Scanner})
4. For each targeted finding:
   a. Find registered Fixer for rule ID
   b. fixer.Plan(ctx, finding, facts) → []Change
5. If --dry-run or --plan: return plan, stop
6. Snapshot original file contents
7. Apply changes to disk
8. Re-scan via injected Scanner function
9. Compare: finding gone? No new findings?
   - Yes → report success
   - No  → rollback to snapshots, report failure
10. Log to .archfit-fix-log.json
```

## Key Design Decisions

### Why explicit registration over auto-discovery

`buildRegistry()` in `main.go` explicitly calls `corepack.Register(reg)` and `agenttool.Register(reg)`. There is no reflection, no `init()` side-effects, no plugin system.

**Rationale**: P3 (shallow explicitness). An agent reading the code can see exactly which packs are active by reading one function. Auto-discovery via reflection or init() is adversarial to agent comprehension.

### Why FactStore is an interface

Resolvers receive `model.FactStore` (interface), not a concrete struct. This enables:
- Test fakes without filesystem access
- Adding new fact types via interface extension (with ADR)
- Clear read-only contract — resolvers cannot mutate facts

### Why collectors and resolvers are separated

Collectors gather data; resolvers interpret it. This separation:
- Enables parallel collector execution
- Keeps resolvers pure (testable without I/O)
- Enforces P5 (aggregation of dangerous capabilities) on archfit itself
- Allows the same facts to be consumed by multiple rules

### AST Collector Design

`internal/collector/ast/` uses `go/parser` from the standard library (no external dependency). Two modes:

- **standard** (default): parses package-level declarations, exported symbols, and function signatures.
- **deep** (`--depth=deep`): additionally resolves struct field types and interface method sets.

A per-file size cap of **1 MiB** prevents pathological files from stalling the scan. Files exceeding the cap produce a `parse_skipped` entry in `ASTFacts` rather than silently dropping.

Go is the only language supported. Other languages are out of scope for the AST collector (see `PROJECT.md` for roadmap).

### Why JSON-in-YAML for config

Phase 1 parses `.archfit.yaml` as JSON. YAML 1.2 is a strict JSON superset, so this works. The benefit: zero external dependencies for config parsing. Full YAML support (anchors, block scalars) deferred to when `yaml.v3` is added.

### Why three LLM backends behind one interface

`llm.Client` with `Explain()` is implemented by Gemini (`real.go`), OpenAI (`openai.go`), and Claude (`anthropic.go`). The composition chain is: `inner → Budget → Cached`. This means:
- Budget enforcement is backend-agnostic
- Cache hits skip the network regardless of backend
- Adding a fourth backend is a single file

## Extension Points

| What to extend | Where | ADR needed? |
|---|---|---|
| New rule in existing pack | `packs/<pack>/resolvers/`, register in `pack.go` | No |
| New rule pack | `packs/<name>/`, register in `main.go:buildRegistry()` | No |
| New collector | `internal/collector/<topic>/`, wire in `scheduler.go` | Yes (FactStore change) |
| New metric | `internal/score/metrics.go`, wire in `scheduler.go` | No |
| New fixer | `internal/fix/static/`, register in `main.go:buildFixEngine()` | No |
| New LLM backend | `internal/adapter/llm/<backend>.go` | No |
| New output format | `internal/report/<format>.go` | No |
| New CLI command | `cmd/archfit/main.go` | No (unless new exit codes) |
| New FactStore method | `internal/model/model.go` | Yes |
| JSON output schema change | `schemas/output.schema.json` | Yes if non-additive |

## File Size Guidelines

Per `CLAUDE.md` and project conventions:
- `SKILL.md`: under 400 lines / 10 KB
- Remediation guides: under 100 lines each
- PR size: ≤ 500 changed lines, ≤ 5 packages
- `main.go`: the only file that imports from all layers — expected to be large
