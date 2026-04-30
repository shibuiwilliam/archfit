# CLAUDE.md — archfit

> Operational contract between Claude Code and this repository.
>
> This file is short on purpose. Strategy, roadmap, and rationale live in `PROJECT.md`. This file tells Claude Code **how to work here**, **what is currently broken on purpose vs. by accident**, and **what "done" looks like** for the most common tasks.
>
> When in doubt: link out, don't inline.

---

## 1. What archfit is (one screen)

archfit is a Go CLI plus a Claude Code agent skill that evaluates whether a repository is shaped for coding agents to work on safely and quickly across seven principles: **locality, spec-first, shallow explicitness, verifiability, aggregation of dangerous capabilities, reversibility, machine-readability**.

Two deliverables, one codebase:

1. The CLI: `cmd/archfit` → binary `archfit`.
2. The skill: `.claude/skills/archfit/` (canonical project-scope skill location, auto-discovered by Claude Code in this repo).

The CLI is the source of truth. The skill is a thin progressive-disclosure wrapper.

For the strategic view (what archfit should become, prioritized roadmap), read `PROJECT.md`. **Do not duplicate that content here.**

---

## 2. Current state at a glance

Claude Code: read this before starting any task. The numbers below should match `make list-rules` and `make list-packs`.

| Item | Value |
|---|---|
| Module path | `github.com/shibuiwilliam/archfit` |
| Go version | **1.24+** (pinned in `go.mod`) |
| Packs implemented | `core` (14 rules), `agent-tool` (3 rules) |
| Total rules | **17**, all `stability: stable` (frozen per ADR 0012) |
| Output formats | `terminal`, `json`, `md`, `sarif` |
| Scan depths | `shallow`, `standard`, `deep` |
| Output schema version | `1.0.0` (frozen per ADR 0012; see §12) |
| LLM enrichment | opt-in via `--with-llm`; Claude / OpenAI / Gemini |
| Self-scan score floor | **must not drop on any PR** (§19) |

If any of these drift, treat the drift itself as a bug and fix the source rather than the document.

---

## 3. The meta-consistency contract (read this second)

archfit must pass its own scan at a high score. Every change is evaluated by: *"would archfit flag this?"*

Concretely:

- **P1 Locality.** Each pack is a vertical slice with `AGENTS.md`, `INTENT.md`, `pack.go`, `resolvers/`, `rules/`, `fixtures/`.
- **P2 Spec-first.** Rules are declared in YAML and validated against `schemas/rule.schema.json`. Go rule definitions are generated from or validated against the YAML — **not the other way around**. (This invariant is in repair; see §7.1.)
- **P3 Shallow explicitness.** No reflection-based plugin systems. No `init()` registration across packages. No interface-per-struct factories. Boring registry, explicit wiring in `cmd/archfit/main.go`.
- **P4 Verifiability.** `make lint` < 5 s, `make test` < 30 s, `make e2e` < 60 s.
- **P5 Aggregation.** All command execution, filesystem traversal, git access, network I/O, and **filesystem writes** go through adapters in `internal/adapter/`. Rule packs cannot import them. The fix engine uses `internal/adapter/fs` for all I/O. Enforced by `.go-arch-lint.yaml`.
- **P6 Reversibility.** Every rule has a `stability` field. `experimental` rules can change shape; `stable` rules require an ADR for any user-visible change.
- **P7 Machine-readability.** `--json` output is the contract; `schemas/output.schema.json` is its specification. They must match byte-for-byte under strict validation.

If a change makes archfit fail one of its own rules, fix the violation, or carry a time-limited `ignore` entry in `.archfit.yaml` with a written rationale and an `expires` date.

---

## 4. Toolchain and dependencies

- Go 1.24+. Do not bump unilaterally.
- No CGO. The release matrix is `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/amd64`.
- Formatting: `gofmt -s` and `goimports`. Non-negotiable.
- Linting: `golangci-lint` with `.golangci.yaml`. New `//nolint` requires a reason comment.
- Boundary linting: `.go-arch-lint.yaml` (see §7.6 — currently being introduced; CI gate will enforce it once landed).
- External dependencies: prefer the standard library. New deps require a justification line in `docs/dependencies.md` and a comment at the first import site.

