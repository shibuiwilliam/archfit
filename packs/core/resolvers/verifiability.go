package resolvers

import (
	"context"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// verificationEntrypoints are the root-level files archfit recognizes as a
// declared fast-verification entrypoint. Keep this list explicit; adding a new
// one is a deliberate policy choice, not a silent expansion.
var verificationEntrypoints = []string{
	"Makefile",
	"makefile",
	"justfile",
	"Justfile",
	"Taskfile.yml",
	"Taskfile.yaml",
	"package.json",
	"pyproject.toml",
	"Cargo.toml",
	"go.mod", // `go test ./...` is a universal fallback.
}

// VerP4VER001 fires when the repo has no recognized fast-verification entrypoint at its root.
func VerP4VER001(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()
	for _, name := range verificationEntrypoints {
		if _, ok := repo.ByPath[name]; ok {
			return nil, nil, nil
		}
	}
	return []model.Finding{{
		Message:    "no verification entrypoint (Makefile, justfile, Taskfile, package.json, pyproject.toml, Cargo.toml, go.mod) at repo root",
		Confidence: 0.98,
		Evidence: map[string]any{
			"looked_for": verificationEntrypoints,
		},
	}}, nil, nil
}
