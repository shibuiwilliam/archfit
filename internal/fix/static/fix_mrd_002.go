package static

import (
	"bytes"
	"context"
	"text/template"

	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// MrdP7MRD002Fixer creates a CHANGELOG.md file for rule P7.MRD.002.
type MrdP7MRD002Fixer struct{}

// NewMrdP7MRD002 returns a new fixer for P7.MRD.002.
func NewMrdP7MRD002() *MrdP7MRD002Fixer {
	return &MrdP7MRD002Fixer{}
}

// RuleID implements fix.Fixer.
func (f *MrdP7MRD002Fixer) RuleID() string { return "P7.MRD.002" }

// NeedsLLM implements fix.Fixer.
func (f *MrdP7MRD002Fixer) NeedsLLM() bool { return false }

// Plan implements fix.Fixer.
func (f *MrdP7MRD002Fixer) Plan(_ context.Context, _ model.Finding, _ model.FactStore) ([]fix.Change, error) {
	tmpl, err := template.ParseFS(templates, "templates/changelog.tmpl")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return nil, err
	}

	return []fix.Change{
		{
			Path:    "CHANGELOG.md",
			Action:  fix.ActionCreate,
			Content: buf.Bytes(),
			Preview: "Create CHANGELOG.md in Keep a Changelog format",
		},
	}, nil
}
