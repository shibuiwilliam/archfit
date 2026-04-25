package static

import (
	"context"

	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

const makefileCreate = `.PHONY: test lint

test:
	@echo "Running tests..."
	go test ./...

lint:
	@echo "Running linters..."
	go vet ./...
`

const makefileAppend = `
.PHONY: test

test:
	@echo "Running tests..."
	go test ./...
`

// VerP4VER001Fixer creates or appends to a Makefile so that a verification
// entrypoint exists.
type VerP4VER001Fixer struct{}

// NewVerP4VER001 returns a new fixer for rule P4.VER.001.
func NewVerP4VER001() *VerP4VER001Fixer {
	return &VerP4VER001Fixer{}
}

// RuleID implements fix.Fixer.
func (f *VerP4VER001Fixer) RuleID() string { return "P4.VER.001" }

// NeedsLLM implements fix.Fixer.
func (f *VerP4VER001Fixer) NeedsLLM() bool { return false }

// Plan implements fix.Fixer.
func (f *VerP4VER001Fixer) Plan(_ context.Context, _ model.Finding, facts model.FactStore) ([]fix.Change, error) {
	_, exists := facts.Repo().ByPath["Makefile"]

	if exists {
		return []fix.Change{
			{
				Path:    "Makefile",
				Action:  fix.ActionAppend,
				Content: []byte(makefileAppend),
				Preview: "Append test target to existing Makefile",
			},
		}, nil
	}

	return []fix.Change{
		{
			Path:    "Makefile",
			Action:  fix.ActionCreate,
			Content: []byte(makefileCreate),
			Preview: "Create Makefile with test and lint targets",
		},
	}, nil
}
