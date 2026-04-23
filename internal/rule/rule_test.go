package rule_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/rule"
)

type stubFacts struct{}

func (stubFacts) Repo() model.RepoFacts       { return model.RepoFacts{} }
func (stubFacts) Git() (model.GitFacts, bool) { return model.GitFacts{}, false }
func (stubFacts) Schemas() model.SchemaFacts  { return model.SchemaFacts{} }

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
