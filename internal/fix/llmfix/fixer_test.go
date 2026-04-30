package llmfix_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/adapter/llm"
	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/fix/llmfix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// stubFixer is a minimal static fixer for testing the LLM wrapper.
type stubFixer struct {
	ruleID  string
	content string
}

func (s *stubFixer) RuleID() string { return s.ruleID }
func (s *stubFixer) NeedsLLM() bool { return false }
func (s *stubFixer) Plan(_ context.Context, _ model.Finding, _ model.FactStore) ([]fix.Change, error) {
	return []fix.Change{{
		Path:    "CLAUDE.md",
		Action:  fix.ActionCreate,
		Content: []byte(s.content),
		Preview: "static scaffold",
	}}, nil
}

type fakeFactStore struct{ repo model.RepoFacts }

func (f *fakeFactStore) Repo() model.RepoFacts                 { return f.repo }
func (f *fakeFactStore) Git() (model.GitFacts, bool)           { return model.GitFacts{}, false }
func (f *fakeFactStore) Schemas() model.SchemaFacts            { return model.SchemaFacts{} }
func (f *fakeFactStore) Commands() (model.CommandFacts, bool)  { return model.CommandFacts{}, false }
func (f *fakeFactStore) DepGraph() (model.DepGraphFacts, bool) { return model.DepGraphFacts{}, false }
func (f *fakeFactStore) Languages() map[string]int             { return nil }
func (f *fakeFactStore) Ecosystems() model.EcosystemFacts      { return model.EcosystemFacts{} }

func TestLLMFixer_EnrichesContent(t *testing.T) {
	fake := llm.NewFake()
	fake.Responses["P1.LOC.001"] = "# CLAUDE.md — my-project\n\nEnriched by LLM.\n"

	base := &stubFixer{ruleID: "P1.LOC.001", content: "# CLAUDE.md\n\nStatic template.\n"}
	var stderr bytes.Buffer
	fixer := llmfix.NewLLMFixer(base, fake, &stderr)

	if fixer.RuleID() != "P1.LOC.001" {
		t.Errorf("RuleID() = %q", fixer.RuleID())
	}
	if !fixer.NeedsLLM() {
		t.Error("NeedsLLM() should be true")
	}

	facts := &fakeFactStore{repo: model.RepoFacts{Root: "/tmp/my-project"}}
	changes, err := fixer.Plan(context.Background(), model.Finding{RuleID: "P1.LOC.001"}, facts)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if !strings.Contains(string(changes[0].Content), "Enriched by LLM") {
		t.Errorf("content should be LLM-enriched: %q", string(changes[0].Content))
	}
	if !strings.Contains(changes[0].Preview, "LLM-enriched") {
		t.Errorf("preview should mention LLM: %q", changes[0].Preview)
	}
}

func TestLLMFixer_FallsBackOnError(t *testing.T) {
	fake := llm.NewFake()
	fake.FailOn = context.DeadlineExceeded

	base := &stubFixer{ruleID: "P1.LOC.001", content: "# Static fallback\n"}
	var stderr bytes.Buffer
	fixer := llmfix.NewLLMFixer(base, fake, &stderr)

	facts := &fakeFactStore{repo: model.RepoFacts{Root: "/tmp/test"}}
	changes, err := fixer.Plan(context.Background(), model.Finding{RuleID: "P1.LOC.001"}, facts)
	if err != nil {
		t.Fatal(err)
	}
	// Should fall back to static content.
	if !strings.Contains(string(changes[0].Content), "Static fallback") {
		t.Errorf("should fall back to static: %q", string(changes[0].Content))
	}
	// Should log to stderr.
	if !strings.Contains(stderr.String(), "falling back") {
		t.Errorf("should log fallback: %q", stderr.String())
	}
}

func TestLLMFixer_FallsBackOnEmptyResponse(t *testing.T) {
	fake := llm.NewFake()
	fake.Responses["P1.LOC.001"] = "" // empty response

	base := &stubFixer{ruleID: "P1.LOC.001", content: "# Static content\n"}
	var stderr bytes.Buffer
	fixer := llmfix.NewLLMFixer(base, fake, &stderr)

	facts := &fakeFactStore{repo: model.RepoFacts{Root: "/tmp/test"}}
	changes, err := fixer.Plan(context.Background(), model.Finding{RuleID: "P1.LOC.001"}, facts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(changes[0].Content), "Static content") {
		t.Errorf("should keep static on empty LLM response: %q", string(changes[0].Content))
	}
}
