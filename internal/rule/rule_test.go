package rule_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/rule"
)

type stubFacts struct{}

func (stubFacts) Repo() model.RepoFacts                 { return model.RepoFacts{} }
func (stubFacts) Git() (model.GitFacts, bool)           { return model.GitFacts{}, false }
func (stubFacts) Schemas() model.SchemaFacts            { return model.SchemaFacts{} }
func (stubFacts) Commands() (model.CommandFacts, bool)  { return model.CommandFacts{}, false }
func (stubFacts) DepGraph() (model.DepGraphFacts, bool) { return model.DepGraphFacts{}, false }
func (stubFacts) Languages() map[string]int             { return nil }
func (stubFacts) Ecosystems() model.EcosystemFacts      { return model.EcosystemFacts{} }
func (stubFacts) AST() (model.ASTFacts, bool)           { return model.ASTFacts{}, false }

func mkRule(id string, fn model.ResolverFunc) model.Rule {
	return model.Rule{
		ID:               id,
		Principle:        model.Principle("P" + string(id[1])),
		Dimension:        id[3:6],
		Title:            id,
		Severity:         model.SeverityWarn,
		EvidenceStrength: model.EvidenceStrong,
		Stability:        model.StabilityExperimental,
		Rationale:        "rationale long enough",
		Remediation:      model.Remediation{Summary: "do the thing"},
		Resolver:         fn,
	}
}

// stubFactsWithLangs returns specific languages from Languages().
type stubFactsWithLangs struct {
	langs map[string]int
}

func (s stubFactsWithLangs) Repo() model.RepoFacts                { return model.RepoFacts{Languages: s.langs} }
func (s stubFactsWithLangs) Git() (model.GitFacts, bool)          { return model.GitFacts{}, false }
func (s stubFactsWithLangs) Schemas() model.SchemaFacts           { return model.SchemaFacts{} }
func (s stubFactsWithLangs) Commands() (model.CommandFacts, bool) { return model.CommandFacts{}, false }
func (s stubFactsWithLangs) DepGraph() (model.DepGraphFacts, bool) {
	return model.DepGraphFacts{}, false
}
func (s stubFactsWithLangs) Languages() map[string]int        { return s.langs }
func (s stubFactsWithLangs) Ecosystems() model.EcosystemFacts { return model.EcosystemFacts{} }
func (s stubFactsWithLangs) AST() (model.ASTFacts, bool)      { return model.ASTFacts{}, false }

func TestEngine_SkipsRulesWithLanguageMismatch(t *testing.T) {
	noop := func(context.Context, model.FactStore) ([]model.Finding, []model.Metric, error) {
		return []model.Finding{{Message: "fired"}}, nil, nil
	}
	// Rule that requires Java.
	javaRule := mkRule("P3.EXP.001", noop)
	javaRule.AppliesTo = model.Applicability{Languages: []string{"java"}}

	// Rule with no language constraint — always runs.
	anyRule := mkRule("P1.LOC.001", noop)

	eng := rule.NewEngine()

	t.Run("Go-only repo skips Java rule", func(t *testing.T) {
		facts := stubFactsWithLangs{langs: map[string]int{"go": 50}}
		res := eng.Evaluate(context.Background(), []model.Rule{javaRule, anyRule}, facts)

		if res.RulesEvaluated != 1 {
			t.Errorf("RulesEvaluated = %d, want 1 (only P1.LOC.001)", res.RulesEvaluated)
		}
		if len(res.SkippedRuleIDs) != 1 || res.SkippedRuleIDs[0] != "P3.EXP.001" {
			t.Errorf("SkippedRuleIDs = %v, want [P3.EXP.001]", res.SkippedRuleIDs)
		}
		if len(res.Findings) != 1 || res.Findings[0].RuleID != "P1.LOC.001" {
			t.Errorf("expected only P1.LOC.001 finding, got %+v", res.Findings)
		}
	})

	t.Run("Java repo runs both rules", func(t *testing.T) {
		facts := stubFactsWithLangs{langs: map[string]int{"java": 30, "go": 10}}
		res := eng.Evaluate(context.Background(), []model.Rule{javaRule, anyRule}, facts)

		if res.RulesEvaluated != 2 {
			t.Errorf("RulesEvaluated = %d, want 2", res.RulesEvaluated)
		}
		if len(res.SkippedRuleIDs) != 0 {
			t.Errorf("SkippedRuleIDs = %v, want empty", res.SkippedRuleIDs)
		}
		if len(res.Findings) != 2 {
			t.Errorf("expected 2 findings, got %d", len(res.Findings))
		}
	})
}

func TestRegistry_DuplicateRejected(t *testing.T) {
	reg := rule.NewRegistry()
	noop := func(context.Context, model.FactStore) ([]model.Finding, []model.Metric, error) {
		return nil, nil, nil
	}
	if err := reg.Register("core", mkRule("P1.LOC.001", noop)); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register("core", mkRule("P1.LOC.001", noop)); err == nil {
		t.Fatal("expected duplicate rule error")
	}
}

func TestEngine_BackfillsAndSortsAndRecoversPanics(t *testing.T) {
	reg := rule.NewRegistry()
	err := reg.Register("core",
		mkRule("P1.LOC.001", func(context.Context, model.FactStore) ([]model.Finding, []model.Metric, error) {
			return []model.Finding{{Path: "z", Message: "m"}, {Path: "a", Message: "m"}}, nil, nil
		}),
		mkRule("P2.SPC.001", func(context.Context, model.FactStore) ([]model.Finding, []model.Metric, error) {
			return nil, nil, errors.New("boom")
		}),
		mkRule("P4.VER.001", func(context.Context, model.FactStore) ([]model.Finding, []model.Metric, error) {
			panic("kaboom")
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	eng := rule.NewEngine()
	res := eng.Evaluate(context.Background(), reg.Rules(), stubFacts{})

	if res.RulesEvaluated != 3 {
		t.Errorf("rulesEvaluated: %d", res.RulesEvaluated)
	}
	if len(res.Findings) != 2 {
		t.Errorf("findings: %d", len(res.Findings))
	}
	if res.Findings[0].Path != "a" || res.Findings[0].RuleID != "P1.LOC.001" {
		t.Errorf("not sorted or unbackfilled: %+v", res.Findings)
	}
	if res.Findings[0].Severity != model.SeverityWarn || res.Findings[0].EvidenceStrength != model.EvidenceStrong {
		t.Errorf("defaults not filled: %+v", res.Findings[0])
	}
	if len(res.Errors) != 2 {
		t.Errorf("expected 2 errors (one plain, one recovered panic), got %d: %+v", len(res.Errors), res.Errors)
	}
}
