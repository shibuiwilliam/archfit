# archfit — Project Document

> Architecture fitness evaluator for the coding-agent era.
>
> This document is the canonical statement of *what archfit is, where it stands today, what is wrong with it, and how it is being driven toward 1.0*. It is a working report, not marketing copy. The roadmap below incorporates a comprehensive review of the v0.3.x state and explicitly tracks the gaps that review surfaced.

---

## 1. Purpose

Coding agents have shifted the center of gravity in software architecture. "Good design" is no longer only about runtime performance, separation of concerns, and human team boundaries. It is increasingly about properties that determine whether *an agent* can change the system without breaking it:

* **P1 Locality** — can a change be understood from a narrow slice of the repo?
* **P2 Spec-first** — are contracts executable artifacts, not prose?
* **P3 Shallow explicitness** — is behavior visible without ten layers of indirection?
* **P4 Verifiability** — can correctness be proven locally in seconds, not hours?
* **P5 Aggregation of danger** — are risky operations concentrated and guarded?
* **P6 Reversibility** — can any change be rolled back cheaply?
* **P7 Machine-readability** — are outputs, errors, logs, and ADRs readable by agents, not only by humans?

archfit measures these seven properties on a repository and produces a structured report. Its place in the toolchain is **above** linters, formatters, and SAST scanners: it consumes their signals where useful, and reports on the *terrain* a repository presents to coding agents.

archfit is not a linter, not a SAST, not a benchmark, and not a cage. It is a fitness instrument — and, increasingly, an instrument whose signal density must keep pace with how quickly agents are changing what "good architecture" means.

---

## 2. Current state (honest)

archfit is **pre-1.0, active development**, currently `v0.3.x`.

### 2.1 What is implemented

| Area | Implementation |
| --- | --- |
| CLI commands | `scan`, `score`, `check`, `explain`, `fix`, `init`, `report`, `diff`, `trend`, `compare`, `contract check`, `contract init`, `list-rules`, `list-packs`, `validate-config`, `validate-pack`, `new-pack`, `test-pack`, `version` |
| Output formats | `terminal`, `json`, `md`, `sarif` |
| Scan depths | `shallow`, `standard`, `deep` (deep runs verification commands) |
| Rule packs | `core` (14 rules), `agent-tool` (3 rules) |
| Total rules | **17**, all `stability: stable` (frozen per ADR 0012; revisited in §3.7) |
| Collectors | `fs`, `git`, `schema`, `depgraph`, `command`, `ecosystem` |
| Fix engine | scan → plan → snapshot → apply → re-scan → rollback-on-regression; static fixers + LLM fixers |
| LLM adapter | Claude / OpenAI / Gemini behind one `llm.Client`; opt-in via `--with-llm`; cache-backed |
| Metrics | `context_span_p50`, `verification_latency_s`, `invariant_coverage`, `parallel_conflict_rate`, `rollback_signal`, `blast_radius_score` |
| Schemas | `rule.schema.json`, `output.schema.json`, `config.schema.json`, `contract.schema.json` |
| Agent skill | `.claude/skills/archfit/` — Claude Code project-scope skill |
| CI | GitHub Actions: build, lint, test, release, self-scan gating |

### 2.2 Rule inventory (v0.3.x)

| ID | Principle | Severity | Evidence | Stability |
| --- | --- | --- | --- | --- |
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

17 rules across 7 principles.

### 2.3 What is *not* shipping but has been claimed elsewhere

The previous `PROJECT.md` and earlier `README.md` revisions implied capabilities the current binary does not provide. They are removed from this document and tracked under §6 Roadmap as concrete deliverables:

* **HTML output format** — claimed; not implemented. Removed from format list.
* **`web-saas`, `iac`, `mobile`, `data-event` packs** — claimed; not implemented. Tracked as Phase 1.5+.
* **Remote pack registry / community publishing** — claimed as "implemented"; only the local pack scaffolding is. Re-classified as planned.
* **Agent Behavior Observatory** — described as next-up; design notes exist in `development/`, no code.

### 2.4 Phase 0 (integrity) — closed

The seven Phase 0 items from the prior plan are resolved:

1. JSON output ↔ schema conformance (ADR 0009).
2. YAML / Go rule unification (`make generate`, `TestRulesSync_*`).
3. Real YAML parsing for `.archfit.yaml` and `.archfit-contract.yaml` (`sigs.k8s.io/yaml`).
4. `FilesChanged` plumbed via `git log --numstat`; `context_span_p50` and `P1.LOC.004` no longer inert.
5. Rule / docs / remediation consistency CI test.
6. Boundary enforcement via `internal/adapter/fs` and `.go-arch-lint.yaml`.
7. Self-scan in CI with score-delta gating.

