package resolvers

import (
	"context"
	"fmt"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// maxReachThreshold is the coupling score above which a finding is emitted.
// A package that transitively reaches more than 10 other packages is
// "context-expensive" — agents touching it must load many neighbors.
const maxReachThreshold = 10

// cliEntryPrefixes are package path prefixes that represent CLI entry points.
// These packages are wiring layers by design — they import everything to
// assemble the application. High transitive reach is expected and not a signal
// of poor locality.
var cliEntryPrefixes = []string{"cmd/", "."}

// isEntryPointPkg returns true if the package is a CLI entry point whose high
// reach is structural, not a coupling problem.
func isEntryPointPkg(pkg string) bool {
	for _, prefix := range cliEntryPrefixes {
		if pkg == prefix || strings.HasPrefix(pkg, prefix) {
			return true
		}
	}
	return false
}

// LocP1LOC003 fires when the dependency graph contains a non-entrypoint package
// with high transitive reach, indicating tight coupling. Agents working on such
// packages must hold a wide context window. CLI entry points (cmd/) are excluded
// because they are wiring layers by design. Silently skips when the depgraph
// collector did not run or found no parseable source.
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

	// If the highest-reach package is a CLI entry point, it's expected.
	// Check whether any non-entrypoint package exceeds the threshold.
	if isEntryPointPkg(dep.MaxReachPkg) {
		// Find the highest-reach non-entrypoint package from the graph.
		// The DepGraphFacts only stores the single max — if that's an
		// entry point, we cannot identify the second-highest here, so
		// we skip. A future improvement could store top-N in DepGraphFacts.
		return nil, metrics, nil
	}

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