Approved network egress at runtime: only the LLM adapter when `--with-llm` is set. Everything else must be local.

Do not introduce:

- Code generators that run implicitly during `go build`. Generation is an explicit `make generate` step with **committed** output.
- `init()` functions that register cross-package state.
- Global mutable state. Pass dependencies through structs.

---

## 5. Repository layout (actually present)

```
archfit/
├── cmd/
│   └── archfit/                 # CLI entry. main.go is the only wiring file.
├── internal/
│   ├── core/                    # Scheduler. Builds FactStore, runs the Engine.
│   ├── model/                   # Shared types: Rule, Finding, Evidence, Metric.
│   ├── config/                  # .archfit.yaml loading and validation.
│   ├── contract/                # Fitness contract loader and check logic.
│   ├── collector/               # Fact gatherers. Pure data, no judgement.
│   │   ├── fs/                  # Filesystem walk, file metadata.
│   │   ├── git/                 # git log sampling.
│   │   ├── schema/              # JSON-Schema detection.
│   │   ├── depgraph/            # Go import graph.
│   │   └── command/             # Verification commands (depth=deep only).
│   ├── adapter/                 # Side-effect boundary.
│   │   ├── exec/                # Subprocess execution.
│   │   └── llm/                 # Claude / OpenAI / Gemini, behind one Client.
│   ├── rule/                    # Engine + Registry. NOT the rules themselves.
│   ├── policy/                  # Organization policy post-processing.
│   ├── packman/                 # Pack scaffolding and structural validation.
│   ├── fix/                     # Fix engine (plan → snapshot → apply → verify).
│   │   ├── static/              # Deterministic fixers (template-based).
│   │   └── llmfix/              # LLM-assisted fixers.
│   ├── report/                  # Renderers: terminal, json, md, sarif, diff.
│   ├── score/                   # Scoring and metric aggregation.
│   └── version/                 # Embedded build version.
├── packs/                       # Rule packs. Each pack = one vertical slice.
│   ├── core/                    # 14 rules (P1, P2, P3, P4, P5, P6, P7).
│   └── agent-tool/              # 3 rules (P2, P7).
├── schemas/                     # rule, output, config, contract.
├── .claude/
│   └── skills/archfit/          # Project-scope agent skill (auto-discovered).
│       ├── SKILL.md             # ≤ 400 lines, ≤ 10 KB; minimal frontmatter.
│       ├── reference/           # Progressive disclosure, including remediation/.
│       ├── scripts/
│       └── templates/
├── testdata/e2e/                # End-to-end golden repos and expected output.
├── docs/
│   ├── adr/                     # ADRs (YAML frontmatter required).
│   ├── rules/                   # One human doc per registered rule.
│   ├── self-scan/               # One JSON snapshot per release tag (§19).
│   └── dependencies.md
├── development/                 # Design notes (not user-facing docs).
├── Makefile
├── .archfit.yaml                # archfit's own config, used by `make self-scan`.
├── .golangci.yaml
├── .go-arch-lint.yaml           # Boundary enforcement (see §7.6).
├── go.mod / go.sum
├── PROJECT.md                   # Strategy, status report, roadmap.
└── CLAUDE.md                    # This file.
```

Boundary rule (machine-enforced once §7.6 lands):

- `packs/*` may import `internal/model` and the public interface of `internal/rule`. Nothing else.
- `packs/*` may not import `internal/adapter`, `internal/collector/command`, `os`, `io/fs`, `os/exec`, or `net/*`. Need a new fact? Add a Collector.

There is **no** `internal/collector/ast/` package. Past versions of this document mentioned tree-sitter; that path is not on the current roadmap.

---

## 6. Core abstractions

Stable. Breaking changes require an ADR.