The existing surface is honest. The next set of issues — surfaced by review — is about *depth, breadth, and empirical validity*, not integrity.

---

## 3. Quality issues from review (the new report part)

These are the issues that v0.4 onward must address. They are not integrity violations of v0.3.x; they are limits of v0.3.x's *evaluative reach*. The same meta-consistency stance applies — archfit must satisfy the principles it measures *as it grows in reach*.

### 3.1 Rule density is too thin to call this a fitness evaluator

17 rules across 7 principles is roughly 2.4 rules per principle. **P3 (shallow explicitness) has exactly one rule** — and it checks for `.env.example`, which is a near-trivial proxy for the actual property. Concepts that the founding documents identify as central to agent-era architecture are absent from the rule set:

* `INTENT.md` per high-risk context
* Branded / nominal types for domain identifiers
* Property-based testing presence
* Idempotency-Key handling on write APIs
* Bidirectional / expand-contract migrations
* Risk-tier file declaring high-risk paths
* OIDC / Workload Identity (vs long-lived credentials)
* CODEOWNERS coverage on declared high-risk paths
* `init()`-based cross-package registration (Go-specific)
* Reflection / metaprogramming density
* Single-implementation interfaces in domain code
* State machines for long-lived flows
* Boundary 4-way alignment (directory ↔ CODEOWNERS ↔ team ↔ package)
* Structured error shape (`code` / `details` / `remediation`)

Effect: a repository can score 100 on archfit while violating most of the principles archfit names. Phase 1 closes this.

### 3.2 Detection depth is too shallow

Almost every rule in v0.3.x is a file-presence or simple-pattern check. There is no AST collector. CLAUDE.md previously stated that an `internal/collector/ast/` package "is not on the current roadmap." Reviewing this against the rules that *should* exist (see §3.1), this stance is the single largest constraint on archfit's evaluative reach. Without AST, archfit cannot:

* Detect `init()` side effects, reflection density, single-implementation interfaces.
* Find branded type patterns in TS, Rust, Go, Python.
* Verify that boundary parsing actually happens at HTTP handler entry points.
* Distinguish a real `INTENT.md` with a forbidden-actions section from an empty one.
* Measure indirection depth.

Effect: most agent-era architecture properties remain invisible to the tool. Compliance theater (creating empty files to pass rules) is not detected.

Resolution path: §6.1 (Phase 1) introduces `internal/collector/ast/` behind an ADR. Tree-sitter for cross-language, `go/parser` for Go-specific accuracy. Cached by content hash, sized-bounded, and fail-soft on parse error (parse failures emit `ParseFailure` findings, never silently zero).

### 3.3 Severity distribution is uncalibrated

All 17 rules are `warn` or `info`. There are zero `error` and zero `critical` findings possible in any scan. This produces three failure modes:

* **Organizational adoption is weak.** A tool that cannot fail a build at the architectural level looks advisory and gets routed around.
* **Evidence asymmetry hidden.** The strongest claims (e.g. "high-risk paths have no CODEOWNERS reviewer") deserve `error` — the bar is calibrated to the wrong axis.
* **Score volatility.** With 17 rules and uniform low severity, score deltas are noisy.

Effect: archfit looks polite, not serious. Phase 1 introduces a small, well-calibrated set of `error` findings (and one `critical`).

### 3.4 Score model has poor signal-to-noise

With ~17 rules, a single failure moves the overall score 5–10 points. Compounding issues:

* Score 100 means "we found nothing wrong" — not "this repo is well-architected." With more rules, the ceiling becomes meaningful.
* Linear weighted average flattens severity. A `critical` and an `info` contribute on the same axis.
* Metrics (`context_span_p50` etc.) are reported but are not first-class evaluation axes; they are visible only as side effects of certain rules.

Effect: teams that hit 100 stop improving; teams that hit 60 cannot tell which axis is most broken. Phase 1 introduces severity pass-rate as a primary signal and promotes metrics to first-class output.

### 3.5 Empirical validation is missing

There is no calibration corpus, no precision/recall measurement per rule, no published per-repo scoring. Thresholds (e.g. P1.LOC.003's max-reach of 10, P1.LOC.004's 8 files) are reasonable defaults, but they are defaults — not data-driven. The risk: as archfit grows beyond 17 rules, false-positive rates on real codebases will grow proportionally if not measured.

