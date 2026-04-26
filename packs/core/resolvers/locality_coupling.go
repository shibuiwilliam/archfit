package resolvers

import (
	"context"
	"fmt"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// maxReachThreshold is the coupling score above which a finding is emitted.
// A package that transitively reaches more than 10 other packages is
// "context-expensive" — agents touching it must load many neighbors.
const maxReachThreshold = 10

// LocP1LOC003 fires when the dependency graph contains a package with high
// transitive reach, indicating tight coupling. Agents working on such packages
// must hold a wide context window. Silently skips when the depgraph collector
// did not run or found no parseable source.
func LocP1LOC003(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	dep, ok := facts.DepGraph()
	if !ok || dep.PackageCount == 0 {
		return nil, nil, nil
	}

	var metrics []model.Metric
	metrics = append(metrics, model.Metric{
		Name:      "coupling_max_reach",
		Value:     float64(dep.MaxReach),
		Unit:      "packages",
		Principle: "P1",
	})

	if dep.MaxReach <= maxReachThreshold {
		return nil, metrics, nil
	}

	return []model.Finding{{
		Message: fmt.Sprintf(
			"package %q has transitive reach of %d (threshold %d) — agents must load many neighbors to understand changes here",
			dep.MaxReachPkg, dep.MaxReach, maxReachThreshold),
		Confidence: 0.75,
		Evidence: map[string]any{
			"max_reach_pkg": dep.MaxReachPkg,
			"max_reach":     dep.MaxReach,
			"package_count": dep.PackageCount,
			"threshold":     maxReachThreshold,
		},
	}}, metrics, nil
}
