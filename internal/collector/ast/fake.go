package ast

import "github.com/shibuiwilliam/archfit/internal/model"

// Fake returns pre-built ASTFacts for use in resolver tests.
// This avoids parsing real files in pack-level tests.
type Fake struct {
	Facts model.ASTFacts
}

// NewFake creates a Fake with the given facts.
func NewFake(facts model.ASTFacts) *Fake {
	return &Fake{Facts: facts}
}

// Result returns the pre-built facts.
func (f *Fake) Result() (model.ASTFacts, bool) {
	return f.Facts, true
}
