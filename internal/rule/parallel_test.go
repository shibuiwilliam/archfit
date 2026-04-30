package rule_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/rule"
)

// makeNRules creates n rules that each emit one finding with a unique message.
// This ensures the parallel path is exercised (threshold = 8).
func makeNRules(n int) []model.Rule {
	rules := make([]model.Rule, n)
	for i := range rules {
		id := fmt.Sprintf("P1.LOC.%03d", i+1)
		msg := fmt.Sprintf("finding-%03d", i+1)
		rules[i] = model.Rule{
			ID:               id,
			Principle:        model.P1Locality,
			Dimension:        "LOC",
			Title:            id,
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "rationale long enough for validation",
			Remediation:      model.Remediation{Summary: "fix it"},
			Resolver: func(_ context.Context, _ model.FactStore) ([]model.Finding, []model.Metric, error) {
				return []model.Finding{{Message: msg, Evidence: map[string]any{"i": i}}}, nil, nil
			},
		}
	}
	return rules
}

// TestEvaluate_ParallelDeterminism runs Evaluate 100 times with -race and
// asserts byte-identical JSON output every iteration. This verifies that the
// parallel path (triggered when len(rules) >= 8) produces deterministic
// output regardless of goroutine scheduling.
func TestEvaluate_ParallelDeterminism(t *testing.T) {
	rules := makeNRules(16) // well above parallelThreshold
	eng := rule.NewEngine()

	var baseline []byte
	for i := 0; i < 100; i++ {
		res := eng.Evaluate(context.Background(), rules, stubFacts{})

		data, err := json.Marshal(res.Findings)
		if err != nil {
			t.Fatal(err)
		}

		if i == 0 {
			baseline = data
			if res.RulesEvaluated != 16 {
				t.Fatalf("expected 16 rules evaluated, got %d", res.RulesEvaluated)
			}
			if len(res.Findings) != 16 {
				t.Fatalf("expected 16 findings, got %d", len(res.Findings))
			}
			continue
		}

		if !bytes.Equal(data, baseline) {
			t.Fatalf("iteration %d produced different output.\nbaseline: %s\ngot:      %s", i, baseline, data)
		}
	}
}

// TestEvaluate_ParallelPanicRecovery verifies that a panicking resolver in the
// parallel path doesn't crash other goroutines.
func TestEvaluate_ParallelPanicRecovery(t *testing.T) {
	rules := makeNRules(10)
	// Replace one resolver with a panicking one.
	rules[5].Resolver = func(context.Context, model.FactStore) ([]model.Finding, []model.Metric, error) {
		panic("test panic in parallel")
	}

	eng := rule.NewEngine()
	res := eng.Evaluate(context.Background(), rules, stubFacts{})

	// 9 rules should produce findings, 1 should produce an error.
	if res.RulesEvaluated != 10 {
		t.Errorf("RulesEvaluated = %d, want 10", res.RulesEvaluated)
	}
	if len(res.Findings) != 9 {
		t.Errorf("Findings = %d, want 9 (one panicked)", len(res.Findings))
	}
	if len(res.Errors) != 1 {
		t.Errorf("Errors = %d, want 1 (the panicking resolver)", len(res.Errors))
	}
}

func BenchmarkEvaluate_Serial(b *testing.B) {
	rules := makeNRules(4) // below threshold → serial
	eng := rule.NewEngine()
	for b.Loop() {
		eng.Evaluate(context.Background(), rules, stubFacts{})
	}
}

func BenchmarkEvaluate_Parallel(b *testing.B) {
	rules := makeNRules(16) // above threshold → parallel
	eng := rule.NewEngine()
	for b.Loop() {
		eng.Evaluate(context.Background(), rules, stubFacts{})
	}
}
