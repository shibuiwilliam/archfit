# IDEA.md -- Why archfit exists

## The shift

For decades, "good architecture" meant something clear: clean module
boundaries for human teams, runtime performance within SLAs, deployment
strategies that match organizational topology. Conway's Law described the
status quo. Architecture reviews asked: *Can the team that owns this service
reason about it and ship it independently?*

That question is no longer sufficient.

Coding agents now read, modify, and verify code at scale. They do not attend
stand-ups. They do not carry institutional memory. They do not intuit that
"we never touch the billing module without a second pair of eyes." They
start with whatever the repository tells them -- and if the repository tells
them nothing, they guess.

The question has shifted: **Can an agent change this system without breaking
it?**

The answer depends less on the code's runtime behavior than on the *shape* of
the repository itself -- the terrain an agent must navigate. archfit measures
that terrain.

---

## The seven properties

Agents succeed or fail based on seven architectural properties. These are not
inventions; they are patterns observed in repositories where agents reliably
produce correct, safe changes -- and their absence in repositories where
agents routinely cause regressions, security holes, or silent data
corruption.

### P1 -- Locality

*Can a change be understood and verified from a narrow slice of the repo?*

An agent's context window is finite. If understanding a change requires
reading 40 files across 12 packages, the agent will either hallucinate
context it hasn't read, or request so many tool calls that latency becomes
unacceptable. Repositories structured as vertical slices -- where each
feature or domain carries its own docs, tests, and contracts in one
directory subtree -- let agents work with confidence inside a bounded scope.

Locality also matters for humans. But agents enforce it brutally: if the
relevant context doesn't fit in the window, the agent *cannot* succeed.
There is no "I'll just remember from last week."

### P2 -- Spec-first

*Are contracts executable artifacts -- schemas, types, generated clients --
rather than prose?*

An agent cannot reason about a contract described in a Confluence page. It
can reason about a JSON Schema, a protobuf definition, or a TypeScript
interface. When the contract is code, the agent can validate its changes
against the contract automatically. When the contract is prose, the agent
must guess -- and guesses compound.

Spec-first also means that changes to contracts are detectable by machines.
A breaking change to an OpenAPI spec triggers a test failure. A breaking
change to a paragraph in a wiki triggers nothing.

### P3 -- Shallow explicitness

*Is behavior visible without chasing reflection or ten layers of
indirection?*

Agents navigate code by reading it, grepping it, and following call chains.
Reflection-based plugin systems, `init()`-based auto-registration across
packages, and deep inheritance hierarchies are adversarial to this workflow.
The agent searches for where a handler is registered -- and finds nothing,
because registration happens via struct tags evaluated at runtime.

Shallow code is not simple code. It is code where the path from intent to
execution is traceable by reading, not by running. The boring explicit
registry that lists every handler in one file is more agent-friendly than
the elegant auto-discovery system that requires understanding the framework's
lifecycle.

### P4 -- Verifiability

*Can correctness be proven locally in seconds to a few minutes?*

An agent's value collapses when the feedback loop is slow. If `make test`
takes 45 minutes, the agent either waits (expensive) or skips verification
(dangerous). Repositories with fast, reliable test suites let agents iterate
in tight loops: change, verify, change, verify. This is how agents produce
correct code -- not by getting it right on the first try, but by closing the
loop quickly.

Verifiability also means the test suite is trustworthy. Flaky tests teach
agents to ignore failures. Mocked tests that diverge from production teach
agents that passing tests mean nothing. Both outcomes are worse than having
no tests at all, because they create false confidence.

### P5 -- Aggregation of dangerous capabilities

*Are auth, billing, migrations, and infra concentrated and guarded?*

When dangerous operations are scattered across the codebase, every change an
agent makes has a non-trivial probability of touching something critical. When
they are aggregated behind clear boundaries -- an `internal/adapter/` package,
a `migrations/` directory, a dedicated auth service -- the agent can be
instructed (or constrained) to avoid those areas, or to treat changes there
with extra scrutiny.

This is not about preventing agents from touching dangerous code. It is
about making dangerous code *identifiable* so that appropriate guardrails
can be applied.

### P6 -- Reversibility

*Can every change be rolled back cheaply? Is blast radius bounded?*

Agents make mistakes. This is not a flaw; it is a statistical certainty when
changes are produced at high volume. The architecture should assume mistakes
and make them cheap. Feature flags, backward-compatible migrations, canary
deployments, and small pull requests all reduce the cost of an incorrect
change from "production incident" to "reverted commit."

Irreversibility is the enemy. A migration that drops a column. A deployment
without a rollback path. A configuration change that requires a full
re-provisioning to undo. These are the scenarios where a single agent mistake
becomes an organizational crisis.

### P7 -- Machine-readability

*Are errors, logs, ADRs, and CLIs readable by machines, not only humans?*

An agent consuming a CLI's output needs structured data: JSON with stable
field names, exit codes with documented semantics, error messages with
machine-parseable codes. An agent reading a project's history needs ADRs
in a predictable format, a changelog that follows a convention, and commit
messages that distinguish features from fixes.

Machine-readability is not about dumbing things down. It is about providing
a structured interface alongside the human one. The terminal output can be
beautiful; the `--json` output must be stable.

---

## What archfit does

archfit scans a repository and evaluates it against these seven properties. It
produces:

- **A score** per principle (0-100, weight-normalized)
- **Findings** with severity, evidence strength, confidence, and concrete
  remediation advice
- **Metrics** that track the terrain over time
- **Diffs** between scans, so teams can gate pull requests on regressions

