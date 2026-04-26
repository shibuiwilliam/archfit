---
id: 0003
title: LLM-assisted explanation via Google Gemini
status: accepted
date: 2026-04-24
tags: [architecture, llm, adapter, dependencies]
---

# ADR 0003 — LLM-assisted explanation via Google Gemini

## Context

archfit's static remediation docs answer "what should I do when a rule fires" in
the general case. They cannot answer "why did *my* repo trigger this, given the
specific evidence" or "what exact change would fix it here."

An LLM closes that gap. The danger is obvious: LLM calls are expensive, slow,
non-deterministic, and a common source of supply-chain and data-leak risk.
CLAUDE.md §13 makes the rule explicit: *"Do not introduce LLM calls on the hot
path. Any LLM-assisted explanation is opt-in via `--with-llm` and lives behind
a clearly isolated adapter."*

This ADR records how that constraint is met, the choice of `google.golang.org/genai` as the
initial provider SDK, and the Go toolchain bump it forced.

## Decision

### 1. Opt-in via `--with-llm`; never on the scan hot path

`archfit scan .`, `archfit score .`, and every command that produces the
core JSON contract are byte-identical whether or not a Gemini API key is
configured. The LLM is invoked only when the user passes `--with-llm` (and
supplies a key).

The golden tests under `testdata/e2e/` assert byte-for-byte equality of the
base JSON output. They run without `--with-llm` and will catch any leakage
of LLM content into the default path.

### 2. One adapter, one network boundary

All Gemini I/O lives under `internal/adapter/llm/`. Every other package
consumes a `llm.Client` interface:

```go
type Client interface {
    Explain(ctx context.Context, prompt Prompt) (Suggestion, error)
}
```

`adapter/llm/real.go` implements `Client` using `google.golang.org/genai`
(the only file that imports it). `adapter/llm/fake.go` returns canned
responses for tests and drives every unit test. `cmd/archfit/main.go` is
the single place `Real` is instantiated, mirroring how `exec.Real` is
wired in Phase 1.

Packs never see the LLM. They cannot — the boundary is enforced both by
`go-arch-lint` and by the fact that `llm.Client` is only available in the
scheduler after a scan completes.

### 3. Configuration by environment only

The API key is read from `GOOGLE_API_KEY` (preferred) or `GEMINI_API_KEY`.
It is never read from `.archfit.yaml`, never written to logs, and never
embedded in the binary. Missing key → the command exits `4` (configuration
error) with a single-sentence message. This matches the contract in
`docs/exit-codes.md`.

### 4. Failure is graceful

LLM errors append a warning to stderr and degrade to static remediation.
They never fail the scan. The CLI exit code is the same as a non-LLM run
for the same finding set. This preserves CI-gate semantics: enabling
`--with-llm` must not flip a green build red because of a transient API
outage.

### 5. Budget and cache in-process

Two safety rails ship in Phase 3a:

- `--llm-budget N` (default 5) — no more than N findings per run receive
  an LLM call. Remaining findings keep their static remediation.
- In-memory response cache keyed by SHA-256 of `(model, prompt)`. Within
  one run, identical prompts are free. A disk cache is Phase 3b.

Per-call timeout: 30 seconds, enforced at the adapter. Longer prompts
are truncated at the boundary — the LLM is not given the whole repo.

### 6. Output is additive

`findings[]` gains an optional `llm_suggestion` field. It is never emitted
when absent, so pre-existing consumers see byte-identical JSON for a
non-`--with-llm` run. `schema_version` remains `0.1.0` (additive within
the minor, per CLAUDE.md §9).

SARIF results carry the LLM suggestion inside
`results[].properties.llm_suggestion` when present — also additive.

### 7. Dependency choice: `google.golang.org/genai`

The SDK is Google's official unified Go client for both the Gemini
Developer API and Vertex AI. It covers the surface area archfit needs
(text generation with prompt caching and safety controls) and is
actively maintained.

Alternatives considered and rejected:

