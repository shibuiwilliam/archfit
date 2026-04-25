package static_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/fix/static"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// fakeFactStore is declared in static_test.go (shared by both test files).

func TestLocP1LOC002_Plan(t *testing.T) {
	fixer := static.NewLocP1LOC002()

	if fixer.RuleID() != "P1.LOC.002" {
		t.Errorf("RuleID() = %q, want %q", fixer.RuleID(), "P1.LOC.002")
	}
	if fixer.NeedsLLM() {
		t.Error("NeedsLLM() = true, want false")
	}

	t.Run("creates AGENTS.md using slice from evidence", func(t *testing.T) {
		finding := model.Finding{
			RuleID:  "P1.LOC.002",
			Message: "vertical slice missing AGENTS.md",
			Evidence: map[string]any{
				"slice": "packs/alpha",
			},
		}
		facts := &fakeFactStore{}
		changes, err := fixer.Plan(context.Background(), finding, facts)
		if err != nil {
			t.Fatalf("Plan() error: %v", err)
		}
		if len(changes) != 1 {
			t.Fatalf("Plan() returned %d changes, want 1", len(changes))
		}

		c := changes[0]
		if c.Path != "packs/alpha/AGENTS.md" {
			t.Errorf("Path = %q, want %q", c.Path, "packs/alpha/AGENTS.md")
		}
		if c.Action != fix.ActionCreate {
			t.Errorf("Action = %q, want %q", c.Action, fix.ActionCreate)
		}
		if !strings.Contains(string(c.Content), "## What this slice does") {
			t.Error("Content missing expected heading")
		}
		if !strings.Contains(string(c.Content), "alpha") {
			t.Error("Content missing slice directory name")
		}
	})

	t.Run("error when slice evidence missing", func(t *testing.T) {
		finding := model.Finding{
			RuleID:   "P1.LOC.002",
			Evidence: map[string]any{},
		}
		_, err := fixer.Plan(context.Background(), finding, &fakeFactStore{})
		if err == nil {
			t.Error("Plan() should return error when slice evidence is missing")
		}
	})
}

func TestVerP4VER001_Plan(t *testing.T) {
	fixer := static.NewVerP4VER001()

	if fixer.RuleID() != "P4.VER.001" {
		t.Errorf("RuleID() = %q, want %q", fixer.RuleID(), "P4.VER.001")
	}
	if fixer.NeedsLLM() {
		t.Error("NeedsLLM() = true, want false")
	}

	t.Run("creates Makefile when none exists", func(t *testing.T) {
		facts := &fakeFactStore{
			repo: model.RepoFacts{
				ByPath: map[string]model.FileFact{},
			},
		}
		finding := model.Finding{RuleID: "P4.VER.001"}
		changes, err := fixer.Plan(context.Background(), finding, facts)
		if err != nil {
			t.Fatalf("Plan() error: %v", err)
		}
		if len(changes) != 1 {
			t.Fatalf("Plan() returned %d changes, want 1", len(changes))
		}

		c := changes[0]
		if c.Path != "Makefile" {
			t.Errorf("Path = %q, want %q", c.Path, "Makefile")
		}
		if c.Action != fix.ActionCreate {
			t.Errorf("Action = %q, want %q", c.Action, fix.ActionCreate)
		}
		content := string(c.Content)
		if !strings.Contains(content, "test:") {
			t.Error("Content missing test target")
		}
		if !strings.Contains(content, "lint:") {
			t.Error("Content missing lint target")
		}
	})

	t.Run("appends to Makefile when it exists", func(t *testing.T) {
		facts := &fakeFactStore{
			repo: model.RepoFacts{
				ByPath: map[string]model.FileFact{
					"Makefile": {Path: "Makefile", Size: 100},
				},
			},
		}
		finding := model.Finding{RuleID: "P4.VER.001"}
		changes, err := fixer.Plan(context.Background(), finding, facts)
		if err != nil {
			t.Fatalf("Plan() error: %v", err)
		}
		if len(changes) != 1 {
			t.Fatalf("Plan() returned %d changes, want 1", len(changes))
		}

		c := changes[0]
		if c.Action != fix.ActionAppend {
			t.Errorf("Action = %q, want %q", c.Action, fix.ActionAppend)
		}
		content := string(c.Content)
		if !strings.Contains(content, "test:") {
			t.Error("Content missing test target")
		}
	})
}