It does this deterministically. The base scan makes no network calls, requires
no API keys, and produces byte-identical output for identical input. An
optional `--with-llm` flag enriches findings with provider-specific
explanations (Claude, OpenAI, or Gemini), but this is strictly additive --
removing it changes nothing about the scan result.

archfit is not a linter. It does not check code style, find bugs, or detect
vulnerabilities. It sits *above* those tools and asks a different question:
given that your linter passes and your tests are green, is the repository
*shaped* in a way that lets an agent keep it that way?

---

## What changes when you use archfit

### Before: architecture is implicit

Most teams have architectural principles. Few encode them. The knowledge
lives in senior engineers' heads, in onboarding docs that drift from reality,
in code review comments that say "we don't do it that way here" without
explaining why. When an agent arrives, none of this context is available.
The agent sees files, not intentions.

**Result**: Agents produce changes that are locally correct but
architecturally wrong. They scatter configuration across modules because
no boundary told them not to. They add a new CLI command without documenting
its exit codes because nothing enforced that convention. Each change erodes
the architecture a little -- and since agents produce changes at high volume,
the erosion is fast.

### After: architecture is measured

archfit makes architectural expectations explicit and measurable. A team runs
`archfit scan .` and gets a score. They add `archfit diff` to their CI
pipeline and block pull requests that introduce architectural regressions.
The agent skill reads the scan output and proposes targeted remediations.

**Result**: The architecture has a feedback loop. Drift is detected at the
speed of CI, not at the speed of code review. Agents receive structured
guidance about what the repository expects, and their changes are validated
against those expectations automatically.

Concretely:

1. **New contributors (human or agent) have an entry point.** `AGENTS.md` or
   `CLAUDE.md` at the repo root tells them where to start, what to avoid,
   and how to verify their changes. P1.LOC.001 checks for this.

2. **Contracts are enforceable.** A versioned JSON Schema for the tool's
   output means an agent can validate its changes against the schema, not
   against a mental model of what the output "should" look like.
   P2.SPC.010 checks for this.

3. **The feedback loop is fast.** If `make test` exists and runs in seconds,
   the agent can verify every change incrementally. P4.VER.001 checks for
   this.

4. **Dangerous areas are visible.** When auth, billing, and infrastructure
   code live behind explicit boundaries, agents (and their supervisors) can
   apply proportional scrutiny. P5 rules (planned) will check for this.

5. **History is machine-readable.** A `CHANGELOG.md` and `docs/adr/`
   directory give agents structured access to the project's decision history,
   not just its code. P7.MRD.002 and P7.MRD.003 check for this.

6. **Regressions are caught.** `archfit diff` compares the current scan
   against a baseline and exits non-zero when new findings appear. In CI,
   this gates pull requests on architectural quality -- the same way linters
   gate on code quality and tests gate on correctness.

### The compounding effect

The value of archfit is not in any single rule. It is in the compounding
effect of measuring architecture continuously.

A repository that starts at score 60 and improves 5 points per quarter will,
within a year, be dramatically easier for agents to work on -- and for humans
too. The improvements are not cosmetic: they are structural changes that
reduce the probability of agent-caused regressions. Each improvement makes
the next improvement easier, because the architecture becomes more regular
and predictable.

Conversely, a repository that never measures its architectural fitness will
drift. Not because anyone intended it, but because drift is the default.
Every "quick hack" that scatters a concern across modules, every
configuration that lives in prose instead of schema, every test suite that
takes too long to run -- these are individually harmless but collectively
lethal to agent productivity.

archfit makes the invisible visible.

---

## The self-scan principle

archfit practices what it measures. The repository passes its own scan at
score 100.0 with all rules enabled. Every architectural decision in archfit's
codebase is evaluated against the rule: *"Would archfit flag this?"*

- **Locality**: each rule pack is a vertical slice with its own `AGENTS.md`,
  `INTENT.md`, tests, and fixtures.
- **Spec-first**: rules are declared in YAML and validated against a JSON
  Schema. The output format conforms to a versioned schema.
- **Shallow explicitness**: no reflection-based plugin systems, no `init()`
  auto-registration. A boring registry in `main.go` lists every pack.
- **Verifiability**: `make test` runs in under 30 seconds. `make lint` in
  under 5.
- **Aggregation**: all subprocess execution, filesystem access, and network
  I/O go through adapters in `internal/adapter/`. Rules never touch these
  directly.
- **Reversibility**: every rule carries a `stability` field. Experimental
  rules are off by default.
- **Machine-readability**: `--json` output is a first-class citizen with a
  versioned schema. Every error has a code, details, and remediation.

This is not vanity. If archfit cannot follow its own principles, those
principles are either wrong or impractical. The self-scan is the forcing
function that keeps the tool honest. If a code change makes archfit fail
its own scan, the change must either fix the self-violation or carry a
time-limited suppression with a written rationale.

---

## Who this is for

**Teams adopting coding agents** who want guardrails that scale. Manual code
review catches architectural drift -- eventually. archfit catches it on every
push.

**Platform teams** building internal developer platforms where agents are
first-class users. archfit provides a measurable quality signal for
repository fitness, analogous to how test coverage provides a signal for
correctness.

**Open-source maintainers** who want external contributors (human or agent)
to succeed without a 30-minute orientation call. The rules encode what the
maintainer would say in code review; the scan checks it before the PR is
opened.

**Individual developers** who want to understand why agents struggle with
their codebase and what to change first. The score gives a starting point;
the findings give a prioritized list.

---

## What archfit is not

archfit does not replace your linter, your SAST scanner, your test suite, or
your code review process. It does not judge your choice of language, framework,
or cloud provider. It does not require an LLM to run. It does not assign blame.

It asks one question: **Is this repository shaped for agents to succeed?**

And it answers with evidence.
