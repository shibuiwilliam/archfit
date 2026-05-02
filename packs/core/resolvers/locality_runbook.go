package resolvers

import (
	"context"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// runbookCandidates are the paths archfit accepts as a runbook for P1.LOC.009.
// This is narrower than rollbackDocCandidates in reversibility.go — it
// specifically requires a RUNBOOK.md, not just any deployment doc.
var runbookCandidates = []string{
	"RUNBOOK.md",
	"runbook.md",
	"docs/RUNBOOK.md",
	"docs/runbook.md",
}

// LocP1LOC009 fires when the repo has deployment artifacts but no RUNBOOK.md.
// Reuses hasDeploymentArtifacts from reversibility.go.
func LocP1LOC009(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	if !hasDeploymentArtifacts(repo) {
		return nil, nil, nil
	}

	for _, candidate := range runbookCandidates {
		if _, ok := repo.ByPath[candidate]; ok {
			return nil, nil, nil
		}
	}

	// Find the first deployment artifact to cite as evidence.
	matchedArtifact := findFirstDeployArtifact(repo)

	return []model.Finding{{
		Confidence: 0.90,
		Path:       "",
		Message:    "repository has deployment artifacts but no RUNBOOK.md",
		Evidence: map[string]any{
			"deployment_artifact": matchedArtifact,
			"checked_paths":       runbookCandidates,
		},
	}}, nil, nil
}

// findFirstDeployArtifact returns the path of the first deployment artifact found.
func findFirstDeployArtifact(repo model.RepoFacts) string {
	for _, name := range deploymentArtifacts {
		if _, ok := repo.ByPath[name]; ok {
			return name
		}
	}
	for _, f := range repo.Files {
		for _, prefix := range deploymentDirPrefixes {
			if len(f.Path) > len(prefix) && f.Path[:len(prefix)] == prefix {
				return f.Path
			}
		}
	}
	return ""
}