func TestMrdP7MRD003_Plan(t *testing.T) {
	fixer := static.NewMrdP7MRD003()

	if fixer.RuleID() != "P7.MRD.003" {
		t.Errorf("RuleID() = %q, want %q", fixer.RuleID(), "P7.MRD.003")
	}
	if fixer.NeedsLLM() {
		t.Error("NeedsLLM() = true, want false")
	}

	t.Run("creates initial ADR", func(t *testing.T) {
		finding := model.Finding{RuleID: "P7.MRD.003"}
		changes, err := fixer.Plan(context.Background(), finding, &fakeFactStore{})
		if err != nil {
			t.Fatalf("Plan() error: %v", err)
		}
		if len(changes) != 1 {
			t.Fatalf("Plan() returned %d changes, want 1", len(changes))
		}

		c := changes[0]
		if c.Path != "docs/adr/0001-initial-architecture.md" {
			t.Errorf("Path = %q, want %q", c.Path, "docs/adr/0001-initial-architecture.md")
		}
		if c.Action != fix.ActionCreate {
			t.Errorf("Action = %q, want %q", c.Action, fix.ActionCreate)
		}
		content := string(c.Content)
		if !strings.Contains(content, "title: \"Initial Architecture\"") {
			t.Error("Content missing YAML frontmatter title")
		}
		if !strings.Contains(content, "## Context") {
			t.Error("Content missing Context section")
		}
		if !strings.Contains(content, "## Decision") {
			t.Error("Content missing Decision section")
		}
		if !strings.Contains(content, "## Consequences") {
			t.Error("Content missing Consequences section")
		}
		// Verify date is present (should match today's date format)
		today := time.Now().Format("2006-01-02")
		if !strings.Contains(content, today) {
			t.Errorf("Content missing today's date %s", today)
		}
	})
}

func TestSpcP2SPC010_Plan(t *testing.T) {
	fixer := static.NewSpcP2SPC010()

	if fixer.RuleID() != "P2.SPC.010" {
		t.Errorf("RuleID() = %q, want %q", fixer.RuleID(), "P2.SPC.010")
	}
	if fixer.NeedsLLM() {
		t.Error("NeedsLLM() = true, want false")
	}

	t.Run("creates output schema JSON", func(t *testing.T) {
		finding := model.Finding{RuleID: "P2.SPC.010"}
		changes, err := fixer.Plan(context.Background(), finding, &fakeFactStore{})
		if err != nil {
			t.Fatalf("Plan() error: %v", err)
		}
		if len(changes) != 1 {
			t.Fatalf("Plan() returned %d changes, want 1", len(changes))
		}

		c := changes[0]
		if c.Path != "schemas/output.schema.json" {
			t.Errorf("Path = %q, want %q", c.Path, "schemas/output.schema.json")
		}
		if c.Action != fix.ActionCreate {
			t.Errorf("Action = %q, want %q", c.Action, fix.ActionCreate)
		}
		content := string(c.Content)
		if !strings.Contains(content, `"$id"`) {
			t.Error("Content missing $id field")
		}
		if !strings.Contains(content, `"schema_version"`) {
			t.Error("Content missing schema_version property")
		}
		if !strings.Contains(content, `"$schema"`) {
			t.Error("Content missing $schema field")
		}
	})
}

// TestAllFixersImplementInterface verifies each fixer satisfies fix.Fixer at compile time.
func TestAllFixersImplementInterface(t *testing.T) {
	var _ fix.Fixer = static.NewLocP1LOC002()
	var _ fix.Fixer = static.NewVerP4VER001()
	var _ fix.Fixer = static.NewMrdP7MRD003()
	var _ fix.Fixer = static.NewSpcP2SPC010()
}
