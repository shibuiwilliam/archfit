# CLAUDE.md — archfit

> Architecture fitness evaluator for the coding-agent era.
> This file is the primary contract between Claude Code and this repository.
> Keep it short. When in doubt, link out rather than inline.

---

## 1. What archfit is

archfit is a CLI tool and an agent skill that evaluates whether a repository follows the architectural principles required by the coding-agent era: **locality, spec-first, shallow explicitness, verifiability, aggregation of dangerous capabilities, reversibility, and machine-readability**.

archfit does **not** replace linters, SAST scanners, or language-specific quality tools. It sits **above** them, reading their outputs when useful, and reports on the **terrain** a repository presents to coding agents.

**Two deliverables, one codebase:**
1. A Go CLI binary: `archfit`
2. A Claude Code agent skill under `.claude/skills/archfit/` that drives the CLI — this is the canonical project-scope skill location per the Agent Skills docs, so Claude Code auto-discovers it when working on this repo.

The CLI is the source of truth. The skill is a thin, progressive-disclosure wrapper.

---

## 2. Meta-consistency rule (read this first)

archfit must **pass its own scan at a high score**. Every architectural decision in this repo is evaluated by the rule: *"Would archfit flag this?"*

Concretely:
- **Locality**: each rule pack is a vertical slice with its own `AGENTS.md`, `INTENT.md`, tests, and fixtures.
- **Spec-first**: rules are declared in YAML and validated against a JSON Schema. Go types are generated from, or validated against, that schema — never the other way around.
- **Shallow explicitness**: no reflection-heavy plugin systems, no init()-based auto-registration across packages, no interface-per-struct factories. Prefer a boring registry with explicit wiring.
- **Verifiability**: `make test` completes under 30s for the default suite. `make lint` under 5s.
- **Aggregation**: all command execution, filesystem traversal, git access, and network I/O go through adapters in `internal/adapter/`. Rules never touch these directly.
- **Reversibility**: every rule carries a `stability` field (`experimental`, `stable`, `deprecated`). Experimental rules are off by default.
- **Machine-readability**: `--json` output is a first-class citizen with a versioned JSON Schema in `schemas/`. Every error returned to the user has `code`, `details`, and `remediation`.

If a change makes archfit fail any of its own rules, the change must either fix the self-violation or carry a time-limited `ignore` entry in `.archfit.yaml` with a written rationale.

---

## 3. Language, toolchain, and versions

- **Go 1.23+** (use the version pinned in `go.mod`; do not bump casually)
- **Module path**: `github.com/<org>/archfit` (replace `<org>` at init)
- **Dependencies**: prefer the standard library. External deps require a short justification comment at the import site on first use. Current approved deps are listed in `docs/dependencies.md`.
- **Build**: `go build ./cmd/archfit`. No CGO. Cross-compilation must work for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`.
- **Formatting**: `gofmt -s` and `goimports`. Non-negotiable.
- **Linting**: `golangci-lint` with the config in `.golangci.yaml`. Do not add `//nolint` without a reason comment.

Do not introduce:
- Code generators that run implicitly during `go build` (generation must be an explicit `make generate` step with committed output).
- `init()` functions that register state across packages. Registration is explicit in `cmd/archfit/main.go` or in the relevant pack's `Register` function called from there.
- Global mutable state. Pass dependencies through structs.

---

## 4. Repository layout

