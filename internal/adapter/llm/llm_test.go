package llm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/adapter/llm"
	"github.com/shibuiwilliam/archfit/internal/model"
)

func sampleRule() model.Rule {
	return model.Rule{
		ID: "P1.LOC.001", Principle: model.P1Locality, Dimension: "LOC",
		Title:    "Agent docs at root",
		Severity: model.SeverityWarn, EvidenceStrength: model.EvidenceStrong,
		Stability: model.StabilityExperimental, Rationale: "agents need an entry point.",
		Remediation: model.Remediation{Summary: "add CLAUDE.md"},
	}
}

func sampleFinding() model.Finding {
	return model.Finding{
		RuleID: "P1.LOC.001", Severity: model.SeverityWarn, Path: "",
		Message:  "no CLAUDE.md at root",
		Evidence: map[string]any{"checked_paths": []string{"CLAUDE.md", "AGENTS.md"}},
	}
}

func TestFake_ReturnsCannedAndRecordsCalls(t *testing.T) {
	f := llm.NewFake()
	f.Responses["P1.LOC.001"] = "add CLAUDE.md with 4 sections and re-scan"

	sug, err := f.Explain(context.Background(), sampleRule(), sampleFinding(), llm.Prompt{})
	if err != nil {
		t.Fatal(err)
	}
	if sug.Text != "add CLAUDE.md with 4 sections and re-scan" {
		t.Errorf("unexpected text: %q", sug.Text)
	}
	if len(f.Calls) != 1 || f.Calls[0].RuleID != "P1.LOC.001" {
		t.Errorf("calls not recorded: %+v", f.Calls)
	}
}

func TestFake_FailOnProducesError(t *testing.T) {
	f := llm.NewFake()
	f.FailOn = errors.New("fake outage")
	_, err := f.Explain(context.Background(), sampleRule(), sampleFinding(), llm.Prompt{})
	if err == nil {
		t.Fatal("expected fake error")
	}
}

func TestCached_SecondCallIsCacheHit(t *testing.T) {
	f := llm.NewFake()
	c := llm.NewCached(f)
	p := llm.Prompt{System: "sys", User: "u"}

	_, err := c.Explain(context.Background(), sampleRule(), sampleFinding(), p)
	if err != nil {
		t.Fatal(err)
	}
	sug2, err := c.Explain(context.Background(), sampleRule(), sampleFinding(), p)
	if err != nil {
		t.Fatal(err)
	}
	if !sug2.CacheHit {
		t.Errorf("expected cache_hit=true on second call, got %+v", sug2)
	}
	if len(f.Calls) != 1 {
		t.Errorf("expected 1 underlying call, got %d", len(f.Calls))
	}
}

func TestBudget_ExhaustsAndReturnsError(t *testing.T) {
	f := llm.NewFake()
	b := llm.NewBudget(f, 2)
	p := llm.Prompt{System: "sys", User: "u"}

	for i := 0; i < 2; i++ {
		// Vary the prompt so we miss the Cached wrapper (not used here) and
		// actually charge the budget each iteration.
		p.User = "u-" + string(rune('a'+i))
		if _, err := b.Explain(context.Background(), sampleRule(), sampleFinding(), p); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if got := b.Remaining(); got != 0 {
		t.Fatalf("remaining=%d, want 0", got)
	}

	p.User = "u-third"
	_, err := b.Explain(context.Background(), sampleRule(), sampleFinding(), p)
	if !errors.Is(err, llm.ErrBudgetExhausted) {
		t.Fatalf("expected ErrBudgetExhausted, got %v", err)
	}
}

func TestBudget_CacheHitsDontConsumeBudget(t *testing.T) {
	// Canonical composition: Cached OUTSIDE Budget so hits bypass the budget.
	f := llm.NewFake()
	budget := llm.NewBudget(f, 1)
	client := llm.NewCached(budget)
	p := llm.Prompt{System: "sys", User: "u"}

	// First call: cache miss → budget consumed (1 → 0).
	if _, err := client.Explain(context.Background(), sampleRule(), sampleFinding(), p); err != nil {
		t.Fatal(err)
	}
	// Second call, same prompt → cache hit → budget NOT touched.
	sug, err := client.Explain(context.Background(), sampleRule(), sampleFinding(), p)
	if err != nil {
		t.Fatal(err)
	}
	if !sug.CacheHit {
		t.Errorf("expected cache hit on repeat call")
	}
	if budget.Remaining() != 0 {
		t.Errorf("budget should still be 0 (not negative) after cache hit; got %d", budget.Remaining())
	}
	// Different prompt → cache miss → budget exhausted.
	p.User = "different"
	_, err = client.Explain(context.Background(), sampleRule(), sampleFinding(), p)
	if !errors.Is(err, llm.ErrBudgetExhausted) {
		t.Errorf("expected ErrBudgetExhausted on third distinct call, got %v", err)
	}
	// And the Fake should have seen exactly 1 real call (the first miss).
	if len(f.Calls) != 1 {
		t.Errorf("Fake expected 1 call, got %d", len(f.Calls))
	}
}

func TestFromEnv_GoogleKeyWins(t *testing.T) {
	env := map[string]string{"GOOGLE_API_KEY": "goog", "GEMINI_API_KEY": "gem"}
	cfg, ok := llm.FromEnv(func(k string) string { return env[k] })
	if !ok || cfg.APIKey != "goog" {
		t.Errorf("want GOOGLE_API_KEY to win, got %+v ok=%v", cfg, ok)
	}
}

func TestFromEnv_GeminiKeyFallback(t *testing.T) {
	env := map[string]string{"GEMINI_API_KEY": "gem"}
	cfg, ok := llm.FromEnv(func(k string) string { return env[k] })
	if !ok || cfg.APIKey != "gem" {
		t.Errorf("want Gemini fallback, got %+v ok=%v", cfg, ok)
	}
}

func TestFromEnv_NoKeyReturnsFalse(t *testing.T) {
	_, ok := llm.FromEnv(func(k string) string { return "" })
	if ok {
		t.Errorf("expected ok=false when no keys set")
	}
}

func TestBuildFindingPrompt_Truncates(t *testing.T) {
	rule := sampleRule()
	f := sampleFinding()
	// Stuff evidence with a huge value to exceed MaxUserBytes.
	big := make([]byte, llm.MaxUserBytes*2)
	for i := range big {
		big[i] = 'x'
	}
	f.Evidence = map[string]any{"big": string(big)}
	p := llm.BuildFindingPrompt(rule, f, []string{"agent-tool"})
	if len(p.User) > llm.MaxUserBytes+32 {
		t.Errorf("BuildFindingPrompt did not truncate: len=%d want <=%d", len(p.User), llm.MaxUserBytes+32)
	}
}

func TestBuildFindingPrompt_StableOrdering(t *testing.T) {
	// Evidence keys must serialize in sorted order so prompts hit the cache
	// across runs.
	rule := sampleRule()
	f := model.Finding{
		RuleID:   "P1.LOC.001",
		Evidence: map[string]any{"b": 1, "a": 2, "c": 3},
	}
	a := llm.BuildFindingPrompt(rule, f, nil)
	b := llm.BuildFindingPrompt(rule, f, nil)
	if a.User != b.User {
		t.Errorf("prompt not deterministic across calls")
	}
}
