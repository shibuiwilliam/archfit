package fix_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/score"
)

// --- fakes ---

// fakeFactStore implements model.FactStore for testing.
type fakeFactStore struct {
	root string
}

func (f *fakeFactStore) Repo() model.RepoFacts {
	return model.RepoFacts{Root: f.root}
}

func (f *fakeFactStore) Git() (model.GitFacts, bool) {
	return model.GitFacts{}, false
}

func (f *fakeFactStore) Schemas() model.SchemaFacts {
	return model.SchemaFacts{}
}

func (f *fakeFactStore) Commands() (model.CommandFacts, bool) {
	return model.CommandFacts{}, false
}

func (f *fakeFactStore) DepGraph() (model.DepGraphFacts, bool) {
	return model.DepGraphFacts{}, false
}

func (f *fakeFactStore) Languages() map[string]int        { return nil }
func (f *fakeFactStore) Ecosystems() model.EcosystemFacts { return model.EcosystemFacts{} }

// fakeFixer implements fix.Fixer for testing.
type fakeFixer struct {
	ruleID   string
	changes  []fix.Change
	needsLLM bool
	planErr  error
}

func (f *fakeFixer) RuleID() string { return f.ruleID }

func (f *fakeFixer) Plan(_ context.Context, _ model.Finding, _ model.FactStore) ([]fix.Change, error) {
	if f.planErr != nil {
		return nil, f.planErr
	}
	return f.changes, nil
}

func (f *fakeFixer) NeedsLLM() bool { return f.needsLLM }

// helpers

func makeFinding(ruleID string) model.Finding {
	return model.Finding{
		RuleID:   ruleID,
		Severity: model.SeverityWarn,
		Path:     "some/path",
		Message:  "test finding for " + ruleID,
		Evidence: map[string]any{},
	}
}

func makeScanner(findings []model.Finding) func(context.Context) (core.ScanResult, error) {
	return func(_ context.Context) (core.ScanResult, error) {
		return core.ScanResult{
			Findings:       findings,
			RulesEvaluated: 1,
			Scores:         score.Scores{Overall: 100, ByPrinciple: map[model.Principle]float64{}},
		}, nil
	}
}

// --- tests ---

