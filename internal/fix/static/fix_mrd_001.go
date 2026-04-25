package static

import (
	"bytes"
	"context"
	"text/template"

	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// MrdP7MRD001Fixer creates a docs/exit-codes.md file for rule P7.MRD.001.
type MrdP7MRD001Fixer struct{}

// NewMrdP7MRD001 returns a new fixer for P7.MRD.001.
func NewMrdP7MRD001() *MrdP7MRD001Fixer {
	return &MrdP7MRD001Fixer{}
}

// RuleID implements fix.Fixer.
func (f *MrdP7MRD001Fixer) RuleID() string { return "P7.MRD.001" }

// NeedsLLM implements fix.Fixer.
func (f *MrdP7MRD001Fixer) NeedsLLM() bool { return false }

// Plan implements fix.Fixer.
func (f *MrdP7MRD001Fixer) Plan(_ context.Context, _ model.Finding, _ model.FactStore) ([]fix.Change, error) {
	tmpl, err := template.ParseFS(templates, "templates/exit_codes.tmpl")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return nil, err
	}

	return []fix.Change{
		{
			Path:    "docs/exit-codes.md",
			Action:  fix.ActionCreate,
			Content: buf.Bytes(),
			Preview: "Create docs/exit-codes.md with exit code table",
		},
	}, nil
}
