![archfit logo](./archfit_logo.png)

# archfit

> **Architecture fitness evaluator for the coding-agent era.**
> Is your repository shaped for coding agents to work on it — *safely* and *quickly*?

![CI](https://github.com/shibuiwilliam/archfit/actions/workflows/ci.yml/badge.svg)
![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)

[Documentation](https://shibuiwilliam.github.io/archfit/) | [Japanese / 日本語](./README.ja.md)

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
# Install (Go 1.24+)
go install github.com/shibuiwilliam/archfit/cmd/archfit@latest

# Or build from source
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit && make build

# Scaffold a config for your repo
archfit init /path/to/your/repo

# Scan it
archfit scan /path/to/your/repo
```

Or run via Docker:

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

### What you'll see

```
archfit 0.1.0 — target . (profile=standard)
rules evaluated: 14 (0 with findings), findings: 0
overall score: 100.0
  P1: 100.0  P2: 100.0  P3: 100.0  P4: 100.0
  P5: 100.0  P6: 100.0  P7: 100.0
no findings
```

When archfit finds something to improve:

```
archfit 0.1.0 — target . (profile=standard)
rules evaluated: 14 (2 with findings), findings: 2
overall score: 84.0
findings:
  [warn] P3.EXP.001  — repository uses .env files but has no .env.example
  [warn] P6.REV.001 docs/ — deployment artifacts detected but no rollback documentation
```

Every finding carries evidence, confidence, and a remediation guide.
You can auto-fix many of them:

```bash
archfit fix P3.EXP.001 .       # fix a specific finding
archfit fix --all .             # fix all fixable findings
archfit fix --dry-run --all .   # preview changes
```

---

## The rule set — 14 rules, all 7 principles

### `core` pack (11 rules) — applies to every repository

| ID | Principle | What it checks | Severity |
|---|---|---|---|
| [P1.LOC.001](./docs/rules/P1.LOC.001.md) | Locality | `CLAUDE.md` or `AGENTS.md` at repo root | warn |
| [P1.LOC.002](./docs/rules/P1.LOC.002.md) | Locality | Vertical-slice directories carry `AGENTS.md` | warn |
| [P1.LOC.003](./docs/rules/P1.LOC.003.md) | Locality | Dependency coupling is bounded (max reach ≤10) | info |
| [P1.LOC.004](./docs/rules/P1.LOC.004.md) | Locality | Commits touch a bounded number of files (≤8) | info |
| [P3.EXP.001](./docs/rules/P3.EXP.001.md) | Explicitness | Config documented (.env, Spring profiles, tfvars, Rails) | warn |
| [P4.VER.001](./docs/rules/P4.VER.001.md) | Verifiability | Verification entrypoint (Makefile, pom.xml, etc. — [26 recognized](#language-and-stack-support)) | warn |
| [P4.VER.002](./docs/rules/P4.VER.002.md) | Verifiability | ≥70% of source directories have test files | info |
| [P4.VER.003](./docs/rules/P4.VER.003.md) | Verifiability | CI configuration present (GitHub Actions, GitLab, etc.) | info |
| [P5.AGG.001](./docs/rules/P5.AGG.001.md) | Aggregation | Security-sensitive files concentrated, not scattered | warn |
| [P6.REV.001](./docs/rules/P6.REV.001.md) | Reversibility | Deployment artifacts → rollback documentation exists | warn |
| [P7.MRD.001](./docs/rules/P7.MRD.001.md) | Machine-readability | CLI repos document exit codes | warn |

### `agent-tool` pack (3 rules) — opt-in, for agent-consumed tools

| ID | Principle | What it checks |
|---|---|---|
| [P2.SPC.010](./docs/rules/P2.SPC.010.md) | Spec-first | Versioned schema with `$id` (also recognizes OpenAPI, Protobuf, GraphQL, Avro) |
| [P7.MRD.002](./docs/rules/P7.MRD.002.md) | Machine-readability | `CHANGELOG.md` at repo root |
| [P7.MRD.003](./docs/rules/P7.MRD.003.md) | Machine-readability | CLI repos record ADRs under `docs/adr/` |

Rule definitions live in YAML under `packs/*/rules/` and are the spec-first source of truth.
Go resolvers are pure functions of a read-only `FactStore`.

---

## Language and stack support

archfit is language-agnostic by design. Here's what each rule recognizes:

**P4.VER.001 — verification entrypoints**: Go, Node/TS, Python, Rust, Java (Maven + Gradle), Ruby, PHP, Elixir, Scala, C/C++ (CMake, Meson), Deno, Bazel, Earthly, and generic task runners (Makefile, justfile, Taskfile).

**P3.EXP.001 — config documentation**: `.env` files, Spring Boot `application-*.yml` profiles, Terraform `*.tfvars`, Rails `config/environments/`.

**P1.LOC.002 — slice containers**: `packs/`, `services/`, `modules/`, `packages/`, `apps/`, `libs/`, `plugins/`, `engines/`, `components/`, `domains/`, `features/`.

**P6.REV.001 — deployment artifacts**: Docker, Kubernetes, Helm, Terraform, AWS CDK, Serverless Framework, Cloud Build, Skaffold, Vercel, Netlify, Fly.io, Render, Railway, and CI systems.

**P2.SPC.010 — spec formats**: JSON Schema, OpenAPI/Swagger, Protobuf, GraphQL, Avro, AsyncAPI.

---

## Commands

```bash
# Scanning
archfit scan [path]                  # full scan (default: .)
archfit check <rule-id> [path]       # single rule
archfit score [path]                 # numbers only
archfit report [path]                # Markdown report

# Fixing
archfit fix [rule-id] [path]         # auto-fix findings
archfit fix --all .                  # fix everything fixable

# Contracts
archfit contract check [path]        # check against .archfit-contract.yaml
archfit contract init [path]         # scaffold contract from current scan

# Comparing
archfit diff <baseline.json>         # PR gate on regressions
archfit trend                        # score history
archfit compare <f1> <f2> [...]      # cross-repo comparison

# Setup
archfit init [path]                  # scaffold .archfit.yaml (stack-aware)
archfit explain <rule-id>            # rule docs + remediation
archfit list-rules                   # all registered rules
```

### Key flags

| Flag | Default | Description |
|---|---|---|
| `--format {terminal\|json\|md\|sarif}` | `terminal` | Output format |
| `--json` | | Shorthand for `--format=json` |
| `--fail-on {info\|warn\|error\|critical}` | `error` | Exit 1 at this severity |
| `--with-llm` | off | Enrich findings with Claude/OpenAI/Gemini explanations |
| `--record <dir>` | | Save JSON + Markdown to timestamped subdirectory |
| `--explain-coverage` | | Show which rules fired vs. passed silently |
| `-C <dir>` | | Change directory before running |

### Exit codes

| Code | Meaning |
|:---:|---|
| 0 | Success (findings below threshold) |
| 1 | Findings at or above `--fail-on` / contract hard violation |
| 2 | Usage error |
| 3 | Runtime error |
| 4 | Configuration error |
| 5 | Contract soft target missed (no hard violations) |

---

## Auto-fix

`archfit fix` ships 7 static fixers with a scan-fix-verify loop:

```bash
archfit fix P1.LOC.001 .             # creates CLAUDE.md
archfit fix --all .                  # fixes everything fixable
archfit fix --plan --all .           # preview without applying
```

Every fix is verified by automatic re-scan. If the finding persists or new
ones appear, changes are rolled back. Actions are logged to `.archfit-fix-log.json`.

LLM-assisted fixers enrich templates with repo-specific context via `--with-llm`.

---

## Fitness contracts

Declare machine-enforceable fitness goals in `.archfit-contract.yaml`:

```json
{
  "version": 1,
  "hard_constraints": [
    { "principle": "overall", "min_score": 80.0, "scope": "**" }
  ],
  "area_budgets": [
    { "path": "src/auth/**", "max_findings": 0, "owner": "@security-team" }
  ],
  "agent_directives": [
    { "when": "finding.severity >= error", "action": "stop and ask the user" }
  ]
}
```

```bash
archfit contract init .      # scaffold from current scan
archfit contract check .     # enforce in CI (exit 0/1/5)
```

---

## CI integration

### SARIF for GitHub Code Scanning

```yaml
- run: archfit scan --format=sarif . > archfit.sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: archfit.sarif
```

### PR gate

```yaml
- run: archfit diff baseline.json   # exits 1 on new findings
```

---

## LLM enrichment (opt-in)

```bash
export ANTHROPIC_API_KEY=sk-...   # or OPENAI_API_KEY or GOOGLE_API_KEY
archfit scan --with-llm .
```

| Provider | Default model | `--llm-backend` |
|---|---|---|
| Claude (Anthropic) | `claude-sonnet-4-20250514` | `claude` |
| OpenAI | `gpt-5.4-mini` | `openai` |
| Google Gemini | `gemini-2.5-flash` | `gemini` |

Safety: opt-in only, budget-capped (default 5 calls), cache-backed, never fails the scan. Only rule metadata + evidence is sent — **no source code**.

---

## Claude Code agent skill

archfit ships a Claude Code skill at [`.claude/skills/archfit/`](./.claude/skills/archfit/)
that drives a scan-fix-verify loop with per-rule remediation decision trees.

To use it in another repo: copy `.claude/skills/archfit/` into that project's `.claude/skills/`.

---

## Install

| Method | Command |
|---|---|
| `go install` | `go install github.com/shibuiwilliam/archfit/cmd/archfit@latest` |
| Source | `git clone ... && make build` |
| Docker | `docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo` |
| Binary | Download from [Releases](https://github.com/shibuiwilliam/archfit/releases) (5 platforms) |

See [Installation guide](./docs/installation.md) for detailed platform instructions.

---

## Development

```bash
make build          # build to ./bin/archfit
make test           # unit + pack tests (with -race)
make lint           # gofmt + go vet + golangci-lint
make self-scan      # archfit on itself — must exit 0
make generate       # regenerate rule definitions from YAML
```

No test performs network I/O. Self-scan is the forcing function: if `archfit scan ./` flags its own code, the change is wrong.

---

## What archfit is *not*

- Not a replacement for language-specific linters.
- Not a SAST tool. Use Semgrep, CodeQL, or Trivy.
- Not a benchmark. The score is for *your* repo over time, not a competition.
- Not dependent on an LLM. The base scan is deterministic and offline.

---

## License

Apache 2.0 — see [LICENSE](./LICENSE).
