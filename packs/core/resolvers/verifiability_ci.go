package resolvers

import (
	"context"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// ciConfigFiles are root-level files whose presence indicates CI is configured.
var ciConfigFiles = []string{
	".gitlab-ci.yml",
	".gitlab-ci.yaml",
	"Jenkinsfile",
	".travis.yml",
	"azure-pipelines.yml",
	"bitbucket-pipelines.yml",
	".woodpecker.yml",
}

// ciConfigDirPrefixes are directory prefixes whose presence indicates CI.
var ciConfigDirPrefixes = []string{
	".github/workflows/",
	".circleci/",
	".buildkite/",
}

// VerP4VER003 fires when the repo has source code and a verification entrypoint
// but no CI configuration. A repo without CI is "locally verifiable but not
// continuously verified."
func VerP4VER003(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	// Only fire if the repo has source files (not a docs-only repo).
	if len(repo.Languages) == 0 {
		return nil, nil, nil
	}

	// Check for CI config files at root.
	for _, name := range ciConfigFiles {
		if _, ok := repo.ByPath[name]; ok {
			return nil, nil, nil
		}
	}

	// Check for CI config directory prefixes.
	for _, f := range repo.Files {
		for _, prefix := range ciConfigDirPrefixes {
			if strings.HasPrefix(f.Path, prefix) {
				return nil, nil, nil
			}
		}
	}

	return []model.Finding{{
		Message:    "no CI configuration detected — the repository is locally verifiable but not continuously verified",
		Confidence: 0.90,
		Evidence: map[string]any{
			"checked_files":        ciConfigFiles,
			"checked_dir_prefixes": ciConfigDirPrefixes,
		},
	}}, nil, nil
}
