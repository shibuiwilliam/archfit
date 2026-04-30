# archfit — Project Document

> Architecture fitness evaluator for the coding-agent era.
>
> This document is the canonical statement of *what archfit is, where it stands today, what is wrong with it, and how it is being driven toward 1.0*.
> It supersedes the previous `PROJECT.md` and is intended to be a working report, not marketing copy.

---

## 1. Purpose

Coding agents have shifted the center of gravity in software architecture. "Good design" is no longer only about runtime performance, separation of concerns, and human team boundaries. It is increasingly about properties that determine whether *an agent* can change the system without breaking it:

- **P1 Locality** — can a change be understood from a narrow slice of the repo?
- **P2 Spec-first** — are contracts executable artifacts, not prose?
- **P3 Shallow explicitness** — is behavior visible without ten layers of indirection?
- **P4 Verifiability** — can correctness be proven locally in seconds, not hours?
- **P5 Aggregation of danger** — are risky operations concentrated and guarded?
- **P6 Reversibility** — can any change be rolled back cheaply?
- **P7 Machine-readability** — are outputs, errors, logs, and ADRs readable by agents, not only by humans?

archfit measures these seven properties on a repository and produces a structured report. Its place in the toolchain is **above** linters, formatters, and SAST scanners: it consumes their signals where useful, and reports on the *terrain* a repository presents to coding agents.

archfit is not a linter, not a SAST, not a benchmark, and not a cage. It is a fitness instrument.

---

## 2. Current state (honest)

archfit is **pre-1.0, active development**, currently `v0.3.x`.

### 2.1 What is implemented

| Area | Implementation |
|---|---|
| CLI commands | `scan`, `score`, `check`, `explain`, `fix`, `init`, `report`, `diff`, `trend`, `compare`, `contract check`, `contract init`, `list-rules`, `list-packs`, `validate-config`, `validate-pack`, `new-pack`, `test-pack`, `version` |
| Output formats | `terminal`, `json`, `md`, `sarif` |
| Scan depths | `shallow`, `standard`, `deep` (deep runs verification commands) |
| Rule packs | `core` (11 rules), `agent-tool` (3 rules) |
| Total rules | **14**, all `stability: experimental` |
| Collectors | `fs`, `git`, `schema`, `depgraph`, `command` |
| Fix engine | scan → plan → snapshot → apply → re-scan → rollback-on-regression; 7 static fixers + LLM fixers |
| LLM adapter | Claude / OpenAI / Gemini behind one `llm.Client`; opt-in via `--with-llm`; cache-backed |
| Metrics | `context_span_p50`, `verification_latency_s`, `invariant_coverage`, `parallel_conflict_rate`, `rollback_signal`, `blast_radius_score` |
| Schemas | `rule.schema.json`, `output.schema.json`, `config.schema.json`, `contract.schema.json` |
| Agent skill | `.claude/skills/archfit/` — Claude Code project-scope skill |
| CI | GitHub Actions: build, lint, test, release |

### 2.2 Rule inventory

| ID | Principle | Severity | Evidence | Stability |
|---|---|---|---|---|
| P1.LOC.001 | Locality | warn | strong | stable |
| P1.LOC.002 | Locality | warn | strong | stable |
| P1.LOC.003 | Locality | info | medium | stable |
| P1.LOC.004 | Locality | info | sampled | stable |
| P2.SPC.001 | Spec-first | warn | strong | stable |
| P2.SPC.010 | Spec-first | warn | strong | stable |
| P3.EXP.001 | Shallow explicitness | warn | strong | stable |
| P4.VER.001 | Verifiability | warn | strong | stable |
| P4.VER.002 | Verifiability | info | medium | stable |
| P4.VER.003 | Verifiability | info | strong | stable |
| P5.AGG.001 | Aggregation | warn | strong | stable |
| P5.AGG.002 | Aggregation | warn | strong | stable |
| P6.REV.001 | Reversibility | warn | strong | stable |
| P6.REV.002 | Reversibility | info | strong | stable |
| P7.MRD.001 | Machine-readability | warn | strong | stable |
| P7.MRD.002 | Machine-readability | warn | strong | stable |
| P7.MRD.003 | Machine-readability | warn | strong | stable |

