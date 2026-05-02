# CLAUDE.md — archfit

> Operational contract between Claude Code and this repository.
>
> This file is short on purpose. Strategy, roadmap, and rationale live in `PROJECT.md`. This file tells Claude Code **how to work here**, **what is currently in flight vs. shipped**, and **what "done" looks like** for the most common tasks.
>
> When in doubt: link out, don't inline.

---

## 1. What archfit is (one screen)

archfit is a Go CLI plus a Claude Code agent skill that evaluates whether a repository is shaped for coding agents to work on safely and quickly across seven principles: **locality, spec-first, shallow explicitness, verifiability, aggregation of dangerous capabilities, reversibility, machine-readability**.

Two deliverables, one codebase:

1. The CLI: `cmd/archfit` → binary `archfit`.
2. The skill: `.claude/skills/archfit/` (canonical project-scope skill location, auto-discovered by Claude Code in this repo).

The CLI is the source of truth. The skill is a thin progressive-disclosure wrapper over the CLI.

For the strategic view (what archfit should become, prioritized roadmap, review findings driving Phase 1), read `PROJECT.md`. **Do not duplicate that content here.**

---

## 2. Current state at a glance

Claude Code: read this before starting any task. The numbers below should match `make list-rules` and `make list-packs`. If they drift, fix the source — do not silently update this table.

| Item | Value |
| --- | --- |
| Module path | `github.com/shibuiwilliam/archfit` |
| Go version | **1.24+** (pinned in `go.mod`) |
| Packs | `core` (14 rules), `agent-tool` (3 rules) |
| Total rules | **17** today; Phase 1 grows to ~27 |
| Stability mix | Most `stable`; Phase 1 walks `P1.LOC.003`, `P1.LOC.004`, `P5.AGG.001` back to `experimental` per ADR 0013 |
| First `error` severity | Phase 1 introduces `P5.AGG.004` (CODEOWNERS on high-risk paths) |
| Output formats | `terminal`, `json`, `md`, `sarif` |
| Scan depths | `shallow`, `standard`, `deep` |
| Output schema version | `1.0.0` today; Phase 1 → `1.1.0` (provenance, additive) |
| LLM enrichment | opt-in via `--with-llm`; `metadata` mode only today; Phase 1.5 adds `file-snippet`, `full-context` |
| Self-scan gate | Phase 1 refines: see §19 |

If any of these drift, treat the drift itself as a bug and fix the source, not the document.

---

## 3. The meta-consistency contract (read this second)

archfit must satisfy the principles it measures. Every change is evaluated by: *"would archfit flag this?"*

* **P1 Locality.** Each pack is a vertical slice with `AGENTS.md`, `INTENT.md`, `pack.go`, `resolvers/`, `rules/`, `fixtures/`.
* **P2 Spec-first.** Rules are declared in YAML and validated against `schemas/rule.schema.json`. Go rule definitions are generated from YAML by `make generate` (committed output) — **not the other way around**.
* **P3 Shallow explicitness.** No reflection-based plugin systems. No `init()` registration across packages. No interface-per-struct factories. Boring registry, explicit wiring in `cmd/archfit/main.go`.
* **P4 Verifiability.** `make lint` < 5 s, `make test` < 30 s, `make e2e` < 60 s.
* **P5 Aggregation.** All command execution, filesystem traversal, git access, network I/O, and **filesystem writes** go through adapters in `internal/adapter/`. Rule packs cannot import them. Enforced by `.go-arch-lint.yaml`.
* **P6 Reversibility.** Every rule has a `stability` field. `experimental` rules can change shape; `stable` rules require an ADR for any user-visible change.
* **P7 Machine-readability.** `--json` output is the contract; `schemas/output.schema.json` is its specification. They must match byte-for-byte under strict validation.

