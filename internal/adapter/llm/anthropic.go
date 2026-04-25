package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	// github.com/anthropics/anthropic-sdk-go — Anthropic's official Go SDK
	// for the Messages API. Third runtime LLM dependency in archfit (after
	// google.golang.org/genai and openai-go). Justified in docs/dependencies.md
	// and ADR 0003. Used only here so the rest of the codebase depends on the
	// local llm.Client interface, not on the SDK types.
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// Anthropic is the production llm.Client backed by github.com/anthropics/anthropic-sdk-go.
//
// Instantiate once at startup from cmd/archfit/main.go — not from packs,
// not from resolvers. A missing API key produces ErrNotConfigured, which
// the CLI maps to exit code 4 (configuration error).
type Anthropic struct {
	client  *anthropic.Client
	model   string
	timeout time.Duration
}

// NewAnthropic builds an Anthropic client from Config. Returns ErrNotConfigured
// when cfg.APIKey is empty.
func NewAnthropic(_ context.Context, cfg Config) (*Anthropic, error) {
	if cfg.APIKey == "" {
		return nil, ErrNotConfigured
	}
	if cfg.Model == "" {
		cfg.Model = DefaultClaudeModel
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))
	return &Anthropic{
		client:  &client,
		model:   cfg.Model,
		timeout: cfg.Timeout,
	}, nil
}

// Explain issues one Messages API call. Errors are wrapped so callers can
// log them without leaking the API key. A per-call timeout bounds the hang.
func (a *Anthropic) Explain(ctx context.Context, rule model.Rule, finding model.Finding, prompt Prompt) (Suggestion, error) {
	callCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	start := time.Now()

	maxTok := int64(prompt.MaxOutputTokens)
	if maxTok <= 0 {
		maxTok = 400
	}

	params := anthropic.MessageNewParams{
		Model:     a.model,
		MaxTokens: maxTok,
		System: []anthropic.TextBlockParam{
			{Text: prompt.System},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt.User)),
		},
	}
	temp := 0.2 // low: we want stable, concise explanations.
	params.Temperature = anthropic.Float(temp)

	resp, err := a.client.Messages.New(callCtx, params)
	if err != nil {
		return Suggestion{}, fmt.Errorf("llm.Explain(%s): %w", rule.ID, err)
	}

	text := extractAnthropicText(resp)
	return Suggestion{
		Text:         text,
		Model:        a.model,
		Truncated:    resp.StopReason == anthropic.StopReasonMaxTokens,
		LatencyMS:    time.Since(start).Milliseconds(),
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}, nil
}

// extractAnthropicText concatenates all text blocks from the response.
func extractAnthropicText(msg *anthropic.Message) string {
	var parts []string
	for i := range msg.Content {
		if v, ok := msg.Content[i].AsAny().(anthropic.TextBlock); ok {
			parts = append(parts, v.Text)
		}
	}
	return strings.Join(parts, "")
}

// Close releases SDK resources. Safe to call more than once.
func (a *Anthropic) Close() error {
	// anthropic-sdk-go does not require explicit cleanup; connections are
	// pooled at the transport level.
	return nil
}
