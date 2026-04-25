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
		Message:    "CLI entrypoint detected (cmd/, bin/, or exe/) but no docs/adr/ directory — irreversible design decisions are not recorded for agents",
		Confidence: 0.95,
		Evidence: map[string]any{
			"looked_for": "docs/adr/",
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

// cliIndicatorBasenames are file basenames that strongly indicate the repo
// ships a CLI, even without a cmd/bin/exe directory.
var cliIndicatorBasenames = []string{
	"__main__.py",
	"cli.go", "cli.py", "cli.ts", "cli.js", "cli.rb",
}

func hasCLIEntrypoint(repo model.RepoFacts) bool {
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

func hasSourceExt(p string) bool {
	for _, ext := range cliSourceExts {
		if strings.HasSuffix(p, ext) {
			return true
		}
	}
	return false
}

func hasADRDir(repo model.RepoFacts) bool {
	for _, f := range repo.Files {
		if strings.HasPrefix(f.Path, "docs/adr/") {
			return true
		}
	}
	return false
}