17 rules across 7 principles, all `stability: stable` (frozen per ADR 0012). Rule IDs are immutable from 1.0 onward.

### 2.3 What is *not* shipping but has been claimed elsewhere

The previous `PROJECT.md` and earlier `README.md` revisions implied capabilities that the current binary does not provide. They are removed from this document and tracked under §6 Roadmap as concrete deliverables:

- **HTML output format** — claimed; not implemented. Removed from format list.
- **`web-saas`, `iac`, `mobile`, `data-event` packs** — claimed; not implemented. Tracked as Phase 2.
- **Remote pack registry / community publishing** — claimed as "implemented"; only the local pack scaffolding is. Re-classified as planned.
- **Agent Behavior Observatory** — described as next-up; design notes exist in `development/`, no code.

---

## 3. Known integrity issues (the report part)

This section is the reason this document calls itself a *report*. archfit's central proposition is **meta-consistency**: the tool must satisfy the principles it measures. A periodic, public list of where we currently violate that promise is more valuable than a clean status badge.

The issues below were identified by reading the source against the documents (`CLAUDE.md`, `PROJECT.md`, `README.md`) and against the output schema. They are tracked as P0/P1 work in §6.

### 3.1 P2 violation — rule definitions are split between YAML and Go and drift

`packs/core/rules/` contains 4 YAML files (`P1.LOC.001`, `P1.LOC.002`, `P4.VER.001`, `P7.MRD.001`). `packs/core/pack.go` declares 11 rules in Go. `packs/agent-tool/rules/` has 3 YAML files matching the Go, but no loader reads any of them. The schema in `schemas/rule.schema.json` exists but does not validate the live rule set on every build.

Effect: archfit's central spec-first claim is currently aspirational. New rules can be added to Go without a YAML record, with no CI signal.

Resolution path: Phase 0 (§6.1).

### 3.2 P7 violation — JSON output does not match its own schema

`internal/report/report.go` emits `summary.rules_with_findings`. `schemas/output.schema.json` declares `additionalProperties: false` on `summary` and lists only three permitted fields. Strict validation of any current scan output therefore fails.

Effect: agents that validate against the published schema treat every archfit run as malformed.

Resolution path: Phase 0 (§6.1).

### 3.3 P3 violation — `.archfit.yaml` is parsed as JSON

`internal/config/config.go` and `internal/contract/contract.go` decode their files with `encoding/json`. JSON is a syntactic subset of YAML 1.2, so a JSON document in `.archfit.yaml` round-trips, but anything idiomatic to YAML — comments, unquoted strings, anchors, block scalars — fails to parse. The file extension and the parser disagree.

Effect: a user writing a real `.archfit.yaml` (with comments) gets a JSON syntax error and no obvious next step.

Resolution path: Phase 0 (§6.1) — adopt `gopkg.in/yaml.v3` or `sigs.k8s.io/yaml` and document the dependency.

### 3.4 Dead metric — `context_span_p50` and rule `P1.LOC.004` never fire

`internal/score/metrics.go` and `packs/core/resolvers/locality_changeset.go` both depend on `model.Commit.FilesChanged`. The git collector (`internal/collector/git/git.go`) calls `git log --pretty=format:%H\t%s` and never `--numstat`, so `FilesChanged` is always 0 and the metric/rule are silently inert.

Effect: a flagship locality metric is reported but is structurally meaningless. The corresponding rule has 0% recall.

Resolution path: Phase 0 (§6.1).

### 3.5 Documentation drift — skill claims rules it does not document

`.claude/skills/archfit/SKILL.md` advertises remediation guides for "all 14 rules". `.claude/skills/archfit/reference/remediation/` contains 10 files. Missing: `P1.LOC.003`, `P1.LOC.004`, `P4.VER.002`, `P4.VER.003`.

