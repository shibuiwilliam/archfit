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

```
archfit scan [path]                  run all enabled rules (default: .)
archfit check <rule-id> [path]       run a single rule against the target
archfit score [path]                 summary only (same scan, no finding list)
archfit report [path]                Markdown report (shorthand for scan --format=md)
archfit diff <baseline.json> [current.json]
                                     compare findings between two scans
archfit fix [rule-id] [path]         auto-fix findings (strong-evidence rules)
archfit trend                        show score trends from archived scans
archfit compare <f1.json> <f2.json>  compare scans across repos
archfit explain <rule-id>            show a rule's rationale and remediation
archfit init [path]                  scaffold .archfit.yaml with defaults
archfit list-rules                   list all registered rules
archfit list-packs                   list all registered rule packs
archfit validate-config [path]       check .archfit.yaml without scanning
archfit validate-pack <path>         check pack structure
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
| `-C <dir>` | Change directory before running | |
| `--policy <file>` | Organization policy file (JSON) | |
| `--with-llm` | Enrich findings with LLM explanations | off |
| `--llm-backend {claude\|openai\|gemini}` | LLM provider | auto-detected |
| `--llm-budget N` | Max LLM calls per run | `5` |

The `fix` command has its own flags: `--all`, `--dry-run`, `--plan`, `--json`.

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

10 rules across 2 packs. All `strong` evidence, `experimental` stability.

### `core` pack (7 rules) — applies to every repository

| ID | Principle | What it checks |
|---|---|---|
| [`P1.LOC.001`](./docs/rules/P1.LOC.001.md) | Locality | `CLAUDE.md` or `AGENTS.md` exists at the repo root |
| [`P1.LOC.002`](./docs/rules/P1.LOC.002.md) | Locality | Vertical-slice directories carry their own `AGENTS.md` |
| [`P3.EXP.001`](./docs/rules/P3.EXP.001.md) | Shallow explicitness | Configuration is documented: `.env` → `.env.example`, Spring `application-*.yml` → `config/README.md`, Terraform `*.tfvars` → `terraform.tfvars.example`, Rails `config/environments/` → config docs |
| [`P4.VER.001`](./docs/rules/P4.VER.001.md) | Verifiability | A fast verification entrypoint exists (`Makefile`, `package.json`, `go.mod`, `pom.xml`, `build.gradle`, `Gemfile`, `Cargo.toml`, and 20+ more) |
| [`P5.AGG.001`](./docs/rules/P5.AGG.001.md) | Aggregation of danger | Security-sensitive files (auth, secrets, migrations, deploy) are concentrated, not scattered |
| [`P6.REV.001`](./docs/rules/P6.REV.001.md) | Reversibility | Deployment artifacts present → rollback documentation must exist |
| [`P7.MRD.001`](./docs/rules/P7.MRD.001.md) | Machine-readability | CLI repos (`cmd/`, `bin/`, `exe/`) document their exit codes |

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
not language syntax.

**P4.VER.001** recognizes build systems for: Go, Node/TypeScript, Python, Rust,
Java (Maven + Gradle), Ruby, PHP, Elixir, Scala, C/C++ (CMake, Meson), Deno,
Bazel, Earthly, and generic task runners (Make, Just, Task).

**P3.EXP.001** checks configuration documentation across four ecosystems:
`.env` (Node, Python, Ruby), Spring Boot profiles (`application-*.yml`),
Terraform variables (`*.tfvars`), and Rails environments
(`config/environments/`).

**P1.LOC.002** recognizes vertical-slice containers used by monorepos
(`packages/`, `apps/`, `libs/`), DDD projects (`domains/`, `features/`),
Rails engines (`engines/`), plugin architectures (`plugins/`, `components/`),
and service-oriented repos (`services/`, `modules/`).

**P6.REV.001** detects deployment artifacts from Docker, Kubernetes, Helm,
Terraform, AWS CDK, Serverless Framework, Cloud Build, Skaffold, Vercel,
Netlify, Fly.io, Render, Railway, Heroku (Procfile), and all major CI systems
(GitHub Actions, CircleCI, GitLab CI, Buildkite).

**P2.SPC.010** recognizes JSON Schema, OpenAPI/Swagger, Protocol Buffers,
GraphQL, Apache Avro, and AsyncAPI as valid spec-first formats.

---

## Auto-fix

`archfit fix` closes the scan → fix → verify loop. It ships 7 static fixers
for all rules that have deterministic fixes:

```bash
# Fix one rule
archfit fix P1.LOC.001 .

# Fix everything fixable
archfit fix --all .

# See the plan without applying
archfit fix --plan --all .

# Dry run — show what would change
archfit fix --dry-run P7.MRD.002 .

# JSON output for automation
archfit fix --json --all .
```

Every fix is **verified by automatic re-scan**. If the finding persists or new
findings appear, changes are rolled back. Fix actions are logged to
`.archfit-fix-log.json` for audit.

LLM-assisted fixers (for context-dependent content) are available via
`--with-llm` — they enrich the static templates with repo-specific context.

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

- **Collectors** gather facts from the filesystem, git history, schemas,
  dependency graphs, and command timing. They observe; they do not judge.
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
├── cmd/archfit/              # CLI entry point — explicit wiring, 17 subcommands
├── internal/
│   ├── core/                 # Scheduler: collectors → FactStore → rules → scores
│   ├── model/                # Rule, Finding, Metric, FactStore, ParseFailure
│   ├── config/               # .archfit.yaml loading + validation
│   ├── collector/            # Fact gatherers: fs, git, schema, depgraph, command
│   ├── adapter/
│   │   ├── exec/             # Fake-able subprocess runner
│   │   └── llm/              # Claude, OpenAI, Gemini behind Client interface
│   ├── fix/                  # Fix engine + 7 static fixers + LLM fixers
│   │   ├── static/           # Deterministic fixers with embedded templates
│   │   └── llmfix/           # LLM-assisted fixers (opt-in via --with-llm)
│   ├── rule/                 # Rule engine core
│   ├── report/               # Renderers: terminal, json, md, sarif
│   └── score/                # Weight-based normalized scoring
├── packs/
│   ├── core/                 # 7 rules covering P1, P3, P4, P5, P6, P7
│   │   ├── resolvers/        # Pure functions of FactStore
│   │   ├── fixtures/         # One golden repo per rule + expected.json
│   │   └── pack_test.go      # Fixture-driven table tests
│   └── agent-tool/           # 3 rules covering P2, P7 (opt-in)
├── schemas/                  # Versioned JSON Schema: rule, config, output
├── testdata/e2e/             # End-to-end golden tests
├── .claude/skills/archfit/   # Claude Code agent skill (auto-discovered)
│   └── reference/remediation/  # 10 per-rule remediation guides
├── .github/workflows/
│   ├── ci.yml                # lint + test + self-scan + cross-build
│   └── release.yml           # binaries + GitHub Release + Docker (ghcr.io)
├── docs/
│   ├── adr/                  # Architecture Decision Records
│   ├── rules/                # Per-rule documentation
│   ├── deployment.md         # Deploy/rollback procedures
│   ├── llm.md                # --with-llm contract
│   └── exit-codes.md         # Exit code contract
├── Dockerfile                # Multi-stage: golang:1.24-alpine → scratch
├── .archfit.yaml             # archfit's own config (self-scan)
├── Makefile
├── CLAUDE.md                 # Contributor contract
├── CHANGELOG.md              # Keep-a-Changelog 1.1.0 format
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
make e2e              # end-to-end golden tests
make lint             # gofmt + go vet (+ golangci-lint if installed)
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