```go
// internal/model — shapes, not literal source

type Rule struct {
    ID               string           // "P1.LOC.001" — pattern P[1-7]\.[A-Z]{3}\.\d{3}
    Principle        Principle        // P1..P7
    Dimension        string           // 3 uppercase letters, e.g. "LOC"
    Title            string
    Severity         Severity         // info | warn | error | critical
    EvidenceStrength EvidenceStrength // strong | medium | weak | sampled
    Stability        Stability        // experimental | stable | deprecated
    AppliesTo        Applicability
    Rationale        string
    Weight           float64          // default 1
    Remediation      Remediation
    Resolver         ResolverFunc
}

type ResolverFunc func(ctx context.Context, facts FactStore) ([]Finding, []Metric, error)

type Finding struct {
    RuleID           string
    Principle        Principle
    Severity         Severity
    EvidenceStrength EvidenceStrength
    Confidence       float64           // 0.0–1.0
    Path             string
    Message          string
    Evidence         map[string]any    // JSON-marshalable
    Remediation      Remediation
    LLMSuggestion    *LLMSuggestion    // omitempty; only when --with-llm
}
```

Two invariants at all times:

1. **Resolvers receive facts; they do not gather them.** No I/O in resolver code.
2. **Validation rejects severity ≥ error with evidence_strength = weak** (`Rule.Validate`). Strong claims need strong evidence.

---

## 7. Known integrity gaps Claude Code is expected to help close

These are accepted, tracked violations of §3. They appear in `PROJECT.md` Phase 0. Until they are closed, every PR Claude Code touches in the affected area must either (a) avoid making the gap worse, or (b) explicitly contribute to closing it.

### 7.1 (closed — YAML is source of truth; `make generate` produces Go; CI sync test enforces)

### 7.2 (closed — see ADR 0009)

### 7.3 (closed — config and contract now parse YAML via sigs.k8s.io/yaml)

### 7.4 (closed — git collector now populates FilesChanged via --numstat)

### 7.5 (closed — CI test enforces docs/rules + remediation guides for every rule)

### 7.6 (closed — internal/adapter/fs adapter added; fix engine refactored; .go-arch-lint.yaml enforces boundaries)

---

## 8. How to add or change a rule (the happy path)

YAML is the source of truth (§7.1 closed). `make generate` produces Go from YAML.

1. Pick the next ID `P<n>.<DIM>.<nnn>`. Check `make list-rules` for collisions.
2. Create `packs/<pack>/rules/P<n>.<DIM>.<nnn>.yaml`. Validate against `schemas/rule.schema.json`. Use `stability: experimental`.
3. Implement the resolver in `packs/<pack>/resolvers/`. Pure function of `FactStore`. No I/O.
4. If a new fact is needed, add a Collector first under `internal/collector/<topic>/` with its own tests and a fake.
5. Add **both** fixtures: `packs/<pack>/fixtures/<id>/positive/` (rule fires) and `.../negative/` (rule does not fire). Each carries `input/` and an `expected.json`.
6. Add a table test in `packs/<pack>/pack_test.go` that runs the rule against both fixtures and diffs `expected.json` byte-for-byte.
7. Add the Go rule declaration in `packs/<pack>/pack.go`. Wire the resolver.
8. Add `docs/rules/<id>.md` and `.claude/skills/archfit/reference/remediation/<id>.md`. Both required.
9. Run `make lint test self-scan`. All three must pass.
10. Open a PR. Note the principle, severity, evidence strength, and stability in the description.

A rule moves from `experimental` to `stable` only after one full release cycle and a passing calibration run on the corpus described in `PROJECT.md` §6.2.

Five solid `strong`-evidence rules outweigh fifty `weak` ones.

---

## 9. How to add a Collector

Collectors observe; they do not judge.

- One topic per package under `internal/collector/<topic>/`.
- Provide a `Fake` (in-package) for use by resolver tests.
- Determinism: same input → same output. Sort outputs in a fixed order.
- Limits: guard against pathological repos (size, depth, byte counts).
- Wire the new collector into `internal/core/scheduler.go` only — the scheduler is the only seam that knows both sides.
- Extend `model.FactStore` only via ADR. Adding a new accessor is a public-surface change.

