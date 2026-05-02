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

# Scaffold a config (auto-detects your stack)
archfit init /path/to/your/repo

# Scan it
archfit scan /path/to/your/repo
```

Or via Docker:

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

### What you'll see

```
archfit 1.0.0 — target . (profile=standard)
rules evaluated: 27 (0 with findings), findings: 0
overall score: 100.0
  P1: 100.0  P2: 100.0  P3: 100.0  P4: 100.0
  P5: 100.0  P6: 100.0  P7: 100.0
  by_severity_class: {critical: 0, error: 0, warn: 0, info: 0}
no findings
```

When archfit finds something to improve:

```
archfit 1.0.0 — target . (profile=standard)
rules evaluated: 27 (2 with findings), findings: 2
overall score: 84.0
findings:
  [warn] P3.EXP.001  — repository uses .env files but has no .env.example
  [warn] P6.REV.001 docs/ — deployment artifacts detected but no rollback documentation
```

Every finding carries evidence, confidence, and a remediation guide.
Auto-fix many of them:

```bash
archfit fix P3.EXP.001 .       # fix one finding
archfit fix --all .             # fix all fixable findings
archfit fix --dry-run --all .   # preview changes
```

---

## The rule set — 27 rules, all 7 principles

### `core` pack (24 rules) — applies to every repository

| ID | Principle | What it checks | Severity |
|---|---|---|---|
| [P1.LOC.001](./docs/rules/P1.LOC.001.md) | Locality | `CLAUDE.md` or `AGENTS.md` at repo root | warn |
| [P1.LOC.002](./docs/rules/P1.LOC.002.md) | Locality | Vertical-slice dirs carry `AGENTS.md` | warn |
| [P1.LOC.003](./docs/rules/P1.LOC.003.md) | Locality | Dependency coupling bounded (max reach ≤10) | info |
| [P1.LOC.004](./docs/rules/P1.LOC.004.md) | Locality | Commits touch bounded files (≤8) | info |
| [P1.LOC.005](./docs/rules/P1.LOC.005.md) | Locality | High-risk paths declare `INTENT.md` | warn |
| [P1.LOC.006](./docs/rules/P1.LOC.006.md) | Locality | Agent docs not bloated (≤400 lines, ≤10 KB) | warn |
| [P1.LOC.009](./docs/rules/P1.LOC.009.md) | Locality | Runbook per high-risk slice | warn |
| [P2.SPC.001](./docs/rules/P2.SPC.001.md) | Spec-first | API boundary has a machine-readable contract | warn |
| [P2.SPC.002](./docs/rules/P2.SPC.002.md) | Spec-first | DB migrations are bidirectional | warn |
| [P2.SPC.004](./docs/rules/P2.SPC.004.md) | Spec-first | ADRs use YAML frontmatter | info |
| [P3.EXP.001](./docs/rules/P3.EXP.001.md) | Explicitness | Config documented (.env, Spring, Terraform, Rails) | warn |
| [P3.EXP.002](./docs/rules/P3.EXP.002.md) | Explicitness | No `init()` cross-package registration (Go, AST) | warn |
| [P3.EXP.003](./docs/rules/P3.EXP.003.md) | Explicitness | Reflection density bounded (Go, AST) | info |
| [P3.EXP.005](./docs/rules/P3.EXP.005.md) | Explicitness | Global mutable state minimized (Go, AST) | info |
| [P4.VER.001](./docs/rules/P4.VER.001.md) | Verifiability | Verification entrypoint exists (26+ build tools) | warn |
| [P4.VER.002](./docs/rules/P4.VER.002.md) | Verifiability | ≥70% source dirs have test files | info |
| [P4.VER.003](./docs/rules/P4.VER.003.md) | Verifiability | CI configuration present | info |
| [P5.AGG.001](./docs/rules/P5.AGG.001.md) | Aggregation | Security-sensitive files concentrated | warn |
| [P5.AGG.002](./docs/rules/P5.AGG.002.md) | Aggregation | Secret scanner runs in CI | warn |
| [P5.AGG.003](./docs/rules/P5.AGG.003.md) | Aggregation | Risk-tier file declared | warn |
| [P5.AGG.004](./docs/rules/P5.AGG.004.md) | Aggregation | High-risk paths protected by CODEOWNERS | error |
| [P6.REV.001](./docs/rules/P6.REV.001.md) | Reversibility | Deployment artifacts → rollback docs | warn |
| [P6.REV.002](./docs/rules/P6.REV.002.md) | Reversibility | Deploying repo uses feature flags | info |
| [P7.MRD.001](./docs/rules/P7.MRD.001.md) | Machine-readability | CLI repos document exit codes | warn |

### `agent-tool` pack (3 rules) — opt-in, for agent-consumed tools

| ID | Principle | What it checks |
|---|---|---|
| [P2.SPC.010](./docs/rules/P2.SPC.010.md) | Spec-first | Versioned schema with `$id` (OpenAPI, Protobuf, GraphQL, Avro) |
| [P7.MRD.002](./docs/rules/P7.MRD.002.md) | Machine-readability | `CHANGELOG.md` at repo root |
| [P7.MRD.003](./docs/rules/P7.MRD.003.md) | Machine-readability | CLI repos record ADRs under `docs/adr/` |

Rule definitions live in YAML under `packs/*/rules/` (spec-first source of truth).
Rules that don't match the repo's detected languages are automatically skipped.

---

## Language and stack support

archfit is language-agnostic by design. Detection adapts to your stack:

**P4.VER.001** — Go, Node/TS, Python, Rust, Java (Maven + Gradle), Ruby, PHP, Elixir, Scala, C/C++ (CMake, Meson), Deno, Bazel, Earthly, and generic task runners.

**P3.EXP.001** — `.env` files, Spring Boot `application-*.yml`, Terraform `*.tfvars`, Rails `config/environments/`. Spring and Rails checks use the ecosystem collector and only fire when the framework is actually detected.

**P3.EXP.002 / P3.EXP.003 / P3.EXP.005** — Go-specific AST analysis via `go/parser`. These rules use the AST collector to detect `init()` cross-package registration, reflection density, and global mutable state at the syntax level.

**P1.LOC.002** — `packs/`, `services/`, `modules/`, `packages/`, `apps/`, `libs/`, `plugins/`, `engines/`, `components/`, `domains/`, `features/`.

**P2.SPC.010** — JSON Schema, OpenAPI/Swagger, Protobuf, GraphQL, Avro, AsyncAPI.

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
archfit pr-check --base <ref>        # PR gate: diff current scan against a base ref
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
| `--depth {shallow\|standard\|deep}` | `standard` | Scan depth (controls collectors and analysis detail) |
| `--with-llm` | off | Enrich findings with Claude/OpenAI/Gemini explanations |
| `--record <dir>` | | Save JSON + Markdown to timestamped subdirectory |
| `--explain-coverage` | | Show which rules fired vs. passed vs. skipped |
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

```bash
archfit fix P1.LOC.001 .             # creates CLAUDE.md
archfit fix --all .                  # fixes everything fixable
archfit fix --plan --all .           # preview without applying
```

Every fix is verified by automatic re-scan. If the finding persists or new ones appear, changes are rolled back. Actions are logged to `.archfit-fix-log.json`.

---

## Fitness contracts

Declare machine-enforceable fitness goals in `.archfit-contract.yaml`:

```bash
archfit contract init .      # scaffold from current scan
archfit contract check .     # enforce in CI (exit 0/1/5)
```

Supports hard constraints, soft targets, area budgets (SRE-style), and agent directives.

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

Safety: opt-in only, budget-capped, cache-backed, never fails the scan. Only rule metadata + evidence is sent — **no source code**.

---

## Claude Code agent skill

archfit ships a Claude Code skill at [`.claude/skills/archfit/`](./.claude/skills/archfit/)
that drives a scan-fix-verify loop with per-rule remediation decision trees.
The skill includes helper scripts under `.claude/skills/archfit/scripts/` for
common operations like scanning, explaining rules, and applying fixes.
To use it in another repo: copy `.claude/skills/archfit/` into that project's `.claude/skills/`.

---

## Install

| Method | Command |
|---|---|
| `go install` | `go install github.com/shibuiwilliam/archfit/cmd/archfit@latest` |
| Source | `git clone ... && make build` |
| Docker | `docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo` |
| Binary | Download from [Releases](https://github.com/shibuiwilliam/archfit/releases) |

See [Installation guide](./docs/installation.md) for detailed platform instructions.

---

## Development

```bash
make build          # build to ./bin/archfit
make test           # unit + pack tests (with -race)
make lint           # gofmt + go vet + golangci-lint
make self-scan      # archfit on itself — must exit 0, score must not drop
make generate       # regenerate rule definitions from YAML
```

No test performs network I/O. Self-scan is the forcing function: if `archfit scan ./` flags its own code, the change is wrong. The `make self-scan` gate enforces that the overall score does not regress compared to the recorded baseline.

---

## What archfit is *not*

- Not a replacement for language-specific linters.
- Not a SAST tool.
- Not a benchmark. The score is for *your* repo over time.
- Not dependent on an LLM. The base scan is deterministic and offline.

---

## License

Apache 2.0 — see [LICENSE](./LICENSE).
