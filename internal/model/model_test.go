package model_test

import (
	"context"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
)

func validRule() model.Rule {
	return model.Rule{
		ID:               "P1.LOC.001",
		Principle:        model.P1Locality,
		Dimension:        "LOC",
		Title:            "test",
		Severity:         model.SeverityWarn,
		EvidenceStrength: model.EvidenceStrong,
		Stability:        model.StabilityExperimental,
		Rationale:        "a reason long enough",
		Remediation:      model.Remediation{Summary: "fix it"},
		Resolver: func(context.Context, model.FactStore) ([]model.Finding, []model.Metric, error) {
			return nil, nil, nil
		},
	}
}

func TestRuleValidate_OK(t *testing.T) {
	if err := validRule().Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuleValidate_BadID(t *testing.T) {
	r := validRule()
	r.ID = "bad"
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for bad id")
	}
}

func TestRuleValidate_WeakErrorForbidden(t *testing.T) {
	r := validRule()
	r.EvidenceStrength = model.EvidenceWeak
	r.Severity = model.SeverityError
	if err := r.Validate(); err == nil {
		t.Fatal("expected error: weak+error is forbidden")
	}
}

func TestSortFindings_DeterministicOrder(t *testing.T) {
	in := []model.Finding{
		{RuleID: "P2.SPC.001", Severity: model.SeverityWarn, Path: "b"},
		{RuleID: "P1.LOC.001", Severity: model.SeverityError, Path: "a"},
		{RuleID: "P1.LOC.001", Severity: model.SeverityError, Path: "b"},
		{RuleID: "P1.LOC.001", Severity: model.SeverityWarn, Path: "a"},
	}
	model.SortFindings(in)
	got := []string{}
	for _, f := range in {
		got = append(got, string(f.Severity)+"/"+f.RuleID+"/"+f.Path)
	}
	want := []string{
		"error/P1.LOC.001/a",
		"error/P1.LOC.001/b",
		"warn/P1.LOC.001/a",
		"warn/P2.SPC.001/b",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pos %d: got %s want %s", i, got[i], want[i])
		}
	}
}

func TestParseFailure_ShapeMatchesClaudeMdRule(t *testing.T) {
	f := model.ParseFailure("P2.SPC.010", "schemas/output.schema.json", "missing $id")
	if f.Severity != model.SeverityWarn {
		t.Errorf("want warn (per CLAUDE.md §13), got %s", f.Severity)
	}
	if f.EvidenceStrength != model.EvidenceStrong {
		t.Errorf("want strong evidence, got %s", f.EvidenceStrength)
	}
	if f.Evidence["parse_failure"] != true {
		t.Errorf("parse_failure flag missing")
	}
}

func TestSeverityRank_MonotonicallyIncreasing(t *testing.T) {
	if model.SeverityInfo.Rank() >= model.SeverityWarn.Rank() ||
		model.SeverityWarn.Rank() >= model.SeverityError.Rank() ||
		model.SeverityError.Rank() >= model.SeverityCritical.Rank() {
		t.Fatal("severity rank is not monotonically increasing")
	}
}
