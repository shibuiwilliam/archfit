package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	// github.com/openai/openai-go/v3 — OpenAI's official Go SDK for the
	// Chat Completions API. Second runtime dependency in archfit (after
	// google.golang.org/genai). Justified in docs/dependencies.md and
	// ADR 0003. Used only here so the rest of the codebase depends on the
	// local llm.Client interface, not on the SDK types.
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// OpenAI is the production llm.Client backed by github.com/openai/openai-go/v3.
//
// Instantiate once at startup from cmd/archfit/main.go — not from packs,
// not from resolvers. A missing API key produces ErrNotConfigured, which
// the CLI maps to exit code 4 (configuration error).
type OpenAI struct {
	client  *openai.Client
	model   string
	timeout time.Duration
}

// NewOpenAI builds an OpenAI client from Config. Returns ErrNotConfigured when
// cfg.APIKey is empty.
func NewOpenAI(_ context.Context, cfg Config) (*OpenAI, error) {
	if cfg.APIKey == "" {
		return nil, ErrNotConfigured
	}
	if cfg.Model == "" {
		cfg.Model = DefaultOpenAIModel
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	client := openai.NewClient(option.WithAPIKey(cfg.APIKey))
	return &OpenAI{
		client:  &client,
		model:   cfg.Model,
		timeout: cfg.Timeout,
	}, nil
}

// Explain issues one Chat Completion call. Errors are wrapped so callers can
// log them without leaking the API key. A per-call timeout bounds the hang.
func (o *OpenAI) Explain(ctx context.Context, rule model.Rule, finding model.Finding, prompt Prompt) (Suggestion, error) {
	callCtx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	start := time.Now()

	params := openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.DeveloperMessage(prompt.System),
			openai.UserMessage(prompt.User),
		},
	}
	if prompt.MaxOutputTokens > 0 {
		params.MaxTokens = openai.Int(int64(prompt.MaxOutputTokens))
	}
	temp := 0.2 // low: we want stable, concise explanations.
	params.Temperature = openai.Float(temp)

	resp, err := o.client.Chat.Completions.New(callCtx, params)
	if err != nil {
		return Suggestion{}, fmt.Errorf("llm.Explain(%s): %w", rule.ID, err)
	}
	if len(resp.Choices) == 0 {
		return Suggestion{}, errors.New("llm.Explain: empty choices")
	}

	text := resp.Choices[0].Message.Content
	maxTok := int32(prompt.MaxOutputTokens)
	return Suggestion{
		Text:      text,
		Model:     o.model,
		Truncated: maxTok > 0 && int32(len(text)) >= maxTok*4, // rough heuristic; tokens ~ 4 bytes
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}

// Close releases SDK resources. Safe to call more than once.
func (o *OpenAI) Close() error {
	// openai-go does not require explicit cleanup; connections are pooled
	// at the transport level.
	return nil
}