```
archfit/
├── cmd/
│   └── archfit/              # CLI entry point. main.go wires everything.
├── internal/
│   ├── core/                 # Scheduler, rule execution, finding aggregation
│   ├── model/                # Shared types: Rule, Finding, Evidence, Metric
│   ├── config/               # .archfit.yaml loading and validation
│   ├── collector/            # Fact gatherers. Pure data, no judgement.
│   │   ├── fs/               # Filesystem walk, file presence, content stats
│   │   ├── git/              # git log sampling, PR size distribution
│   │   ├── ast/              # tree-sitter-based lightweight parsing
│   │   ├── depgraph/         # Import graph construction
│   │   ├── command/          # Runs `make test`, `go build`, etc. with timing
│   │   └── schema/           # OpenAPI/JSON Schema/protobuf detection & parse
│   ├── adapter/              # Side-effect boundary: exec, net, fs-write
│   ├── rule/                 # Rule engine core (NOT the rules themselves)
│   ├── report/               # Output renderers: json, md, sarif, terminal
│   └── score/                # Scoring and metric aggregation
├── packs/                    # Rule packs. Each pack = one vertical slice.
│   ├── core/
│   │   ├── AGENTS.md
│   │   ├── INTENT.md
│   │   ├── context.yaml
│   │   ├── rules/            # YAML rule definitions
│   │   ├── resolvers/        # Go resolver functions (the "detect" logic)
│   │   ├── fixtures/         # Golden repos for pack-level tests
│   │   └── pack_test.go
│   ├── web-saas/
│   ├── iac/
│   ├── mobile/
│   ├── data-event/
│   └── agent-tool/
├── schemas/                  # JSON Schemas: rule.schema.json, output.schema.json, config.schema.json
├── .claude/
│   └── skills/
│       └── archfit/          # The Claude Code agent skill (canonical project-scope
│           ├── SKILL.md      # location per the Agent Skills docs; auto-discovered)
│           ├── reference/
│           ├── scripts/
│           └── templates/
├── testdata/                 # Golden input repos and expected JSON outputs
├── docs/
│   ├── adr/                  # Architecture Decision Records (YAML frontmatter required)
│   ├── rules/                # Human docs per rule, auto-generated from YAML + hand notes
│   └── dependencies.md
├── scripts/
├── Makefile
├── .archfit.yaml             # archfit's own config, for self-scan
├── .golangci.yaml
├── go.mod / go.sum
└── CLAUDE.md                 # this file
```

**Boundary rule**: `packs/*` may import from `internal/model` and the public interfaces in `internal/rule`. They must not import from `internal/adapter`, `internal/collector/command`, or anything else that performs I/O. If a pack needs a new kind of fact, add a Collector — do not widen a pack's capabilities.

This boundary is enforced by `go-arch-lint` (config in `.go-arch-lint.yaml`). Violations fail CI.

---

## 5. Core abstractions

Keep these stable. Breaking changes here require an ADR.

```go
// internal/model/rule.go (shape, not literal)

type Rule struct {
    ID               string          // "P5.RSK.002"
    Principle        Principle       // enum: P1..P7
    Dimension        string
    Title            string
    Severity         Severity        // info, warn, error, critical
    EvidenceStrength EvidenceStrength // strong, medium, weak, sampled
    Stability        Stability       // experimental, stable, deprecated
    AppliesTo        Applicability
    Resolver         ResolverFunc
}

type ResolverFunc func(ctx context.Context, facts FactStore) ([]Finding, []Metric, error)

type Finding struct {
    RuleID           string
    Severity         Severity
    Confidence       float64           // 0.0–1.0
    EvidenceStrength EvidenceStrength
    Path             string
    Message          string
    Evidence         map[string]any    // structured, must be JSON-marshalable
    Remediation      Remediation
}
```

**Resolvers receive facts; they do not gather them.** This is how locality and the aggregation principle are enforced in archfit itself.

`FactStore` is a read-only view populated by Collectors before rule execution begins. Collectors run in parallel where their dependency DAG allows.

---

## 6. How to add a rule (the happy path)

1. Decide the principle and dimension. Pick the next free ID: `P<n>.<DIM>.<nnn>`.
2. Create `packs/<pack>/rules/P<n>.<DIM>.<nnn>.yaml` using the schema in `schemas/rule.schema.json`. Include `rationale`, `evidence_strength`, `stability: experimental`, and a `remediation` block.
3. Implement the resolver in `packs/<pack>/resolvers/`. It must be a pure function of `FactStore`. If you need a new fact, first add a Collector (step 4). Do not reach into the filesystem from the resolver.
4. If a new Collector is required, add it under `internal/collector/<topic>/` with its own tests. Collectors must be deterministic given identical inputs.
5. Add a golden fixture repo under `packs/<pack>/fixtures/P<n>.<DIM>.<nnn>/` with `input/` (a minimal repo that should trigger the rule) and `expected.json` (the expected finding shape).
6. Add a table test in `pack_test.go` that runs the rule against the fixture and diffs against `expected.json`.
7. Run `make self-scan` to confirm no self-regressions.
8. Update `docs/rules/` (auto-generation is fine for the skeleton; hand-edit the "When to care" section).
9. Mark `stability: stable` only after at least one release cycle at `experimental` and review.

