package report_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/report"
)

func finding(rule, path, msg string) model.Finding {
	return model.Finding{RuleID: rule, Path: path, Message: msg, Severity: model.SeverityWarn}
}

func TestDiff_ClassifiesCorrectly(t *testing.T) {
	baseline := []model.Finding{
		finding("P1.LOC.001", "", "missing CLAUDE.md"),
		finding("P4.VER.001", "", "no verification entrypoint"),
	}
	current := []model.Finding{
		finding("P1.LOC.001", "", "missing CLAUDE.md"),              // unchanged
		finding("P7.MRD.001", "docs/", "exit codes not documented"), // new
		// P4.VER.001 disappears → fixed
	}
	d := report.Diff(baseline, current)
	if len(d.Unchanged) != 1 || d.Unchanged[0].RuleID != "P1.LOC.001" {
		t.Errorf("unchanged: %+v", d.Unchanged)
	}
	if len(d.New) != 1 || d.New[0].RuleID != "P7.MRD.001" {
		t.Errorf("new: %+v", d.New)
	}
	if len(d.Fixed) != 1 || d.Fixed[0].RuleID != "P4.VER.001" {
		t.Errorf("fixed: %+v", d.Fixed)
	}
}

func TestDiff_JSONIsDeterministic(t *testing.T) {
	d := report.Diff(
		[]model.Finding{finding("P1.LOC.001", "", "a"), finding("P4.VER.001", "", "b")},
		[]model.Finding{finding("P4.VER.001", "", "b"), finding("P7.MRD.001", "x", "c")},
	)
	var a, b bytes.Buffer
	if err := report.RenderDiffJSON(&a, d); err != nil {
		t.Fatal(err)
	}
	if err := report.RenderDiffJSON(&b, d); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("diff JSON is not deterministic")
	}
	var doc map[string]any
	if err := json.Unmarshal(a.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"new", "fixed", "unchanged", "summary"} {
		if _, ok := doc[k]; !ok {
			t.Errorf("missing %q in diff output", k)
		}
	}
}

func TestLoadBaseline_AcceptsCurrentOutputFormat(t *testing.T) {
	data := []byte(`{
      "schema_version": "0.1.0",
      "findings": [{"rule_id":"P1.LOC.001","severity":"warn","path":"","message":"m","evidence":{},"remediation":{"summary":"fix"},"principle":"P1","evidence_strength":"strong","confidence":0.9}]
    }`)
	doc, err := report.LoadBaseline(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Findings) != 1 || doc.Findings[0].RuleID != "P1.LOC.001" {
		t.Errorf("unexpected parse: %+v", doc)
	}
}
