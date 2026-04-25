package static

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

const agentsMDTemplate = `# %s

## What this slice does

<!-- Describe the purpose and responsibility of this vertical slice. -->

## Key files

<!-- List the most important files an agent should read first. -->

## Testing

<!-- How to run tests for this slice. -->
`

// LocP1LOC002Fixer creates AGENTS.md in vertical-slice directories that lack one.
type LocP1LOC002Fixer struct{}

// NewLocP1LOC002 returns a new fixer for rule P1.LOC.002.
func NewLocP1LOC002() *LocP1LOC002Fixer {
	return &LocP1LOC002Fixer{}
}

// RuleID implements fix.Fixer.
func (f *LocP1LOC002Fixer) RuleID() string { return "P1.LOC.002" }

// NeedsLLM implements fix.Fixer.
func (f *LocP1LOC002Fixer) NeedsLLM() bool { return false }

// Plan implements fix.Fixer.
func (f *LocP1LOC002Fixer) Plan(_ context.Context, finding model.Finding, _ model.FactStore) ([]fix.Change, error) {
	slice, ok := finding.Evidence["slice"].(string)
	if !ok || slice == "" {
		return nil, fmt.Errorf("P1.LOC.002 fixer: finding.Evidence[\"slice\"] missing or not a string")
	}

	dirName := filepath.Base(slice)
	content := fmt.Sprintf(agentsMDTemplate, dirName)

	return []fix.Change{
		{
			Path:    filepath.Join(slice, "AGENTS.md"),
			Action:  fix.ActionCreate,
			Content: []byte(content),
			Preview: fmt.Sprintf("Create AGENTS.md in %s", slice),
		},
	}, nil
}
