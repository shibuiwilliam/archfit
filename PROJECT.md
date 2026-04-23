# archfit

> **Architecture fitness evaluator for the coding-agent era.**
> Measure how well your repository is shaped for coding agents to work on *safely* and *quickly*.

[![Go Reference](https://pkg.go.dev/badge/github.com/your-org/archfit.svg)](https://pkg.go.dev/github.com/your-org/archfit)
[![CI](https://github.com/your-org/archfit/actions/workflows/ci.yml/badge.svg)](https://github.com/your-org/archfit/actions)
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

**Early development.** archfit is pre-1.0. The rule IDs, configuration schema, and JSON output are stabilizing but may still change. See [`CHANGELOG.md`](./CHANGELOG.md) and the "Stability" section below.

---

## Install

### From release binaries

```bash
# Linux/macOS
curl -sSL https://github.com/your-org/archfit/releases/latest/download/archfit_$(uname -s)_$(uname -m).tar.gz \
  | tar -xz -C /usr/local/bin archfit

# Verify
archfit --version
```

Pre-built binaries are provided for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, and `windows/amd64`.

### From source

```bash
go install github.com/your-org/archfit/cmd/archfit@latest
```

Requires Go 1.23 or newer. No CGO.

### Via Homebrew

```bash
brew install your-org/tap/archfit
```

### Via Docker

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/your-org/archfit:latest scan /repo
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

Each principle expands into dimensions, and dimensions into concrete rules with IDs like `P5.RSK.002`. See [`docs/principles.md`](./docs/principles.md) and [`docs/rules/`](./docs/rules/) for the full catalog.

---

## How archfit works

```
          ┌──────────────────────────────┐
          │          archfit CLI          │
          └───────────────┬───────────────┘
                          │
     ┌────────────────────┼────────────────────┐
     │                    │                    │
┌────▼─────┐    ┌─────────▼────────┐    ┌──────▼──────┐
│ Collectors│    │  Rule Packs      │    │  Renderers  │
│  fs,git,  │    │  core,web-saas,  │    │ term,json,  │
│  ast,exec │    │  iac,mobile,…    │    │ md,sarif,html│
└───────────┘    └──────────────────┘    └─────────────┘
```

- **Collectors** gather facts from the filesystem, git history, ASTs, command execution, and schema files. They do not judge — they only observe.
- **Rule packs** contain rules declared as YAML plus Go resolvers. Resolvers receive a read-only `FactStore` and emit findings.
- **Renderers** produce human- and machine-readable output.

Collectors and rule packs are separated so resolvers never perform I/O. This enforces archfit's own locality and aggregation principles on itself.

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

GitHub Actions:

```yaml
- uses: your-org/archfit-action@v1
  with:
    fail-on: error
    depth: standard
    sarif-output: archfit.sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: archfit.sarif
```

Examples for GitLab CI, CircleCI, and Buildkite are in [`docs/ci/`](./docs/ci/).

---

## Rule packs

| Pack | Covers |
|---|---|
| `core` | Principles that apply to every repository |
| `web-saas` | OpenAPI/GraphQL contracts, branded types, tenant isolation, feature flags |
| `mobile` | View/logic separation, state machines for flows, snapshot tests, OS-capability adapters |
| `desktop` | IPC surfaces, auto-update safety, credential store handling |
| `iac` | Terraform/CDK layering, policy-as-code, plan/apply separation, drift detection |
| `data-event` | Schema registry, idempotency, DLQ, replay harnesses |
| `agent-tool` | For tools meant to be used by agents: `--json` output, structured errors, versioned schemas |

Packs are opt-in via `.archfit.yaml`. You can write your own — see [`docs/authoring-packs.md`](./docs/authoring-packs.md).

---

## Metrics

archfit emits a set of metrics alongside findings. These are the numbers worth tracking over time, more than the overall score:

- **`context_span_p50`** — median number of files touched in a typical change
- **`verification_latency_s`** — time to run typecheck, lint, and unit tests
- **`invariant_coverage`** — fraction of stated invariants that are machine-enforced
- **`parallel_conflict_rate`** — merge-conflict rate across recent PRs
- **`rollback_signal`** — revert-commit frequency and time-to-revert
- **`blast_radius_score`** — estimated maximum reach of a change in each area

See [`docs/metrics.md`](./docs/metrics.md) for definitions and how they are computed.

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

See [`docs/stability.md`](./docs/stability.md).

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

Report security issues privately to `security@your-org.example` or via GitHub's private vulnerability reporting. Do not open public issues for vulnerabilities. See [`SECURITY.md`](./SECURITY.md) for details and our disclosure timeline.

archfit executes commands like `go build`, `make test`, and `git log` against the repository being scanned. Run it only on repositories you trust, or inside a sandbox. The `--depth=shallow` mode performs no command execution and is safe for untrusted input.

---

## Roadmap

Tracked in the [`Roadmap`](https://github.com/your-org/archfit/projects) project board. Near-term priorities:

- **0.x → 0.y**: Stabilize rule IDs in `core` and `web-saas` packs. Freeze JSON schema at v1.
- **1.0**: Public API stability. Complete metric definitions. SARIF output certified against GitHub Code Scanning.
- **Post-1.0**: Remote rule-pack registry, organization policy profiles, `.agent-trace/` ingestion for replay-based findings.

---

## Acknowledgments

archfit grew out of an analysis of how Anthropic, OpenAI, and GitHub's coding-agent documentation converges on a consistent set of architectural prerequisites, and how standards like NIST SSDF, SLSA, and OPA translate those prerequisites into enforceable safety controls. The tool is an attempt to make those prerequisites *measurable*.

---

## License

Apache License 2.0. See [`LICENSE`](./LICENSE).