Do not mass-add rules. Prefer five solid `strong` rules over fifty `weak` ones.

---

## 7. Testing strategy

Three layers, in order of strictness:

- **Unit** (target: < 5s total). Pure logic in `internal/model`, `internal/score`, `internal/config`, individual resolvers against in-memory `FactStore`.
- **Pack tests** (target: < 20s total). Each rule runs against its fixture repo. Output is diffed against `expected.json` byte-for-byte (after canonicalization). This is the primary correctness bar.
- **End-to-end** (target: < 60s total, runs in CI only by default). `archfit scan` on a small set of golden repos in `testdata/e2e/`. Asserts on the full JSON output shape, overall score, and exit code.

Property-based tests (`gopter` or hand-rolled) are preferred for the scoring and config-merge logic.

**Do not** write tests that depend on network, real git remotes, or the host's installed toolchains. Shell out only through `internal/adapter/exec`, which is faked in tests.

**Golden test updates**: if a change intentionally alters output, regenerate with `make update-golden` and review the diff carefully. Do not commit golden updates and code changes in the same commit without a review note explaining what moved.

---

## 8. CLI surface (do not change casually)

```
archfit scan [path]              # run all enabled rules
archfit check <rule-id> [path]   # run one rule
archfit score [path]             # summary only
archfit explain <rule-id>        # show rule docs + remediation
archfit fix <rule-id> [path]     # auto-fix (strong-evidence rules only)
archfit init                     # scaffold .archfit.yaml
archfit report [path]            # Markdown report
archfit diff <baseline.json>     # diff against a baseline scan
archfit contract check [path]    # check scan results against .archfit-contract.yaml
archfit contract init [path]     # scaffold contract from current scan
archfit list-rules
archfit list-packs
archfit validate-config
```

Global flags:
- `--json` / `--format={terminal,json,md,sarif,html}`
- `--depth={shallow,standard,deep}` (default: `standard`)
- `--fail-on={info,warn,error,critical}` (default: `error`)
- `--profile={strict,standard,permissive}`
- `-C <dir>` (work-in directory, like `git -C`)

Exit codes are part of the contract — see `docs/exit-codes.md`. **Never change an exit code without an ADR and a major-version bump.**

---

## 9. Output format contract

`--json` output conforms to `schemas/output.schema.json`. The schema is versioned with `schema_version`. Agents consume this; changing it silently breaks them.

Rules for output stability:
- Additive changes (new fields) are fine within a minor version.
- Renaming, removing, or retyping fields requires a major bump.
- Ordering of `findings[]` must be deterministic (sort by `severity desc, rule_id asc, path asc`).
- Floating-point scores are rounded to one decimal in output; internal math stays in float64.

Every `finding` must carry `evidence` that is sufficient for a human or agent to verify the claim without re-running archfit. If you cannot produce such evidence, the rule is not ready.

---

## 10. Agent skill (`.claude/skills/archfit/`)

The skill lives at `.claude/skills/archfit/` — the canonical project-scope location per the [Agent Skills docs](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview), so Claude Code auto-discovers it when working inside this repo. Do not move it to `skills/` or anywhere else; Claude Code will not find it there.

`SKILL.md` must carry YAML frontmatter with exactly two fields — `name` (lowercase / digits / hyphens, ≤64 chars, no reserved words `anthropic`/`claude`) and `description` (≤1024 chars, must state both *what* the skill does and *when* to use it, so it triggers reliably from Level-1 metadata alone). No other frontmatter fields.

Treat `SKILL.md` like a repo's top-level `AGENTS.md`: short, task-oriented, and linking out. The explicit rule: **if `SKILL.md` exceeds 10 KB or 400 lines, it has failed its own principle.** Level-2 content (the SKILL.md body) should stay under ~5k tokens; push deeper material into Level-3 files.

Deep material lives in `.claude/skills/archfit/reference/`. Agents load it on demand via progressive disclosure; do not inline it.

When adding a rule, also add `.claude/skills/archfit/reference/remediation/<rule-id>.md` with:
- One-sentence summary of what the rule checks
- A decision tree: what to do when this finding appears (including *when to ask the user* vs *when to proceed*)
- A minimal code/config snippet for the fix