---

## 10. Testing strategy

Three layers:

| Layer | Bar | Lives in |
|---|---|---|
| Unit | < 5 s total | next to the code under test |
| Pack | < 20 s total | `packs/<pack>/pack_test.go`, with fixtures in `packs/<pack>/fixtures/` |
| End-to-end | < 60 s total, CI by default | `testdata/e2e/`, runs the real `archfit scan` |

Discipline:

- Property-based tests preferred for `internal/score`, `internal/config`, `internal/contract`.
- No tests depending on network, real git remotes, or installed toolchains. All exec goes through `internal/adapter/exec`, which has a fake.
- Golden updates: regenerate via `make update-golden` and **review the diff carefully**. Never commit a golden update and a code change in the same commit without a review note.

A change that intentionally alters output requires:

- The corresponding `expected.json` files updated.
- The schema validated against the new output.
- A note in `CHANGELOG.md`.

---

## 11. CLI surface (do not change casually)

```
archfit scan [path]                      # run all enabled rules
archfit check <rule-id> [path]           # run a single rule
archfit score [path]                     # summary only
archfit explain <rule-id>                # show rule docs + remediation
archfit fix [rule-id] [path]             # auto-fix; --plan, --all, --dry-run
archfit init [path]                      # scaffold .archfit.yaml
archfit report [path]                    # Markdown report
archfit diff <baseline.json> [current.json]
archfit trend                            # score trends from --record archives
archfit compare <f1.json> <f2.json> [...]
archfit contract check [path]            # validate scan against .archfit-contract.yaml
archfit contract init [path]             # scaffold contract from current scan
archfit list-rules
archfit list-packs
archfit validate-config
archfit validate-pack <path>
archfit new-pack <name> [path]
archfit test-pack <path>
archfit version | --version | -v
archfit help    | --help    | -h
```

Global flags (where applicable):

- `--format {terminal|json|md|sarif}` (default `terminal`)
- `--json` (shorthand for `--format=json`)
- `--fail-on {info|warn|error|critical}` (default `error`)
- `--depth {shallow|standard|deep}` (default `standard`)
- `-C <dir>` (work-in directory, like `git -C`)
- `--config <file>` (default `.archfit.yaml` in target)
- `--profile {strict|standard|permissive}`
- `--policy <file>` (organization policy post-processing)
- `--with-llm`, `--llm-backend {claude|openai|gemini}`, `--llm-budget N`
- `--record <dir>`, `--explain-coverage`

Exit codes are part of the contract — see `docs/exit-codes.md` and `PROJECT.md` §5.1. Do not change them without an ADR and a major-version bump. Exit code 5 (advisory) is under review (see `PROJECT.md` §5.1).

---

## 12. Output format contract

`--json` output conforms to `schemas/output.schema.json`. The schema is versioned via `schema_version`. Agents consume this; silent breakage is the worst kind of regression archfit can ship.

Stability rules:

- Pre-1.0: additive field changes are minor; any rename, removal, or retype is major.
- `findings[]` is sorted by severity desc, then `rule_id` asc, then `path` asc. Renderers do not re-sort.
- Float scores are rounded to one decimal in output; internal math stays in `float64`.
- Every `finding.evidence` carries enough to verify the claim *without* re-running archfit.

If you change anything under `internal/report/`: bump `schema_version`, update `schemas/output.schema.json`, regenerate goldens, run schema-validation tests. **All four**, in the same PR.

---

## 13. Agent skill (`.claude/skills/archfit/`)

The skill lives at `.claude/skills/archfit/` because that is the canonical project-scope location per the Agent Skills documentation; Claude Code auto-discovers it inside this repo.

`SKILL.md` carries minimal YAML frontmatter — `name` (lowercase, digits, hyphens, ≤ 64 chars; do not use `anthropic` or `claude` as a name) and `description` (≤ 1024 chars; states *what* and *when*). No other frontmatter fields.

Hard limits:

- `SKILL.md` ≤ 400 lines, ≤ 10 KB. If you bust it, the skill itself fails the principle it teaches.
- Body of `SKILL.md` ≤ ~5k tokens (Level-2 content). Push deeper material to `reference/`.
- Each `reference/remediation/<rule-id>.md` ≤ 100 lines. Link out to `docs/rules/<id>.md` for depth.

When you add a rule (§8), you add the corresponding remediation file in the same PR. CI will enforce this once §7.5 lands; until then, treat it as a hard rule of the PR template.

---

## 14. Commit and PR discipline

- Conventional Commits: `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`, `test:`, `pack:`.
- One logical change per PR. A new rule and an engine refactor do not belong together.
- PR description must include:
  - Which principle(s) are touched.
  - Whether `schemas/output.schema.json` changes (and the `schema_version` delta).
  - The result of `make self-scan`.
- Any change under `internal/model/` or `schemas/` requires an ADR in `docs/adr/` (YAML frontmatter mandatory).
- Changes to exit codes, CLI flag names, or the JSON output schema require a `BREAKING CHANGE:` footer and a migration note.
- PR size budget: **≤ 500 changed lines, ≤ 5 packages**, unless the PR is a pure rename or generated-code update (label `chore: codegen` or `refactor: rename`).

---

## 15. What Claude Code should do by default

1. **Read the relevant pack's `AGENTS.md` and `INTENT.md` before editing.** State the plan before code. For a new rule, the plan must name the rule ID, the pack, and the fixture strategy.
2. **Prefer editing existing files over creating new ones.** New utility packages need justification.
3. **Run the fast loop after every non-trivial change.** `make lint test` must pass locally. `make self-scan` before claiming a feature complete.
4. **Respect boundaries.** If a task seems to require a pack importing from `internal/adapter`, propose a Collector instead. If a task seems to require I/O outside `internal/adapter/`, propose adding a new adapter.
5. **Refuse implicit magic.** `init()` registration, reflection-based discovery, interface-per-struct factories are out of scope. Push back and propose the explicit alternative.
6. **Write fixtures first for rule work.** Build the minimal positive and negative fixture, write the expected JSON, then implement the resolver until the fixture passes.
7. **Keep output deterministic.** Verify under `-race` and with shuffled input order. Non-determinism is a bug.
8. **Ask before widening scope.** Adding a dependency, a new Collector type, a new CLI subcommand, or a new output format is a design decision. Surface it; don't ship it silently.
9. **When closing a Phase 0 gap (§7), prefer the smallest committed change that closes it.** A single PR per gap is the target.

When unsure whether a change is in scope, the default is to ask. A concise question with two concrete alternatives is almost always better than a large speculative PR.

---

## 16. What not to do

- Do not add rules with `evidence_strength: weak` and `severity: error`. `Rule.Validate` rejects this; do not work around it.
- Do not couple scoring to rule count. Adding rules must not mechanically lower scores; scoring is weight-based and normalized per applicable rule set.
- Do not let a rule silently skip on parse failure. Parse failures are findings (`severity: warn`, `evidence_strength: strong`), not zero-finding returns. Use `model.ParseFailure`.
- Do not shell out to tools we can parse ourselves. `git log` is fine (well-specified). `terraform plan` output goes behind an adapter with an interface, never ad-hoc regex in a resolver.
- Do not hardcode paths like `src/` or `internal/` inside resolvers. Path patterns come from `.archfit.yaml`, the rule's `applies_to.path_globs`, or pack-level globs.
- Do not introduce LLM calls on the hot path. Anything that consults an LLM is opt-in via `--with-llm`, lives behind `internal/adapter/llm`, and degrades gracefully when the API call fails.
- Do not edit files under `docs/self-scan/` by hand. Those are generated by release tooling.

---

## 17. Definition of done (per task type)

**New rule.**

