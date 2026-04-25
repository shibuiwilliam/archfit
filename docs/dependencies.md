# Runtime and build dependencies

Per `CLAUDE.md` §3, archfit prefers the standard library. Any external dep
requires (a) a justification comment at the import site on first use, (b) an
entry below, and (c) an ADR when the dep affects the architecture.

## Direct runtime dependencies

| Module | Version | First used in | Why | ADR |
|---|---|---|---|---|
| `google.golang.org/genai` | `v1.54.0` | `internal/adapter/llm/real.go` (Phase 3a) | Official Google SDK for the Gemini Developer API and Vertex AI. Used behind the `llm.Client` adapter to implement `--with-llm`. LLM calls are strictly opt-in and off the hot path (CLAUDE.md §13). | [ADR 0003](./adr/0003-llm-explanation.md) |
| `github.com/openai/openai-go/v3` | `v3.32.0` | `internal/adapter/llm/openai.go` (Phase 3b) | Official OpenAI Go SDK for the Chat Completions API. Second LLM backend behind `llm.Client`. Selected when `OPENAI_API_KEY` is set or `--llm-backend=openai` is passed. Same opt-in/budget/cache model as the Gemini backend. | [ADR 0003](./adr/0003-llm-explanation.md) |
| `github.com/anthropics/anthropic-sdk-go` | `v1.38.0` | `internal/adapter/llm/anthropic.go` (Phase 3b) | Official Anthropic Go SDK for the Messages API. Third LLM backend behind `llm.Client`. Selected when `ANTHROPIC_API_KEY` is set or `--llm-backend=claude` is passed. Same opt-in/budget/cache model as the other backends. | [ADR 0003](./adr/0003-llm-explanation.md) |

## Transitive dependencies

These are pulled in by `google.golang.org/genai` and `github.com/openai/openai-go/v3`
and never imported directly by archfit's own code. They are listed for
auditability, not for use:

- `cloud.google.com/go`, `cloud.google.com/go/auth`, `cloud.google.com/go/compute/metadata` — Google auth / ADC support for Vertex backend.
- `google.golang.org/grpc`, `google.golang.org/genproto/...`, `google.golang.org/protobuf` — RPC transport for Vertex.
- `github.com/gorilla/websocket` — streaming transport.
- `github.com/golang/groupcache`, `github.com/google/go-cmp`, `github.com/google/s2a-go`, `github.com/googleapis/enterprise-certificate-proxy` — auth / TLS support.
- `go.opencensus.io` — SDK-internal tracing.
- `golang.org/x/{crypto,net,sys,text}` — standard-library-adjacent modules the SDK relies on.
- `github.com/tidwall/{gjson,match,pretty,sjson}` — JSON manipulation used by the OpenAI and Anthropic SDKs.

## Go toolchain

Minimum: **Go 1.24** (bumped from 1.23 in Phase 3a because `google.golang.org/genai`
requires it). See ADR 0003 for the rationale.

## Not planned

- Reflection-based YAML / JSON libraries — violate P3 (shallow explicitness).
- CLI frameworks larger than `flag` (`cobra`, `urfave/cli`) — Phase 1's
  commands are small enough that `flag` is enough and the explicit dispatch
  in `main.go` is the feature, not the bug.
- A vendor-neutral LLM abstraction library (LangChain Go, etc.) — the
  `llm.Client` interface inside `internal/adapter/llm/` is enough abstraction.
  All three backends (Gemini, OpenAI, Claude) share the same interface, budget,
  and cache layers without any third-party abstraction framework.

## Planned for later phases

- `gopkg.in/yaml.v3` — proper YAML parsing for `.archfit.yaml` with anchors,
  block scalars, and comments. Phase 3b.
- `github.com/santhosh-tekuri/jsonschema/v5` — schema validation for rule YAML
  and output JSON. Phase 3c.

Each addition will land with its own ADR.