Keep remediation files under 100 lines each. If more is needed, link to `docs/rules/<rule-id>.md`.

---

## 11. Commit and PR discipline

- Conventional Commits (`feat:`, `fix:`, `refactor:`, `docs:`, `chore:`, `test:`, `pack:`).
- One logical change per PR. Mixing a new rule with a refactor of the engine is not acceptable.
- PR description must include: which principle(s) the change touches, whether it affects the JSON output schema, and the result of `make self-scan`.
- Any change under `internal/model/` or `schemas/` requires an ADR in `docs/adr/`.
- Changes to exit codes, CLI flag names, or the output JSON schema require a `BREAKING CHANGE:` footer and a migration note.

PR size budget: **≤ 500 changed lines, ≤ 5 packages touched**, unless the PR is a pure rename or generated-code update (labeled `chore: codegen` or `refactor: rename`).

---

## 12. What Claude Code should do by default

When asked to work on this repository, Claude Code should:

1. **Read, then plan.** Before editing, read the relevant pack's `AGENTS.md` and `INTENT.md`, then state the plan. For a new rule, the plan must name the rule ID, pack, and the fixture strategy before any code is written.
2. **Prefer editing over creating.** New files are fine when they fit the layout; proliferating helper files or utility packages is not.
3. **Run the fast loop after every non-trivial change.** `make lint test` must pass locally before declaring a change done. `make self-scan` before claiming a feature is complete.
4. **Respect boundaries.** If a task seems to require a pack importing from `internal/adapter`, stop and propose a Collector instead.
5. **Refuse implicit magic.** If the natural solution involves `init()` registration, reflection to discover rules, or interface-per-struct, push back and propose the explicit alternative.
6. **Write the fixture first for rule work.** Build the minimal repo that should trigger the rule and the expected JSON, then implement the resolver until the fixture passes.
7. **Keep output deterministic.** Any time you produce output that flows into `--json`, verify it under `-race` and with shuffled input order. Non-determinism is a bug.
8. **Ask before widening scope.** Adding a dependency, a new Collector type, a new CLI subcommand, or a new output format is a design decision — surface it, do not quietly ship it.

When Claude Code is unsure whether a change is in scope, the default is to ask. A concise question with two concrete alternatives is almost always better than a large speculative PR.

---

## 13. What not to do

- Do not add rules whose evidence is only `weak` and whose severity is `error`. High severity requires high evidence.
- Do not couple scoring to rule count. Adding rules must not make the score artificially go down for existing repositories — scoring is weight-based and normalized per applicable rule set.
- Do not let a rule silently skip on parse failure. Parse failures are a finding (`P<n>.<DIM>.<nnn>` with severity `warn` and `evidence_strength: strong`), not a reason to return zero findings.
- Do not shell out to tools we can parse ourselves. `git log` is fine (well-specified); `terraform plan` output parsing goes behind an adapter with an interface, never ad-hoc regex in a resolver.
- Do not hardcode paths like `src/` or `internal/`. Every path pattern comes from `.archfit.yaml` or the pack's declared globs.
- Do not introduce LLM calls on the hot path. Any LLM-assisted explanation is opt-in via `--with-llm` and lives behind a clearly isolated adapter.

---

## 14. Definition of done (per task type)

**New rule**: YAML present and schema-valid · resolver implemented · fixture + `expected.json` committed · table test passing · remediation doc under `.claude/skills/archfit/reference/remediation/` · `stability: experimental` · `make self-scan` clean or with documented waiver · `docs/rules/<rule-id>.md` exists.

**New Collector**: lives under `internal/collector/<topic>/` · fake implementation available for tests · unit tests against representative fixtures · no direct use from packs (only through `FactStore`) · documented in `internal/collector/README.md`.

**New output format**: conforms to a schema under `schemas/` · golden test against `testdata/e2e/` · documented in `docs/outputs.md` · exit-code behavior unchanged.

**Refactor**: `make lint test self-scan` all green · no change to public types in `internal/model/` without ADR · no change to JSON output bytes on golden fixtures without an update note.

---

## 15. Fast reference

