package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	// google.golang.org/genai — Google's official unified Go SDK for the
	// Gemini Developer API and Vertex AI. First and only non-stdlib runtime
	// dependency in archfit. Justified in docs/dependencies.md and ADR 0003.
	// Used only here so the rest of the codebase depends on the local
	// llm.Client interface, not on the SDK types.
	"google.golang.org/genai"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// Real is the production llm.Client, backed by google.golang.org/genai.
//
// Instantiate once at startup from cmd/archfit/main.go — not from packs,
// not from resolvers. A missing API key produces ErrNotConfigured, which
// the CLI maps to exit code 4 (configuration error).
type Real struct {
	client  *genai.Client
	model   string
	timeout time.Duration
}

// NewReal builds a Real client from Config. Returns ErrNotConfigured when
// cfg.APIKey is empty.
func NewReal(ctx context.Context, cfg Config) (*Real, error) {
	if cfg.APIKey == "" {
		return nil, ErrNotConfigured
	}
	if cfg.Model == "" {
		cfg.Model = DefaultGeminiModel
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}
	return &Real{
		client:  client,
		model:   cfg.Model,
		timeout: cfg.Timeout,
	}, nil
}

// Explain issues one GenerateContent call. Errors are wrapped so callers can
// log them without leaking the API key. A per-call timeout bounds the hang.
func (r *Real) Explain(ctx context.Context, rule model.Rule, finding model.Finding, prompt Prompt) (Suggestion, error) {
	callCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	start := time.Now()

	sys := &genai.Content{Parts: []*genai.Part{{Text: prompt.System}}}
	maxTok := int32(prompt.MaxOutputTokens)
	temp := float32(0.2) // low: we want stable, concise explanations.

	resp, err := r.client.Models.GenerateContent(callCtx, r.model, genai.Text(prompt.User), &genai.GenerateContentConfig{
		SystemInstruction: sys,
		MaxOutputTokens:   maxTok,
		Temperature:       &temp,
	})
	if err != nil {
		return Suggestion{}, fmt.Errorf("llm.Explain(%s): %w", rule.ID, err)
	}
	if resp == nil {
		return Suggestion{}, errors.New("llm.Explain: nil response")
	}
	text := resp.Text()
	return Suggestion{
		Text:      text,
		Model:     r.model,
		Truncated: maxTok > 0 && int32(len(text)) >= maxTok*4, // rough heuristic; tokens ≈ 4 bytes
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}

// Close releases SDK resources. Safe to call more than once.
func (r *Real) Close() error {
	// google.golang.org/genai.Client has no Close method in the current SDK
	// version; connections are pooled at the transport level. We keep the
	// method on our interface so swapping in a provider that does require
	// Close is non-breaking.
	return nil
}
