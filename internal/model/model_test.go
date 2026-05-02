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

func TestRuleValidate_SeverityEvidenceMatrix(t *testing.T) {
	// CLAUDE.md §6 invariant 2:
	//   critical → strong only
	//   error    → strong only
	//   warn     → strong, medium, sampled OK; weak rejected
	//   info     → any OK
	tests := []struct {
		severity model.Severity
		evidence model.EvidenceStrength
		wantErr  bool
	}{
		// critical
		{model.SeverityCritical, model.EvidenceStrong, false},
		{model.SeverityCritical, model.EvidenceMedium, true},
		{model.SeverityCritical, model.EvidenceWeak, true},
		{model.SeverityCritical, model.EvidenceSampled, true},
		// error
		{model.SeverityError, model.EvidenceStrong, false},
		{model.SeverityError, model.EvidenceMedium, true},
		{model.SeverityError, model.EvidenceWeak, true},
		{model.SeverityError, model.EvidenceSampled, true},
		// warn
		{model.SeverityWarn, model.EvidenceStrong, false},
		{model.SeverityWarn, model.EvidenceMedium, false},
		{model.SeverityWarn, model.EvidenceSampled, false},
		{model.SeverityWarn, model.EvidenceWeak, true},
		// info
		{model.SeverityInfo, model.EvidenceStrong, false},
		{model.SeverityInfo, model.EvidenceMedium, false},
		{model.SeverityInfo, model.EvidenceSampled, false},
		{model.SeverityInfo, model.EvidenceWeak, false},
	}
	for _, tt := range tests {
		name := string(tt.severity) + "/" + string(tt.evidence)
		t.Run(name, func(t *testing.T) {
			r := validRule()
			r.Severity = tt.severity
			r.EvidenceStrength = tt.evidence
			err := r.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("expected validation error for %s+%s", tt.severity, tt.evidence)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %s+%s: %v", tt.severity, tt.evidence, err)
			}
		})
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
