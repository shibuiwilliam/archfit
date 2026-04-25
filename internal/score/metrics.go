// Package score — metric computation helpers.
//
// These functions derive archfit metrics from collected facts. Each returns a
// single model.Metric. They are called from the scheduler (internal/core) after
// fact collection completes.
package score

import (
	"math"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// ContextSpanP50 computes the median number of files touched per commit.
func ContextSpanP50(git model.GitFacts) model.Metric {
	if len(git.RecentCommits) == 0 {
		return model.Metric{Name: "context_span_p50", Value: 0, Unit: "files", Principle: "P1"}
	}
	var counts []int
	for _, c := range git.RecentCommits {
		if c.FilesChanged > 0 {
			counts = append(counts, c.FilesChanged)
		}
	}
	if len(counts) == 0 {
		return model.Metric{Name: "context_span_p50", Value: 0, Unit: "files", Principle: "P1"}
	}
	sortInts(counts)
	p50 := float64(counts[len(counts)/2])
	return model.Metric{Name: "context_span_p50", Value: p50, Unit: "files", Principle: "P1"}
}

// VerificationLatency computes the wall-clock test execution time in seconds.
// When multiple commands were run, it reports the maximum duration.
func VerificationLatency(cmds model.CommandFacts) model.Metric {
	if len(cmds.Results) == 0 {
		return model.Metric{Name: "verification_latency_s", Value: 0, Unit: "seconds", Principle: "P4"}
	}
	var maxMS int64
	for _, r := range cmds.Results {
		if r.DurationMS > maxMS {
			maxMS = r.DurationMS
		}
	}
	secs := math.Round(float64(maxMS)/100) / 10 // ms to s, 1 decimal
	return model.Metric{Name: "verification_latency_s", Value: secs, Unit: "seconds", Principle: "P4"}
}

// InvariantCoverage estimates the fraction of stated invariants that are
// machine-enforced. Approximated as 1 - (rules with error+ findings) / (total rules).
func InvariantCoverage(findings []model.Finding, rules []model.Rule) model.Metric {
	if len(rules) == 0 {
		return model.Metric{Name: "invariant_coverage", Value: 1, Unit: "ratio", Principle: "P4"}
	}
	violated := map[string]bool{}
	for _, f := range findings {
		if f.Severity.Rank() >= model.SeverityError.Rank() {
			violated[f.RuleID] = true
		}
	}
	coverage := 1.0 - float64(len(violated))/float64(len(rules))
	if coverage < 0 {
		coverage = 0
	}
	return model.Metric{Name: "invariant_coverage", Value: math.Round(coverage*1000) / 1000, Unit: "ratio", Principle: "P4"}
}

// ParallelConflictRate estimates merge conflict frequency from git history.
// Heuristic: fraction of recent commits that are merge commits (subjects
// starting with "Merge").
func ParallelConflictRate(git model.GitFacts) model.Metric {
	if len(git.RecentCommits) == 0 {
		return model.Metric{Name: "parallel_conflict_rate", Value: 0, Unit: "ratio", Principle: "P1"}
	}
	merges := 0
	for _, c := range git.RecentCommits {
		if strings.HasPrefix(c.Subject, "Merge") {
			merges++
		}
	}
	rate := float64(merges) / float64(len(git.RecentCommits))
	return model.Metric{Name: "parallel_conflict_rate", Value: math.Round(rate*1000) / 1000, Unit: "ratio", Principle: "P1"}
}

// RollbackSignal computes revert-commit frequency.
// Heuristic: fraction of recent commits whose subject starts with "Revert".
func RollbackSignal(git model.GitFacts) model.Metric {
	if len(git.RecentCommits) == 0 {
		return model.Metric{Name: "rollback_signal", Value: 0, Unit: "ratio", Principle: "P6"}
	}
	reverts := 0
	for _, c := range git.RecentCommits {
		if strings.HasPrefix(c.Subject, "Revert") {
			reverts++
		}
	}
	rate := float64(reverts) / float64(len(git.RecentCommits))
	return model.Metric{Name: "rollback_signal", Value: math.Round(rate*1000) / 1000, Unit: "ratio", Principle: "P6"}
}

// BlastRadius computes the max transitive reach normalized by total packages.
// A value of 1.0 means one package can transitively reach every other package.
func BlastRadius(depGraph model.DepGraphFacts) model.Metric {
	if depGraph.PackageCount <= 1 {
		return model.Metric{Name: "blast_radius_score", Value: 0, Unit: "ratio", Principle: "P5"}
	}
	ratio := float64(depGraph.MaxReach) / float64(depGraph.PackageCount-1)
	if ratio > 1 {
		ratio = 1
	}
	return model.Metric{Name: "blast_radius_score", Value: math.Round(ratio*1000) / 1000, Unit: "ratio", Principle: "P5"}
}

// sortInts sorts a slice of ints in ascending order (avoids importing sort for a trivial case).
func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
