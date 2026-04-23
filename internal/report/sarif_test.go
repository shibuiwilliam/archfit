package report_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/report"
)

func sampleRule() model.Rule {
	return model.Rule{
		ID: "P1.LOC.001", Principle: model.P1Locality, Dimension: "LOC",
		Title: "Agent docs at root", Severity: model.SeverityWarn,
		EvidenceStrength: model.EvidenceStrong, Stability: model.StabilityExperimental,
		Rationale:   "rationale long enough",
		Remediation: model.Remediation{Summary: "add CLAUDE.md"},
		Resolver: func(context.Context, model.FactStore) ([]model.Finding, []model.Metric, error) {
			return nil, nil, nil
		},
	}
}

func TestRenderSARIF_MinimalConformance(t *testing.T) {
	res := core.ScanResult{
		Root: "/repo",
		Findings: []model.Finding{{
			RuleID: "P1.LOC.001", Principle: model.P1Locality,
			Severity: model.SeverityWarn, EvidenceStrength: model.EvidenceStrong,
			Confidence: 0.99, Path: "", Message: "missing CLAUDE.md",
			Evidence: map[string]any{"checked": "root"},
		}},
		RulesEvaluated: 1,
	}

	var buf bytes.Buffer
	if err := report.RenderSARIF(&buf, res, []model.Rule{sampleRule()}, "0.2.0"); err != nil {
		t.Fatal(err)
	}

	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("SARIF output is not valid JSON: %v", err)
	}
	if doc["version"] != "2.1.0" {
		t.Errorf("version: %v", doc["version"])
	}
	runs := doc["runs"].([]any)
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	run := runs[0].(map[string]any)
	tool := run["tool"].(map[string]any)
	driver := tool["driver"].(map[string]any)
	if driver["name"] != "archfit" {
		t.Errorf("tool.driver.name: %v", driver["name"])
	}
	rules := driver["rules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].(map[string]any)["id"] != "P1.LOC.001" {
		t.Errorf("rule id: %v", rules[0])
	}

	results := run["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0].(map[string]any)
	if r["level"] != "warning" {
		t.Errorf("result.level: %v (want warning for severity=warn)", r["level"])
	}
	locs := r["locations"].([]any)
	if len(locs) != 1 {
		t.Fatalf("expected 1 location (synthetic root), got %d", len(locs))
	}
}

func TestRenderSARIF_IsDeterministic(t *testing.T) {
	res := core.ScanResult{Root: "/r"}
	var a, b bytes.Buffer
	if err := report.RenderSARIF(&a, res, []model.Rule{sampleRule()}, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if err := report.RenderSARIF(&b, res, []model.Rule{sampleRule()}, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("SARIF output is not deterministic")
	}
}

func TestSeverityMapping(t *testing.T) {
	cases := map[model.Severity]string{
		model.SeverityInfo:     "note",
		model.SeverityWarn:     "warning",
		model.SeverityError:    "error",
		model.SeverityCritical: "error",
	}
	for sev, want := range cases {
		res := core.ScanResult{
			Findings: []model.Finding{{RuleID: "P1.LOC.001", Severity: sev, Path: "x", Message: "m"}},
		}
		var buf bytes.Buffer
		if err := report.RenderSARIF(&buf, res, []model.Rule{sampleRule()}, "0.2.0"); err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		_ = json.Unmarshal(buf.Bytes(), &doc)
		level := doc["runs"].([]any)[0].(map[string]any)["results"].([]any)[0].(map[string]any)["level"]
		if level != want {
			t.Errorf("severity %s → %v, want %s", sev, level, want)
		}
	}
}

func TestParseFormat_SARIF(t *testing.T) {
	f, err := report.ParseFormat("sarif")
	if err != nil || f != report.FormatSARIF {
		t.Errorf("ParseFormat(sarif) = (%v, %v)", f, err)
	}
}
