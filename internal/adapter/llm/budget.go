package llm

import (
	"context"
	"sync"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// Budget wraps a Client with a per-run call limit. Once exhausted, subsequent
// Explain calls return ErrBudgetExhausted — callers handle this by falling
// back to static remediation for the remaining findings.
//
// Composition order matters: Budget must be wrapped *inside* Cached so that
// cache hits bypass the budget entirely. The canonical construction is:
//
//	client := llm.NewCached(llm.NewBudget(real, limit))
//
// Under that composition, Cached serves repeated prompts for free, and only
// cache misses (the calls that would actually hit the API) consume budget.
type Budget struct {
	inner  Client
	mu     sync.Mutex
	remain int
	limit  int
}

// NewBudget wraps inner with a budget of `limit` calls. If limit <= 0, the
// budget is unlimited (useful for tests).
func NewBudget(inner Client, limit int) *Budget {
	return &Budget{inner: inner, remain: limit, limit: limit}
}

// Explain consumes one budget unit per successful call.
func (b *Budget) Explain(ctx context.Context, rule model.Rule, finding model.Finding, prompt Prompt) (Suggestion, error) {
	b.mu.Lock()
	if b.limit > 0 && b.remain <= 0 {
		b.mu.Unlock()
		return Suggestion{}, ErrBudgetExhausted
	}
	b.mu.Unlock()

	sug, err := b.inner.Explain(ctx, rule, finding, prompt)
	if err != nil {
		// Failed calls do not consume budget — retries after transient
		// errors stay possible within the run.
		return sug, err
	}

	b.mu.Lock()
	b.remain--
	b.mu.Unlock()
	return sug, nil
}

// Remaining returns the calls left in the current run.
func (b *Budget) Remaining() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.remain
}

// Close forwards to the underlying Client.
func (b *Budget) Close() error { return b.inner.Close() }
