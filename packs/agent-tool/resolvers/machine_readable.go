package resolvers

import (
	"context"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// P7MRD002 fires when the repo has no CHANGELOG.md at the root.
// Case-insensitive match for common variants (Changelog.md, changelog.md, CHANGES.md).
func P7MRD002(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()
	for _, name := range []string{"CHANGELOG.md", "Changelog.md", "changelog.md", "CHANGES.md"} {
		if _, ok := repo.ByPath[name]; ok {
			return nil, nil, nil
		}
	}
	return []model.Finding{{
		Message:    "no CHANGELOG.md at repo root — agents cannot diff tool versions machine-readably",
		Confidence: 0.98,
		Evidence: map[string]any{
			"looked_for": []string{"CHANGELOG.md", "Changelog.md", "changelog.md", "CHANGES.md"},
		},
	}}, nil, nil
}

// P7MRD003 fires when the repo ships a CLI (has cmd/ source) but no docs/adr/ directory.
func P7MRD003(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()
	if !hasCLIEntrypoint(repo) {
		return nil, nil, nil
	}
	if hasADRDir(repo) {
		return nil, nil, nil
	}
	return []model.Finding{{
		Path:       "docs/adr/",
		Message:    "CLI entrypoint detected but no docs/adr/ directory — irreversible design decisions are not recorded for agents",
		Confidence: 0.95,
		Evidence: map[string]any{
			"looked_for": "docs/adr/",
		},
	}}, nil, nil
}

func hasCLIEntrypoint(repo model.RepoFacts) bool {
	for _, f := range repo.Files {
		if strings.HasPrefix(f.Path, "cmd/") && hasSourceExt(f.Path) {
			return true
		}
	}
	return false
}

func hasSourceExt(p string) bool {
	return strings.HasSuffix(p, ".go") ||
		strings.HasSuffix(p, ".py") ||
		strings.HasSuffix(p, ".ts") ||
		strings.HasSuffix(p, ".rs") ||
		strings.HasSuffix(p, ".js")
}

func hasADRDir(repo model.RepoFacts) bool {
	for _, f := range repo.Files {
		if strings.HasPrefix(f.Path, "docs/adr/") {
			return true
		}
	}
	return false
}
