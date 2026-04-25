package fix_test

import (
	"strings"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

func TestPlan_Summary_Empty(t *testing.T) {
	p := fix.Plan{}
	if got := p.Summary(); got != "no fixable findings" {
		t.Errorf("unexpected summary for empty plan: %q", got)
	}
}

func TestPlan_Summary_WithFixes(t *testing.T) {
	p := fix.Plan{
		Fixes: []fix.PlannedFix{
			{
				RuleID:  "P1.LOC.001",
				Finding: model.Finding{Message: "missing CLAUDE.md"},
				Changes: []fix.Change{
					{Path: "CLAUDE.md", Action: fix.ActionCreate, Preview: "scaffold from template"},
				},
			},
			{
				RuleID:  "P7.MRD.002",
				Finding: model.Finding{Message: "no CHANGELOG.md"},
				Changes: []fix.Change{
					{Path: "CHANGELOG.md", Action: fix.ActionCreate, Preview: "Keep a Changelog skeleton"},
				},
			},
		},
	}
	s := p.Summary()
	if !strings.Contains(s, "2 finding(s)") {
		t.Errorf("summary should mention 2 findings: %q", s)
	}
	if !strings.Contains(s, "P1.LOC.001") || !strings.Contains(s, "P7.MRD.002") {
		t.Errorf("summary should list rule IDs: %q", s)
	}
	if !strings.Contains(s, "create CLAUDE.md") {
		t.Errorf("summary should show change action and path: %q", s)
	}
}

func TestPlan_FilePaths_Deduplicates(t *testing.T) {
	p := fix.Plan{
		Fixes: []fix.PlannedFix{
			{
				Changes: []fix.Change{
					{Path: "CLAUDE.md"},
					{Path: "docs/exit-codes.md"},
				},
			},
			{
				Changes: []fix.Change{
					{Path: "CLAUDE.md"}, // duplicate
					{Path: "CHANGELOG.md"},
				},
			},
		},
	}
	paths := p.FilePaths()
	if len(paths) != 3 {
		t.Errorf("expected 3 unique paths, got %d: %v", len(paths), paths)
	}
}
