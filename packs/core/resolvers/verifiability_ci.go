package resolvers

import (
	"context"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// VerP4VER003 fires when the repo has source code but no CI configuration.
// Uses the centralized ecosystem collector (ADR 0011) instead of private
// keyword tables.
func VerP4VER003(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	// Only fire if the repo has source files (not a docs-only repo).
	if len(repo.Languages) == 0 {
		return nil, nil, nil
	}

	if facts.Ecosystems().HasCI() {
		return nil, nil, nil
	}

	return []model.Finding{{
		Message:    "no CI configuration detected — the repository is locally verifiable but not continuously verified",
		Confidence: 0.90,
		Evidence: map[string]any{
			"checked_for": "CI platforms (GitHub Actions, GitLab CI, CircleCI, etc.)",
		},
	}}, nil, nil
}
