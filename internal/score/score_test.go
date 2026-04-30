package score_test

import (
	"context"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/score"
)

func mkRule(id string, p model.Principle, sev model.Severity, weight float64) model.Rule {
	return model.Rule{
		ID: id, Principle: p, Dimension: "LOC", Title: id,
		Severity: sev, EvidenceStrength: model.EvidenceStrong, Stability: model.StabilityExperimental,
		Rationale: "rationale long enough", Weight: weight,
		Remediation: model.Remediation{Summary: "x"},
		Resolver: func(context.Context, model.FactStore) ([]model.Finding, []model.Metric, error) {
			return nil, nil, nil
		},
	}
}

func TestCompute_PerfectScoreWithNoFindings(t *testing.T) {
	rules := []model.Rule{mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1)}
	s := score.Compute(rules, nil)
	if s.Overall != 100 {
		t.Errorf("expected 100, got %v", s.Overall)
	}
}

func TestCompute_WarnPenaltyIs40Pct(t *testing.T) {
	rules := []model.Rule{mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1)}
	findings := []model.Finding{{RuleID: "P1.LOC.001", Severity: model.SeverityWarn}}
	s := score.Compute(rules, findings)
	if s.Overall != 60.0 {
		t.Errorf("expected 60.0 (100 - 40%% penalty), got %v", s.Overall)
	}
}

func TestCompute_MultipleFindingsForSameRuleDontCompound(t *testing.T) {
	rules := []model.Rule{mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1)}
	findings := []model.Finding{
		{RuleID: "P1.LOC.001", Severity: model.SeverityWarn},
		{RuleID: "P1.LOC.001", Severity: model.SeverityWarn},
		{RuleID: "P1.LOC.001", Severity: model.SeverityWarn},
	}
	s := score.Compute(rules, findings)
	if s.Overall != 60.0 {
		t.Errorf("want 60.0 (noisy rules do not stack), got %v", s.Overall)
	}
}

func TestCompute_AddingRulesWithoutFindingsDoesNotLowerExistingScore(t *testing.T) {
	// CLAUDE.md §13: adding rules must not make the score artificially go down.
	baseline := []model.Rule{mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1)}
	findings := []model.Finding{{RuleID: "P1.LOC.001", Severity: model.SeverityError}}
	baseScore := score.Compute(baseline, findings).Overall

	extended := make([]model.Rule, len(baseline))
	copy(extended, baseline)
	extended = append(extended, mkRule("P2.SPC.001", model.P2SpecFirst, model.SeverityWarn, 1))
	extScore := score.Compute(extended, findings).Overall

	if extScore < baseScore {
		t.Errorf("adding a rule that produced no findings lowered the score: %v < %v", extScore, baseScore)
	}
}

func TestCompute_WorstSeverityWinsPerRule(t *testing.T) {
	rules := []model.Rule{mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1)}
	findings := []model.Finding{
		{RuleID: "P1.LOC.001", Severity: model.SeverityWarn},
		{RuleID: "P1.LOC.001", Severity: model.SeverityError},
	}
	s := score.Compute(rules, findings)
	// error severity → 80% penalty → 20.0
	if s.Overall != 20.0 {
		t.Errorf("want 20.0 from error severity, got %v", s.Overall)
	}
}

func TestCompute_SkippedRulesExcludedFromWeight(t *testing.T) {
	rules := []model.Rule{
		mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1),
		mkRule("P3.EXP.001", model.P3ShallowExplicitness, model.SeverityWarn, 1),
	}
	// P3.EXP.001 has a finding, but was skipped (applies_to mismatch).
	// Its weight should not count, so the score stays 100 for P3.
	findings := []model.Finding{
		{RuleID: "P3.EXP.001", Severity: model.SeverityWarn},
	}
	s := score.Compute(rules, findings, "P3.EXP.001")

	if s.Overall != 100 {
		t.Errorf("overall = %v, want 100 (skipped rule's finding should not penalize)", s.Overall)
	}
	if _, hasP3 := s.ByPrinciple[model.P3ShallowExplicitness]; hasP3 {
		t.Errorf("P3 should be absent from by_principle when its only rule is skipped")
	}
}

func TestCompute_SkippedRuleDoesNotInflateScore(t *testing.T) {
	rules := []model.Rule{
		mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1),
		mkRule("P3.EXP.001", model.P3ShallowExplicitness, model.SeverityWarn, 1),
	}
	// P1.LOC.001 has a finding (warn → 40% penalty). P3.EXP.001 is skipped.
	findings := []model.Finding{
		{RuleID: "P1.LOC.001", Severity: model.SeverityWarn},
	}
	s := score.Compute(rules, findings, "P3.EXP.001")

	// With P3 skipped, only P1's weight=1 counts. warn penalty → 60.0.
	if s.Overall != 60 {
		t.Errorf("overall = %v, want 60 (only P1's weight counts)", s.Overall)
	}
}