Resolution path: §6.1 builds and publishes a 30-repo corpus with nightly scans, per-rule precision/recall, and a public dashboard.

### 3.6 Language and stack coverage is uneven

Detection is well-developed for Go, Node/TS, Python. It is partial for Java, Ruby, PHP, Rust. It is absent for Swift/Kotlin (mobile), Scala (deep), C/C++, Dart/Flutter, and the IaC ecosystem (Terraform/CDK/Pulumi). `applies_to.languages` was added to `FactStore` in Phase 0, but few rules use it strictly enough — repositories in unsupported languages can pass with most rules silently skipped, and the surface presents as "100" with no warning.

Effect: archfit gives a misleading verdict for stacks it does not understand. Phase 1 enforces strict applicability and `--explain-coverage` becomes part of every scan summary.

### 3.7 ADR 0012's blanket "stable" freeze is premature

ADR 0012 promoted all 17 rules to `stability: stable` and froze rule IDs for 1.0. Reviewing this in light of §3.1 and §3.5: many of the v0.3 rules are at the right level of design, but their *thresholds* (P1.LOC.003: ≤10, P1.LOC.004: ≤8) and their *evidence interpretation* (P5.AGG.001's positive on fixture/testdata paths) are uncalibrated. Freezing them freezes the wrong things and constrains Phase 1 unnecessarily.

Resolution path: §6.1 includes ADR 0013 that downgrades rules with known calibration gaps (P1.LOC.003, P1.LOC.004, P5.AGG.001) to `stability: experimental`, while keeping ID stability. ID-level promises hold; behavioral promises follow data.

### 3.8 Self-scan gate distorts incentives

CLAUDE.md §19 enforces that self-scan score "must not drop on any PR." This rule, well-intended, creates a perverse pressure against expanding the rule set — adding a rule that fires on archfit's own code lowers its score. Phase 0 added `score-gate` CI; Phase 1 must refine it so that *additive rule introduction* is recognized as improvement, not regression.

Resolution path: §6.1 refines the gate. A PR may lower the score iff the lower score is fully explained by newly introduced rules; the score *attributable to old rules* must not drop.

### 3.9 Skill is a document, not an actor

`.claude/skills/archfit/` ships SKILL.md and `reference/` markdown. There are no executable scripts, no decision trees in machine-readable form, no `triage → plan → fix → verify` loop wired up. Agents have to assemble the workflow from prose. This is below the bar archfit itself sets in `agent-tool` rules.

Resolution path: §6.1 ships `scripts/triage.sh`, `scripts/plan_remediation.sh`, `scripts/verify_loop.sh`. Each remediation file gains a structured `decision_tree` block that agents can step through.

### 3.10 LLM enrichment is too cautious to be useful

The `--with-llm` mode sends rule metadata and finding evidence — never source code. Safe, but the LLM has no view of the actual code under question, so its suggestions tend to restate the rule. The differentiator archfit is well-positioned to deliver — *static facts plus LLM context* — is unrealized.

Resolution path: §6.2 introduces `--with-llm-mode={metadata|file-snippet|full-context}` with explicit per-call user confirmation for snippet/full modes, plus a prompt-injection sanitizer in `internal/adapter/llm/`.

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
|  fs, git,     |  |  core, agent-   |  | terminal, json|
|  schema,      |  |  tool, (iac,    |  |  md, SARIF    |
|  depgraph,    |  |  data-event,    |  +-------+-------+
|  command,     |  |  mobile…)       |          |
|  ecosystem,   |  +--------+--------+   +------v---------+
|  ast (P1)     |           |            |  LLM adapter   |
+-------+-------+   +-------v--------+   | Claude/OpenAI/ |
        |           |   Fix engine   |   | Gemini (opt-in)|
        |           | static + LLM   |   +----------------+
        |           +----------------+
+-------v-------+
|  FactStore    |  read-only view passed to resolvers
+---------------+
```

Key invariants:

* Resolvers are pure functions of `FactStore`. They do not perform I/O. New facts require a new collector.
* Rule packs do not import `internal/adapter`. All side effects live there. Enforced by `.go-arch-lint.yaml`.
* All registration is explicit in `cmd/archfit/main.go`. No `init()` auto-discovery. No reflection-based plugin loading.
* JSON output is a versioned contract (§5.2).

The architecture is sound. Phase 1 extends — does not redesign — it: an `ast` collector, an enriched `ecosystem` collector, a `synth` pass for cross-rule meta-findings.

---

## 5. Stability and contracts

archfit is pre-1.0. The contracts below are how we keep moving without breaking consumers.

### 5.1 Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success, or findings below `--fail-on` threshold |
| 1 | Findings at or above threshold, or contract hard-constraint violation |
| 2 | Usage error |
| 3 | Runtime error |
| 4 | Configuration error |
| 5 | Contract soft-target missed (advisory) |

Exit codes change only with an ADR and a major-version bump. **Exit code 5 is under review** (§6.4) — advisory-as-non-zero is friction-heavy in most CI systems. Replacement candidate: a JSON field in scan output, with exit 0.

### 5.2 JSON output schema

* `schemas/output.schema.json` is authoritative.
* `schema_version` in the output identifies the contract version.
* Pre-1.0: minor bumps are additive; any rename, removal, or retype is a major bump.
* `findings[]` order is deterministic: severity desc, then `rule_id` asc, then `path` asc.
* Numeric scores are rounded to one decimal in output; internal math is `float64`.
* Phase 1 introduces typed `evidence` variants (file-presence / file-pattern / ast-pattern / git-sample / command-result / cross-reference); current free-form `map[string]any` becomes one variant.

### 5.3 Configuration schema

`.archfit.yaml` carries a top-level `version:`. Migration notes accompany every breaking change. `ignore` entries require a `reason` and an `expires` date; expired ignores surface as warnings on every scan. Phase 1 introduces v2 with richer `risk_tiers`, `verification`, and `agent_directives` blocks (§8); v1 remains accepted.

### 5.4 Rule schema

`schemas/rule.schema.json` is the source of truth (Phase 0). Go rule definitions are generated from YAML on every build (`make generate`) and CI's `TestRulesSync_*` fails on drift.

### 5.5 Stability tiers

* **Rule ID**: stable from first ship. Renumbering requires a major bump.
* **Rule behavior**: experimental → stable after one full release cycle *and* a passing calibration run on the public corpus (§6.1). Phase 1 walks back ADR 0012's blanket freeze: rules with known calibration gaps (P1.LOC.003, P1.LOC.004, P5.AGG.001) revert to `experimental` while their IDs stay stable. Behavior changes within `experimental` are allowed; within `stable` they require an ADR.

---

## 6. Roadmap

The roadmap is organized by gate, not by version. Phases beyond Phase 0 only begin once the prior gate is closed.

### Phase 0 — Integrity (closed)

See §2.4. All seven items resolved.

### Phase 1 — Sharpness (P0, 0–4 weeks)

**Goal: depth and severity calibration. The tool must be able to distinguish well-architected repositories from those that merely appear to be.**

#### 6.1.1 AST collector

ADR 0014: introduce `internal/collector/ast/`.

* `treesitter/` for cross-language (Go, TypeScript, Python, Rust, Java, Ruby).
* `goast/` for Go-only deep analysis using `go/parser`.
* Content-hashed cache under `.archfit-cache/ast/` (opt-in).
* File size cap (default 1 MiB), per-file timeout (default 5 s).
* Parse failures emit `ParseFailure` findings; never silently zero.
* `--depth=standard` runs structural mode (declarations only); `--depth=deep` runs full body analysis.

#### 6.1.2 Rule expansion to ~30 (preserve ID stability for the 17)

Rule additions, all initially `stability: experimental`:

| New ID | Principle | Title | Detection | Evidence | Severity |
| --- | --- | --- | --- | --- | --- |
| P1.LOC.005 | P1 | High-risk paths declare `INTENT.md` | file_presence on `risk_tiers.high` paths | strong | warn |
| P1.LOC.006 | P1 | `AGENTS.md`/`CLAUDE.md` not bloated (≤400 lines, ≤10 KB) | file_metrics | strong | warn |
| P1.LOC.007 | P1 | Boundary 4-way alignment (dir ↔ CODEOWNERS ↔ go.mod) | cross_reference | medium | warn |
| P1.LOC.009 | P1 | `runbook.md` exists for each high-risk slice | file_presence | strong | warn |
| P2.SPC.002 | P2 | DB migrations are bidirectional | migration_pattern | strong | warn |
| P2.SPC.004 | P2 | ADR uses YAML frontmatter | parsed_frontmatter | strong | info |
| P2.SPC.005 | P2 | Branded / nominal types in domain layer | ast_pattern | medium | info |
| P3.EXP.002 | P3 | No `init()` cross-package registration (Go) | ast_pattern | strong | warn |
| P3.EXP.003 | P3 | Reflection / metaprogramming density bounded | ast_pattern | medium | info |
| P3.EXP.004 | P3 | Single-implementation interfaces flagged (Go) | ast_pattern | weak | info |
| P3.EXP.005 | P3 | Global mutable state minimized | ast_pattern | medium | info |
| P5.AGG.003 | P5 | Risk-tier file declared | file_presence | strong | warn |
| **P5.AGG.004** | **P5** | **High-risk paths protected by CODEOWNERS** | **cross_reference** | **strong** | **error** |
| P5.AGG.005 | P5 | Idempotency-Key handling on write APIs | ast_pattern | weak | info |
| P5.AGG.006 | P5 | Long-lived static credentials avoided (OIDC) | ecosystem | medium | warn |
| P6.REV.003 | P6 | Feature flag actually wired to changed code | ast_pattern | medium | info |
| P6.REV.005 | P6 | Soft-delete pattern for user-facing entities | schema_pattern | weak | info |

Phase 1 ships **at least 10 of these 17 candidates**, including P3.EXP.002, P3.EXP.005, P5.AGG.003, P5.AGG.004, P1.LOC.005, P1.LOC.006. Severity distribution post-Phase 1:

```
critical:  0
error:     1   (P5.AGG.004)
warn:    ~14
info:    ~12
```

#### 6.1.3 Severity calibration enforcement

`Rule.Validate` already rejects `severity ≥ error` paired with `evidence_strength: weak`. Phase 1 also rejects `severity: critical` paired with anything below `strong`, and adds CI test `TestSeverityCalibration_*` that walks the registry.

#### 6.1.4 Stability re-tiering (ADR 0013)

Walk back the blanket-stable freeze for rules with known calibration gaps:

* `P1.LOC.003` (max reach ≤10) → `experimental` until corpus data sets a defensible threshold.
* `P1.LOC.004` (commit fan-out ≤8) → `experimental` for the same reason.
* `P5.AGG.001` (deploy-artifact false positive on fixtures/testdata) → `experimental`; resolver fix is a Phase 1 deliverable.

Rule IDs remain stable. Only behavioral promises move.

#### 6.1.5 Calibration corpus v0

Build `calibration/` with 10 permissively-licensed repositories covering Go, TS, Python, Rust, Java, IaC. Nightly job runs every rule, computes precision/recall against hand-annotated `ground_truth/`, publishes a Markdown report under `docs/calibration/`. Initial bar: precision ≥ 0.85 to promote a rule from `experimental` to `stable`.

Candidate corpus (final list in `calibration/corpus.yaml`):

```
gin-gonic/gin            — Go, web framework
fastapi/fastapi          — Python, web framework
charmbracelet/bubbletea  — Go, CLI
strapi/strapi            — TS/Node, large monorepo
posthog/posthog          — TS/Python mixed
hashicorp/terraform-aws-modules  — Terraform
sharkdp/bat              — Rust, CLI
mlflow/mlflow            — Python, MLOps
ktor/ktor                — Kotlin, server
expo/expo                — TS/JS, mobile
```

#### 6.1.6 Score model v2

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

`severity_class.error_pass_rate` becomes the **primary** signal — 1.0 is the wall every PR must clear. `overall` becomes a continuous secondary indicator. Documentation and skill rewritten around this distinction.

Score formula refined:

```
contribution_i = passed_i × weight_i × evidence_factor_i
score = 100 × Σ contribution_i / Σ weight_applied_i

evidence_factor:
  strong → 1.0
  medium → 0.85
  weak   → 0.70
  sampled → 0.80
```

This prevents `weak`-evidence rules from dominating score deltas.

#### 6.1.7 PR mode (pulled in from previous Phase 2)

`archfit pr-check --base <ref>` — scans base ref in a git worktree, scans HEAD in working dir, reports only new findings. Ships with `.github/actions/archfit-pr-check/action.yml`. Schema at `schemas/pr-check.schema.json`. Exits 1 on any new `error`+ finding regardless of `--fail-on`.

#### 6.1.8 Skill becomes executable

`.claude/skills/archfit/scripts/`:

```
triage.sh             # archfit scan --json | jq filter critical+error, top 5
plan_remediation.sh   # propose fix order accounting for dependencies between rules
apply_safe_fixes.sh   # invoke `archfit fix` for auto-fixable findings only
verify_loop.sh        # fix → re-scan → diff loop, stops on regression
```

Each `reference/remediation/<id>.md` adopts a structured `decision_tree` block. Schema at `schemas/remediation.schema.json` so coverage and shape are machine-checkable. CI fails on missing `decision_tree`.

#### 6.1.9 Self-scan gate refinement

Old gate: "score must not drop." Replaced with:

```
A PR passes the self-scan gate iff:
  score(PR_HEAD, rules_on_main) >= score(main, rules_on_main)
  AND there are no new error+ findings produced by rules that exist on main.
Newly introduced rules may produce findings without failing the gate;
those findings appear in the PR comment as "expected from new rules X, Y".
```

This removes the disincentive against rule expansion that the prior gate created.

#### 6.1.10 Provenance fields in output

Output gains `tool`, `config`, `environment`, `scan_id`. `schema_version` bumps to `1.1.0` (additive). Existing consumers continue working; new fields are reference material (e.g. `scan_id` becomes the primary key for `record/diff/trend`).

#### Phase 1 deliverables summary

* AST collector (ADR 0014)
* +10 rules (rule total ~27), severity now includes 1 `error`
* Stability re-tiering (ADR 0013)
* Calibration corpus v0 (10 repos) + nightly precision/recall
* Score model v2 with severity pass rates
* PR mode shipping
* Skill scripts shipping
* Self-scan gate refined
* Output schema 1.1 (additive)

Self-scan score on archfit itself is expected to drop from 100 (under the old 17-rule set) to ~88 under the new 27-rule set. **This is the intended outcome.** The drop is documented in `docs/self-scan/` with the breakdown by new rule. archfit's honesty about its own architecture is the marketing.

### Phase 1.5 — Coverage (P1, 4–10 weeks)

Goal: expand beyond `core` and `agent-tool`, raise corpus quality, ship the first external pack.

* **Rule expansion to ~44.** Add the remaining Phase 1 candidates plus 5–7 new rules driven by corpus signals (false-positive analysis of v0.4 will identify gaps).
* **`iac` pack — first external pack.** Eight rules covering layered IaC (raw / hardened / blueprint / app-stack), policy-as-code presence, plan-only-for-PRs, remote/locked state, secret-by-reference, module versioning. Ships as a separately-versioned pack to validate the publishing workflow end-to-end.
* **Calibration corpus v1.** Grow to 30 repositories. Publish per-rule precision/recall dashboard at `https://shibuiwilliam.github.io/archfit/calibration/`.
* **`Applies_to` strict mode.** When a rule's `applies_to.languages` does not match the repo, the rule is excluded from scoring weight (it does not silently pass). `--explain-coverage` becomes part of every scan summary terminal output, not just an opt-in flag.
* **Cross-rule synthesis (`internal/synth`).** Two-pass evaluation produces meta-findings:
  * `META.001` "Compliance theater suspected"
  * `META.002` "High-risk paths not under defense in depth"
* **Typed evidence variants.** Output schema bumps to `1.2.0`. Existing free-form `evidence` becomes the `Generic` variant; new findings emit typed variants.
* **LLM mode expansion.** `--with-llm-mode={metadata|file-snippet|full-context}` with per-snippet confirmation. Prompt-injection sanitizer in `internal/adapter/llm/sanitizer.go`. Documented in `docs/llm.md`.

### Phase 2 — Reach (P2, 10–24 weeks)

Goal: scale evaluation to monorepos, large repos, and deep verification.

* **Monorepo / workspace mode.** `archfit scan --workspace` understands pnpm/yarn workspaces, Cargo workspaces, Go workspaces, Nx, Turborepo. Per-package scores aggregate to a workspace score; per-package config and pack selection.
* **Incremental scan.** `archfit scan --since=<ref>` evaluates only rules whose `applies_to.path_globs` intersect the changed file set. Baseline cache under `.archfit-cache/`.
* **Stratified verification.** `verification:` block in `.archfit.yaml` declares named layers with commands and timeouts. `CollectLayers` runs them in order (fail-fast). Per-layer `verification_latency_s.<layer>` metrics emitted. Rules P4.VER.005/006 land here.
* **Second external pack: `data-event`.** Schema registry, idempotency, DLQ, replay harness, outbox.
* **Parallel resolver execution.** Bounded goroutine pool when `len(rules) >= 8`. Per-rule slots merged in deterministic order. 100-iteration race test enforces.
* **Adaptive Rule Engine v0.** Local-only telemetry: `archfit feedback <rule-id> --suppress` with a reason field, recorded under `.archfit-stats.json`. Effective confidence per rule is updated from the local distribution. No remote telemetry without explicit opt-in.

### Phase 3 — Toward 1.0 (P3, ongoing)

* **Rule ID freeze.** All rules promoted to `stability: stable` once they pass a calibration cycle. `TestStability_AllRulesAreStable` becomes a hard CI gate.
* **JSON output schema v1.** `schema_version: "2.0.0"`. Field set frozen. Migration guide at `docs/migration/1.x-to-2.0.md`.
* **Exit code 5 resolution.** Either keep with documented CI integration patterns or replace with a JSON field — decided by ADR 0015 informed by Phase 1.5 user feedback.
* **Public API statement.** ADR 0016 documents the frozen surface: rule IDs, schema fields, exit codes, CLI commands/flags, config schema.
* **`mobile` pack.** SwiftUI / Jetpack Compose / React Native — view-logic separation, screenshot diff, OS-capability adapters, state machines for offline sync.
* **OpenTelemetry export.** `--metrics-otlp=<endpoint>` exports scan execution as OTel traces; per-rule spans for performance debugging.

### Research tracks (not on the roadmap until Phase 2 closes)

* **Agent Behavior Observatory** (`development/agent-observatory.md`) — observe what real agents do on a repo and feed behavioral metrics back into scoring.
* **Adaptive Rule Engine v1** — organization-level telemetry behind explicit opt-in. Tunes thresholds globally.

---

## 7. Quality bars

These are the bars every PR must clear. They are enforced in CI and in `make` targets, not by review alone.

| Gate | Bar |
| --- | --- |
| `make lint` | < 5 s |
| `make test` | < 30 s, all packages |
| `make e2e` | < 60 s |
| `make self-scan` | exit 0 under the refined gate (§6.1.9) |
| Output schema validation | every `expected.json` validates against `schemas/output.schema.json` |
| Rule / docs consistency | every registered rule has `docs/rules/<id>.md` and a remediation guide |
| Remediation structure | every remediation file passes `schemas/remediation.schema.json` |
| Severity calibration | no rule violates the severity ↔ evidence matrix |
| Boundary check | `go-arch-lint` passes |
| Self-scan trend | published per release tag in `docs/self-scan/` |
| Calibration trend | per-rule precision ≥ 0.85 before promotion to `stable` |
| PR size | ≤ 500 changed lines, ≤ 5 packages, unless explicitly labeled |

A PR that adds a new rule additionally requires:

* YAML in `packs/<pack>/rules/`
* Resolver in `packs/<pack>/resolvers/`
* Positive fixture
* Negative fixture
* Expected JSON
* `docs/rules/<id>.md`
* `.claude/skills/archfit/reference/remediation/<id>.md` with valid `decision_tree`
* `stability: experimental`
* Severity ↔ evidence matrix respected
* A note in `CHANGELOG.md`

A PR that promotes a rule from `experimental` to `stable` additionally requires:

* Calibration data showing precision ≥ 0.85 across the corpus
* At least one full release cycle with the rule active
* ADR if the rule's user-visible behavior is changing as part of promotion

---

## 8. Configuration sketch

```yaml
version: 2                                  # v1 still accepted
project_type: [web-saas]
profile: standard

risk_tiers:
  critical:
    paths: ["src/auth/**", "migrations/**"]
    require_codeowners: true
    require_intent_md: true
    require_runbook: true
  high:
    paths: ["src/billing/**", "infra/**"]
    require_codeowners: true
  medium:
    paths: ["src/features/**"]
  low:
    paths: ["docs/**", "tests/**"]

packs:
  enabled: [core, agent-tool]
  external:
    - source: github.com/shibuiwilliam/archfit-pack-iac@v0.1.0
      enabled: true

verification:                                # Phase 2
  lint:        { command: "make lint",       timeout_s: 5,    layer: 1 }
  typecheck:   { command: "make typecheck",  timeout_s: 10,   layer: 1 }
  unit:        { command: "make test",       timeout_s: 60,   layer: 2 }
  integration: { command: "make e2e",        timeout_s: 300,  layer: 3 }

agent_directives:                            # Phase 1.5
  forbidden_paths: ["secrets/**", "third_party/vendored/**"]
  caution_paths:   ["src/auth/**"]
  default_review_required: ["src/billing/**"]

overrides:
  P4.VER.003:
    timeout_seconds: 60

ignore:
  - rule: P5.AGG.001
    paths: ["src/legacy/**"]
    reason: "Legacy area on a documented migration path"
    expires: 2026-09-30
```

The `verification:` block is a Phase 2 deliverable. The `agent_directives:` block ships in Phase 1.5. v1 configurations continue to work without changes.

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

# PR mode (Phase 1 deliverable)
- name: PR check
  uses: shibuiwilliam/archfit/actions/pr-check@v0
  with:
    base: origin/main
    fail_on_error: true        # always fail on new error+ findings
```

Exit-code 5 (advisory) is **not** treated as a CI failure. Phase 3 will revisit this.

---

## 10. What archfit is not

* Not a replacement for `golangci-lint`, `ruff`, `eslint`, or any other language-specific linter.
* Not a SAST tool. Use Semgrep, CodeQL, or Trivy. archfit can *consume* their outputs but does not duplicate them.
* Not a benchmark for cross-repo competition. Scores are signals about *your* repository over time. Calibration corpus scores are published for transparency, not ranking.
* Not a cage. Suppression is a feature; expiry is the discipline.
* Not a tool that runs untrusted repositories without a sandbox. `git log` runs on the target; `--depth=deep` runs configured commands; `--with-llm-mode=file-snippet` (Phase 1.5) sends snippets to a third-party LLM with explicit confirmation.

---

## 11. Security

See `SECURITY.md` for reporting instructions.

archfit performs read-only filesystem access by default and shells out to `git log` against the scanned repository. `--depth=deep` runs verification commands defined in the project. Treat untrusted repositories with a sandbox.

`--with-llm` (Phase 1 mode `metadata`) sends *rule metadata and finding evidence* to the configured provider. **Source code and file contents are not transmitted in the default mode.** Phase 1.5 introduces `file-snippet` and `full-context` modes that do send source; both require explicit per-call confirmation and pass through `internal/adapter/llm/sanitizer.go` for prompt-injection scrubbing. The exact contract lives in `docs/llm.md` and is tested as part of the LLM adapter's golden cases.

---

## 12. Contributing

Before opening a PR, read in this order:

1. `CLAUDE.md` — operational rules for changes.
2. `CONTRIBUTING.md` — workflow, commit conventions, PR-size budget.
3. `docs/authoring-rules.md` — the golden path for adding a rule.
4. The relevant pack's `AGENTS.md` and `INTENT.md`.

High-value contributions, in rough priority:

1. Anything in §6.1 Phase 1 (sharpness work) — especially AST-based rules and corpus expansion.
2. Pair fixtures (negative cases) for newly added rules.
3. Calibration ground-truth annotations for repositories already in the corpus.
4. New rules that satisfy the §7 quality bars.
5. Translations of the skill's `reference/` documentation.
6. External packs (`iac`, `data-event`, `mobile`) once Phase 1.5 lands.

All contributors follow the `CODE_OF_CONDUCT.md`.

---

## 13. Acknowledgments

archfit is an attempt to make measurable a set of architectural prerequisites that converged across Anthropic, OpenAI, and GitHub coding-agent documentation, and that intersect with NIST SSDF, SLSA, and OPA-style policy work. The seven-principle decomposition, the meta-consistency stance, and any errors in execution are this project's responsibility.

---

## 14. License

Apache License 2.0. See `LICENSE`.

---

## 15. Change log of this document

* **2026-05-02 — Phase 1 plan grounded in review findings.** Added §3 Quality issues from review (10 issues, replacing the prior integrity list now that Phase 0 is closed). Re-scoped Phase 1 from "Coverage" to "Sharpness" with concrete deliverables: AST collector (ADR 0014), +10 rules including the first `error` severity (P5.AGG.004), score model v2, PR mode pulled in from previous Phase 2, skill scripts, refined self-scan gate, output provenance. Added §3.7 walking back ADR 0012's blanket-stable freeze for rules with calibration gaps (ADR 0013). Reorganized the roadmap into Phase 1 (sharpness), Phase 1.5 (coverage), Phase 2 (reach), Phase 3 (toward 1.0). Added severity-calibration and remediation-structure to §7 quality bars. Configuration sketch updated for v2 with `agent_directives`. README and IDEA.md realignment around "score 100 is not the goal; severity_class.error_pass_rate = 1.0 is" tracked separately.
* **2026-04-30 — Rewrite as a working report.** Removed claims that did not match the current binary (HTML output, "implemented" remote pack registry, missing remediation guides). Added Known integrity issues. Re-organized the roadmap into Phase 0 (integrity), Phase 1 (coverage), Phase 2 (reach), Phase 3 (toward 1.0). Added Quality bars as enforceable gates rather than aspirations.
