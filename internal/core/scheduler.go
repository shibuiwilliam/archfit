// Package core is the scheduler. It runs collectors, builds a FactStore, and
// evaluates the rule engine. Resolvers never reach through FactStore into
// collectors — this package is the only seam that knows both sides.
package core

import (
	"context"
	"time"

	"github.com/shibuiwilliam/archfit/internal/adapter/exec"
	collectcmd "github.com/shibuiwilliam/archfit/internal/collector/command"
	collectdep "github.com/shibuiwilliam/archfit/internal/collector/depgraph"
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
	Depth  string      // "shallow", "standard", "deep"; default "standard"
}

// ScanResult is what the CLI formats into the chosen renderer.
type ScanResult struct {
	Root              string
	Findings          []model.Finding
	Metrics           []model.Metric
	RulesEvaluated    int
	RulesWithFindings int // how many rules produced ≥1 finding
	Scores            score.Scores
	Git               model.GitFacts
	GitAvailable      bool
	Errors            []rule.EvalError
}

// Scan collects facts and evaluates rules, returning the aggregated result.
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

	// Dependency graph: lightweight, always runs for parseable source.
	var depFacts model.DepGraphFacts
	depOK := false
	graph, gerr := collectdep.Collect(repo)
	if gerr == nil && graph.PackageCount() > 0 {
		maxPkg, maxReach := graph.MaxReach()
		depFacts = model.DepGraphFacts{
			PackageCount: graph.PackageCount(),
			MaxReach:     maxReach,
			MaxReachPkg:  maxPkg,
		}
		depOK = true
	}

	// Command timing: expensive, only at --depth=deep.
	var cmdFacts model.CommandFacts
	cmdOK := false
	if in.Depth == "deep" && in.Runner != nil {
		results := collectcmd.Collect(ctx, in.Runner, in.Root, 120*time.Second)
		if len(results) > 0 {
			for _, r := range results {
				cmdFacts.Results = append(cmdFacts.Results, model.CommandResult{
					Command:    r.Command,
					DurationMS: r.DurationMS,
					ExitCode:   r.ExitCode,
				})
			}
			cmdOK = true
		}
	}

	facts := newFactStore(repo, gitFacts, gitOK, schemas, cmdFacts, cmdOK, depFacts, depOK)
	ev := rule.NewEngine().Evaluate(ctx, in.Rules, facts)

	// Compute metrics from collected facts.
	var metrics []model.Metric
	metrics = append(metrics, ev.Metrics...)
	if gitOK {
		metrics = append(metrics,
			score.ContextSpanP50(gitFacts),
			score.ParallelConflictRate(gitFacts),
			score.RollbackSignal(gitFacts),
		)
	}
	if cmdOK {
		metrics = append(metrics, score.VerificationLatency(cmdFacts))
	}
	if depOK {
		metrics = append(metrics, score.BlastRadius(depFacts))
	}
	metrics = append(metrics, score.InvariantCoverage(ev.Findings, in.Rules))

	sc := score.Compute(in.Rules, ev.Findings)

	return ScanResult{
		Root:              in.Root,
		Findings:          ev.Findings,
		Metrics:           metrics,
		RulesEvaluated:    ev.RulesEvaluated,
		RulesWithFindings: ev.RulesWithFindings,
		Scores:            sc,
		Git:               gitFacts,
		GitAvailable:      gitOK,
		Errors:            ev.Errors,
	}, nil
}

// factStore is the read-only view exposed to resolvers.
type factStore struct {
	repo    model.RepoFacts
	git     model.GitFacts
	gitOK   bool
	schemas model.SchemaFacts
	cmds    model.CommandFacts
	cmdsOK  bool
	dep     model.DepGraphFacts
	depOK   bool
}

func newFactStore(repo model.RepoFacts, git model.GitFacts, gitOK bool, schemas model.SchemaFacts, cmds model.CommandFacts, cmdsOK bool, dep model.DepGraphFacts, depOK bool) model.FactStore {
	return &factStore{repo: repo, git: git, gitOK: gitOK, schemas: schemas, cmds: cmds, cmdsOK: cmdsOK, dep: dep, depOK: depOK}
}

// Repo returns the collected repo-wide facts.
func (f *factStore) Repo() model.RepoFacts { return f.repo }

// Git returns git facts and whether they are available.
func (f *factStore) Git() (model.GitFacts, bool) { return f.git, f.gitOK }

// Schemas returns the collected schema facts.
func (f *factStore) Schemas() model.SchemaFacts { return f.schemas }

// Commands returns command timing facts and whether they are available.
func (f *factStore) Commands() (model.CommandFacts, bool) { return f.cmds, f.cmdsOK }

// DepGraph returns dependency graph facts and whether they are available.
func (f *factStore) DepGraph() (model.DepGraphFacts, bool) { return f.dep, f.depOK }
