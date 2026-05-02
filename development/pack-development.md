# Pack Development Guide

## What is a Pack?

A rule pack is a vertical slice of rules targeting a specific domain or project type. Each pack contains rules, resolvers, fixtures, and documentation.

## Built-in Packs

| Pack | Directory | Rules | Scope |
|---|---|---|---|
| `core` | `packs/core/` | 24 | Universal principles for any repository |
| `agent-tool` | `packs/agent-tool/` | 3 | Tools consumed by coding agents |

## Pack Structure

```
packs/<pack-name>/
├── AGENTS.md           # Agent-facing documentation (required)
├── INTENT.md           # Scope, design rationale (required)
├── pack.go             # Register() function + rule definitions (required)
├── pack_test.go        # Fixture-driven table tests
├── resolvers/          # One resolver per rule (required dir)
│   ├── p1_loc_001.go
│   └── p1_loc_002.go
├── fixtures/           # Golden test data (required dir)
│   ├── P1.LOC.001/
│   │   ├── input/      # minimal repo that triggers the rule
│   │   └── expected.json
│   └── P1.LOC.002/
│       ├── input/
│       └── expected.json
└── rules/              # YAML rule definitions (optional, recommended)
    ├── P1.LOC.001.yaml
    └── P1.LOC.002.yaml
```

## Creating a New Pack

### Using the SDK

```bash
archfit new-pack my-pack           # creates ./my-pack/
archfit new-pack my-pack ./packs/  # creates ./packs/my-pack/
```

This scaffolds: `AGENTS.md`, `INTENT.md`, `pack.go`, `pack_test.go`, `resolvers/`, `fixtures/`.

### Manual Steps After Scaffolding

1. **Define rules** in `pack.go`:
   ```go
   func Rules() []model.Rule {
       return []model.Rule{
           {
               ID:               "P3.EXP.002",
               Principle:        model.P3ShallowExplicitness,
               Dimension:        "EXP",
               Title:            "Spring config profiles documented",
               Severity:         model.SeverityWarn,
               EvidenceStrength: model.EvidenceStrong,
               Stability:        model.StabilityExperimental,
               Weight:           1.0,
               Rationale:        "...",
               Remediation:      model.Remediation{Summary: "..."},
               Resolver:         resolvers.ResolveP3EXP002,
           },
       }
   }
   ```

2. **Implement resolver** in `resolvers/`:
   ```go
   func ResolveP3EXP002(ctx context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
       repo := facts.Repo()
       // Pure logic on facts — no I/O
       return findings, nil, nil
   }
   ```

3. **Create fixture** in `fixtures/P3.EXP.002/input/` (minimal repo that triggers the rule)

4. **Generate expected.json**: run the resolver, capture output

5. **Write table test** in `pack_test.go`

6. **Register** in `cmd/archfit/main.go`:
   ```go
   if err := mypack.Register(reg); err != nil {
       return nil, err
   }
   ```

7. **Add documentation**:
   - `docs/rules/P3.EXP.002.md`
   - `.claude/skills/archfit/reference/remediation/P3.EXP.002.md`

## Validating a Pack

```bash
archfit validate-pack ./packs/my-pack
```

Checks:
- `AGENTS.md` exists
- `INTENT.md` exists
- At least one `.go` file in root directory
- `resolvers/` directory exists
- `fixtures/` directory exists with at least one `fixtures/*/input/` subdirectory

## Testing a Pack

```bash
archfit test-pack ./packs/my-pack  # runs go test -race -count=1
```

Or directly:
```bash
go test -race -count=1 ./packs/my-pack/
```

## Pack Rules

From `CLAUDE.md`:

- Packs may import `internal/model` and `internal/rule`
- Packs MUST NOT import `internal/adapter/`, `internal/collector/command/`, or anything that performs I/O
- If a resolver needs a new kind of fact, add a Collector — do not widen the pack's capabilities
- This boundary is enforced by `.go-arch-lint.yaml`

## Rule Naming Convention

```
P<principle>.<dimension>.<number>
```

- `P` = principle number (1-7)
- `<dimension>` = 3 uppercase letters (LOC, SPC, EXP, VER, AGG, REV, MRD, etc.)
- `<number>` = 3 digits, sequential within the dimension

Examples: `P1.LOC.001`, `P2.SPC.010`, `P7.MRD.003`

## Rule Quality Bar

- `error` severity requires `strong` evidence
- Every rule ships with: resolver, fixture + expected.json, table test, remediation doc, docs/rules/ entry
- Start at `stability: experimental`; promote to `stable` only after at least one release cycle
- Prefer 5 solid `strong` rules over 50 `weak` ones
