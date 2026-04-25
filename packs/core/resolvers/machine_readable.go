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
		Message:    "CLI entrypoint detected (cmd/, bin/, or exe/) but exit codes are not documented",
		Confidence: 0.95,
		Evidence: map[string]any{
			"looked_for": exitCodeDocCandidates,
		},
	}}, nil, nil
}

// cliDirPrefixes are directory prefixes that conventionally hold CLI entrypoints.
var cliDirPrefixes = []string{"cmd/", "bin/", "exe/"}

// cliSourceExts are file extensions recognized as CLI source files.
var cliSourceExts = []string{
	".go", ".py", ".ts", ".js", ".rs",
	".rb", ".java", ".kt", ".swift", ".php", ".sh",
}

// cliIndicatorBasenames are file basenames (without path) that strongly
// indicate the repo ships a CLI, even without a cmd/bin/exe directory.
// These catch Python CLIs (__main__.py), and explicitly named CLI modules.
var cliIndicatorBasenames = []string{
	"__main__.py", // Python CLI convention (python -m pkg)
	"cli.go", "cli.py", "cli.ts", "cli.js", "cli.rb",
}

func hasCLIEntrypoint(repo model.RepoFacts) bool {
	// Check conventional CLI directories.
	for _, f := range repo.Files {
		for _, prefix := range cliDirPrefixes {
			if strings.HasPrefix(f.Path, prefix) {
				for _, ext := range cliSourceExts {
					if strings.HasSuffix(f.Path, ext) {
						return true
					}
				}
			}
		}
	}
	// Check for CLI indicator basenames at any depth.
	for _, f := range repo.Files {
		base := f.Path
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		for _, indicator := range cliIndicatorBasenames {
			if base == indicator {
				return true
			}
		}
	}
	return false
}
