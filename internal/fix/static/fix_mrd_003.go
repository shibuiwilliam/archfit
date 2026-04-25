package static

import (
	"context"
	"fmt"
	"time"

	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

const adrTemplate = `---
id: "0001"
title: "Initial Architecture"
status: "accepted"
date: "%s"
---

# ADR 0001: Initial Architecture

## Context

This project needs a documented record of significant architectural decisions.
This is the first ADR, establishing the practice of recording decisions.

## Decision

We will use Architecture Decision Records (ADRs) to capture important
architectural decisions made in this project. Each ADR will be stored in
` + "`docs/adr/`" + ` with a sequential numeric prefix.

## Consequences

- All significant architectural decisions will be documented.
- New team members can understand the reasoning behind past decisions.
- The ADR directory serves as a lightweight architecture journal.
`

// MrdP7MRD003Fixer creates docs/adr/ with an initial ADR template.
type MrdP7MRD003Fixer struct {
	// nowFunc is injectable for testing; defaults to time.Now.
	nowFunc func() time.Time
}

// NewMrdP7MRD003 returns a new fixer for rule P7.MRD.003.
func NewMrdP7MRD003() *MrdP7MRD003Fixer {
	return &MrdP7MRD003Fixer{nowFunc: time.Now}
}

// RuleID implements fix.Fixer.
func (f *MrdP7MRD003Fixer) RuleID() string { return "P7.MRD.003" }

// NeedsLLM implements fix.Fixer.
func (f *MrdP7MRD003Fixer) NeedsLLM() bool { return false }

// Plan implements fix.Fixer.
func (f *MrdP7MRD003Fixer) Plan(_ context.Context, _ model.Finding, _ model.FactStore) ([]fix.Change, error) {
	date := f.nowFunc().Format("2006-01-02")
	content := fmt.Sprintf(adrTemplate, date)

	return []fix.Change{
		{
			Path:    "docs/adr/0001-initial-architecture.md",
			Action:  fix.ActionCreate,
			Content: []byte(content),
			Preview: "Create initial ADR in docs/adr/",
		},
	}, nil
}
