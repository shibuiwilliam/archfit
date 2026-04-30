package resolvers

import (
	"context"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// secretScannerKeywords are substrings searched for in CI config file paths
// to detect secret-scanning tools.
var secretScannerKeywords = []string{
	"gitleaks",
	"trufflehog",
	"detect-secrets",
	"secretlint",
	"talisman",
	"git-secrets",
}

// AggP5AGG002 fires when the repo has CI configuration but no secret scanner
// is detected. Uses the centralized ecosystem collector for CI file detection
// (ADR 0011) instead of a private keyword table.
func AggP5AGG002(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	eco := facts.Ecosystems()
	if !eco.HasCI() {
		return nil, nil, nil // no CI → rule does not apply
	}

	ciFiles := eco.CIFiles()

	// Check if any CI file path contains a scanner keyword.
	for _, ciPath := range ciFiles {
		lower := strings.ToLower(ciPath)
		for _, kw := range secretScannerKeywords {
			if strings.Contains(lower, kw) {
				return nil, nil, nil // scanner detected
			}
		}
	}

	return []model.Finding{{
		Message:    "CI configuration detected but no secret scanner (gitleaks, trufflehog, detect-secrets, etc.) is wired — leaked credentials are the highest-impact aggregation failure",
		Confidence: 0.85,
		Evidence: map[string]any{
			"ci_files":            truncateSlice(ciFiles, 10),
			"looked_for_scanners": secretScannerKeywords,
		},
	}}, nil, nil
}
