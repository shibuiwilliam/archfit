![archfit logo](./archfit_logo.png)

# archfit

> **Architecture fitness evaluator for the coding-agent era.**
> Is your repository shaped for coding agents to work on it — *safely* and *quickly*?

![CI](https://github.com/shibuiwilliam/archfit/actions/workflows/ci.yml/badge.svg)
![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)

[日本語](./README.ja.md)

---

Most tools check your *code*. archfit checks the **terrain** your code sits on —
the entry points an agent reads first, the speed of the feedback loop it relies
on, and the places where a single bad change could quietly break everything.

It evaluates seven architectural properties that determine whether a coding
agent (and a new human contributor) can succeed without a senior engineer
reviewing every diff:

| | Principle | The question it asks |
|---|---|---|
| **P1** | Locality | Can a change be understood from a narrow slice of the repo? |
| **P2** | Spec-first | Are contracts schemas and types, not prose? |
| **P3** | Shallow explicitness | Is behaviour visible without chasing reflection or deep indirection? |
| **P4** | Verifiability | Can correctness be proven locally in seconds? |
| **P5** | Aggregation of danger | Are auth, secrets, and migrations concentrated and guarded? |
| **P6** | Reversibility | Can every change be rolled back cheaply? |
| **P7** | Machine-readability | Are errors, ADRs, and CLIs readable by machines? |

archfit is **not** a linter, and **not** a SAST scanner. It sits *above* those
tools and reports on architectural properties they do not measure.

---

## Quick start

```bash
# Build from source (Go 1.24+, no CGO)
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit
make build

# Scaffold a config for your repo
./bin/archfit init /path/to/your/repo

# Scan it
./bin/archfit scan /path/to/your/repo
```

Or run via Docker:

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

### What you'll see

A clean scan:

```
archfit dev — target . (profile=standard)
rules evaluated: 10, findings: 0
overall score: 100.0
  P1: 100.0
  P2: 100.0
  P3: 100.0
  P4: 100.0
  P5: 100.0
  P6: 100.0
  P7: 100.0
no findings
```

When archfit finds something to improve:

```
archfit dev — target . (profile=standard)
rules evaluated: 10, findings: 2
overall score: 84.0
  P1: 100.0
  P3: 60.0
  P6: 60.0
  ...
findings:
  [warn] P3.EXP.001  — repository uses .env files but has no .env.example
  [warn] P6.REV.001 docs/ — deployment artifacts detected but no rollback documentation
```

Every finding carries evidence, confidence, and a remediation guide.
You can auto-fix many of them:

```bash
# Fix a specific finding
./bin/archfit fix P3.EXP.001 .

# Fix all fixable findings at once
./bin/archfit fix --all .

# Preview what would change
./bin/archfit fix --dry-run --all .
```

---

## Commands

### Scanning and analysis

```
archfit scan [path]                  run all enabled rules (default: .)
archfit check <rule-id> [path]       run a single rule against the target
archfit score [path]                 summary only (same scan, no finding list)
archfit report [path]                Markdown report (shorthand for scan --format=md)
archfit explain <rule-id>            show a rule's rationale and remediation
```

### Comparing and tracking

```
archfit diff <baseline.json> [current.json]
                                     compare findings between two scans — exits 1 on regressions
archfit trend                        show score trends from archived scans
archfit compare <f1.json> <f2.json> [...]
                                     compare scans across repos side by side
```

### Fixing

```
archfit fix [rule-id] [path]         auto-fix findings (strong-evidence rules)
```

### Contracts

```
archfit contract check [path]        check scan results against .archfit-contract.yaml
archfit contract init [path]         scaffold a contract from current scan results
```

### Configuration and packs

```
archfit init [path]                  scaffold .archfit.yaml with defaults
archfit validate-config [path]       check .archfit.yaml without scanning
archfit list-rules                   list all registered rules
archfit list-packs                   list all registered rule packs
archfit validate-pack <path>         check pack structure (AGENTS.md, resolvers/, fixtures/)
archfit new-pack <name> [path]       scaffold a new rule pack
archfit test-pack <path>             run pack tests
archfit version                      print the version
```

### Key flags

| Flag | Description | Default |
|---|---|---|
| `--format {terminal\|json\|md\|sarif}` | Output format | `terminal` |
| `--json` | Shorthand for `--format=json` | |
| `--fail-on {info\|warn\|error\|critical}` | Exit `1` at this severity | `error` |
| `--config <file>` | Path to config file | `.archfit.yaml` in target dir |
| `--depth {shallow\|standard\|deep}` | Scan depth (`deep` runs verification commands) | `standard` |
| `--policy <file>` | Organization policy file (JSON) | |
| `-C <dir>` | Change directory before running | |
| `--with-llm` | Enrich findings with LLM explanations | off |
| `--llm-backend {claude\|openai\|gemini}` | LLM provider | auto-detected |
| `--llm-budget N` | Max LLM calls per run | `5` |
| `--record <dir>` | Save scan results (JSON + Markdown) to a timestamped subdirectory | |

The `fix` command adds: `--all`, `--dry-run`, `--plan`, `--json`.
The `trend` command adds: `--history <dir>`, `--since <date>`, `--format {terminal|json|csv}`.
The `compare` command adds: `--format {terminal|json|csv|md}`, `--sort {overall|name}`.

### Exit codes

| Code | Meaning |
|:---:|---|
| `0` | Success (or: all findings below `--fail-on` threshold) |
| `1` | Findings at or above `--fail-on` |
| `2` | Usage error |
| `3` | Runtime error |
| `4` | Configuration error |

Exit codes are part of the stability contract — see [`docs/exit-codes.md`](./docs/exit-codes.md).
Treat `1` as "read the JSON output", not as a crash.

---

## The rule set — all 7 principles covered

12 rules across 2 packs. All `experimental` stability.

### `core` pack (9 rules) — applies to every repository

| ID | Principle | What it checks | Severity / Evidence |
|---|---|---|---|
| [`P1.LOC.001`](./docs/rules/P1.LOC.001.md) | Locality | `CLAUDE.md` or `AGENTS.md` exists at the repo root | warn / strong |
| [`P1.LOC.002`](./docs/rules/P1.LOC.002.md) | Locality | Vertical-slice directories carry their own `AGENTS.md` | warn / strong |
| [`P3.EXP.001`](./docs/rules/P3.EXP.001.md) | Shallow explicitness | Configuration is documented: `.env` files, Spring Boot profiles, Terraform tfvars, Rails environments (see [details below](#language-and-stack-support)) | warn / strong |
| [`P4.VER.001`](./docs/rules/P4.VER.001.md) | Verifiability | A fast verification entrypoint exists (Makefile, package.json, go.mod, pom.xml, build.gradle, Gemfile, Cargo.toml, and [20+ more](#language-and-stack-support)) | warn / strong |
| [`P4.VER.002`](./docs/rules/P4.VER.002.md) | Verifiability | Source directories have test files alongside code | info / medium |
| [`P4.VER.003`](./docs/rules/P4.VER.003.md) | Verifiability | Repository has CI configuration (GitHub Actions, GitLab CI, Jenkins, etc.) | info / strong |
| [`P5.AGG.001`](./docs/rules/P5.AGG.001.md) | Aggregation of danger | Security-sensitive files (auth, secrets, migrations, deploy) are concentrated, not scattered | warn / strong |
| [`P6.REV.001`](./docs/rules/P6.REV.001.md) | Reversibility | Deployment artifacts present → rollback documentation must exist | warn / strong |
| [`P7.MRD.001`](./docs/rules/P7.MRD.001.md) | Machine-readability | CLI repos document their exit codes | warn / strong |

### `agent-tool` pack (3 rules) — opt-in, for agent-consumed tools

| ID | Principle | What it checks |
|---|---|---|
| [`P2.SPC.010`](./docs/rules/P2.SPC.010.md) | Spec-first | Tool ships a versioned JSON Schema with `$id` (also recognizes OpenAPI, Protobuf, GraphQL, Avro, AsyncAPI) |
| [`P7.MRD.002`](./docs/rules/P7.MRD.002.md) | Machine-readability | `CHANGELOG.md` exists at the repo root |
| [`P7.MRD.003`](./docs/rules/P7.MRD.003.md) | Machine-readability | CLI repos record ADRs under `docs/adr/` |

More packs (`web-saas`, `iac`, `mobile`, `data-event`) are planned.
Ten solid `strong`-evidence rules beat a hundred weak ones.

---

## Language and stack support

archfit is language-agnostic by design. Its rules check architectural terrain,
not language syntax. Here's what each rule recognizes across stacks:

### P4.VER.001 — verification entrypoints

Go (`go.mod`), Node/TypeScript (`package.json`), Python (`pyproject.toml`),
Rust (`Cargo.toml`), Java (`pom.xml`, `build.gradle`, `build.gradle.kts`,
`settings.gradle`, `settings.gradle.kts`), Ruby (`Gemfile`, `Rakefile`),
PHP (`composer.json`), Elixir (`mix.exs`), Scala (`build.sbt`),
C/C++ (`CMakeLists.txt`, `meson.build`), Deno (`deno.json`, `deno.jsonc`),
Bazel (`BUILD.bazel`), Earthly (`Earthfile`), and generic task runners
(`Makefile`, `justfile`, `Taskfile.yml`).

### P3.EXP.001 — configuration documentation

Four config ecosystems are checked independently:

| Ecosystem | Config files detected | Documentation expected |
|---|---|---|
| Node/Python/Ruby | `.env`, `.env.*` | `.env.example`, `.env.sample`, or `.env.template` |
| Spring Boot | `application-*.yml`, `application-*.yaml`, `application-*.properties` | `config/README.md` or `docs/config.md` |
| Terraform | `*.tfvars` | `terraform.tfvars.example` or `example.tfvars` |
| Rails | `config/environments/*.rb` | `config/README.md` or `docs/config.md` |

### P1.LOC.002 — vertical-slice containers

`packs/`, `services/`, `modules/`, `packages/`, `apps/`, `libs/`,
`plugins/`, `engines/`, `components/`, `domains/`, `features/` — covering
monorepos (NX, Turborepo, Lerna), DDD projects, Rails engines, plugin
architectures, and service-oriented repos.

### P6.REV.001 — deployment artifacts

Docker (`Dockerfile`, `docker-compose.yml`, `compose.yml`),
Kubernetes (`kubernetes/`, `k8s/`), Helm (`helm/`),
Terraform (`terraform/`), AWS CDK (`cdk.json`, `cdk/`),
Serverless Framework (`serverless.yml`), Cloud Build (`cloudbuild.yaml`),
Skaffold (`skaffold.yaml`), PaaS (Vercel, Netlify, Fly.io, Render,
Railway, Heroku `Procfile`), and CI systems (GitHub Actions, CircleCI,
GitLab CI, Buildkite).

### P7.MRD.001 — CLI detection

CLI entrypoints are detected via `cmd/`, `bin/`, or `exe/` directories
containing source files in any of 11 languages (`.go`, `.py`, `.ts`, `.js`,
`.rs`, `.rb`, `.java`, `.kt`, `.swift`, `.php`, `.sh`), plus indicator files
like `__main__.py`, `cli.go`, `cli.py`, `cli.ts`, `cli.js`, `cli.rb`.

### P2.SPC.010 — spec-first formats

JSON Schema (`schemas/*.schema.json` with `$id`), OpenAPI/Swagger
(`openapi.yaml`, `swagger.json`), Protocol Buffers (`.proto`),
GraphQL (`.graphql`, `.gql`), Apache Avro (`.avsc`), and
AsyncAPI (`.asyncapi`).

---

## Auto-fix

`archfit fix` closes the scan → fix → verify loop. It ships 7 static fixers —
one for every rule with a deterministic fix:

| Rule | What the fixer creates |
|---|---|
| P1.LOC.001 | `CLAUDE.md` at repo root |
| P1.LOC.002 | `AGENTS.md` in each slice directory |
| P4.VER.001 | `Makefile` with test and lint targets |
| P7.MRD.001 | `docs/exit-codes.md` |
| P7.MRD.002 | `CHANGELOG.md` (Keep a Changelog format) |
| P7.MRD.003 | `docs/adr/0001-initial-architecture.md` |
| P2.SPC.010 | `schemas/output.schema.json` with `$id` |

```bash
archfit fix P1.LOC.001 .             # fix one rule
archfit fix --all .                  # fix everything fixable
archfit fix --plan --all .           # see the plan without applying
archfit fix --dry-run P7.MRD.002 .   # show what would change
archfit fix --json --all .           # JSON output for automation
```

Every fix is **verified by automatic re-scan**. If the finding persists or new
findings appear, changes are rolled back. Fix actions are logged to
`.archfit-fix-log.json` for audit.

LLM-assisted fixers enrich the static templates with repo-specific context
when `--with-llm` is set.

---

## Tracking fitness over time

### Diff — PR gate on regressions

```bash
# Compare against a baseline (exits 1 if new findings appear)
archfit diff baseline.json current.json
archfit diff baseline.json              # current from stdin
```

### Trend — score history

```bash
# Archive scans into a directory, then view trends
archfit scan --json . > .archfit-history/2026-04-25.json
archfit trend --history .archfit-history
archfit trend --since 2026-01-01 --format csv
```

### Compare — cross-repo scoreboard

```bash
# Compare multiple repos side by side
archfit compare api.json frontend.json infra.json
archfit compare --format md --sort name *.json
```

---

## Evidence, not verdict

Every finding carries four qualities:

- **Severity** — how bad is it if true? (`info` / `warn` / `error` / `critical`)
- **Evidence strength** — how deterministic is the detection? (`strong` / `medium` / `weak` / `sampled`)
- **Confidence** — a numeric 0.0–1.0
- **Remediation** — a summary plus a link to a detailed guide

archfit is deliberately conservative: `error` severity requires `strong`
evidence. **False positives are treated as bugs.**

JSON output is deterministic (severity desc, rule_id asc, path asc) so agents
can make stable references and `archfit diff` produces reliable deltas.

### Scoring

Each finding penalizes the score for its principle based on severity:

| Severity | Penalty (% of rule weight) |
|---|---|
| info | 10% |
| warn | 40% |
| error | 80% |
| critical | 100% |

Scores are 0–100, computed per-principle and overall as
`100 × (1 − penalty / totalWeight)`. Adding rules does not inflate
scores — scoring is weight-based and normalized per applicable rule set.

### Metrics

Six quantitative metrics are computed alongside findings:

| Metric | Principle | What it measures |
|---|---|---|
| `context_span_p50` | P1 | Median files touched per commit |
| `parallel_conflict_rate` | P1 | Merge commit frequency |
| `verification_latency_s` | P4 | Wall-clock test execution time (deep scan only) |
| `invariant_coverage` | P4 | Fraction of rules with no error+ findings |
| `blast_radius_score` | P5 | Max transitive package reach |
| `rollback_signal` | P6 | Revert commit frequency |

---

## Configuration

archfit reads `.archfit.yaml` from the target directory, or you can point to a
specific file:

```bash
archfit scan .                              # default discovery
archfit scan --config .archfit.all.yaml .   # explicit config
```

Generate a starter config:

```bash
archfit init .
```

```json
{
  "version": 1,
  "project_type": [],
  "profile": "standard",
  "packs": { "enabled": ["core"] },
  "ignore": []
}
```

A richer example with both packs, risk tiers, and expiring suppressions:

```json
{
  "version": 1,
  "project_type": ["agent-tool"],
  "profile": "standard",
  "risk_tiers": {
    "high":   ["src/auth/**", "infra/**", "migrations/**"],
    "medium": ["src/features/**"],
    "low":    ["docs/**", "tests/**"]
  },
  "packs": { "enabled": ["core", "agent-tool"] },
  "ignore": [
    {
      "rule": "P1.LOC.002",
      "paths": ["packs/legacy-*"],
      "reason": "Legacy slices on a documented deletion path",
      "expires": "2026-12-31"
    }
  ]
}
```

Every `ignore` entry requires a `reason` and an `expires` date. Expired
suppressions surface as warnings — they cannot silently rot.

Full reference: [`docs/configuration.md`](./docs/configuration.md).

### Fitness contracts

For teams that want machine-enforceable fitness goals, archfit supports
contracts in `.archfit-contract.yaml`:

```json
{
  "version": 1,
  "hard_constraints": [
    { "principle": "overall", "min_score": 80.0, "scope": "**" }
  ],
  "soft_targets": [
    { "principle": "P4", "target_score": 95.0, "deadline": "2026-06-30" }
  ],
  "area_budgets": [
    { "path": "src/auth/**", "max_findings": 0, "owner": "@security-team" }
  ]
}
```

Hard constraints fail the scan. Soft targets are aspirational. Area budgets
give teams SRE-style finding allowances per directory.

### Organization policies

The `--policy` flag loads an org-wide policy that enforces minimum scores,
required packs, and rule severity overrides across all repos:

```bash
archfit scan --policy org-policy.json .
```

---

## How it works

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
| schema,   |   |  agent-tool (3)  |   | md, SARIF 2.1.0|
| depgraph, |   +--------+---------+   +--------+-------+
| command   |            |                      |
+-----------+  +---------+--------+   +---------v-------+
               |    Fix Engine    |   |   LLM Adapter   |
               | 7 static fixers  |   | Claude | OpenAI |
               | + LLM-assisted   |   | Gemini          |
               +------------------+   +-----------------+
                                        (opt-in only)
```

- **Collectors** gather facts from the filesystem, git history, JSON schemas,
  dependency graphs (Go), and command timing. They observe; they do not judge.
- **Rule packs** declare rules and implement resolver functions. Resolvers are
  pure functions of a read-only `FactStore` — no I/O. This is archfit's own
  P5 (aggregation) enforced on itself.
- **Fix engine** produces deterministic file changes for each finding, then
  re-scans to verify. Static fixers handle templated fixes; LLM fixers
  generate contextual content when `--with-llm` is set.
- **Renderers** produce output in multiple formats. JSON conforms to
  [`schemas/output.schema.json`](./schemas/output.schema.json); SARIF 2.1.0
  integrates with GitHub Code Scanning.
- **LLM adapter** is the single network boundary. Three backends — Claude,
  OpenAI, Gemini — behind one `llm.Client` interface. Only engaged via
  `--with-llm`; the base scan is identical with or without an API key.

Rule registration is explicit in `cmd/archfit/main.go`. No reflection, no
`init()` auto-discovery, no plugin magic.

Design rationale:
[ADR 0001](./docs/adr/0001-architecture-overview.md),
[ADR 0002](./docs/adr/0002-phase2-dogfood-and-sarif.md),
[ADR 0003](./docs/adr/0003-llm-explanation.md),
[ADR 0004](./docs/adr/0004-fix-engine.md).

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

## LLM-assisted explanation (opt-in)

Static remediation guides tell you what to do *in general*. `--with-llm` tells
you why *your specific repo* triggered the rule and what *exact change* would
fix it — without touching the default scan path.

### Supported providers

| Provider | Env var | Default model | `--llm-backend` |
|---|---|---|---|
| Claude (Anthropic) | `ANTHROPIC_API_KEY` | `claude-sonnet-4-6-20250627` | `claude` |
| OpenAI | `OPENAI_API_KEY` | `gpt-5.4-mini` | `openai` |
| Google Gemini | `GOOGLE_API_KEY` / `GEMINI_API_KEY` | `gemini-2.5-flash` | `gemini` |

Auto-detection priority: `ANTHROPIC_API_KEY` > `OPENAI_API_KEY` > `GOOGLE_API_KEY`.

```bash
export ANTHROPIC_API_KEY=sk-...
archfit scan --with-llm .                # enrich findings
archfit explain --with-llm P3.EXP.001   # explain a rule
archfit fix --with-llm --all .           # contextual auto-fix
```

### Safety guarantees

- **Opt-in only.** Base `archfit scan` makes zero LLM calls.
- **Bounded cost.** `--llm-budget N` caps calls per run (default 5). Cache hits are free.
- **Never fails the scan.** API errors degrade gracefully to static remediation.
- **Minimal data sent.** Rule metadata + finding evidence only. No source code, no file contents, no git history.

Full contract: [`docs/llm.md`](./docs/llm.md).

---

## Claude Code agent skill

archfit ships with a Claude Code agent skill at
[`.claude/skills/archfit/`](./.claude/skills/archfit/) — auto-discovered when
Claude Code runs inside this repo. The skill drives a scan → fix → verify loop:

1. **Run**: `archfit scan --json .`
2. **Read**: the `findings[]` array
3. **Fix**: `archfit fix <rule-id>` or load the remediation guide from `reference/remediation/`
4. **Verify**: re-scan — the re-scan is the proof, not the claim

10 remediation guides ship under `.claude/skills/archfit/reference/remediation/`,
one per rule. Each contains a decision tree that tells the agent when to fix
automatically and when to ask the user first.

To use the skill in another repo, copy `.claude/skills/archfit/` into that
project's `.claude/skills/` directory.

---

## Install

### From source

```bash
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit
make build
./bin/archfit version
```

Requires **Go 1.24+**. No CGO. Cross-compiles to `linux/{amd64,arm64}`,
`darwin/{amd64,arm64}`, and `windows/amd64`.

### From release binaries

```bash
# Linux/macOS
curl -sSL https://github.com/shibuiwilliam/archfit/releases/latest/download/archfit-<version>-linux-amd64.tar.gz \
  | tar xz
./archfit version
```

Pre-built binaries for all 5 platforms and SHA-256 checksums are published with
each [GitHub Release](https://github.com/shibuiwilliam/archfit/releases).

### Via Docker

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

Multi-arch image (`linux/amd64` + `linux/arm64`) published to
[GitHub Container Registry](https://github.com/shibuiwilliam/archfit/pkgs/container/archfit).

---

## Repository layout

```
archfit/
├── cmd/archfit/              # CLI entry point — explicit wiring, 18 subcommands
├── internal/
│   ├── core/                 # Scheduler: collectors → FactStore → rules → scores
│   ├── model/                # Rule, Finding, Metric, FactStore, ParseFailure
│   ├── config/               # .archfit.yaml loading + validation
│   ├── contract/             # Fitness contracts (.archfit-contract.yaml)
│   ├── policy/               # Organization policies (--policy)
│   ├── collector/            # Fact gatherers: fs, git, schema, depgraph, command
│   ├── adapter/
│   │   ├── exec/             # Fake-able subprocess runner
│   │   └── llm/              # Claude, OpenAI, Gemini behind Client interface
│   ├── fix/                  # Fix engine + 7 static fixers + LLM fixers
│   │   ├── static/           # Deterministic fixers with embedded templates
│   │   └── llmfix/           # LLM-assisted fixers (opt-in via --with-llm)
│   ├── packman/              # Pack validation (validate-pack command)
│   ├── rule/                 # Rule engine core
│   ├── report/               # Renderers: terminal, json, md, sarif
│   └── score/                # Weight-based normalized scoring + 6 metrics
├── packs/
│   ├── core/                 # 9 rules covering P1, P3, P4, P5, P6, P7
│   │   ├── resolvers/        # Pure functions of FactStore
│   │   ├── fixtures/         # One golden repo per rule + expected.json
│   │   └── pack_test.go      # Fixture-driven table tests
│   └── agent-tool/           # 3 rules covering P2, P7 (opt-in)
├── schemas/                  # Versioned JSON Schema: rule, config, output, contract
├── testdata/e2e/             # End-to-end golden tests
├── .claude/skills/archfit/   # Claude Code agent skill (auto-discovered)
│   └── reference/remediation/  # Per-rule remediation guides
├── .github/workflows/
│   ├── ci.yml                # lint + test + self-scan + cross-build (5 platforms)
│   ├── auto-release.yml      # auto patch-bump, tag, release on main push
│   └── release.yml           # manual release on v* tag push
├── docs/
│   ├── adr/                  # Architecture Decision Records
│   ├── rules/                # Per-rule documentation
│   ├── deployment.md         # Deploy/rollback procedures
│   ├── llm.md                # --with-llm contract
│   └── exit-codes.md         # Exit code contract
├── Dockerfile                # Multi-stage: golang:1.24-alpine → scratch
├── VERSION                   # SemVer source of truth (read by Makefile + CI)
├── .archfit.yaml             # archfit's own config (self-scan)
├── Makefile
├── CLAUDE.md                 # Contributor contract
├── CHANGELOG.md              # Keep-a-Changelog 1.1.0 format
├── CONTRIBUTING.md           # Contribution guide
├── SECURITY.md               # Security policy
└── LICENSE                   # Apache 2.0
```

**Boundary rule**: `packs/*` may import `internal/model` and `internal/rule`,
but never anything that performs I/O. If a rule needs a new fact, it grows a
Collector. Enforced by [`.go-arch-lint.yaml`](./.go-arch-lint.yaml).

---

## Development

```bash
make build            # build to ./bin/archfit
make test             # unit + pack tests (with -race)
make test-short       # quick tests (skip long)
make e2e              # end-to-end golden tests
make lint             # gofmt + go vet + golangci-lint + go-arch-lint
make self-scan        # archfit on itself — must exit 0
make self-scan-json   # same, JSON to stdout
make update-golden    # regenerate expected.json (review the diff!)
make clean
```

No test performs network I/O. The LLM `Fake` client is used throughout; real
clients are instantiated only in `main.go`. No API keys needed.

The **self-scan** is the forcing function: if `archfit scan ./` flags archfit's
own code, the change is wrong.

---

## Contributing

Before opening a PR, read [`CLAUDE.md`](./CLAUDE.md) and
[`CONTRIBUTING.md`](./CONTRIBUTING.md). Key rules:

- PR budget: ≤ 500 changed lines, ≤ 5 packages
- Every new rule ships with: resolver, fixture + `expected.json`, table test,
  rule doc, and a remediation guide
- No `init()` registration, no reflection, no global mutable state
- No I/O inside `packs/*` — add a Collector instead
- No LLM calls on the default scan path

---

## Security

See [`SECURITY.md`](./SECURITY.md). Two things to know:

- archfit runs `git log` against the scanned repo. Use a sandbox for untrusted repos.
- `--with-llm` sends rule metadata and finding evidence to the LLM provider.
  **Source code and file contents are never sent.**
  Full contract: [`docs/llm.md`](./docs/llm.md).

---

## Versioning and releases

archfit follows [SemVer 2.0](https://semver.org/spec/v2.0.0.html). The current
version lives in the [`VERSION`](./VERSION) file at the repo root.

### Automatic patch releases

Every push to `main` (except docs-only changes) triggers the
[auto-release](./.github/workflows/auto-release.yml) workflow:

1. Read `VERSION`, find the latest `v0.1.*` tag, bump the patch
2. Run `make lint`, `make test`, `make self-scan` as a quality gate
3. Tag the commit (`v0.1.1`, `v0.1.2`, ...)
4. Cross-compile binaries for 5 platforms
5. Create a GitHub Release with binaries, checksums, and a self-scan report
6. Build and push a multi-arch Docker image to `ghcr.io`
7. Open a PR to update `VERSION` for the next cycle

### Manual minor/major releases

To bump the minor or major version, edit `VERSION` (e.g., `0.2.0` or `1.0.0`)
before merging to `main`. The auto-release workflow respects the higher number.
You can also push a `v*` tag directly to trigger the
[manual release](./.github/workflows/release.yml) workflow.

### Each release includes

- Pre-built binaries: `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/amd64`
- SHA-256 checksums (`checksums-sha256.txt`)
- Self-scan JSON report for that version
- Multi-arch Docker image at `ghcr.io/shibuiwilliam/archfit:<version>`

---

## What archfit is *not*

- Not a replacement for language-specific linters (`golangci-lint`, `eslint`, `ruff`).
- Not a SAST tool. Use Semgrep, CodeQL, or Trivy for that.
- Not a benchmarking tool. The score is for *your* repo over time, not a competition.
- Not a cage. Suppression exists — with a reason and an expiry — on purpose.
- Not dependent on an LLM. The base scan is deterministic and offline.

---

## Roadmap

Detailed plan in [`DEVELOPMENT_PLAN.md`](./DEVELOPMENT_PLAN.md):

- **0.1.0**: Foundation, `core` pack (4 rules), JSON/Markdown, self-scan.
- **0.2.0**: `init`/`check`/`report`/`diff`, SARIF 2.1.0, `agent-tool` pack, e2e tests, CI.
- **0.3.x**: Multi-provider LLM (Claude, OpenAI, Gemini). P3/P5/P6 rules. `--config` flag. `archfit fix` with 7 static fixers. Dockerfile + release workflow.
- **Next**: `web-saas`/`iac`/`mobile`/`data-event` packs, metrics pipeline, release binaries, additional collectors (AST, depgraph, command).
- **1.0**: Rule IDs frozen, JSON schema v1, SARIF certified.

---

## License

Apache 2.0 — see [`LICENSE`](./LICENSE).