If a change makes archfit fail one of its own rules, fix the violation, or carry a time-limited `ignore` entry in `.archfit.yaml` with a written rationale and an `expires` date. **The Phase 1 score drop from ~100 to ~88 is intentional**: it reflects new rules firing on archfit's own code. Do not paper over those findings — fix them.

---

## 4. Toolchain and dependencies

* Go 1.24+. Do not bump unilaterally.
* No CGO. The release matrix is `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/amd64`.
* Formatting: `gofmt -s` and `goimports`. Non-negotiable.
* Linting: `golangci-lint` with `.golangci.yaml`. New `//nolint` requires a reason comment.
* Boundary linting: `.go-arch-lint.yaml` enforces the import rules in §5.
* External dependencies: prefer the standard library. New deps require a justification line in `docs/dependencies.md` and a comment at the first import site.

Approved network egress at runtime: only the LLM adapter when `--with-llm` is set. Everything else must be local.

Do not introduce:

* Code generators that run implicitly during `go build`. Generation is an explicit `make generate` step with **committed** output.
* `init()` functions that register cross-package state.
* Global mutable state. Pass dependencies through structs.

---

## 5. Repository layout

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
│   │   ├── git/                 # git log sampling (--numstat populated).
│   │   ├── schema/              # JSON-Schema/OpenAPI/protobuf detection.
│   │   ├── depgraph/            # Go import graph.
│   │   ├── command/             # Verification commands (depth=deep only).
│   │   ├── ecosystem/           # CI/deploy/framework/migration detection.
│   │   └── ast/                 # Phase 1 — see §7.1.
│   ├── adapter/                 # Side-effect boundary.
│   │   ├── exec/                # Subprocess execution.
│   │   ├── fs/                  # Filesystem writes (Real + Memory).
│   │   └── llm/                 # Claude / OpenAI / Gemini behind one Client.
│   ├── rule/                    # Engine + Registry. NOT the rules themselves.
│   ├── synth/                   # Phase 1.5 — cross-rule meta-findings (§7.2).
│   ├── policy/                  # Organization policy post-processing.
│   ├── packman/                 # Pack scaffolding and structural validation.
│   ├── fix/                     # Fix engine (plan → snapshot → apply → verify).
│   │   ├── static/              # Deterministic fixers (template-based).
│   │   └── llmfix/              # LLM-assisted fixers.
│   ├── report/                  # Renderers: terminal, json, md, sarif, diff.
│   ├── score/                   # Scoring (severity_class first, overall second).
│   └── version/                 # Embedded build version.
├── packs/                       # Rule packs. Each pack = one vertical slice.
│   ├── core/                    # 14 rules today.
│   └── agent-tool/              # 3 rules today.
├── schemas/                     # rule, output, config, contract, remediation, pr-check.
├── calibration/                 # Phase 1 — corpus + ground truth (§7.3).
├── .claude/
│   └── skills/archfit/          # Project-scope agent skill (auto-discovered).
│       ├── SKILL.md             # ≤ 400 lines, ≤ 10 KB; minimal frontmatter.
│       ├── reference/           # Progressive disclosure, including remediation/.
│       ├── scripts/             # Phase 1 — executable agent helpers (§7.4).
│       └── templates/
├── testdata/e2e/                # End-to-end golden repos and expected output.
├── docs/
│   ├── adr/                     # ADRs (YAML frontmatter required).
│   ├── rules/                   # One human doc per registered rule.
│   ├── self-scan/               # JSON snapshot per release tag (§19).
│   ├── calibration/             # Phase 1 — per-rule precision/recall reports.
│   └── dependencies.md
├── development/                 # Design notes (not user-facing docs).
├── Makefile
├── .archfit.yaml                # archfit's own config, used by `make self-scan`.
├── .golangci.yaml
├── .go-arch-lint.yaml           # Boundary enforcement.
├── go.mod / go.sum
├── PROJECT.md                   # Strategy, status report, roadmap.
└── CLAUDE.md                    # This file.
```

Boundary rule (machine-enforced):

* `packs/*` may import `internal/model` and the public interface of `internal/rule`. Nothing else.
* `packs/*` may not import `internal/adapter`, `internal/collector/command`, `os`, `io/fs`, `os/exec`, or `net/*`. Need a new fact? Add a Collector.
* `internal/synth` (Phase 1.5) sees findings, not file system. It runs after the engine.

---

## 6. Core abstractions

Stable. Breaking changes require an ADR.

```
// internal/model — shapes, not literal source

type Rule struct {
    ID               string           // "P1.LOC.001" — pattern P[1-7]\.[A-Z]{3}\.\d{3}
    Principle        Principle        // P1..P7
    Dimension        string           // 3 uppercase letters, e.g. "LOC"
    Title            string
    Severity         Severity         // info | warn | error | critical
    EvidenceStrength EvidenceStrength // strong | medium | weak | sampled
    Stability        Stability        // experimental | stable | deprecated
    AppliesTo        Applicability    // languages, ecosystems, path globs
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
    Evidence         Evidence          // typed variants from Phase 1.5; Generic today
    Remediation      Remediation
    LLMSuggestion    *LLMSuggestion    // omitempty; only when --with-llm
}
```

Three invariants at all times:

1. **Resolvers receive facts; they do not gather them.** No I/O in resolver code.
2. **Severity ↔ evidence matrix is enforced** (`Rule.Validate`):
   * `severity: critical` requires `evidence_strength: strong` (Phase 1).
   * `severity: error` requires `evidence_strength: strong`.
   * `severity: warn` requires at least `medium`.
   * `severity: info` may be `weak`.
   * Strong claims need strong evidence. CI test `TestSeverityCalibration_*` walks the registry.
3. **Applicability is honored.** A rule whose `applies_to.languages` does not match the repo is **excluded from scoring weight**, not silently passed. Surface it via `--explain-coverage` (Phase 1.5 makes this default-visible in summary).

---

## 7. Phase 1 work in flight

These are the active workstreams Claude Code will most often touch. PROJECT.md §6.1 has full context; this section is the operational handle.

### 7.1 AST collector (ADR 0014)

`internal/collector/ast/` is being introduced to enable rules that look beyond file presence:

```
internal/collector/ast/
├── ast.go                # FactStore extension
├── treesitter/           # Cross-language: Go, TS, Python, Rust, Java, Ruby
│   └── ...
├── goast/                # Go-only deep analysis using go/parser
└── README.md
```

* `--depth=standard`: structural mode (declarations only).
* `--depth=deep`: full body + call graph.
* Cache by content SHA-256 under `.archfit-cache/ast/`.
* File size cap (default 1 MiB) and per-file timeout (default 5 s).
* Parse failures emit `model.ParseFailure` findings; **never silently zero**.

When Claude Code adds a rule that needs AST, it goes through this collector — not by re-parsing inside the resolver.

### 7.2 Cross-rule synthesis (`internal/synth`)

Two-pass evaluation. After the engine produces findings, `synth` runs over them to emit meta-findings (e.g. `META.001` "Compliance theater suspected"). Resolvers in packs do not see other rules' findings — only `synth` does.

### 7.3 Calibration corpus

`calibration/corpus.yaml` lists 10 (Phase 1) → 30 (Phase 1.5) permissively-licensed repositories. `ground_truth/<repo>/expected_findings.yaml` contains hand annotations. A nightly job runs every rule and publishes precision/recall to `docs/calibration/`.

A rule is promoted from `experimental` to `stable` only after it passes a calibration cycle with **precision ≥ 0.85** across the corpus.

### 7.4 Skill scripts

`.claude/skills/archfit/scripts/`:

* `triage.sh` — top-N critical+error findings as JSON.
* `plan_remediation.sh` — proposes fix order, accounting for cross-rule dependencies.
* `apply_safe_fixes.sh` — wraps `archfit fix` for auto-fixable findings only.
* `verify_loop.sh` — fix → re-scan → diff loop, stops on regression.

Each `reference/remediation/<id>.md` carries a structured `decision_tree` block validated by `schemas/remediation.schema.json`.

### 7.5 Score model v2

Output gains:

```json
"scores": {
  "overall": 78.4,
  "by_principle": { "P1": 80, ... },
  "by_dimension": { "P1.LOC": 75, ... },
  "by_severity_class": {
    "critical_pass_rate": 1.0,
    "error_pass_rate": 0.95,
    "warn_pass_rate": 0.80,
    "info_pass_rate": 0.60
  }
}
```

`severity_class.error_pass_rate` is the **primary** signal. `overall` is secondary. The scoring formula uses an evidence factor:

```
contribution_i = passed_i × weight_i × evidence_factor_i
evidence_factor: strong=1.0, medium=0.85, weak=0.7, sampled=0.8
score = 100 × Σ contribution_i / Σ weight_applied_i
```

When you change anything in `internal/score/`, regenerate goldens, validate against `schemas/output.schema.json`, and bump `schema_version` if the change is anything other than additive.

### 7.6 PR mode

`archfit pr-check --base <ref>` scans base in a worktree, scans HEAD, reports only new findings. Exit 1 on any new `error+` finding regardless of `--fail-on`. Schema at `schemas/pr-check.schema.json`. Ships with `.github/actions/archfit-pr-check/action.yml`.

When Claude Code touches PR-mode code, the determinism rules of normal scan apply doubly: two scans run, two outputs are compared, any source of non-determinism becomes a diff bug.

### 7.7 Stability re-tiering (ADR 0013)

Phase 1 walks back ADR 0012's blanket `stable` freeze for rules with known calibration gaps:

* `P1.LOC.003` → `experimental` (max-reach threshold uncalibrated)
* `P1.LOC.004` → `experimental` (commit fan-out threshold uncalibrated)
* `P5.AGG.001` → `experimental` (false positives on fixture/testdata paths)

Rule IDs remain stable. Only behavioral promises move. Do not re-promote without calibration data.

---

## 8. How to add or change a rule (the happy path)

YAML is the source of truth. `make generate` produces Go from YAML.

1. Pick the next ID `P<n>.<DIM>.<nnn>`. Check `make list-rules` for collisions.
2. Create `packs/<pack>/rules/P<n>.<DIM>.<nnn>.yaml`. Validate against `schemas/rule.schema.json`. **Always start at `stability: experimental`**.
3. Honor the severity ↔ evidence matrix (§6 invariant 2). `Rule.Validate` rejects mismatches; do not work around it.
4. Set `applies_to.languages` and `applies_to.ecosystems` accurately. Rules that fire on every repo regardless of language are rare and require explicit justification in the YAML's `rationale`.
5. Implement the resolver in `packs/<pack>/resolvers/`. Pure function of `FactStore`. No I/O.
6. If a new fact is needed, add a Collector first under `internal/collector/<topic>/` with its own tests and a fake. AST-based facts go through `internal/collector/ast/` (§7.1) — do not parse files in the resolver.
7. Add **both** fixtures: `packs/<pack>/fixtures/<id>/positive/` (rule fires) and `.../negative/` (rule does not fire). Each carries `input/` and an `expected.json`.
8. Add a table test in `packs/<pack>/pack_test.go` that runs the rule against both fixtures and diffs `expected.json` byte-for-byte.
9. Add the Go rule declaration in `packs/<pack>/pack.go`. Wire the resolver. Run `make generate`.
10. Add `docs/rules/<id>.md`.
11. Add `.claude/skills/archfit/reference/remediation/<id>.md` with a `decision_tree` block validated by `schemas/remediation.schema.json`.
12. Run `make lint test self-scan`. All three must pass under the refined gate (§19).
13. Open a PR. Note in the description: principle, severity, evidence strength, stability, and the corpus repos you tested against.

A rule moves from `experimental` to `stable` only after one full release cycle **and** corpus precision ≥ 0.85.

Five solid `strong`-evidence rules outweigh fifty `weak` ones.

---

## 9. How to add a Collector

Collectors observe; they do not judge.

* One topic per package under `internal/collector/<topic>/`.
* Provide a `Fake` (in-package) for use by resolver tests.
* Determinism: same input → same output. Sort outputs in a fixed order.
* Limits: guard against pathological repos (size, depth, byte counts, per-file timeouts).
* Wire the new collector into `internal/core/scheduler.go` only — the scheduler is the only seam that knows both sides.
* Extend `model.FactStore` only via ADR. Adding a new accessor is a public-surface change.

---

## 10. Testing strategy

Three layers:

| Layer | Bar | Lives in |
| --- | --- | --- |
| Unit | < 5 s total | next to the code under test |
| Pack | < 20 s total | `packs/<pack>/pack_test.go`, with fixtures in `packs/<pack>/fixtures/` |
| End-to-end | < 60 s total, CI by default | `testdata/e2e/`, runs the real `archfit scan` |

Discipline:

* Property-based tests preferred for `internal/score`, `internal/config`, `internal/contract`, `internal/synth`.
* No tests depending on network, real git remotes, or installed toolchains. All exec goes through `internal/adapter/exec`, which has a fake. AST tests use small in-memory inputs.
* Golden updates: regenerate via `make update-golden` and **review the diff carefully**. Never commit a golden update and a code change in the same commit without a review note.
* New AST-backed rule? **Add a fixture for the parse-failure path** as well as positive and negative. Silent collector failure is a bug.

A change that intentionally alters output requires:

* The corresponding `expected.json` files updated.
* The schema validated against the new output.
* A note in `CHANGELOG.md`.

---

## 11. CLI surface (do not change casually)

```
archfit scan [path]                      # run all enabled rules
archfit pr-check --base <ref> [path]     # PR mode (Phase 1)
archfit check <rule-id> [path]           # run a single rule
archfit score [path]                     # summary only
archfit explain <rule-id>                # show rule docs + remediation
archfit fix [rule-id] [path]             # auto-fix; --plan, --all, --dry-run
archfit init [path]                      # scaffold .archfit.yaml
archfit report [path]                    # Markdown report
archfit diff <baseline.json> [current.json]
archfit trend                            # score / metric trends from --record archives
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

* `--format {terminal|json|md|sarif}` (default `terminal`)
* `--json` (shorthand for `--format=json`)
* `--fail-on {info|warn|error|critical}` (default `error`)
* `--depth {shallow|standard|deep}` (default `standard`)
* `-C <dir>` (work-in directory, like `git -C`)
* `--config <file>` (default `.archfit.yaml` in target)
* `--profile {strict|standard|permissive}`
* `--policy <file>` (organization policy post-processing)
* `--with-llm`, `--llm-backend {claude|openai|gemini}`, `--llm-budget N`
* `--with-llm-mode {metadata|file-snippet|full-context}` (Phase 1.5; `metadata` only today)
* `--record <dir>`, `--explain-coverage`

Exit codes are part of the contract — see `docs/exit-codes.md` and `PROJECT.md` §5.1. Do not change them without an ADR and a major-version bump. Exit code 5 (advisory) is under review (`PROJECT.md` §6.4).

---

## 12. Output format contract

`--json` output conforms to `schemas/output.schema.json`. The schema is versioned via `schema_version`. Agents consume this; silent breakage is the worst kind of regression archfit can ship.

Stability rules:

* Pre-1.0: additive field changes are minor; any rename, removal, or retype is major.
* `findings[]` is sorted by severity desc, then `rule_id` asc, then `path` asc. Renderers do not re-sort.
* Float scores are rounded to one decimal in output; internal math stays in `float64`.
* Every `finding.evidence` carries enough to verify the claim *without* re-running archfit.
* Phase 1 adds `tool`, `config`, `environment`, `scan_id` provenance fields. **Additive — bumps `schema_version` to `1.1.0`.**
* Phase 1.5 introduces typed evidence variants. Existing free-form `evidence` becomes the `Generic` variant. **Additive — bumps to `1.2.0`.**

If you change anything under `internal/report/`: bump `schema_version`, update `schemas/output.schema.json`, regenerate goldens, run schema-validation tests. **All four**, in the same PR.

---

## 13. Agent skill (`.claude/skills/archfit/`)

The skill lives at `.claude/skills/archfit/` because that is the canonical project-scope location per the Agent Skills documentation; Claude Code auto-discovers it inside this repo.

`SKILL.md` carries minimal YAML frontmatter — `name` (lowercase, digits, hyphens, ≤ 64 chars; do not use `anthropic` or `claude` as a name) and `description` (≤ 1024 chars; states *what* and *when*). No other frontmatter fields.

Hard limits:

* `SKILL.md` ≤ 400 lines, ≤ 10 KB. If you bust it, the skill itself fails the principle it teaches.
* Body of `SKILL.md` ≤ ~5k tokens (Level-2 content). Push deeper material to `reference/`.
* Each `reference/remediation/<rule-id>.md` ≤ 100 lines and **must validate against `schemas/remediation.schema.json`**, including a `decision_tree` block (Phase 1).
* `scripts/` files (Phase 1) must be POSIX-compatible shell, no external dependencies beyond `jq` and `archfit` itself.

When you add a rule (§8), you add the corresponding remediation file in the same PR. CI enforces this — there is no warmup grace.

---

## 14. Commit and PR discipline

* Conventional Commits: `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`, `test:`, `pack:`.
* One logical change per PR. A new rule and an engine refactor do not belong together.
* PR description must include:
  + Which principle(s) are touched.
  + Whether `schemas/output.schema.json` changes (and the `schema_version` delta).
  + The result of `make self-scan` and the score delta versus `main`.
  + For new rules: the corpus repos tested against and the precision observed.
* Any change under `internal/model/` or `schemas/` requires an ADR in `docs/adr/` (YAML frontmatter mandatory).
* Changes to exit codes, CLI flag names, or the JSON output schema (beyond additive) require a `BREAKING CHANGE:` footer and a migration note.
* PR size budget: **≤ 500 changed lines, ≤ 5 packages**, unless the PR is a pure rename or generated-code update (label `chore: codegen` or `refactor: rename`).

---

## 15. What Claude Code should do by default

1. **Read the relevant pack's `AGENTS.md` and `INTENT.md` before editing.** State the plan before code. For a new rule, the plan must name the rule ID, the pack, the fixture strategy, and the corpus repos used for sanity-checking.
2. **Prefer editing existing files over creating new ones.** New utility packages need justification.
3. **Run the fast loop after every non-trivial change.** `make lint test` must pass locally. `make self-scan` before claiming a feature complete. Score delta documented.
4. **Respect boundaries.** If a task seems to require a pack importing from `internal/adapter`, propose a Collector instead. If a task seems to require I/O outside `internal/adapter/`, propose adding a new adapter behind an ADR.
5. **Use AST through the collector, never inline.** A resolver that calls `go/parser` directly is a boundary violation. Push the analysis into `internal/collector/ast/` and consume it through `FactStore`.
6. **Refuse implicit magic.** `init()` registration, reflection-based discovery, interface-per-struct factories are out of scope. Push back and propose the explicit alternative.
7. **Write fixtures first for rule work.** Build positive, negative, and (for AST rules) a parse-failure fixture. Write the expected JSON. Then implement the resolver until all three pass.
8. **Keep output deterministic.** Verify under `-race` and with shuffled input order. Non-determinism is a bug. PR mode runs two scans — any non-determinism becomes user-visible.
9. **Honor `applies_to`.** A rule that fires on languages it does not support is worse than no rule. Test the rule against repositories of unsupported languages and confirm it is excluded from scoring.
10. **Ask before widening scope.** Adding a dependency, a new Collector type, a new CLI subcommand, or a new output format is a design decision. Surface it; don't ship it silently.
11. **Phase 1 mindset: the score is *expected* to drop.** New rules should fire on archfit's own code. Fix the violations they expose; don't suppress them just to keep the score green.

When unsure whether a change is in scope, the default is to ask. A concise question with two concrete alternatives is almost always better than a large speculative PR.

---

## 16. What not to do

* Do not add rules with `evidence_strength: weak` and `severity: error` (or above). `Rule.Validate` rejects this; do not work around it.
* Do not couple scoring to rule count. Adding rules must not mechanically lower scores; the formula in §7.5 normalizes per applicable rule set.
* Do not let a rule silently skip on parse failure. Parse failures are findings (`severity: warn`, `evidence_strength: strong`), not zero-finding returns. Use `model.ParseFailure`.
* Do not shell out to tools we can parse ourselves. `git log` is fine (well-specified). `terraform plan` output goes behind an adapter with an interface, never ad-hoc regex in a resolver.
* Do not hardcode paths like `src/` or `internal/` inside resolvers. Path patterns come from `.archfit.yaml`, the rule's `applies_to.path_globs`, or pack-level globs.
* Do not introduce LLM calls on the hot path. Anything that consults an LLM is opt-in via `--with-llm`, lives behind `internal/adapter/llm`, and degrades gracefully when the API call fails. The Phase 1 `metadata` mode is the default; do not promote `file-snippet` or `full-context` to default behavior.
* Do not edit files under `docs/self-scan/` or `docs/calibration/` by hand. Those are generated by release tooling and the nightly calibration job.
* Do not promote a rule from `experimental` to `stable` without corpus precision data. Promotion is a deliberate event with evidence, not a cleanup task.

---

## 17. Definition of done (per task type)

**New rule.**

* YAML in `packs/<pack>/rules/<id>.yaml`, schema-valid, `stability: experimental`.
* Severity ↔ evidence matrix respected; `applies_to` filled in.
* Resolver in `packs/<pack>/resolvers/`. Pure function of `FactStore`.
* Positive fixture + `expected.json`.
* Negative fixture + `expected.json`.
* If AST-based: parse-failure fixture + `expected.json`.
* Table test passing in `pack_test.go`.
* Resolver wired in `packs/<pack>/pack.go` resolver map + `make generate`.
* `docs/rules/<id>.md`.
* `.claude/skills/archfit/reference/remediation/<id>.md` with valid `decision_tree`.
* `make lint test self-scan` clean under the refined gate (§19).
* `CHANGELOG.md` entry.
* PR description records corpus repos used for sanity-checking and the observed false-positive count.

**New Collector.**

* Lives under `internal/collector/<topic>/`.
* Fake implementation for tests in the same package.
* Unit tests against representative fixtures.
* No direct use from packs (only through `FactStore`).
* `model.FactStore` extension with ADR if required.
* Documented in `internal/collector/README.md`.
* Bounded resource use (size cap, timeout) where relevant.

**New output format / schema change.**

* Conforms to a schema under `schemas/`.
* Golden test against `testdata/e2e/`.
* Documented in `docs/outputs.md`.
* Exit-code behavior unchanged unless ADR.
* `schema_version` bumped according to the rules in §12.

**Rule promotion `experimental` → `stable`.**

* Corpus precision ≥ 0.85 with raw data committed under `docs/calibration/<rule-id>/`.
* At least one full release cycle as `experimental`.
* ADR if user-visible behavior changes as part of the promotion.
* CHANGELOG entry calling out the promotion.

**Refactor.**

* `make lint test self-scan` all green.
* No change to `internal/model/` public types without ADR.
* No change to JSON output bytes on golden fixtures without an update note in the PR.

---

## 18. Fast reference

```
make dev            # bootstrap: deps, generate, build
make build          # build the CLI
make test           # unit + pack tests
make e2e            # end-to-end against testdata/
make lint           # gofmt, goimports, golangci-lint, go-arch-lint
make generate       # regenerate code from schemas (committed output)
make self-scan      # archfit scan ./ — must pass under the refined gate
make update-golden  # regenerate expected.json files (review carefully)
make list-rules     # registered rules and their metadata
make list-packs     # registered packs
make calibrate      # Phase 1 — run rules against the corpus, write reports
```

---

## 19. Quality gates (every PR)

* `make lint` < 5 s, clean.
* `make test` < 30 s, clean.
* `make e2e` < 60 s, clean.
* `make self-scan` exits 0.
* **Refined self-scan gate** (replaces the old "score must not drop" rule):
  * `score(PR_HEAD, rules_on_main) >= score(main, rules_on_main)` — the score *attributable to old rules* must not drop.
  * No new `error+` finding from any rule that exists on `main`.
  * Newly introduced rules may produce findings without failing the gate; CI labels them as "expected from new rules X, Y" in the PR comment.
* No new `//nolint` without reason comment.
* No new dependency without justification + `docs/dependencies.md` entry.
* No changes to `internal/model/` without ADR.
* No changes to JSON output schema without `schema_version` consideration.
* No new I/O outside `internal/adapter/`.
* PR ≤ 500 changed lines, ≤ 5 packages touched.
* `CHANGELOG.md` updated when the change is user-visible.
* For new rules: remediation file with valid `decision_tree`, fixtures present, corpus sanity-check noted in PR description.
* For severity changes on existing rules: ADR.
* For stability changes (`experimental` ↔ `stable`): ADR + calibration evidence.

---

## 20. When this file and reality disagree

This file is the contract, not the reality. If you find the repository doing something this file forbids, the first move is **not** to update this file to match. Raise the contradiction, decide which side is right, and only then change one of them.

When this file forbids something currently happening, the contract says the *reality* is wrong. Your job is to help close the gap, not to weaken the contract.

Silent drift between `CLAUDE.md` and the code is exactly the kind of thing archfit itself is meant to prevent.

---

## 21. Change log of this document

* **2026-05-02 — Phase 1 alignment.** Updated §2 status table to reflect Phase 1 in flight (rule expansion to ~27, first `error` severity, score model v2, AST collector, PR mode, skill scripts). Added §6 invariant 2 (severity ↔ evidence matrix) and invariant 3 (applicability honored in scoring). Added §7 "Phase 1 work in flight" with sub-sections for AST collector (7.1), `synth` (7.2), calibration (7.3), skill scripts (7.4), score model (7.5), PR mode (7.6), stability re-tiering (7.7). Tightened §8 to require corpus sanity-check on new rules. Added §15.5 (use AST through the collector), §15.10 (honor applies_to), §15.11 (Phase 1 score-drop is expected). Added §16 prohibitions on promoting rules without calibration data. Added §17 "Rule promotion" definition of done. Refined §19 self-scan gate to allow score drops attributable to newly introduced rules.
* **2026-04-30 — Rewrite as an operational report.** Updated factual fields to match the live repository: Go 1.24+, module path, 17 rules across `core` and `agent-tool`, removed the never-existed `internal/collector/ast/` reference, addition of `internal/contract/`, `internal/policy/`, `internal/packman/`, `internal/fix/`, `internal/adapter/llm/`. Added "Current state at a glance" so Claude Code can orient in one screen. Added "Known integrity gaps Claude Code is expected to help close" — six accepted, tracked violations of the meta-consistency contract with target states. Tightened "Definition of done" to require pair fixtures and remediation docs in the same PR. Added "Quality gates" with measurable bars. Cross-linked `PROJECT.md` rather than duplicating roadmap content here.
