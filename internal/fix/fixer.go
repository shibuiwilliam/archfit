// Package fix implements the scan-fix-verify remediation engine for archfit.
//
// The engine orchestrates three phases: Plan (propose changes), Apply (write
// files), and Verify (re-scan to confirm the fix worked). If verification
// fails, changes are rolled back.
//
// Two fixer classes exist:
//   - Static fixers (internal/fix/static/): deterministic file scaffolding
//     for strong-evidence rules. Safe for --all without confirmation.
//   - LLM-assisted fixers (internal/fix/llmfix/): wrap a static fixer and
//     enrich output via llm.Client. Fall back to static on failure.
//
// Fixers are registered explicitly in cmd/archfit/main.go — no reflection,
// no init(). See ADR 0004.
package fix

import (
	"context"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// Fixer produces file changes that remediate a specific finding.
// Static fixers return deterministic changes; LLM fixers call the adapter.
type Fixer interface {
	// RuleID returns which rule this fixer handles (e.g. "P1.LOC.001").
	RuleID() string

	// Plan proposes changes without applying them. The finding and facts
	// provide context for generating the fix content.
	Plan(ctx context.Context, finding model.Finding, facts model.FactStore) ([]Change, error)

	// NeedsLLM reports whether this fixer requires --with-llm to produce
	// useful output. Static fixers return false.
	NeedsLLM() bool
}

// Change is a single file-level modification proposed by a fixer.
type Change struct {
	// Path is relative to the repository root.
	Path string `json:"path"`
	// Action is one of create, modify, or append.
	Action ChangeAction `json:"action"`
	// Content is the new file content (create/modify) or appended bytes.
	Content []byte `json:"content"`
	// Preview is a human-readable summary of what changes (for --plan output).
	Preview string `json:"preview"`
}

// ChangeAction describes how a file is modified.
type ChangeAction string

// Supported change actions.
const (
	ActionCreate ChangeAction = "create"
	ActionModify ChangeAction = "modify"
	ActionAppend ChangeAction = "append"
)
