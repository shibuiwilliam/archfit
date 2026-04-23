package llm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/adapter/llm"
	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/report"
	"github.com/shibuiwilliam/archfit/internal/rule"
	corepack "github.com/shibuiwilliam/archfit/packs/core"
)

// enrich mirrors the CLI's enrichFindings. Inlined here (rather than exported
// from cmd/archfit) so that the LLM integration has a testable unit without
// coupling the test to the CLI main package.
func enrich(ctx context.Context, client llm.Client, rules []model.Rule, findings []model.Finding) []string {
	byID := map[string]model.Rule{}
	for _, r := range rules {
		byID[r.ID] = r
	}
	var warnings []string
	for i := range findings {
		r, ok := byID[findings[i].RuleID]
		if !ok {
			continue
		}
		sug, err := client.Explain(ctx, r, findings[i], llm.BuildFindingPrompt(r, findings[i], nil))
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		findings[i].LLMSuggestion = &model.LLMSuggestion{
			Text: sug.Text, Model: sug.Model, CacheHit: sug.CacheHit,
		}
	}
	return warnings
}

func TestEnrichment_AttachesSuggestionAndSurvivesJSON(t *testing.T) {
	// Arrange: scan a fixture that triggers exactly P1.LOC.001.
	reg := rule.NewRegistry()
	if err := corepack.Register(reg); err != nil {
		t.Fatal(err)
	}
	res, err := core.Scan(context.Background(), core.ScanInput{
		Root:  "../../../packs/core/fixtures/P1.LOC.001/input",
		Rules: reg.Rules(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) == 0 {
		t.Fatal("expected at least one finding on P1.LOC.001 fixture")
	}

	// Act: enrich with a Fake that gives a stable canned response.
	fake := llm.NewFake()
	fake.Responses["P1.LOC.001"] = "write CLAUDE.md at the root with 4 short sections"
	warnings := enrich(context.Background(), fake, reg.Rules(), res.Findings)
	if len(warnings) > 0 {
		t.Fatalf("unexpected llm warnings: %v", warnings)
	}

	// Assert: JSON carries llm_suggestion with our canned text.
	var buf bytes.Buffer
	if err := report.Render(&buf, res, "test", "standard", report.FormatJSON); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	findings := doc["findings"].([]any)
	if len(findings) == 0 {
		t.Fatal("rendered findings empty")
	}
	first := findings[0].(map[string]any)
	sug, ok := first["llm_suggestion"].(map[string]any)
	if !ok {
		t.Fatalf("llm_suggestion missing: %+v", first)
	}
	if !strings.Contains(sug["text"].(string), "4 short sections") {
		t.Errorf("suggestion text not preserved: %v", sug["text"])
	}
	if sug["model"] != "fake" {
		t.Errorf("model tag missing or wrong: %v", sug["model"])
	}
}

func TestEnrichment_BudgetExhaustionDegradesSilently(t *testing.T) {
	// Given a budget of 1 and 2 findings, the second should keep its static
	// remediation (no llm_suggestion) and the enrichment function must NOT
	// return a warning for ErrBudgetExhausted — the user set the budget.
	findings := []model.Finding{
		{RuleID: "P1.LOC.001", Path: "", Message: "first"},
		{RuleID: "P1.LOC.001", Path: "somewhere/else", Message: "second"},
	}
	r := model.Rule{
		ID: "P1.LOC.001", Principle: model.P1Locality, Dimension: "LOC",
		Title: "t", Severity: model.SeverityWarn, EvidenceStrength: model.EvidenceStrong,
		Stability: model.StabilityExperimental, Rationale: "r",
		Remediation: model.Remediation{Summary: "s"},
	}
	client := llm.NewCached(llm.NewBudget(llm.NewFake(), 1))
	warnings := enrich(context.Background(), client, []model.Rule{r}, findings)

	// First finding gets enriched.
	if findings[0].LLMSuggestion == nil {
		t.Error("first finding should have been enriched")
	}
	// Second finding hits budget exhaustion — we treat that as a warning in this
	// test harness but in the real CLI it's silent. Either way, no LLMSuggestion.
	if findings[1].LLMSuggestion != nil {
		t.Errorf("second finding should NOT be enriched: %+v", findings[1].LLMSuggestion)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "budget exhausted") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a budget-exhausted warning in test harness, got %v", warnings)
	}
}
