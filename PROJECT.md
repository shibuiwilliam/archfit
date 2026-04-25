# archfit

> **Architecture fitness evaluator for the coding-agent era.**
> Measure how well your repository is shaped for coding agents to work on *safely* and *quickly*.

[![CI](https://github.com/shibuiwilliam/archfit/actions/workflows/ci.yml/badge.svg)](https://github.com/shibuiwilliam/archfit/actions)
[![Self-scan](https://img.shields.io/badge/self--scan-passing-brightgreen)](./docs/self-scan.md)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](./LICENSE)

---

## Why archfit exists

Coding agents have shifted the center of gravity in software architecture. "Good design" is no longer only about runtime performance, separation of concerns, and human team boundaries. It is increasingly about properties that determine whether an *agent* can change the system without breaking it:

- **Locality** — can a change be understood from a narrow slice of the repo?
- **Spec-first** — are contracts executable artifacts, not prose?
- **Shallow explicitness** — is behavior visible without following ten indirections?
- **Verifiability** — can correctness be proven locally in minutes, not hours?
- **Aggregation of dangerous capabilities** — are risky operations concentrated and guarded?
- **Reversibility** — can any change be rolled back quickly?
- **Machine-readability** — are errors, logs, ADRs, and CLIs readable by machines?

Most existing tools check *code*. archfit checks the **terrain** the code sits on — the shape of the repository, the contracts at its boundaries, the safety of its defaults, the latency of its feedback loops. It tells you whether your repository gives coding agents a place where they can succeed.

archfit is not a linter. It is not a security scanner. It sits **above** those tools, consumes their signals where useful, and reports on architectural properties they do not measure.

---

## What archfit does

archfit scans a repository and produces a structured report with:

- A **score per principle** (P1–P7) and an overall score
- A list of **findings**, each with evidence, severity, confidence, and remediation
- A set of **metrics**: context span, verification latency, invariant coverage, blast-radius score, and more
- A **diff mode** for tracking how a repository's fitness changes over time
- A **Claude agent skill** so agents can run archfit, read the output, and act on it

Output is available as terminal text, Markdown, JSON, SARIF, and HTML.

---

## Status

**Active development (v0.3.x).** archfit is pre-1.0 with 10 rules across 2 packs, a working fix engine, multi-provider LLM support, SARIF output, metrics pipeline, and a Claude Code agent skill. Rule IDs, configuration schema, and JSON output are stabilizing but may still change. See [`CHANGELOG.md`](./CHANGELOG.md) and the "Stability" section below.

---

## Install

### From source

```bash
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit
make build
./bin/archfit version
```

Requires **Go 1.24+**. No CGO. Cross-compiles to `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, and `windows/amd64`.

### From release binaries

```bash
curl -sSL https://github.com/shibuiwilliam/archfit/releases/latest/download/archfit-<version>-linux-amd64.tar.gz \
  | tar xz
./archfit version
```

### Via Docker

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

---

## Quick start

```bash
# In your repository
cd my-project
archfit init           # scaffold .archfit.yaml
archfit scan           # run the default scan
archfit score          # just the numbers
archfit explain P5.RSK.002   # learn about a specific rule
```

Scan in a CI job and produce JSON for downstream tooling:

```bash
archfit scan --json --fail-on=error > archfit.json
```

Track fitness over time:

```bash
archfit scan --json > main.json          # on main
archfit scan --json > pr.json            # on a PR
archfit diff main.json pr.json --format=markdown
```

---

## The principles archfit evaluates

| Principle | What it asks |
|---|---|
| **P1. Locality** | Can a change be understood and verified from a narrow slice of the repo? |
| **P2. Spec-first** | Are contracts executable artifacts — schemas, types, generated clients — rather than prose? |
| **P3. Shallow explicitness** | Is behavior visible without following reflection, metaclasses, or deep indirection? |
| **P4. Verifiability** | Can correctness be proven locally in seconds to a few minutes? |
| **P5. Aggregation of danger** | Are auth, billing, migrations, and infra *concentrated* and protected? |
| **P6. Reversibility** | Can every change be rolled back cheaply? Is blast radius bounded? |
| **P7. Machine-readability** | Are outputs, errors, logs, and ADRs readable by agents, not just humans? |

Each principle expands into dimensions, and dimensions into concrete rules with IDs like `P1.LOC.001`. See [`docs/rules/`](./docs/rules/) for the full catalog.

---

## How archfit works

```
          +------------------------------+
          |          archfit CLI          |
          +--------------+---------------+
                         |
     +-------------------+---------------------+
     |                   |                      |
+----v------+   +--------v---------+   +--------v-------+
| Collectors|   |    Rule Packs    |   |   Renderers    |
| fs, git,  |   |  core (7 rules)  |   | terminal, json,|
|  schema,  |   |  agent-tool (3)  |   | md, SARIF 2.1.0|
| depgraph, |   +--------+---------+   +--------+-------+
|  command  |            |                      |
+-----------+   +--------v-------+   +----------v--------+
                |   Fix Engine   |   |   LLM Adapter     |
                | 7 static fixers|   | Claude | OpenAI   |
                | + LLM fixers   |   | Gemini            |
                +----------------+   +-------------------+
                                       (opt-in only)
```

- **Collectors** gather facts from the filesystem, git history, schemas, dependency graphs, and command execution. They observe; they do not judge.
- **Rule packs** contain rules and resolver functions. Resolvers are pure functions of a read-only `FactStore` — no I/O.
- **Fix engine** produces deterministic file changes for each finding, then re-scans to verify. Static fixers handle templates; LLM fixers generate contextual content when `--with-llm` is set.
- **Renderers** produce output in multiple formats. JSON conforms to `schemas/output.schema.json`; SARIF 2.1.0 integrates with GitHub Code Scanning.
- **LLM adapter** is the single network boundary. Three backends behind one `llm.Client` interface. Only engaged via `--with-llm`.

Rule registration is explicit in `cmd/archfit/main.go`. No reflection, no `init()` auto-discovery.

---

## Evidence, not verdict

Every finding archfit produces carries four qualities:

- **Severity** — how bad is it if true? (`info` / `warn` / `error` / `critical`)
- **Evidence strength** — how deterministic is the detection? (`strong` / `medium` / `weak` / `sampled`)
- **Confidence** — a numeric 0.0–1.0
- **Remediation** — what to do, with auto-fix when available

archfit is deliberately conservative: `error` severity requires strong evidence. Heuristic findings are clearly marked. The tool is designed to inform decisions, not to override them. **False positives are treated as bugs.**

---

## Claude agent skill

archfit ships with a Claude Code agent skill in [`.claude/skills/archfit/`](./.claude/skills/archfit/) — the canonical project-scope skill location per the [Agent Skills docs](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview), so Claude Code auto-discovers it when invoked inside the repo. The skill includes:

- A short `SKILL.md` entry point (under 400 lines, on purpose)
- Per-rule remediation guides the agent loads on demand
- Decision trees for interpreting findings
- Templates for `INTENT.md`, `AGENTS.md`, `context.yaml`, and `runbook.md`

To use it in another repo, copy or symlink `.claude/skills/archfit/` into that project's `.claude/skills/` directory (or into `~/.claude/skills/` for personal use across every repo). The skill drives the CLI, reads `--json`, and proposes changes that are verifiable by re-running the scan.

---

## Configuration

archfit is configured via `.archfit.yaml` at the repository root:

```yaml
version: 1
project_type: [web-saas]
profile: standard
risk_tiers:
  high:    ["src/auth/**", "src/billing/**", "infra/**", "migrations/**"]
  medium:  ["src/features/**"]
  low:     ["docs/**", "tests/**"]
packs:
  enabled: [core, web-saas, agent-tool]
overrides:
  P4.VER.003:
    timeout_seconds: 60
ignore:
  - rule: P3.MAG.002
    paths: ["src/legacy/**"]
    reason: "Legacy area on a documented migration path"
    expires: 2026-09-30
```

`ignore` entries require a `reason` and `expires`; expired ignores surface as warnings so suppressions cannot silently rot. Full reference in [`docs/configuration.md`](./docs/configuration.md).

---

## CI integration

### SARIF for GitHub Code Scanning

```yaml
- name: Build archfit
  run: go install github.com/shibuiwilliam/archfit/cmd/archfit@latest

- name: Scan
  run: archfit scan --format=sarif . > archfit.sarif

- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: archfit.sarif
```

### PR gate on new findings only

```yaml
- name: Baseline (main)
  run: archfit scan --json . > baseline.json

- name: Diff (PR)
  run: archfit diff baseline.json   # exits 1 when new findings appear
```

### Auto-fix in CI

```yaml
- name: Fix and commit
  run: |
    archfit fix --all .
    git diff --quiet || git commit -am "chore: archfit auto-fix"
```

---

## Rule packs

### Implemented

| Pack | Rules | Covers |
|---|---|---|
| `core` | 7 (P1, P3, P4, P5, P6, P7) | Principles that apply to every repository |
| `agent-tool` | 3 (P2, P7) | For tools consumed by agents: versioned schemas, CHANGELOG, ADRs |

### Planned

| Pack | Covers |
|---|---|
| `web-saas` | OpenAPI/GraphQL contracts, branded types, tenant isolation, feature flags |
| `iac` | Terraform/CDK layering, policy-as-code, plan/apply separation, drift detection |
| `mobile` | View/logic separation, state machines for flows, snapshot tests, OS-capability adapters |
| `data-event` | Schema registry, idempotency, DLQ, replay harnesses |

Packs are opt-in via `.archfit.yaml`. You can write your own with `archfit new-pack <name>`.

---

## Metrics

archfit emits a set of metrics alongside findings. These are the numbers worth tracking over time, more than the overall score:

- **`context_span_p50`** — median number of files touched in a typical change
- **`verification_latency_s`** — time to run typecheck, lint, and unit tests
- **`invariant_coverage`** — fraction of stated invariants that are machine-enforced
- **`parallel_conflict_rate`** — merge-conflict rate across recent PRs
- **`rollback_signal`** — revert-commit frequency and time-to-revert
- **`blast_radius_score`** — estimated maximum reach of a change in each area

Metrics are computed from collected facts and included in the JSON output. See [`development/metrics-and-scoring.md`](./development/metrics-and-scoring.md) for computation details.

---

## Design principles of archfit itself

archfit must score well under its own scan. Concretely:

- **Locality**: each rule pack is a vertical slice with its own `AGENTS.md`, `INTENT.md`, tests, and fixtures.
- **Spec-first**: rules are declared in YAML validated by JSON Schema; Go types track the schema, not the other way around.
- **Shallow explicitness**: no reflection-based auto-registration, no cross-package `init()` side effects, no interface-per-struct factories.
- **Verifiability**: `make test` under 30s, `make lint` under 5s.
- **Aggregation**: all exec, filesystem writes, and network calls live in `internal/adapter/` and are unavailable to rule packs.
- **Reversibility**: every rule has a `stability` field; experimental rules are off by default.
- **Machine-readability**: JSON output conforms to a versioned schema in [`schemas/`](./schemas/).

Self-scan results are published in [`docs/self-scan.md`](./docs/self-scan.md) and updated on every release.

---

## Stability guarantees

archfit follows semantic versioning from 1.0 onward. Until then, expect:

- **Rule IDs**: may be renumbered before 1.0 with release notes.
- **Configuration schema** (`.archfit.yaml`): changes are versioned via the top-level `version:` field; migration notes accompany each change.
- **JSON output schema**: versioned via `schema_version` in output. Additive changes are minor; renames/removals are major.
- **CLI flags and exit codes**: exit codes are contract and will not change without a major version bump.

See [`docs/exit-codes.md`](./docs/exit-codes.md) for exit code contract details.

---

## What archfit is not

- Not a replacement for `golangci-lint`, `ruff`, `eslint`, or any other language-specific linter.
- Not a SAST tool. Use Semgrep, CodeQL, or Trivy for that. archfit can *consume* their outputs but does not duplicate them.
- Not a benchmarking tool. The score is a signal for *your own* repository over time, not a number to compete on.
- Not a cage. Rules are advice backed by evidence. Suppression exists on purpose.

---

## Contributing

Contributions are welcome. Before opening a PR please read:

1. [`CLAUDE.md`](./CLAUDE.md) — how this repository is built and the rules that apply to changes.
2. [`CONTRIBUTING.md`](./CONTRIBUTING.md) — workflow, commit conventions, PR size budget (≤500 lines, ≤5 packages).
3. [`docs/authoring-rules.md`](./docs/authoring-rules.md) — the golden path for adding a rule.

High-value contributions, in rough priority:

- New rule packs for ecosystems we do not yet cover well
- Fixture repositories for under-tested rules
- Remediation guides for existing rules
- Translations of the skill's `reference/` docs
- Real-world case studies (anonymized) to calibrate severities

All contributors are expected to follow the [Code of Conduct](./CODE_OF_CONDUCT.md).

---

## Security

See [`SECURITY.md`](./SECURITY.md) for reporting instructions.

archfit runs `git log` against the scanned repo. Use a sandbox for untrusted repos. `--with-llm` sends rule metadata and finding evidence to the LLM provider — **source code and file contents are never sent**. Full contract: [`docs/llm.md`](./docs/llm.md).

---

## Roadmap

### Completed

- **0.1.0**: Foundation, `core` pack (7 rules), JSON/Markdown output, self-scan, schemas.
- **0.2.0**: `init`/`check`/`report`/`diff`, SARIF 2.1.0, `agent-tool` pack (3 rules), e2e tests, CI.
- **0.3.x**: Multi-provider LLM (Claude, OpenAI, Gemini). P3/P5/P6 rules. `--config` flag. `archfit fix` with 7 static fixers + LLM-assisted fixers. Dockerfile + release workflow. `archfit trend` and `archfit compare`. Pack SDK (`new-pack`, `validate-pack`, `test-pack`). Organization policy profiles (`--policy`). Metrics pipeline (6 metrics). Dependency graph and command collectors.

### Infrastructure: Remediation, Metrics, and Ecosystem

These capabilities are largely implemented and provide the foundation for the strategic elements.

- **Fix engine** (implemented): `archfit fix` with 7 static fixers + LLM-assisted fixers. Scan → fix → verify loop with automatic rollback. Remaining: fix conflict resolution, richer LLM prompts.
- **Metrics pipeline** (implemented): 6 metrics (`context_span_p50`, `verification_latency_s`, `invariant_coverage`, `parallel_conflict_rate`, `rollback_signal`, `blast_radius_score`), `archfit trend`, `archfit compare`.
- **Ecosystem platform** (implemented): Pack SDK, organization policies (`--policy`), cross-repo compare. Remaining: remote pack registry, community publishing.

### Next: Three Strategic Elements

Three capabilities transform archfit from a point-in-time audit tool into continuous architecture infrastructure. See `development/` for detailed implementation guides.

#### Element 1 — Agent Behavior Observatory

Watch real agents work on a repo, measure what actually causes failures, and feed behavioral metrics back into scoring. This is a genuinely new category — no competitor observes the *agent* as it works.

- **Trace collection**: ingest agent session traces (files read/written, commands run, retries)
- **Behavioral metrics**: `agent_context_efficiency`, `agent_retry_rate`, `agent_time_to_first_verify`, `agent_cross_boundary_reads`, `agent_dangerous_touches`, `agent_rollback_frequency`
- **Hotspot analysis**: cross-reference behavioral data with static findings to identify areas where agents struggle most

Status: not started. See [`development/agent-observatory.md`](./development/agent-observatory.md).

#### Element 2 — Fitness Contract as Code

Move from "run archfit and read the report" to "declare fitness requirements in a machine-executable contract that agents, CI, and IDEs consume continuously."

- **`.archfit-contract.yaml`**: hard constraints (must satisfy), soft targets (aspirational), area budgets (SRE-style finding budgets per path), agent directives (machine-readable instructions for coding agents)
- **Contract enforcement**: `archfit contract check` for CI gating
- **Agent integration**: Claude Code skill reads the contract before starting work, respects area budgets, follows directives

Status: contract types and checking logic implemented (`internal/contract/`). CLI wiring next. See [`development/fitness-contract.md`](./development/fitness-contract.md).

#### Element 3 — Adaptive Rule Engine

Let archfit learn from fix outcomes, suppress history, and repo characteristics to tune confidence and thresholds automatically.

- **Fix outcome tracking**: extend audit log with repo signals
- **Adaptive confidence**: adjust finding confidence based on fix success rates and suppress rates
- **Threshold adaptation**: scale numeric thresholds (e.g., P5.AGG.001's `maxTopLevelDirs`) based on repo size
- **Community telemetry** (future): anonymized, opt-in data sharing for cross-project calibration

Status: not started. See [`development/adaptive-engine.md`](./development/adaptive-engine.md).

### Toward 1.0

- Stabilize rule IDs in `core` and `agent-tool` packs.
- Freeze JSON output schema at v1.
- SARIF output certified against GitHub Code Scanning.
- Public API stability statement.
- Additional packs: `web-saas`, `iac`, `mobile`, `data-event`.
- Cross-stack detection improvements (Java, Ruby, PHP, Terraform — see [`development/cross-stack-improvements.md`](./development/cross-stack-improvements.md)).

---

## Acknowledgments

archfit grew out of an analysis of how Anthropic, OpenAI, and GitHub's coding-agent documentation converges on a consistent set of architectural prerequisites, and how standards like NIST SSDF, SLSA, and OPA translate those prerequisites into enforceable safety controls. The tool is an attempt to make those prerequisites *measurable*.

---

## License

Apache License 2.0. See [`LICENSE`](./LICENSE).
