package static_test

import (
	"context"
	"strings"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/fix/static"
	"github.com/shibuiwilliam/archfit/internal/model"
)

type fakeFactStore struct {
	repo model.RepoFacts
}

func (f *fakeFactStore) Repo() model.RepoFacts                 { return f.repo }
func (f *fakeFactStore) Git() (model.GitFacts, bool)           { return model.GitFacts{}, false }
func (f *fakeFactStore) Schemas() model.SchemaFacts            { return model.SchemaFacts{} }
func (f *fakeFactStore) Commands() (model.CommandFacts, bool)  { return model.CommandFacts{}, false }
func (f *fakeFactStore) DepGraph() (model.DepGraphFacts, bool) { return model.DepGraphFacts{}, false }
func (f *fakeFactStore) Languages() map[string]int             { return nil }
func (f *fakeFactStore) Ecosystems() model.EcosystemFacts      { return model.EcosystemFacts{} }
func (f *fakeFactStore) AST() (model.ASTFacts, bool)           { return model.ASTFacts{}, false }

func TestLocP1LOC001Fixer(t *testing.T) {
	fixer := static.NewLocP1LOC001()

	if fixer.RuleID() != "P1.LOC.001" {
		t.Errorf("RuleID = %q, want %q", fixer.RuleID(), "P1.LOC.001")
	}
	if fixer.NeedsLLM() {
		t.Error("NeedsLLM should be false")
	}

	facts := &fakeFactStore{
		repo: model.RepoFacts{Root: "/home/user/my-project"},
	}

	changes, err := fixer.Plan(context.Background(), model.Finding{}, facts)
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("got %d changes, want 1", len(changes))
	}

	c := changes[0]
	if c.Path != "CLAUDE.md" {
		t.Errorf("Path = %q, want %q", c.Path, "CLAUDE.md")
	}
	if c.Action != fix.ActionCreate {
		t.Errorf("Action = %q, want %q", c.Action, fix.ActionCreate)
	}
	if len(c.Content) == 0 {
		t.Error("Content is empty")
	}

	content := string(c.Content)
	for _, want := range []string{"## What this project is", "my-project"} {
		if !strings.Contains(content, want) {
			t.Errorf("Content missing %q", want)
		}
	}
}

func TestMrdP7MRD001Fixer(t *testing.T) {
	fixer := static.NewMrdP7MRD001()

	if fixer.RuleID() != "P7.MRD.001" {
		t.Errorf("RuleID = %q, want %q", fixer.RuleID(), "P7.MRD.001")
	}
	if fixer.NeedsLLM() {
		t.Error("NeedsLLM should be false")
	}

	facts := &fakeFactStore{
		repo: model.RepoFacts{Root: "/home/user/my-project"},
	}

	changes, err := fixer.Plan(context.Background(), model.Finding{}, facts)
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("got %d changes, want 1", len(changes))
	}

	c := changes[0]
	if c.Path != "docs/exit-codes.md" {
		t.Errorf("Path = %q, want %q", c.Path, "docs/exit-codes.md")
	}
	if c.Action != fix.ActionCreate {
		t.Errorf("Action = %q, want %q", c.Action, fix.ActionCreate)
	}
	if len(c.Content) == 0 {
		t.Error("Content is empty")
	}

	content := string(c.Content)
	for _, want := range []string{"# Exit codes", "Success", "Error"} {
		if !strings.Contains(content, want) {
			t.Errorf("Content missing %q", want)
		}
	}
}

func TestMrdP7MRD002Fixer(t *testing.T) {
	fixer := static.NewMrdP7MRD002()

	if fixer.RuleID() != "P7.MRD.002" {
		t.Errorf("RuleID = %q, want %q", fixer.RuleID(), "P7.MRD.002")
	}
	if fixer.NeedsLLM() {
		t.Error("NeedsLLM should be false")
	}

	facts := &fakeFactStore{
		repo: model.RepoFacts{Root: "/home/user/my-project"},
	}

	changes, err := fixer.Plan(context.Background(), model.Finding{}, facts)
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("got %d changes, want 1", len(changes))
	}

	c := changes[0]
	if c.Path != "CHANGELOG.md" {
		t.Errorf("Path = %q, want %q", c.Path, "CHANGELOG.md")
	}
	if c.Action != fix.ActionCreate {
		t.Errorf("Action = %q, want %q", c.Action, fix.ActionCreate)
	}
	if len(c.Content) == 0 {
		t.Error("Content is empty")
	}

	content := string(c.Content)
	for _, want := range []string{"# Changelog", "Keep a Changelog", "[Unreleased]"} {
		if !strings.Contains(content, want) {
			t.Errorf("Content missing %q", want)
		}
	}
}

// TestFixerInterface verifies all three fixers satisfy the Fixer interface.
func TestFixerInterface(t *testing.T) {
	var _ fix.Fixer = static.NewLocP1LOC001()
	var _ fix.Fixer = static.NewMrdP7MRD001()
	var _ fix.Fixer = static.NewMrdP7MRD002()
}