```bash
make dev          # bootstrap: deps, generate, build
make build        # build the CLI
make test         # unit + pack tests
make e2e          # end-to-end against testdata/
make lint         # gofmt, goimports, golangci-lint, go-arch-lint
make generate     # regenerate code from schemas
make self-scan    # archfit scan ./ — must pass
make update-golden  # regenerate expected.json files (review carefully)
```

---

## 16. Implementation guide for ongoing work

Each numbered step is a self-contained PR-sized unit of work. For detailed technical specs, see `development/`.

### Fix engine (`internal/fix/`)

- **Fixer interface**: `Plan(ctx, finding, facts) → []Change`. Static fixers return deterministic changes; LLM fixers call the adapter. Both registered explicitly in `buildFixEngine()` in `main.go`.
- **Engine loop**: scan → plan → snapshot originals → apply → re-scan → rollback if regression.
- **Static fixers** in `internal/fix/static/` use `//go:embed` + `text/template`.
- **LLM fixers** in `internal/fix/llmfix/` wrap static fixers and enrich via `llm.Client`. Fallback on LLM failure.
- **Adding a fixer**: implement `fix.Fixer`, add to `buildFixEngine()`, add unit test, update remediation doc.

### Metrics and collectors

- **New collector**: add under `internal/collector/<topic>/` with fake for tests. Wire into `core.Scan()` in `scheduler.go`. Update `FactStore` interface — this requires an ADR.
- **New metric**: pure function in `internal/score/metrics.go`. Property-based tests preferred.

### Ecosystem and packs

- **External packs** are Go modules compiled into a custom binary. Runtime plugin loading is forbidden.
- **Organization policies** (`internal/policy/`): post-processing only, does not affect how rules run.
- **Adding a pack**: use `archfit new-pack <name>`, register in `buildRegistry()` in `main.go`.

### Fitness contract (`internal/contract/`)

- **Contract types**: `Contract`, `Constraint`, `AreaBudget`, `AgentDirective` in `internal/contract/contract.go`.
- **Check function**: `Check(contract, scores, findings) → CheckResult`. Pure function, no I/O.
- **Loading**: same JSON-in-YAML pattern as `internal/config/`.
- **CLI** (implemented): `archfit contract check` (exit 0/1/5) and `archfit contract init` (generates contract from current scan). ADR 0008.
- **Agent integration** (next step): skill reads contract before starting work, respects area budgets, follows directives.
- See `development/fitness-contract.md` for full design.

### Agent observatory (`internal/observer/` — not yet started)

- **Trace types**: `Trace`, `Event`, `EventType` in `internal/observer/trace.go`.
- **Behavioral metrics**: pure functions computing `agent_context_efficiency`, `agent_retry_rate`, etc. from traces.
- **Hotspot analysis**: cross-reference traces with static findings.
- **CLI**: `archfit observe --trace-dir .agent-traces/`. Read-only, exit 0 always.
- **Boundary**: observer reads trace files. It never instruments the agent.
- See `development/agent-observatory.md` for full design.

### Adaptive engine (`internal/adaptive/` — not yet started)

- **Confidence adjustment**: post-processing layer that adjusts findings AFTER resolvers run.
- **Opt-in**: `--adaptive` flag or `adaptive: true` in `.archfit.yaml`.
- **Fix outcome tracking**: extend `internal/fix/log.go` with `RepoSignals`.
- **Threshold adaptation**: context-aware numeric thresholds. May need optional `FactStore` method (ADR required).
- Resolvers remain pure. The adaptive layer never modifies resolver functions.
- See `development/adaptive-engine.md` for full design.

### Quality gates (every step)

- [ ] `make lint` passes (< 5s)
- [ ] `make test` passes (< 30s)
- [ ] `make self-scan` exits 0
- [ ] No new `//nolint` without reason comment
- [ ] No new dependency without justification + `docs/dependencies.md` entry
- [ ] No changes to `internal/model/` without ADR
- [ ] No changes to JSON output schema without `schema_version` consideration
- [ ] PR ≤ 500 changed lines, ≤ 5 packages touched

---

## 17. When this file and reality disagree

This file is the contract, not the reality. If you find the repository doing something this file forbids, the first move is **not** to update this file to match. Raise the contradiction, decide which side is right, and only then change one of them. Silent drift between `CLAUDE.md` and the code is exactly the kind of thing archfit itself is meant to prevent.