- **Hand-rolled HTTP client against `generativelanguage.googleapis.com`**:
  one fewer dep, but we re-implement error taxonomy, auth refresh, rate-
  limit parsing, and streaming every time the SDK adds a feature. Not
  worth it for the first provider.
- **A vendor-neutral SDK (e.g., LangChain Go, OpenAI SDK adapted)**:
  adds abstraction we don't need yet. A clean `llm.Client` interface
  inside `internal/adapter/llm/` gives us the same flexibility to add
  OpenAI/Anthropic later — see Phase 3b.

### 8. Go toolchain bump: 1.23 → 1.24

`google.golang.org/genai v1.54.0` requires Go 1.24. CLAUDE.md §3 says
"do not bump casually"; this is not casual, it is dependency-forced and
the alternative is dropping the SDK. The bump is explicitly scoped to
Phase 3a and noted in `CHANGELOG.md` and `docs/dependencies.md`.

Cross-compilation targets (`linux/{amd64,arm64}`, `darwin/{amd64,arm64}`,
`windows/amd64`) are unchanged; Go 1.24 supports all of them.

### 9. What about `CLAUDE.md` §3 "prefer the standard library"?

This is archfit's first non-stdlib runtime dependency. It triggers all
three §3 requirements:

- Justification comment at the import site (`adapter/llm/real.go`).
- Entry in `docs/dependencies.md` listing transitive deps and why.
- This ADR.

Every indirect dep (`cloud.google.com/go/auth`, `google.golang.org/grpc`,
etc.) is pulled by `genai`. They are not imported directly anywhere in
archfit's own code.

## Consequences

**Positive**

- Consumers get a dramatically better "why did my repo fail P7.MRD.001?"
  answer when they opt in.
- The adapter's shape (`Client` interface + Fake + Real) mirrors the
  `exec` adapter from Phase 1, so the pattern is familiar to contributors.
- Cost is bounded by the per-run budget; a user cannot accidentally
  run a 10,000-finding scan with `--with-llm` and receive a surprise bill.

**Negative**

- `go.sum` grows. `google.golang.org/genai` transitively pulls in
  `cloud.google.com/go`, `grpc`, `protobuf`, `x/crypto`, etc. We accept
  this as the cost of a first-class provider SDK.
- The `go.mod` minimum is now 1.24. Consumers on older toolchains must
  upgrade to build from source. Pre-built binaries are unaffected.
- Non-determinism enters the product. This is contained — base scan
  output is deterministic, `--with-llm` output is not, and the two paths
  are tested separately.

### 10. Additional backends: OpenAI and Claude (Phase 3b)

OpenAI support was added using `github.com/openai/openai-go/v3`, the
official OpenAI Go SDK. Claude support was added using
`github.com/anthropics/anthropic-sdk-go`, the official Anthropic Go SDK.
Both follow the same pattern as the Gemini backend:

- `adapter/llm/openai.go` implements `Client` using the Chat Completions
  API. It is the only file that imports the OpenAI SDK.
- `adapter/llm/anthropic.go` implements `Client` using the Messages API.
  It is the only file that imports the Anthropic SDK.
- Configuration via `OPENAI_API_KEY` / `ANTHROPIC_API_KEY` environment
  variables respectively.
- Default models: `gpt-5.4-mini` (OpenAI), `claude-sonnet-4-20250514`
  (Claude).
- Auto-detection: `FromEnv()` detects the backend from whichever API key
  is present. Priority: `ANTHROPIC_API_KEY` > `OPENAI_API_KEY` >
  `GOOGLE_API_KEY` > `GEMINI_API_KEY`.
- Explicit override: `--llm-backend={gemini|openai|claude}` forces a
  specific backend regardless of which keys are present.
- The same Budget and Cached wrappers apply — no provider-specific
  special-casing outside of the concrete client.

The `llm.Client` interface proved sufficient without modification for
all three providers, validating the original design decision to use a
local interface rather than a vendor-neutral SDK.

## Status

Accepted. Gemini, OpenAI, and Claude backends are available. Additional
backends (local Ollama, etc.) can be added by implementing `llm.Client`
without modifying existing code.
