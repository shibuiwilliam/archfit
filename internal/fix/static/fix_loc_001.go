package static

import (
	"bytes"
	"context"
	"path/filepath"
	"text/template"

	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// LocP1LOC001Fixer creates a CLAUDE.md file for rule P1.LOC.001.
type LocP1LOC001Fixer struct{}

// NewLocP1LOC001 returns a new fixer for P1.LOC.001.
func NewLocP1LOC001() *LocP1LOC001Fixer {
	return &LocP1LOC001Fixer{}
}

// RuleID implements fix.Fixer.
func (f *LocP1LOC001Fixer) RuleID() string { return "P1.LOC.001" }

// NeedsLLM implements fix.Fixer.
func (f *LocP1LOC001Fixer) NeedsLLM() bool { return false }

// Plan implements fix.Fixer.
func (f *LocP1LOC001Fixer) Plan(_ context.Context, _ model.Finding, facts model.FactStore) ([]fix.Change, error) {
	tmpl, err := template.ParseFS(templates, "templates/claude_md.tmpl")
	if err != nil {
		return nil, err
	}

	projectName := filepath.Base(facts.Repo().Root)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"ProjectName": projectName,
	}); err != nil {
		return nil, err
	}

	return []fix.Change{
		{
			Path:    "CLAUDE.md",
			Action:  fix.ActionCreate,
			Content: buf.Bytes(),
			Preview: "Create CLAUDE.md with project scaffold",
		},
	}, nil
}
