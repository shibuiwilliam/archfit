package resolvers

import (
	"context"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// exitCodeDocCandidates are the paths archfit accepts as "exit codes documented".
// Repos that document exit codes inside a catch-all reference (e.g. docs/cli.md
// with a section on exit codes) are not detected in Phase 1 — that heuristic is
// too weak for `strong` evidence.
var exitCodeDocCandidates = []string{
	"docs/exit-codes.md",
	"docs/EXIT_CODES.md",
	"docs/exit_codes.md",
	"EXIT_CODES.md",
}

// MrdP7MRD001 fires when the repo ships a CLI entrypoint (cmd/ or bin/ directory
// with source files) but does not document exit codes at one of the known paths.
func MrdP7MRD001(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()
	if !hasCLIEntrypoint(repo) {
		return nil, nil, nil
	}
	for _, c := range exitCodeDocCandidates {
		if _, ok := repo.ByPath[c]; ok {
			return nil, nil, nil
		}
	}
	return []model.Finding{{
		Path:       "docs/",
		Message:    "CLI entrypoint detected (cmd/ or bin/) but exit codes are not documented",
		Confidence: 0.95,
		Evidence: map[string]any{
			"looked_for": exitCodeDocCandidates,
		},
	}}, nil, nil
}

func hasCLIEntrypoint(repo model.RepoFacts) bool {
	for _, f := range repo.Files {
		if strings.HasPrefix(f.Path, "cmd/") || strings.HasPrefix(f.Path, "bin/") {
			if strings.HasSuffix(f.Path, ".go") || strings.HasSuffix(f.Path, ".py") || strings.HasSuffix(f.Path, ".ts") || strings.HasSuffix(f.Path, ".rs") {
				return true
			}
		}
	}
	return false
}
