// Package core is the scheduler. It runs collectors, builds a FactStore, and
// evaluates the rule engine. Resolvers never reach through FactStore into
// collectors — this package is the only seam that knows both sides.
package core

import (
	"context"

	"github.com/shibuiwilliam/archfit/internal/adapter/exec"
	collectfs "github.com/shibuiwilliam/archfit/internal/collector/fs"
	collectgit "github.com/shibuiwilliam/archfit/internal/collector/git"
	collectschema "github.com/shibuiwilliam/archfit/internal/collector/schema"
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/rule"
	"github.com/shibuiwilliam/archfit/internal/score"
)

// ScanInput is what a caller (typically the CLI) provides.
type ScanInput struct {
	Root   string
	Rules  []model.Rule
	Runner exec.Runner // if nil, git facts are silently unavailable (shallow mode).
}

// ScanResult is what the CLI formats into the chosen renderer.
type ScanResult struct {
	Root           string
	Findings       []model.Finding
	Metrics        []model.Metric
	RulesEvaluated int
	Scores         score.Scores
	Git            model.GitFacts
	GitAvailable   bool
	Errors         []rule.RuleError
}

func Scan(ctx context.Context, in ScanInput) (ScanResult, error) {
	repo, err := collectfs.Collect(in.Root)
	if err != nil {
		return ScanResult{}, err
	}
	var gitFacts model.GitFacts
	gitOK := false
	if in.Runner != nil {
		g, gerr := collectgit.Collect(ctx, in.Runner, in.Root)
		if gerr == nil {
			gitFacts = g
			gitOK = true
		}
	}

	schemas := collectschema.Collect(repo)

	facts := newFactStore(repo, gitFacts, gitOK, schemas)
	ev := rule.NewEngine().Evaluate(ctx, in.Rules, facts)
	sc := score.Compute(in.Rules, ev.Findings)

	return ScanResult{
		Root:           in.Root,
		Findings:       ev.Findings,
		Metrics:        ev.Metrics,
		RulesEvaluated: ev.RulesEvaluated,
		Scores:         sc,
		Git:            gitFacts,
		GitAvailable:   gitOK,
		Errors:         ev.Errors,
	}, nil
}

// factStore is the read-only view exposed to resolvers.
type factStore struct {
	repo    model.RepoFacts
	git     model.GitFacts
	gitOK   bool
	schemas model.SchemaFacts
}

func newFactStore(repo model.RepoFacts, git model.GitFacts, gitOK bool, schemas model.SchemaFacts) model.FactStore {
	return &factStore{repo: repo, git: git, gitOK: gitOK, schemas: schemas}
}

func (f *factStore) Repo() model.RepoFacts       { return f.repo }
func (f *factStore) Git() (model.GitFacts, bool) { return f.git, f.gitOK }
func (f *factStore) Schemas() model.SchemaFacts  { return f.schemas }
