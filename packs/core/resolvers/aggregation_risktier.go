package resolvers

import (
	"context"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// riskTierCandidates are files that declare risk tiers.
var riskTierCandidates = []string{
	"RISK_TIERS.md",
	"risk_tiers.md",
	"SECURITY.md",
	"security.md",
}

// AggP5AGG003 fires when the repository has no risk-tier declaration.
// A risk-tier file (RISK_TIERS.md, SECURITY.md, or .archfit.yaml with
// risk_tiers) tells agents which paths are dangerous.
func AggP5AGG003(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	// Check for dedicated risk-tier files.
	for _, name := range riskTierCandidates {
		if _, ok := repo.ByPath[name]; ok {
			return nil, nil, nil
		}
	}

	// Check for .archfit.yaml containing risk_tiers (the string "risk_tiers"
	// in the config file). We don't parse the config here — that would require
	// importing internal/config, which packs cannot do. Instead, check if
	// the config file exists and contains the keyword via file content.
	// Since we only have file metadata in FactStore (not content), we check
	// for the config file's presence as a minimal signal.
	// Future: when config is exposed in FactStore, check risk_tiers field.
	for _, name := range []string{".archfit.yaml", ".archfit.yml"} {
		if _, ok := repo.ByPath[name]; ok {
			// Config exists — can't check content from FactStore, so give
			// benefit of the doubt. The rule fires only when NO risk-tier
			// declaration exists at all.
			return nil, nil, nil
		}
	}

	return []model.Finding{
		{
			Confidence: 0.90,
			Message:    "no risk-tier declaration found — agents cannot distinguish high-risk paths from safe ones",
			Evidence: map[string]any{
				"looked_for": append(riskTierCandidates, ".archfit.yaml"),
			},
		},
	}, nil, nil
}
