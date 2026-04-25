// Package llmfix wraps static fixers with LLM-generated content enrichment.
//
// An LLMFixer delegates to its base static fixer for the plan structure,
// then calls the llm.Client to enrich the file content with context-aware
// text. If the LLM call fails, it falls back silently to the static
// template — LLM failures never block a fix.
package llmfix

import (
	"context"
	"fmt"
	"io"

	"github.com/shibuiwilliam/archfit/internal/adapter/llm"
	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// LLMFixer wraps a static fixer and enriches its output with LLM-generated
// content. Falls back to the static fixer's template when the LLM is
// unavailable or returns an error.
type LLMFixer struct {
	base   fix.Fixer
	client llm.Client
	stderr io.Writer
}

// NewLLMFixer wraps base with LLM enrichment. The client must not be nil.
// stderr receives fallback warnings when the LLM call fails.
func NewLLMFixer(base fix.Fixer, client llm.Client, stderr io.Writer) *LLMFixer {
	return &LLMFixer{base: base, client: client, stderr: stderr}
}

// RuleID delegates to the base fixer.
func (f *LLMFixer) RuleID() string { return f.base.RuleID() }

// NeedsLLM always returns true for LLM fixers.
func (f *LLMFixer) NeedsLLM() bool { return true }

// Plan calls the base fixer for the static plan, then enriches each change's
// content with an LLM call. On LLM failure, the static content is kept.
func (f *LLMFixer) Plan(ctx context.Context, finding model.Finding, facts model.FactStore) ([]fix.Change, error) {
	changes, err := f.base.Plan(ctx, finding, facts)
	if err != nil {
		return nil, err
	}

	repo := facts.Repo()
	for i := range changes {
		enriched, enrichErr := f.enrich(ctx, finding, changes[i], repo)
		if enrichErr != nil {
			fmt.Fprintf(f.stderr, "llmfix: %s: falling back to static template (%v)\n",
				f.base.RuleID(), enrichErr)
			continue
		}
		changes[i].Content = []byte(enriched)
		changes[i].Preview += " (LLM-enriched)"
	}
	return changes, nil
}

func (f *LLMFixer) enrich(ctx context.Context, finding model.Finding, change fix.Change, repo model.RepoFacts) (string, error) {
	prompt := buildEnrichPrompt(f.base.RuleID(), change, repo)

	rule := model.Rule{ID: f.base.RuleID()}
	sug, err := f.client.Explain(ctx, rule, finding, prompt)
	if err != nil {
		return "", err
	}
	if sug.Text == "" {
		return "", fmt.Errorf("empty LLM response")
	}
	return sug.Text, nil
}
