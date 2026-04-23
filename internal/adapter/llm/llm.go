// Package llm is archfit's single network boundary for LLM calls.
//
// CLAUDE.md §4 and §13 require both that network I/O lives in internal/adapter/
// and that LLM calls are strictly opt-in (`--with-llm`) and off the default
// scan path. Every other package depends on the `Client` interface defined
// here — never on a concrete SDK type.
//
// Two implementations:
//
//   - Fake (fake.go): returns canned responses. Drives every unit test.
//   - Real (real.go): wraps google.golang.org/genai. Instantiated only from
//     cmd/archfit/main.go when the user passes --with-llm and an API key is
//     configured.
//
// Errors are non-fatal to the caller by convention: when the Client returns
// an error, the CLI logs it to stderr and degrades to static remediation.
// The scan's base exit code is never flipped by an LLM failure.
package llm

import (
	"context"
	"errors"
	"time"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// DefaultTimeout bounds any single Explain call. Individual implementations
// may choose a shorter timeout. 30s is long enough for Gemini's p99 but short
// enough that a hung call does not block a CI job.
const DefaultTimeout = 30 * time.Second

// DefaultModel is Gemini's current cost/quality sweet spot for short
// explanations. Override via LLM_MODEL env var at runtime.
const DefaultModel = "gemini-2.5-flash"

// Prompt is the LLM adapter's input. The caller constructs it from a Rule
// and a Finding; the adapter is not concerned with archfit's scoring model.
type Prompt struct {
	// System is the system instruction (role-setting text). Stable per call
	// site, so identical System strings hit the in-run cache.
	System string
	// User is the task-specific body. Contains the rule definition and the
	// specific evidence from the finding. Truncated by the adapter if it
	// exceeds MaxUserBytes.
	User string
	// MaxOutputTokens bounds the response. Keep this tight — short
	// suggestions are more useful than long ones, and cheaper.
	MaxOutputTokens int
}

// MaxUserBytes caps the prompt body. Prevents the caller from accidentally
// sending the entire repo to the LLM.
const MaxUserBytes = 8 * 1024

// Suggestion is the adapter's output. A Client populates exactly these fields.
type Suggestion struct {
	Text      string `json:"text"`
	Model     string `json:"model"`
	Truncated bool   `json:"truncated,omitempty"`
	CacheHit  bool   `json:"cache_hit,omitempty"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
}

// Client is the LLM boundary. Only this interface is visible to callers.
type Client interface {
	// Explain returns a suggestion for the given finding, given its rule.
	// Implementations must be safe for concurrent use.
	Explain(ctx context.Context, rule model.Rule, finding model.Finding, prompt Prompt) (Suggestion, error)

	// Close releases any underlying resources. Idempotent. Callers should
	// defer Close() even when Client is a Fake — the contract should not
	// depend on the implementation.
	Close() error
}

// ErrNotConfigured is returned by constructors when required configuration
// (e.g., an API key) is missing. The CLI maps this to exit code 4.
var ErrNotConfigured = errors.New("llm: not configured (set GOOGLE_API_KEY or GEMINI_API_KEY)")

// ErrBudgetExhausted is returned when a caller exceeds its per-run LLM budget.
var ErrBudgetExhausted = errors.New("llm: per-run budget exhausted")

// Config is the adapter-level configuration shared by Real and any future backends.
type Config struct {
	APIKey  string
	Model   string
	Timeout time.Duration
}

// FromEnv builds a Config from process environment variables. Callers get a
// "(Config, ok=false)" result when no API key is present so they can emit
// a clear "llm not configured" message without needing to sniff env vars
// themselves.
func FromEnv(getenv func(string) string) (Config, bool) {
	key := getenv("GOOGLE_API_KEY")
	if key == "" {
		key = getenv("GEMINI_API_KEY")
	}
	cfg := Config{
		APIKey:  key,
		Model:   orDefault(getenv("LLM_MODEL"), DefaultModel),
		Timeout: DefaultTimeout,
	}
	return cfg, key != ""
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
