package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// Cached wraps a Client with an in-process response cache keyed by
// (model, system, user). Identical prompts within one run are free.
//
// The cache is per-instance (one per run). Phase 3b adds a disk-backed cache.
type Cached struct {
	inner Client
	mu    sync.Mutex
	store map[string]Suggestion
}

// NewCached wraps inner with an in-memory cache. Safe for concurrent use.
func NewCached(inner Client) *Cached {
	return &Cached{inner: inner, store: map[string]Suggestion{}}
}

// Explain looks up (or computes) the suggestion.
func (c *Cached) Explain(ctx context.Context, rule model.Rule, finding model.Finding, prompt Prompt) (Suggestion, error) {
	key := cacheKey(prompt)

	c.mu.Lock()
	if hit, ok := c.store[key]; ok {
		c.mu.Unlock()
		hit.CacheHit = true
		return hit, nil
	}
	c.mu.Unlock()

	sug, err := c.inner.Explain(ctx, rule, finding, prompt)
	if err != nil {
		return sug, err
	}

	c.mu.Lock()
	c.store[key] = sug
	c.mu.Unlock()
	return sug, nil
}

// Close forwards to the underlying Client.
func (c *Cached) Close() error { return c.inner.Close() }

func cacheKey(p Prompt) string {
	h := sha256.New()
	h.Write([]byte(p.System))
	h.Write([]byte{0})
	h.Write([]byte(p.User))
	return hex.EncodeToString(h.Sum(nil))
}