func TestFix_DryRun_ReturnsPlanWithoutApplying(t *testing.T) {
	root := t.TempDir()

	eng := fix.NewEngine()
	eng.Register(&fakeFixer{
		ruleID: "P1.LOC.001",
		changes: []fix.Change{
			{Path: "AGENTS.md", Action: fix.ActionCreate, Content: []byte("# Agents\n"), Preview: "create AGENTS.md"},
		},
	})

	result, err := eng.Fix(context.Background(), fix.Input{
		Root:     root,
		DryRun:   true,
		Facts:    &fakeFactStore{root: root},
		Findings: []model.Finding{makeFinding("P1.LOC.001")},
		Scanner:  makeScanner(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Plan.Fixes) != 1 {
		t.Fatalf("expected 1 planned fix, got %d", len(result.Plan.Fixes))
	}
	if result.Plan.Fixes[0].RuleID != "P1.LOC.001" {
		t.Errorf("expected rule ID P1.LOC.001, got %s", result.Plan.Fixes[0].RuleID)
	}

	// File should NOT exist because dry-run does not apply.
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Error("expected AGENTS.md to not exist in dry-run mode")
	}

	if len(result.Applied) != 0 {
		t.Errorf("expected no applied fixes in dry-run, got %d", len(result.Applied))
	}
}

func TestFix_AppliesAndVerifies(t *testing.T) {
	root := t.TempDir()

	eng := fix.NewEngine()
	eng.Register(&fakeFixer{
		ruleID: "P1.LOC.001",
		changes: []fix.Change{
			{Path: "AGENTS.md", Action: fix.ActionCreate, Content: []byte("# Agents\n"), Preview: "create AGENTS.md"},
		},
	})

	// Scanner returns no findings after fix — verification passes.
	result, err := eng.Fix(context.Background(), fix.Input{
		Root:     root,
		Facts:    &fakeFactStore{root: root},
		Findings: []model.Finding{makeFinding("P1.LOC.001")},
		Scanner:  makeScanner(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Verified {
		t.Error("expected verified to be true")
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied fix, got %d", len(result.Applied))
	}
	if result.Applied[0].RuleID != "P1.LOC.001" {
		t.Errorf("expected applied rule ID P1.LOC.001, got %s", result.Applied[0].RuleID)
	}
	if !result.Applied[0].Verified {
		t.Error("expected applied fix to be marked verified")
	}

	// File should exist.
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("expected AGENTS.md to exist: %v", err)
	}
	if string(data) != "# Agents\n" {
		t.Errorf("unexpected content: %q", data)
	}
}

func TestFix_RollsBackOnVerificationFailure(t *testing.T) {
	root := t.TempDir()

	eng := fix.NewEngine()
	eng.Register(&fakeFixer{
		ruleID: "P1.LOC.001",
		changes: []fix.Change{
			{Path: "AGENTS.md", Action: fix.ActionCreate, Content: []byte("# Agents\n"), Preview: "create AGENTS.md"},
		},
	})

	// Scanner still returns the same finding — verification fails.
	result, err := eng.Fix(context.Background(), fix.Input{
		Root:     root,
		Facts:    &fakeFactStore{root: root},
		Findings: []model.Finding{makeFinding("P1.LOC.001")},
		Scanner:  makeScanner([]model.Finding{makeFinding("P1.LOC.001")}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verified {
		t.Error("expected verified to be false when scanner still reports finding")
	}

	// File should have been rolled back (removed since it was created).
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Error("expected AGENTS.md to be rolled back (removed)")
	}

	// Applied should show unverified.
	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied fix, got %d", len(result.Applied))
	}
	if result.Applied[0].Verified {
		t.Error("expected applied fix to be marked unverified after rollback")
	}
}

func TestFix_RollsBackOnNewIssues(t *testing.T) {
	root := t.TempDir()

	eng := fix.NewEngine()
	eng.Register(&fakeFixer{
		ruleID: "P1.LOC.001",
		changes: []fix.Change{
			{Path: "AGENTS.md", Action: fix.ActionCreate, Content: []byte("# Agents\n"), Preview: "create AGENTS.md"},
		},
	})

	// Scanner returns a new finding for a different rule — regression.
	newFinding := makeFinding("P2.SPC.001")
	result, err := eng.Fix(context.Background(), fix.Input{
		Root:     root,
		Facts:    &fakeFactStore{root: root},
		Findings: []model.Finding{makeFinding("P1.LOC.001")},
		Scanner:  makeScanner([]model.Finding{newFinding}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verified {
		t.Error("expected verified to be false when new issues appeared")
	}

	if len(result.NewIssues) != 1 {
		t.Fatalf("expected 1 new issue, got %d", len(result.NewIssues))
	}
	if result.NewIssues[0].RuleID != "P2.SPC.001" {
		t.Errorf("expected new issue rule ID P2.SPC.001, got %s", result.NewIssues[0].RuleID)
	}

	// File should have been rolled back.
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Error("expected AGENTS.md to be rolled back")
	}
}

func TestFix_MultipleFixers(t *testing.T) {
	root := t.TempDir()

	eng := fix.NewEngine()
	eng.Register(&fakeFixer{
		ruleID: "P1.LOC.001",
		changes: []fix.Change{
			{Path: "AGENTS.md", Action: fix.ActionCreate, Content: []byte("# Agents\n"), Preview: "create AGENTS.md"},
		},
	})
	eng.Register(&fakeFixer{
		ruleID: "P2.SPC.001",
		changes: []fix.Change{
			{Path: "INTENT.md", Action: fix.ActionCreate, Content: []byte("# Intent\n"), Preview: "create INTENT.md"},
		},
	})

	result, err := eng.Fix(context.Background(), fix.Input{
		Root:  root,
		Facts: &fakeFactStore{root: root},
		Findings: []model.Finding{
			makeFinding("P1.LOC.001"),
			makeFinding("P2.SPC.001"),
		},
		Scanner: makeScanner(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Plan.Fixes) != 2 {
		t.Fatalf("expected 2 planned fixes, got %d", len(result.Plan.Fixes))
	}
	if len(result.Applied) != 2 {
		t.Fatalf("expected 2 applied fixes, got %d", len(result.Applied))
	}
	if !result.Verified {
		t.Error("expected verified to be true")
	}

	// Both files should exist.
	for _, name := range []string{"AGENTS.md", "INTENT.md"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		}
	}
}

func TestFix_FiltersByRuleIDs(t *testing.T) {
	root := t.TempDir()

	eng := fix.NewEngine()
	eng.Register(&fakeFixer{
		ruleID: "P1.LOC.001",
		changes: []fix.Change{
			{Path: "AGENTS.md", Action: fix.ActionCreate, Content: []byte("# Agents\n"), Preview: "create AGENTS.md"},
		},
	})
	eng.Register(&fakeFixer{
		ruleID: "P2.SPC.001",
		changes: []fix.Change{
			{Path: "INTENT.md", Action: fix.ActionCreate, Content: []byte("# Intent\n"), Preview: "create INTENT.md"},
		},
	})

	// Only fix P1.LOC.001, even though both findings and fixers are present.
	result, err := eng.Fix(context.Background(), fix.Input{
		Root:    root,
		RuleIDs: []string{"P1.LOC.001"},
		Facts:   &fakeFactStore{root: root},
		Findings: []model.Finding{
			makeFinding("P1.LOC.001"),
			makeFinding("P2.SPC.001"),
		},
		Scanner: makeScanner(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Plan.Fixes) != 1 {
		t.Fatalf("expected 1 planned fix, got %d", len(result.Plan.Fixes))
	}
	if result.Plan.Fixes[0].RuleID != "P1.LOC.001" {
		t.Errorf("expected planned fix for P1.LOC.001, got %s", result.Plan.Fixes[0].RuleID)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied fix, got %d", len(result.Applied))
	}

	// Only AGENTS.md should exist.
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Errorf("expected AGENTS.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "INTENT.md")); !os.IsNotExist(err) {
		t.Error("expected INTENT.md to NOT exist (filtered out)")
	}
}

func TestFix_ModifyAction(t *testing.T) {
	root := t.TempDir()

	// Pre-create a file to modify.
	original := []byte("old content\n")
	if err := os.WriteFile(filepath.Join(root, "README.md"), original, 0o644); err != nil {
		t.Fatal(err)
	}

	eng := fix.NewEngine()
	eng.Register(&fakeFixer{
		ruleID: "P7.MRD.001",
		changes: []fix.Change{
			{Path: "README.md", Action: fix.ActionModify, Content: []byte("new content\n"), Preview: "update README.md"},
		},
	})

	result, err := eng.Fix(context.Background(), fix.Input{
		Root:     root,
		Facts:    &fakeFactStore{root: root},
		Findings: []model.Finding{makeFinding("P7.MRD.001")},
		Scanner:  makeScanner(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Verified {
		t.Error("expected verified")
	}

	data, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content\n" {
		t.Errorf("expected modified content, got %q", data)
	}
}

func TestFix_ModifyRollsBackToOriginal(t *testing.T) {
	root := t.TempDir()

	original := []byte("original content\n")
	if err := os.WriteFile(filepath.Join(root, "README.md"), original, 0o644); err != nil {
		t.Fatal(err)
	}

	eng := fix.NewEngine()
	eng.Register(&fakeFixer{
		ruleID: "P7.MRD.001",
		changes: []fix.Change{
			{Path: "README.md", Action: fix.ActionModify, Content: []byte("bad content\n"), Preview: "update README.md"},
		},
	})

	// Scanner returns the same finding — rollback.
	_, err := eng.Fix(context.Background(), fix.Input{
		Root:     root,
		Facts:    &fakeFactStore{root: root},
		Findings: []model.Finding{makeFinding("P7.MRD.001")},
		Scanner:  makeScanner([]model.Finding{makeFinding("P7.MRD.001")}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should be restored to original.
	data, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original content\n" {
		t.Errorf("expected original content after rollback, got %q", data)
	}
}

func TestFix_AppendAction(t *testing.T) {
	root := t.TempDir()

	original := []byte("line1\n")
	if err := os.WriteFile(filepath.Join(root, "log.txt"), original, 0o644); err != nil {
		t.Fatal(err)
	}

	eng := fix.NewEngine()
	eng.Register(&fakeFixer{
		ruleID: "P4.VER.001",
		changes: []fix.Change{
			{Path: "log.txt", Action: fix.ActionAppend, Content: []byte("line2\n"), Preview: "append to log.txt"},
		},
	})

	result, err := eng.Fix(context.Background(), fix.Input{
		Root:     root,
		Facts:    &fakeFactStore{root: root},
		Findings: []model.Finding{makeFinding("P4.VER.001")},
		Scanner:  makeScanner(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Verified {
		t.Error("expected verified")
	}

	data, err := os.ReadFile(filepath.Join(root, "log.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "line1\nline2\n" {
		t.Errorf("expected appended content, got %q", data)
	}
}

func TestFix_NoMatchingFixer(t *testing.T) {
	root := t.TempDir()

	eng := fix.NewEngine()
	// No fixers registered.

	result, err := eng.Fix(context.Background(), fix.Input{
		Root:     root,
		Facts:    &fakeFactStore{root: root},
		Findings: []model.Finding{makeFinding("P1.LOC.001")},
		Scanner:  makeScanner(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Plan.Fixes) != 0 {
		t.Errorf("expected empty plan when no fixer matches, got %d fixes", len(result.Plan.Fixes))
	}
	if !result.Verified {
		t.Error("expected verified to be true when there is nothing to fix")
	}
}
