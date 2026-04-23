# archfit

> **Architecture fitness evaluator for the coding-agent era.**
> How well-shaped is your repository for coding agents to work on it — *safely* and *quickly*?

![CI](https://github.com/shibuiwilliam/archfit/actions/workflows/ci.yml/badge.svg)
![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)

archfit scans a repository and reports on the **terrain** it presents to coding
agents. Not the code's runtime behavior, not its bug count — the *shape* of the
repo itself. The entry points an agent reads first. The speed of the feedback
loop it relies on. The places where a single bad change could quietly take
production down.

It evaluates seven properties that determine whether an agent (and, frankly, a
new human contributor) can succeed without a senior engineer reading every
diff:

| | Principle | The question it asks |
|---|---|---|
| **P1** | Locality | Can a change be understood and verified from a narrow slice of the repo? |
| **P2** | Spec-first | Are contracts executable artifacts — schemas, types, generated clients — rather than prose? |
| **P3** | Shallow explicitness | Is behavior visible without chasing reflection or ten layers of indirection? |
| **P4** | Verifiability | Can correctness be proven locally in seconds to a few minutes? |
| **P5** | Aggregation of danger | Are auth, billing, migrations, and infra *concentrated* and guarded? |
| **P6** | Reversibility | Can every change be rolled back cheaply? Is blast radius bounded? |
| **P7** | Machine-readability | Are errors, logs, ADRs, and CLIs readable by machines, not only humans? |

archfit is **not** a linter, and **not** a SAST scanner. It sits *above* those
tools, consumes their signals where useful, and reports on architectural
properties they do not measure. See [`PROJECT.md`](./PROJECT.md) for the full
thesis.

---

## Status — Phase 2 (0.2.0)

archfit scans a repo, runs the `core` and `agent-tool` packs, emits
terminal / JSON / Markdown / **SARIF** output, and passes its own self-scan
with 7 `strong`-evidence rules at score **100.0**.

| Layer | 0.1.0 | 0.2.0 | Later |
|---|---|---|---|
| CLI: `scan`, `score`, `explain`, `list-rules`, `list-packs`, `validate-config`, `version` | ready | ready | |
| CLI: `init`, `check`, `report`, `diff` | | ready | |
| Rule engine with explicit (no-`init`) registration | ready | ready | |
| Collectors: filesystem, git, **schema** | fs + git | + schema | ast, depgraph, command |
| Schemas: `rule`, `config`, `output` (versioned JSON Schema) | ready | ready | |
| Renderers: terminal, JSON, Markdown, **SARIF 2.1.0** | term + json + md | + SARIF | HTML |
| `core` pack — 4 rules | ready | ready | |
| `agent-tool` pack — 3 rules (opt-in) | | ready | |
| Packs: `web-saas`, `iac`, `mobile`, `data-event` | | | Phase 3+ |
| `archfit fix` (auto-remediation) | | | Phase 3 |
| Metrics pipeline (`context_span_p50`, `blast_radius_score`, …) | | | Phase 3 |
| `go-arch-lint` + `golangci-lint` configs (contracts) | | ready | |
| End-to-end golden tests (`testdata/e2e/`) | | ready | |
| GitHub Actions CI: lint + test + self-scan + cross-build matrix | | ready | |

The full plan is in [`DEVELOPMENT_PLAN.md`](./DEVELOPMENT_PLAN.md). The release
log is in [`CHANGELOG.md`](./CHANGELOG.md).

---

## Install

### From source

```bash
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit
make build
./bin/archfit version
```

Requires Go 1.23 or newer. No CGO, no external dependencies — the 0.2.0
binary is still pure standard library.

Pre-built binaries, Homebrew tap, and Docker image are Phase 3.

---

## Quick start

```bash
# Scaffold .archfit.yaml for a new repo
./bin/archfit init .

# Scan the current directory
./bin/archfit scan .

# Run a single rule
./bin/archfit check P1.LOC.001 .

# Summary only (no finding list)
./bin/archfit score .

# Full JSON — the agent-facing contract
./bin/archfit scan --json . | jq .

# Markdown report (same as: scan --format=md)
./bin/archfit report .

# SARIF 2.1.0 for GitHub Code Scanning
./bin/archfit scan --format=sarif . > archfit.sarif

# Compare two scans — PR gate on *new* findings only
./bin/archfit scan --json . > main.json
# ... make changes ...
./bin/archfit diff main.json

# See every registered rule with its severity
./bin/archfit list-rules

# Learn about a specific rule
./bin/archfit explain P2.SPC.010

# Check .archfit.yaml without scanning
./bin/archfit validate-config .
```

### Example output

```
archfit 0.2.0 — target . (profile=standard)
rules evaluated: 7, findings: 0
overall score: 100.0
  P1: 100.0
  P2: 100.0
  P4: 100.0
  P7: 100.0
no findings
```

### Exit codes

Exit codes are part of the stability contract —
see [`docs/exit-codes.md`](./docs/exit-codes.md):

| Code | Meaning |
|:---:|---|
| `0` | Success (or: all findings below the `--fail-on` threshold) |
| `1` | Findings at or above `--fail-on` (default: `error`) |
| `2` | Usage error |
| `3` | Runtime error |
| `4` | Configuration error |

Treat `1` as a signal to re-read the JSON, not as a crash. `archfit diff`
uses the same convention — exit `1` when new findings appear vs. the
baseline, which is the intended PR-gate behavior.

---

## How it works

```
          ┌──────────────────────────────┐
          │          archfit CLI         │
          └──────────────┬───────────────┘
                         │
     ┌───────────────────┼───────────────────────┐
     │                   │                       │
┌────▼─────┐   ┌─────────▼────────┐   ┌──────────▼────────┐
│Collectors│   │    Rule Packs    │   │    Renderers      │
│ fs, git, │   │  core, agent-    │   │ terminal, json,   │
│  schema  │   │  tool (YAML + Go │   │ md, SARIF 2.1.0   │
│(via exec │   │  resolvers)      │   │   (HTML later)    │
│ adapter) │   └──────────────────┘   └───────────────────┘
└──────────┘
```

- **Collectors** gather facts from the filesystem, git history, and JSON
  Schema files. They *observe*; they do not judge. AST, dependency-graph,
  and command-execution collectors arrive in Phase 3.
- **Rule packs** declare rules in YAML and implement resolvers in Go.
  Resolvers are pure functions of a read-only `FactStore` — they never perform
  I/O. This is archfit's own aggregation principle (P5), enforced on itself
  and locked in by [`.go-arch-lint.yaml`](./.go-arch-lint.yaml).
- **Renderers** produce human- and machine-readable output. JSON is the
  primary machine contract and conforms to [`schemas/output.schema.json`](./schemas/output.schema.json);
  SARIF 2.1.0 is the CI/code-scanning contract.

Rule registration is explicit — `cmd/archfit/main.go` calls each pack's
`Register` function. There is no reflection, no `init()`-based auto-discovery,
no plugin magic. Adding a pack is a two-line diff.

The design rationale is in
[`docs/adr/0001-architecture-overview.md`](./docs/adr/0001-architecture-overview.md)
and
[`docs/adr/0002-phase2-dogfood-and-sarif.md`](./docs/adr/0002-phase2-dogfood-and-sarif.md).

---

## Evidence, not verdict

Every finding archfit produces carries four qualities:

- **Severity** — how bad is it if true? (`info` / `warn` / `error` / `critical`)
- **Evidence strength** — how deterministic is the detection? (`strong` / `medium` / `weak` / `sampled`)
- **Confidence** — a numeric 0.0–1.0
- **Remediation** — a summary plus a link to a guide, with auto-fix when available

archfit is deliberately conservative: `error` severity requires `strong`
evidence. Heuristic findings are clearly marked. **False positives are treated
as bugs.** The JSON output is ordered deterministically
(severity desc, rule_id asc, path asc) so agents can make stable references
into it and `archfit diff` can produce a stable delta.

When a collector or resolver encounters malformed input it was asked to
interpret, it emits a `ParseFailure` finding rather than silently skipping —
see `model.ParseFailure` and [`CLAUDE.md`](./CLAUDE.md) §13.

---

## The current rule set

All 7 rules ship at `strong` evidence and `experimental` stability.

### `core` pack — universal

| ID | Principle | What it checks |
|---|---|---|
| [`P1.LOC.001`](./docs/rules/P1.LOC.001.md) | Locality | Top-level `AGENTS.md` or `CLAUDE.md` exists at the repo root |
| [`P1.LOC.002`](./docs/rules/P1.LOC.002.md) | Locality | Declared vertical-slice directories carry their own `AGENTS.md` |
| [`P4.VER.001`](./docs/rules/P4.VER.001.md) | Verifiability | A fast verification entrypoint (`Makefile`, `justfile`, `package.json` scripts, …) names a `test` target |
| [`P7.MRD.001`](./docs/rules/P7.MRD.001.md) | Machine-readability | Repos that ship a CLI document their exit codes |

### `agent-tool` pack — opt-in, for repos whose consumers are agents

| ID | Principle | What it checks |
|---|---|---|
| [`P2.SPC.010`](./docs/rules/P2.SPC.010.md) | Spec-first | The tool ships a versioned JSON output schema (`$id` + `schema_version`) |
| [`P7.MRD.002`](./docs/rules/P7.MRD.002.md) | Machine-readability | A `CHANGELOG.md` exists at the repo root |
| [`P7.MRD.003`](./docs/rules/P7.MRD.003.md) | Machine-readability | A repo with a `cmd/` binary records ADRs under `docs/adr/` |

More packs (`web-saas`, `iac`, `mobile`, `data-event`) arrive in Phase 3.
Breadth is deliberately paced: seven solid `strong`-evidence rules beat a
hundred weak ones.

---

## Configuration

archfit reads `.archfit.yaml` (or `.archfit.yml` / `.archfit.json`) from the
repository root. In Phase 1 and 2 the file is parsed as JSON — YAML 1.2 is a
strict superset, so the JSON document round-trips through any YAML tool. Full
YAML syntax (anchors, block scalars, unquoted strings) lands in Phase 3 when
`yaml.v3` is introduced.

The easy way to get started:

```bash
./bin/archfit init .
```

That produces:

```json
{
  "version": 1,
  "project_type": [],
  "profile": "standard",
  "packs": { "enabled": ["core"] },
  "ignore": []
}
```

A richer example with the `agent-tool` pack, risk tiers, overrides, and
expiring suppressions:

```json
{
  "version": 1,
  "project_type": ["agent-tool"],
  "profile": "standard",
  "risk_tiers": {
    "high":   ["src/auth/**", "src/billing/**", "infra/**", "migrations/**"],
    "medium": ["src/features/**"],
    "low":    ["docs/**", "tests/**"]
  },
  "packs": { "enabled": ["core", "agent-tool"] },
  "overrides": { "P4.VER.003": { "timeout_seconds": 60 } },
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

Every `ignore` entry must carry a `reason` and an `expires` date. Expired
suppressions surface as warnings on the next scan, so they cannot silently
rot. Full reference: [`docs/configuration.md`](./docs/configuration.md).

---

## CI integration

archfit's SARIF output plugs directly into GitHub Code Scanning:

```yaml
- name: Build archfit
  run: go install github.com/shibuiwilliam/archfit/cmd/archfit@latest

- name: Scan
  run: archfit scan --format=sarif . > archfit.sarif

- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: archfit.sarif
```

To gate pull requests on *new* findings only (not the existing backlog), use
`archfit diff` against a baseline captured on `main`:

```yaml
- name: Capture baseline (on main)
  run: archfit scan --json . > baseline.json
  # Stash this in a cache / artifact / committed baseline file.

- name: Diff (on PR)
  run: archfit diff baseline.json   # exits 1 when new findings appear
```

The repo's own CI is in
[`.github/workflows/ci.yml`](./.github/workflows/ci.yml): it runs
`make lint`, `make test`, `make self-scan`, and a cross-build matrix over the
five platforms CLAUDE.md §3 pins (`linux/{amd64,arm64}`,
`darwin/{amd64,arm64}`, `windows/amd64`).

---

## Claude agent skill

archfit ships with a Claude Code agent skill at
[`.claude/skills/archfit/SKILL.md`](./.claude/skills/archfit/SKILL.md) — the
canonical project-scope location per the
[Agent Skills docs](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview),
so Claude Code auto-discovers it when you run inside this repo. The skill is
a thin, progressive-disclosure wrapper: the `SKILL.md` entry point stays
under 400 lines, and per-rule remediation guides live under
`.claude/skills/archfit/reference/remediation/` and are loaded on demand.

To use the skill in another repository, copy or symlink
`.claude/skills/archfit/` into that project's `.claude/skills/` directory, or
drop it under `~/.claude/skills/` for personal use across every repo. The
skill's core loop is:

1. **Run**: `archfit scan --json . > /tmp/archfit.json`
2. **Read**: the deterministic `findings[]` array
3. **Propose**: load the matching remediation guide, follow its decision tree
4. **Verify**: re-run the scan — the re-scan is the contract, not the claim

The skill never silently mass-fixes findings. Some remediations require asking
the user first; the guides say which.

---

## Repository layout

```
archfit/
├── cmd/archfit/              # CLI entry point — explicit wiring lives here
├── internal/
│   ├── core/                 # Scheduler, rule execution, finding aggregation
│   ├── model/                # Rule, Finding, Evidence, Metric, FactStore, ParseFailure
│   ├── config/               # .archfit.yaml loading + schema validation
│   ├── collector/            # Fact gatherers (fs, git, schema). Pure data, no judgement.
│   ├── adapter/              # Side-effect boundary: exec, fs-write, net
│   ├── rule/                 # Rule engine core (not the rules themselves)
│   ├── report/               # Renderers: terminal, json, markdown, sarif
│   ├── score/                # Weight-based, normalized scoring
│   └── version/              # Build-time version info
├── packs/
│   ├── core/                 # rules/ resolvers/ fixtures/ AGENTS.md INTENT.md pack_test.go
│   └── agent-tool/           # same shape — opt-in, for agent-facing tools
├── schemas/                  # Versioned JSON Schema: rule / config / output
├── testdata/e2e/             # End-to-end golden tests (`make e2e`)
├── .claude/skills/archfit/   # Claude Code agent skill (canonical location,
│                             # auto-discovered by Claude Code)
├── .github/workflows/ci.yml  # Lint + test + self-scan + cross-build matrix
├── docs/
│   ├── adr/                  # Architecture Decision Records (0001, 0002)
│   ├── rules/                # Human docs per rule
│   ├── configuration.md
│   ├── exit-codes.md
│   └── dependencies.md
├── .archfit.yaml             # archfit's own config — used by `make self-scan`
├── .golangci.yaml            # Lint config (contract even when not executed)
├── .go-arch-lint.yaml        # Boundary rule (`packs/*` cannot import I/O)
├── Makefile
├── CLAUDE.md                 # Contributor contract
├── CONTRIBUTING.md           # Workflow, commit conventions, PR checklist
├── SECURITY.md               # Disclosure policy
├── CHANGELOG.md              # Keep-a-Changelog 1.1.0 format
├── PROJECT.md                # Long-form project overview
└── DEVELOPMENT_PLAN.md       # Phased roadmap
```

The boundary is load-bearing: `packs/*` may import `internal/model` and the
public interfaces in `internal/rule`, but **not** anything that performs I/O.
A rule that needs a new fact grows a Collector, not a new filesystem call.

---

## Development

```bash
make build            # build the CLI into ./bin/archfit
make test             # unit + pack tests (with -race, deterministic)
make e2e              # end-to-end golden tests under testdata/e2e/
make update-golden    # regenerate testdata/e2e/*/expected.json (review the diff!)
make lint             # gofmt + go vet (+ golangci-lint + go-arch-lint if installed)
make self-scan        # run archfit on itself — must exit 0 under --fail-on=error
make self-scan-json   # same, but emit JSON to stdout
make clean
```

Every push and pull request runs the same targets in GitHub Actions, plus a
cross-build matrix for `linux/amd64`, `linux/arm64`, `darwin/amd64`,
`darwin/arm64`, and `windows/amd64`. The self-scan artifact is uploaded for
debugging.

The **self-scan** is the forcing function. If `archfit scan ./` flags
archfit's own code, the change is wrong: either the code needs to be fixed or
the rule needs to be revised. Silent drift is exactly what archfit is
designed to prevent.

---

## Contributing

Before opening a PR, read:

1. [`CLAUDE.md`](./CLAUDE.md) — the contributor contract (binding for both
   humans and agents).
2. [`CONTRIBUTING.md`](./CONTRIBUTING.md) — workflow, commit conventions, PR
   checklist.
3. [`DEVELOPMENT_PLAN.md`](./DEVELOPMENT_PLAN.md) — what belongs to which
   phase, so your change lands in scope.

Highlights:

- **PR size budget**: ≤ 500 changed lines, ≤ 5 packages touched
- **Every new rule** ships with: YAML + schema-valid · resolver · fixture +
  `expected.json` · table test · remediation guide · `docs/rules/<id>.md`
- **Every new Collector** ships with: tests against representative fixtures
  and a fake implementation for downstream tests
- **No** `init()` cross-package registration, reflection-based discovery, or
  global mutable state
- **No** rules that bypass the collector boundary — if a pack needs I/O, add
  a Collector

---

## Security

See [`SECURITY.md`](./SECURITY.md) for the disclosure policy. archfit executes
subprocesses like `git log` against the repository being scanned, so run it
only on repositories you trust or inside a sandbox.

---

## What archfit is *not*

- Not a replacement for `golangci-lint`, `ruff`, `eslint`, or any other
  language-specific linter.
- Not a SAST tool. Use Semgrep, CodeQL, or Trivy for that. archfit can
  *consume* their outputs but does not duplicate them.
- Not a benchmarking tool. The score is a signal for *your own* repository
  over time, not a number to compete on.
- Not a cage. Rules are advice backed by evidence. Suppression exists — with
  a reason and an expiry date — on purpose.

---

## Roadmap

Full plan in [`DEVELOPMENT_PLAN.md`](./DEVELOPMENT_PLAN.md):

- **Phase 1 — 0.1.0** ✅ foundation, `core` pack, 4 rules, JSON/Markdown,
  self-scan green.
- **Phase 2 — 0.2.0** ✅ `init` / `check` / `report` / `diff`, SARIF 2.1.0,
  `agent-tool` pack, end-to-end golden tests, CI, `.golangci.yaml` +
  `.go-arch-lint.yaml` contracts.
- **Phase 3 — next** `iac` / `mobile` / `data-event` / `web-saas` packs,
  metrics pipeline (`context_span_p50`, `blast_radius_score`, …), `archfit
  fix`, release binaries, Docker image, Homebrew tap, YAML config (`yaml.v3`).
- **Phase 4 — 1.0** rule IDs frozen in `core` and `web-saas`, JSON schema v1
  certified, SARIF certified against GitHub Code Scanning, public API
  stability statement.

---

## License

Apache 2.0 — see [`LICENSE`](./LICENSE).