- [ ] YAML in `packs/<pack>/rules/<id>.yaml`, schema-valid.
- [ ] Resolver in `packs/<pack>/resolvers/`.
- [ ] Positive fixture + `expected.json`.
- [ ] Negative fixture + `expected.json`.
- [ ] Table test passing in `pack_test.go`.
- [ ] Resolver wired in `packs/<pack>/pack.go` resolver map + `make generate`.
- [ ] `docs/rules/<id>.md`.
- [ ] `.claude/skills/archfit/reference/remediation/<id>.md`.
- [ ] `stability: experimental` (promote to `stable` after one release cycle).
- [ ] `make lint test self-scan` clean.
- [ ] `CHANGELOG.md` entry.

**New Collector.**

- [ ] Lives under `internal/collector/<topic>/`.
- [ ] Fake implementation for tests in the same package.
- [ ] Unit tests against representative fixtures.
- [ ] No direct use from packs (only through `FactStore`).
- [ ] `model.FactStore` extension with ADR if required.
- [ ] Documented in `internal/collector/README.md`.

**New output format.**

- [ ] Conforms to a schema under `schemas/`.
- [ ] Golden test against `testdata/e2e/`.
- [ ] Documented in `docs/outputs.md`.
- [ ] Exit-code behavior unchanged.
- [ ] `schema_version` considered.

**Refactor.**

- [ ] `make lint test self-scan` all green.
- [ ] No change to `internal/model/` public types without ADR.
- [ ] No change to JSON output bytes on golden fixtures without an update note in the PR.

**Phase 0 gap closure (§7).**

- [ ] Single PR per gap.
- [ ] Adds the corresponding CI check that prevents regression.
- [ ] Updates `PROJECT.md` §6.1 to mark the item closed.
- [ ] Removes or updates the matching subsection in this file's §7.

---

## 18. Fast reference

```
make dev          # bootstrap: deps, generate, build
make build        # build the CLI
make test         # unit + pack tests
make e2e          # end-to-end against testdata/
make lint         # gofmt, goimports, golangci-lint, go-arch-lint
make generate     # regenerate code from schemas (committed output)
make self-scan    # archfit scan ./ — must pass
make update-golden  # regenerate expected.json files (review carefully)
make list-rules   # registered rules and their metadata
make list-packs   # registered packs
```

---

## 19. Quality gates (every PR)

- [ ] `make lint` < 5 s, clean.
- [ ] `make test` < 30 s, clean.
- [ ] `make self-scan` exits 0.
- [ ] Self-scan **overall score is not lower** than the score on `main` (CI publishes this delta in PR comments once tooling lands).
- [ ] No new `//nolint` without reason comment.
- [ ] No new dependency without justification + `docs/dependencies.md` entry.
- [ ] No changes to `internal/model/` without ADR.
- [ ] No changes to JSON output schema without `schema_version` consideration.
- [ ] No new I/O outside `internal/adapter/`.
- [ ] PR ≤ 500 changed lines, ≤ 5 packages touched.
- [ ] `CHANGELOG.md` updated when the change is user-visible.

---

## 20. When this file and reality disagree

This file is the contract, not the reality. If you find the repository doing something this file forbids, the first move is **not** to update this file to match. Raise the contradiction, decide which side is right, and only then change one of them.

When this file forbids something that is currently happening (the integrity gaps in §7), the contract says the *reality* is wrong. Your job is to help close the gap, not to weaken the contract.

Silent drift between `CLAUDE.md` and the code is exactly the kind of thing archfit itself is meant to prevent.

---

## 21. Change log of this document

- **2026-04-30 — Rewrite as an operational report.**
  Updated factual fields to match the live repository: Go 1.24+, module path `github.com/shibuiwilliam/archfit`, 14 rules across `core` (11) and `agent-tool` (3), removal of the never-existed `internal/collector/ast/` reference, addition of `internal/contract/`, `internal/policy/`, `internal/packman/`, `internal/fix/`, `internal/adapter/llm/`. Added §2 "Current state at a glance" so Claude Code can orient in one screen. Added §7 "Known integrity gaps Claude Code is expected to help close" — six accepted, tracked violations of §3 with target states. Tightened §17 "Definition of done" to require pair fixtures and remediation docs in the same PR. Added §19 "Quality gates" with measurable bars. Cross-linked `PROJECT.md` rather than duplicating roadmap content here.
