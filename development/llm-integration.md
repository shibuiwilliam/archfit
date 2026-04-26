# LLM Integration

## Architecture

The LLM subsystem lives entirely in `internal/adapter/llm/`. It is the single network boundary for all LLM calls.

```
internal/adapter/llm/
├── llm.go           # Client interface, Config, Backend enum, FromEnv()
├── llm_test.go      # Interface contract tests
├── real.go          # Google Gemini backend
├── openai.go        # OpenAI backend
├── anthropic.go     # Anthropic Claude backend
├── fake.go          # Canned responses for tests
├── budget.go        # Budget wrapper (limits calls per run)
├── cache.go         # In-memory cache (SHA-256 keyed)
└── prompt.go        # Prompt construction helpers
```

## Client Interface

```go
type Client interface {
    Explain(ctx context.Context, rule model.Rule, finding model.Finding, prompt Prompt) (Suggestion, error)
    Close() error
}
```

All backends implement this single method. The caller does not know which backend is active.

## Composition Chain

Clients are composed as: `inner → Budget → Cached`

```go
inner := llm.NewReal(ctx, cfg)      // or NewOpenAI() or NewAnthropic()
budgeted := llm.NewBudget(inner, 5) // max 5 calls
cached := llm.NewCached(budgeted)   // SHA-256 dedup
```

This means:
1. Cache check first (free, no network)
2. Budget check second (returns `ErrBudgetExhausted` if exceeded)
3. Actual API call last

## Backend Selection

Auto-detection priority (from `FromEnv()`):
1. `ANTHROPIC_API_KEY` → Claude backend
2. `OPENAI_API_KEY` → OpenAI backend
3. `GOOGLE_API_KEY` or `GEMINI_API_KEY` → Gemini backend

Override with `--llm-backend={claude|openai|gemini}`.

## Backend Details

### Gemini (`real.go`)

- SDK: `google.golang.org/genai` v1.54.0
- Default model: `gemini-2.5-flash`
- Override: `LLM_MODEL` env var
- 30s timeout per call

### OpenAI (`openai.go`)

- SDK: `github.com/openai/openai-go/v3` v3.32.0
- Default model: `gpt-5.4-mini`
- Override: `LLM_MODEL` env var
- Chat Completions API

### Claude (`anthropic.go`)

- SDK: `github.com/anthropics/anthropic-sdk-go` v1.38.0
- Default model: `claude-sonnet-4-20250514`
- Override: `LLM_MODEL` env var
- Messages API

## Prompt Design

Prompts are constructed in `llm.go`:

```go
func BuildFindingPrompt(rule model.Rule, finding model.Finding, projectType []string) Prompt
func BuildRulePrompt(rule model.Rule, projectType []string) Prompt
```

The prompt includes:
- Rule ID, title, severity, rationale, static remediation
- Finding path, message, evidence map
- Repo's `project_type` from config

The prompt does NOT include:
- Source code or file contents
- Environment variables
- Git history or author information
- Evidence values longer than 8 KiB (truncated with `[truncated]` marker)

LLM is instructed to produce ≤200 words in three sections: *why it matters here*, *concrete fix*, *when to suppress*.

## Safety Guarantees

1. **Opt-in only**: no LLM calls without `--with-llm`
2. **Bounded cost**: `--llm-budget N` caps calls (default 5)
3. **Never fails the scan**: API errors degrade to static remediation
4. **Minimal data**: rule metadata + finding evidence only
5. **No source code sent**: ever

## Adding a New Backend

1. Create `internal/adapter/llm/<backend>.go`
2. Implement `Client` interface
3. Add `Backend<Name>` constant to `llm.go`
4. Add case to `FromEnvWithBackend()` and `buildLLMClient()` in `main.go`
5. Add SDK dependency to `go.mod` with justification in `docs/dependencies.md`
6. Test with `llm.NewFake()` — no real API calls in tests

## Testing

All LLM code paths use `llm.NewFake()`:

```go
fake := llm.NewFake(llm.FakeConfig{
    Response: "Fake explanation text",
    Model:    "fake-model",
})
```

The fake returns canned responses immediately. No network calls in any test.

Budget and cache wrappers have their own unit tests with the fake inner client.