Effect: an agent following the skill expects files that aren't there.

Resolution path: Phase 0 (§6.1).

### 3.6 Boundary enforcement promised, not active

`CLAUDE.md` states that `internal/adapter` may not be imported from rule packs and that this is enforced by `go-arch-lint` via `.go-arch-lint.yaml`. No such configuration is present. `internal/fix/engine.go` performs `os.WriteFile` directly rather than going through an `adapter/fs` helper, so the "all writes go through adapter" claim is partially violated already.

Effect: the architectural rule that protects archfit's own correctness is held by code review alone.

Resolution path: Phase 0 (§6.1).

---

## 4. Architecture

```
            +-----------------------------+
            |          archfit CLI         |
            +--------------+--------------+
                           |
        +------------------+------------------+
        |                  |                  |
+-------v-------+  +-------v---------+  +-----v---------+
|  Collectors   |  |   Rule Packs    |  |   Renderers   |
|  fs, git,     |  |  core (11)      |  | terminal, json|
|  schema,      |  |  agent-tool (3) |  |  md, SARIF    |
|  depgraph,    |  +--------+--------+  +-------+-------+
|  command      |           |                   |
+-------+-------+   +-------v--------+   +------v---------+
        |           |   Fix engine   |   |  LLM adapter   |
        |           | static + LLM   |   | Claude/OpenAI/ |
        |           +----------------+   | Gemini (opt-in)|
        |                                +----------------+
+-------v-------+
|  FactStore    |  read-only view passed to resolvers
+---------------+
```

Key invariants:

- **Resolvers are pure functions of `FactStore`.** They do not perform I/O. New facts require a new collector.
- **Rule packs do not import `internal/adapter`.** All side effects live there.
- **All registration is explicit in `cmd/archfit/main.go`.** No `init()` auto-discovery. No reflection-based plugin loading.
- **JSON output is a versioned contract.** Field changes follow the rules in §5.2.

The architecture itself is sound. The integrity issues in §3 are about fidelity between what the architecture *promises* and what the code currently *implements*.

---

## 5. Stability and contracts

archfit is pre-1.0. The contracts below are how we keep moving without breaking consumers.

### 5.1 Exit codes

| Code | Meaning |
|---|---|
| 0 | Success, or findings below `--fail-on` threshold |
| 1 | Findings at or above threshold, or contract hard-constraint violation |
| 2 | Usage error |
| 3 | Runtime error |
| 4 | Configuration error |
| 5 | Contract soft-target missed (advisory) |

Exit codes change only with an ADR and a major-version bump. **Exit code 5 is under review** (§6.2): treating advisory state as a non-zero exit code is friction-heavy with most CI systems and may be replaced with a JSON field in a future minor release.

### 5.2 JSON output schema

- `schemas/output.schema.json` is authoritative.
- `schema_version` in the output identifies the contract version.
- Pre-1.0: minor bumps are additive; any rename, removal, or retype is a major bump.
- `findings[]` order is deterministic: severity desc, then `rule_id` asc, then `path` asc.
- Numeric scores are rounded to one decimal in output; internal math is `float64`.

### 5.3 Configuration schema

`.archfit.yaml` carries a top-level `version:`. Migration notes accompany every breaking change. `ignore` entries require a `reason` and an `expires` date; expired ignores surface as warnings on every scan.

### 5.4 Rule schema

`schemas/rule.schema.json` defines the YAML shape. Once Phase 0 lands (§6.1), Go rule definitions are generated from or validated against this schema on every build, removing the drift described in §3.1.

---

## 6. Roadmap

The roadmap is organized by gate, not by version. Phases 1 and 2 only begin once the prior gate is closed.

### Phase 0 — Integrity (P0, 0–4 weeks)

**Goal: archfit must satisfy archfit at the level its own documents already promise.** Until this is done, every other improvement compounds existing inconsistency.

