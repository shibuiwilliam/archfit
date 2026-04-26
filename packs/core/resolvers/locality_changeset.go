package resolvers

import (
	"context"
	"fmt"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// changeSetThreshold is the median files-per-commit above which a finding is
// emitted. Repos where typical commits touch >8 files force agents to hold a
// wide context for routine changes.
const changeSetThreshold = 8

// LocP1LOC004 fires when the median number of files changed per commit exceeds
// the threshold, indicating poor change isolation. Silently skips when git
// facts are unavailable or the sample has no commits with file-change data.
func LocP1LOC004(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	git, ok := facts.Git()
	if !ok || len(git.RecentCommits) == 0 {
		return nil, nil, nil
	}

	span := medianFilesChanged(git.RecentCommits)
	if span == 0 {
		return nil, nil, nil // no file-change data in sampled commits
	}

	if span <= changeSetThreshold {
		return nil, nil, nil
	}

	return []model.Finding{{
		Message: fmt.Sprintf(
			"typical commits touch %.0f files (threshold %d) — agents must hold wide context for routine changes",
			span, changeSetThreshold),
		Confidence: 0.75,
		Evidence: map[string]any{
			"context_span_p50": span,
			"threshold":        changeSetThreshold,
			"sample_size":      len(git.RecentCommits),
		},
	}}, nil, nil
}

// medianFilesChanged computes the median FilesChanged across commits that have
// this data (FilesChanged > 0). Returns 0 when no commits carry the data.
func medianFilesChanged(commits []model.Commit) float64 {
	var counts []int
	for _, c := range commits {
		if c.FilesChanged > 0 {
			counts = append(counts, c.FilesChanged)
		}
	}
	if len(counts) == 0 {
		return 0
	}
	// Insertion sort — commit sample is small (<100).
	for i := 1; i < len(counts); i++ {
		for j := i; j > 0 && counts[j] < counts[j-1]; j-- {
			counts[j], counts[j-1] = counts[j-1], counts[j]
		}
	}
	return float64(counts[len(counts)/2])
}
