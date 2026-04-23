package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// Fake is the test-only Client. It returns canned responses keyed by rule ID,
// records every call for assertions, and never performs I/O.
//
// Tests that want a specific response set it via Responses[ruleID]. Any call
// for a rule without a canned response returns a deterministic synthetic
// suggestion so tests that don't care about content still work.
type Fake struct {
	mu        sync.Mutex
	Responses map[string]string
	Calls     []FakeCall
	// FailOn, if set, causes Explain to return this error on every call. Used
	// to exercise the graceful-degradation path.
	FailOn error
}

// FakeCall is recorded for each Explain invocation.
type FakeCall struct {
	RuleID string
	Path   string
}

// NewFake returns an empty Fake ready for tests to populate.
func NewFake() *Fake {
	return &Fake{Responses: map[string]string{}}
}

// Explain implements Client.
func (f *Fake) Explain(_ context.Context, rule model.Rule, finding model.Finding, _ Prompt) (Suggestion, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, FakeCall{RuleID: rule.ID, Path: finding.Path})
	if f.FailOn != nil {
		return Suggestion{}, f.FailOn
	}
	text, ok := f.Responses[rule.ID]
	if !ok {
		text = fmt.Sprintf("synthetic suggestion for %s at %q (fake client)", rule.ID, finding.Path)
	}
	return Suggestion{Text: text, Model: "fake"}, nil
}

// Close implements Client. Fake holds no resources; Close is a no-op.
func (f *Fake) Close() error { return nil }