- [x] **JSON output / schema reconciliation.** Added `rules_with_findings` to `output.schema.json`, bumped `schema_version` to `0.2.0`, added CI schema conformance test. See [ADR 0009](./docs/adr/0009-output-schema-rules-with-findings.md).
- [x] **YAML / Go rule unification.** YAML under `packs/<pack>/rules/` is the source of truth. `make generate` produces `generated_rules.go` (committed). `pack.go` merges generated metadata with resolver map. CI test `TestRulesSync_*` in `internal/packman/` fails if sets diverge.
- [x] **Real YAML parsing.** Adopted `sigs.k8s.io/yaml` for `.archfit.yaml` and `.archfit-contract.yaml`. JSON configs continue to work (YAML 1.2 superset). Documented in `docs/dependencies.md`. Tests cover comments, unquoted strings, and unknown-field rejection.
- [x] **`FilesChanged` plumbing.** Git collector now calls `git log --numstat` and populates `Commit.FilesChanged`. Unit tests cover 3-file, merge, and binary commits. `context_span_p50` reports non-zero values; `P1.LOC.004` fires when median exceeds 8.
- [x] **Skill / docs / rule consistency.** CI test `TestDocsSync_AllRulesHaveDocumentation` in `internal/packman/` enforces both files exist for every registered rule. SKILL.md no longer hardcodes a rule list — it points to `archfit list-rules`.
- [x] **Boundary enforcement.** `internal/adapter/fs` added with `Real` and `Memory` implementations. `internal/fix/engine.go` refactored to use the adapter. `.go-arch-lint.yaml` updated with all components and boundary rules.
- [x] **Self-scan in CI.** `make self-scan` runs on every PR. `score-gate` CI job fails if PR drops overall score by > 1.0 and posts a delta comment. `docs/self-scan/` carries one JSON snapshot per release. `make self-scan-record` generates it locally.

**Phase 0 is closed.** All seven integrity items are resolved. The existing surface is honest.

### Phase 1 — Coverage (P1, 4–10 weeks)

**Goal: raise rule coverage where it is most lopsided, without increasing false-positive rates.**

- [x] **Pair fixtures.** Every rule has a `*-negative/` fixture alongside its positive fixture. `TestCorePack_PairFixtures` and `TestAgentToolPack_PairFixtures` fail CI if either is missing. Rules requiring runtime data (P1.LOC.003, P1.LOC.004) are documented exceptions. Known false positive: P5.AGG.001 counts fixture/testdata paths as real deploy artifacts — resolver fix deferred.
- [ ] **Calibration corpus.** Curate a small set (10–30) of permissively-licensed open-source repositories. A nightly job runs every rule and tracks precision/recall per rule across the corpus. Findings drive rule tuning.
- [ ] **New rules — P2 (spec-first).** Candidates: API-boundary contract presence (OpenAPI / GraphQL / protobuf), bidirectional DB migrations, ADR with YAML frontmatter, JSON Schema for tool outputs.
- [ ] **New rules — P5 (aggregation).** Candidates: secret-scanner CI presence (gitleaks, trufflehog), policy-as-code presence for IaC (Conftest, Checkov), Idempotency-Key handling on write APIs.
- [ ] **New rules — P6 (reversibility).** Candidates: feature-flag library dependency, expand/contract migration pattern, canary/blue-green configuration in deploy manifests.
- [x] **`Applies_to` activation.** `Languages()` added to `FactStore` (ADR 0010). Rule engine skips rules whose `applies_to.languages` don't match the repo. Skipped rules excluded from scoring weight. P3.EXP.001 tagged with supported languages. `genrules` emits `AppliesTo`.
- [x] **Ecosystem collector.** `internal/collector/ecosystem` centralizes CI, deployment, and framework detection. `Ecosystems()` added to `FactStore` (ADR 0011). Resolvers `verifiability_ci`, `aggregation_secrets`, and `explicitness` migrated to use it. Single-pass detection replaces duplicate file walks.

### Phase 2 — Reach (P2, 10–24 weeks)

**Goal: scale evaluation to monorepos, PR workflows, and depth-stratified verification.**

