package resolvers

import (
	"context"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// featureFlagKeywords are substrings matched against file paths (case-insensitive)
// to detect feature-flag libraries and local flag implementations.
var featureFlagKeywords = []string{
	// Managed services
	"launchdarkly",
	"unleash",
	"openfeature",
	"configcat",
	"growthbook",
	"optimizely",
	"gofeatureflag",
	"go-feature-flag",
	"flipt",
	"flagsmith",
	"split.io",
	"splitio",
	// Local/generic patterns
	"featureflag",
	"feature-flag",
	"feature_flag",
	"featureflags",
	"feature-flags",
	"feature_flags",
	"feature-toggle",
	"feature_toggle",
}

// RevP6REV002 fires when the repo deploys (has deployment artifacts) but
// shows no evidence of a feature-flag mechanism. Info-severity because
// feature flags are aspirational — not every service needs them — but their
// absence in deploying repos is a reversibility signal worth surfacing.
func RevP6REV002(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	if !hasDeploymentArtifacts(repo) {
		return nil, nil, nil // not a deploying repo
	}

	// Search all file paths for feature-flag indicators.
	for _, f := range repo.Files {
		if isFixtureOrTestdata(f.Path) {
			continue
		}
		lower := strings.ToLower(f.Path)
		for _, kw := range featureFlagKeywords {
			if strings.Contains(lower, kw) {
				return nil, nil, nil // flag mechanism detected
			}
		}
	}

	return []model.Finding{{
		Message:    "repository deploys but shows no feature-flag mechanism — changes cannot be toggled off without a full rollback",
		Confidence: 0.75,
		Evidence: map[string]any{
			"looked_for": featureFlagKeywords[:7], // top-level names only for readability
		},
	}}, nil, nil
}
