package score_test

import (
	"context"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/score"
)

func mkRule(id string, p model.Principle, sev model.Severity, weight float64) model.Rule {
	return mkRuleWithEvidence(id, p, sev, model.EvidenceStrong, weight)
}

func mkRuleWithEvidence(id string, p model.Principle, sev model.Severity, ev model.EvidenceStrength, weight float64) model.Rule {
	return model.Rule{
		ID: id, Principle: p, Dimension: "LOC", Title: id,
		Severity: sev, EvidenceStrength: ev, Stability: model.StabilityExperimental,
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

func TestCompute_EvidenceFactorReducesWeight(t *testing.T) {
	// A medium-evidence rule with weight=1 has effective weight 0.85.
	// Two rules: one strong (w=1.0), one medium (w=0.85). Total = 1.85.
	// The medium rule has a warn finding → penalty = 0.40 * 0.85 = 0.34.
	// Score = 100 * (1 - 0.34/1.85) = 100 * 0.8162... = 81.6.
	rules := []model.Rule{
		mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1),
		mkRuleWithEvidence("P1.LOC.002", model.P1Locality, model.SeverityInfo, model.EvidenceMedium, 1),
	}
	findings := []model.Finding{
		{RuleID: "P1.LOC.002", Severity: model.SeverityInfo},
	}
	s := score.Compute(rules, findings)
	// penalty = 0.10 * 0.85 = 0.085; total = 1.85; score = 100*(1-0.085/1.85) = 95.4
	if s.Overall != 95.4 {
		t.Errorf("overall = %v, want 95.4", s.Overall)
	}
}

func TestCompute_SeverityClassPassRates(t *testing.T) {
	rules := []model.Rule{
		mkRule("P1.LOC.001", model.P1Locality, model.SeverityWarn, 1),
		mkRule("P4.VER.001", model.P4Verifiability, model.SeverityWarn, 1),
		mkRule("P5.AGG.004", model.P5Aggregation, model.SeverityError, 1),
		mkRuleWithEvidence("P7.MRD.001", model.P7MachineReadability, model.SeverityInfo, model.EvidenceMedium, 1),
	}
	findings := []model.Finding{
		{RuleID: "P1.LOC.001", Severity: model.SeverityWarn},
		// P5.AGG.004 passes (no finding), P4.VER.001 passes, P7.MRD.001 passes.
	}
	s := score.Compute(rules, findings)

	// error: 1 total, 1 passed → 1.0
	if s.BySeverityClass.ErrorPassRate != 1.0 {
		t.Errorf("error_pass_rate = %v, want 1.0", s.BySeverityClass.ErrorPassRate)
	}
	// warn: 2 total, 1 passed → 0.5
	if s.BySeverityClass.WarnPassRate != 0.5 {
		t.Errorf("warn_pass_rate = %v, want 0.5", s.BySeverityClass.WarnPassRate)
	}
	// info: 1 total, 1 passed → 1.0
	if s.BySeverityClass.InfoPassRate != 1.0 {
		t.Errorf("info_pass_rate = %v, want 1.0", s.BySeverityClass.InfoPassRate)
	}
	// critical: 0 total → 1.0 (no rules at this tier)
	if s.BySeverityClass.CriticalPassRate != 1.0 {
		t.Errorf("critical_pass_rate = %v, want 1.0", s.BySeverityClass.CriticalPassRate)
	}
}

func TestCompute_SeverityClassAllFailing(t *testing.T) {
	rules := []model.Rule{
		mkRule("P5.AGG.004", model.P5Aggregation, model.SeverityError, 1),
	}
	findings := []model.Finding{
		{RuleID: "P5.AGG.004", Severity: model.SeverityError},
	}
	s := score.Compute(rules, findings)

	if s.BySeverityClass.ErrorPassRate != 0.0 {
		t.Errorf("error_pass_rate = %v, want 0.0", s.BySeverityClass.ErrorPassRate)
	}
}