- [ ] **Monorepo / workspace mode.** `archfit scan --workspace` understands pnpm/yarn workspaces, Cargo workspaces, Go workspaces, Nx/Turborepo. Per-package scores aggregate to a workspace score.
- [x] **PR mode.** `archfit pr-check --base <ref>` scans base in a git worktree, scans head in working dir, diffs, reports only new findings. Ships with `.github/actions/archfit-pr-check/action.yml`. Schema at `schemas/pr-check.schema.json`.
- [x] **Stratified verification (PR A).** `.archfit.yaml` `verification:` block declares named layers with commands and timeouts. `CollectLayers` runs them in order (fail-fast). Per-layer `verification_latency_s.<layer>` metrics emitted. Rules P4.VER.005/006 deferred to PR B.
- [ ] **First external pack.** Ship one of `web-saas`, `iac`, or `data-event` as a separately-versioned pack. Document the pack-publishing workflow end to end.
- [x] **Parallel resolver execution.** Bounded goroutine pool (semaphore = NumCPU) when len(rules) >= 8. Per-rule slots merged in order for deterministic output. 100-iteration race test enforces. Documented in `development/perf.md`.

### Phase 3 — Toward 1.0 (P3, ongoing)

- [x] **Rule ID freeze.** All 17 rules promoted to `stability: stable`. `TestStability_AllRulesAreStable` CI gate prevents regression. No renumbering without 2.0.
- [x] **JSON output schema v1.** `schema_version: "1.0.0"`. Field set frozen. Migration guide at `docs/migration/0.x-to-1.0.md`.
- [x] **Contract check exit-code revisit.** Exit code 5 documented in `docs/exit-codes.md`. ADR 0012 freezes codes 0–5.
- [x] **Public API statement.** ADR 0012 (`docs/adr/0012-1.0-stability.md`) documents the frozen surface: rule IDs, schema fields, exit codes, CLI commands/flags, config schema.
- [ ] **Cross-stack improvements.** Java, Ruby, PHP, Terraform — track in `development/cross-stack-improvements.md`.

The two strategic ideas previously listed under "Next: Three Strategic Elements" are kept as **research tracks**, not roadmap items, until Phase 0 closes:

- **Agent Behavior Observatory** (`development/agent-observatory.md`) — observing what real agents do on a repo and feeding behavioral metrics back into scoring.
- **Adaptive Rule Engine** (`development/adaptive-engine.md`) — using fix outcomes and suppress history to tune confidence and thresholds.

The Fitness Contract (`development/fitness-contract.md`) is partially shipped; its CLI is wired and its agent-skill integration is under §6.2 Phase 1.

---

## 7. Quality bars

These are the bars every PR must clear. They are enforced in CI and in `make` targets, not by review alone.

| Gate | Bar |
|---|---|
| `make lint` | < 5 s |
| `make test` | < 30 s, all packages |
| `make e2e` | < 60 s |
| `make self-scan` | exit 0; overall score must not drop versus `main` |
| Output schema validation | every `expected.json` validates against `schemas/output.schema.json` |
| Rule / docs consistency | every registered rule has `docs/rules/<id>.md` and a remediation guide |
| Boundary check | `go-arch-lint` (or equivalent) passes |
| Self-scan trend | published per release tag in `docs/self-scan/` |
| PR size | ≤ 500 changed lines, ≤ 5 packages, unless explicitly labeled |

A PR that adds a new rule additionally requires:

- YAML in `packs/<pack>/rules/`
- Resolver in `packs/<pack>/resolvers/`
- Positive fixture
- Negative fixture
- Expected JSON
- `docs/rules/<id>.md`
- `.claude/skills/archfit/reference/remediation/<id>.md`
- `stability: experimental`
- A note in `CHANGELOG.md`

---

## 8. Configuration sketch

