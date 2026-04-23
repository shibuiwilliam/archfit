package report_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/report"
	"github.com/shibuiwilliam/archfit/internal/score"
)

func sampleResult() core.ScanResult {
	return core.ScanResult{
		Root:           "/repo",
		RulesEvaluated: 2,
		Findings: []model.Finding{
			{
				RuleID:           "P1.LOC.001",
				Principle:        model.P1Locality,
				Severity:         model.SeverityError,
				EvidenceStrength: model.EvidenceStrong,
				Confidence:       0.95,
				Path:             "",
				Message:          "missing CLAUDE.md",
				Evidence:         map[string]any{"checked": "root"},
				Remediation:      model.Remediation{Summary: "add CLAUDE.md"},
			},
		},
		Metrics: []model.Metric{},
		Scores: score.Scores{
			Overall:     80.0,
			ByPrinciple: map[model.Principle]float64{model.P1Locality: 80.0},
		},
	}
}

func TestRenderJSON_IsDeterministicAndSchemaShaped(t *testing.T) {
	res := sampleResult()
	var a, b bytes.Buffer
	if err := report.Render(&a, res, "0.1.0", "standard", report.FormatJSON); err != nil {
		t.Fatal(err)
	}
	if err := report.Render(&b, res, "0.1.0", "standard", report.FormatJSON); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("JSON output is non-deterministic")
	}
	var doc map[string]any
	if err := json.Unmarshal(a.Bytes(), &doc); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	for _, k := range []string{"schema_version", "tool", "target", "summary", "scores", "findings", "metrics"} {
		if _, ok := doc[k]; !ok {
			t.Errorf("missing field %q in JSON output", k)
		}
	}
	if doc["schema_version"].(string) != "0.1.0" {
		t.Errorf("schema_version: %v", doc["schema_version"])
	}
}

func TestRenderTerminal_Readable(t *testing.T) {
	var out bytes.Buffer
	if err := report.Render(&out, sampleResult(), "0.1.0", "standard", report.FormatTerminal); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "P1.LOC.001") || !strings.Contains(s, "overall score") {
		t.Errorf("terminal output missing expected pieces:\n%s", s)
	}
}

func TestRenderJSON_OmitsLLMSuggestionWhenAbsent(t *testing.T) {
	// CLAUDE.md §9 + ADR 0003: base JSON must be byte-identical when LLM not used.
	res := sampleResult() // helper defined below; no LLMSuggestion set.
	var buf bytes.Buffer
	if err := report.Render(&buf, res, "0.1.0", "standard", report.FormatJSON); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "llm_suggestion") {
		t.Errorf("unused llm_suggestion field leaked into JSON output:\n%s", buf.String())
	}
}

func TestRenderJSON_IncludesLLMSuggestionWhenPresent(t *testing.T) {
	res := sampleResult()
	res.Findings[0].LLMSuggestion = &model.LLMSuggestion{
		Text:     "add CLAUDE.md with the four required sections",
		Model:    "fake",
		CacheHit: false,
	}
	var buf bytes.Buffer
	if err := report.Render(&buf, res, "0.1.0", "standard", report.FormatJSON); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "llm_suggestion") {
		t.Errorf("llm_suggestion missing from JSON:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "four required sections") {
		t.Errorf("suggestion text not preserved:\n%s", buf.String())
	}
}

func TestRenderTerminal_ShowsLLMSuggestion(t *testing.T) {
	res := sampleResult()
	res.Findings[0].LLMSuggestion = &model.LLMSuggestion{
		Text:     "line one\nline two",
		Model:    "fake",
		CacheHit: true,
	}
	var buf bytes.Buffer
	if err := report.Render(&buf, res, "0.1.0", "standard", report.FormatTerminal); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "llm") || !strings.Contains(out, "(cached)") {
		t.Errorf("terminal output missing llm section with cached tag:\n%s", out)
	}
	if !strings.Contains(out, "line one") || !strings.Contains(out, "line two") {
		t.Errorf("multiline LLM text not preserved:\n%s", out)
	}
}

func TestParseFormat(t *testing.T) {
	cases := map[string]report.Format{
		"":         report.FormatTerminal,
		"terminal": report.FormatTerminal,
		"json":     report.FormatJSON,
		"md":       report.FormatMarkdown,
	}
	for in, want := range cases {
		got, err := report.ParseFormat(in)
		if err != nil || got != want {
			t.Errorf("ParseFormat(%q)=(%v,%v), want (%v,nil)", in, got, err, want)
		}
	}
	if _, err := report.ParseFormat("xml"); err == nil {
		t.Error("expected error for unknown format")
	}
}