```yaml
version: 1
project_type: [web-saas]
profile: standard
risk_tiers:
  high:    ["src/auth/**", "src/billing/**", "infra/**", "migrations/**"]
  medium:  ["src/features/**"]
  low:     ["docs/**", "tests/**"]
packs:
  enabled: [core, agent-tool]
verification:                    # Phase 2
  lint:        { command: "make lint",       timeout_s: 5 }
  typecheck:   { command: "make typecheck",  timeout_s: 10 }
  unit:        { command: "make test",       timeout_s: 60 }
  integration: { command: "make e2e",        timeout_s: 300 }
overrides:
  P4.VER.003:
    timeout_seconds: 60
ignore:
  - rule: P5.AGG.001
    paths: ["src/legacy/**"]
    reason: "Legacy area on a documented migration path"
    expires: 2026-09-30
```

The `verification:` block is a Phase 2 deliverable; current builds ignore it.

---

## 9. CI integration

```yaml
# Strict gating
- name: Build archfit
  run: go install github.com/shibuiwilliam/archfit/cmd/archfit@latest
- name: Scan
  run: archfit scan --format=sarif . > archfit.sarif
- uses: github/codeql-action/upload-sarif@v3
  with: { sarif_file: archfit.sarif }

# PR-only mode (Phase 2 once landed)
- name: Baseline
  run: archfit scan --json . > baseline.json
- name: Diff
  run: archfit diff baseline.json    # exit 1 on new findings
```

Exit-code 5 (advisory) is **not** treated as a CI failure. Phase 3 will revisit whether to keep it as an exit code or surface it solely in JSON.

---

## 10. What archfit is not

- Not a replacement for `golangci-lint`, `ruff`, `eslint`, or any other language-specific linter.
- Not a SAST tool. Use Semgrep, CodeQL, or Trivy. archfit can *consume* their outputs but does not duplicate them.
- Not a benchmark for cross-repo competition. Scores are signals about *your* repository over time.
- Not a cage. Suppression is a feature; expiry is the discipline.
- Not a tool that runs untrusted repositories without a sandbox. `git log` runs on the target; `--depth=deep` runs configured commands.

---

## 11. Security

See `SECURITY.md` for reporting instructions.

archfit performs read-only filesystem access by default and shells out to `git log` against the scanned repository. `--depth=deep` runs verification commands defined in the project. Treat untrusted repositories with a sandbox.

`--with-llm` sends *rule metadata and finding evidence* to the configured provider. **Source code and file contents are not transmitted.** The exact contract lives in `docs/llm.md` and is tested as part of the LLM adapter's golden cases.

---

## 12. Contributing

Before opening a PR, read in this order:

1. `CLAUDE.md` — operational rules for changes.
2. `CONTRIBUTING.md` — workflow, commit conventions, PR-size budget.
3. `docs/authoring-rules.md` — the golden path for adding a rule.
4. The relevant pack's `AGENTS.md` and `INTENT.md`.

High-value contributions, in rough priority:

1. Anything in §6.1 Phase 0 (integrity work).
2. Pair fixtures (negative cases) for existing rules.
3. New rules that satisfy the §7 quality bars.
4. Calibration repositories for the §6.2 corpus.
5. Translations of the skill's `reference/` documentation.

All contributors follow the `CODE_OF_CONDUCT.md`.

---

## 13. Acknowledgments

archfit is an attempt to make measurable a set of architectural prerequisites that converged across Anthropic, OpenAI, and GitHub coding-agent documentation, and that intersect with NIST SSDF, SLSA, and OPA-style policy work. The seven-principle decomposition and the meta-consistency stance are this project's responsibility.

---

## 14. License

Apache License 2.0. See `LICENSE`.

---

## 15. Change log of this document

- **2026-04-30 — Rewrite as a working report.** Removed claims that did not match the current binary (HTML output, "implemented" remote pack registry, missing remediation guides). Added §3 Known integrity issues. Re-organized the roadmap into Phase 0 (integrity), Phase 1 (coverage), Phase 2 (reach), Phase 3 (toward 1.0). Added §7 Quality bars as enforceable gates rather than aspirations.
