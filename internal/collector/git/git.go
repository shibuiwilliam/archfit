// Package git collects git facts by shelling out through internal/adapter/exec.
//
// It is intentionally lightweight: commit count, recent subjects, current ref.
// Anything that needs PR-size distribution or churn lives in Phase 2 behind
// another collector that takes this one's output as input.
package git

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/adapter/exec"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// MaxRecentCommits bounds the slice returned; larger repos are sampled, not scanned.
const MaxRecentCommits = 50

// ErrNoGit is returned when the target is not a git working tree. Collectors
// surface this as "no git facts available" rather than a fatal error.
var ErrNoGit = errors.New("git: not a working tree")

func Collect(ctx context.Context, runner exec.Runner, root string) (model.GitFacts, error) {
	if _, err := runner.Run(ctx, root, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
		return model.GitFacts{}, ErrNoGit
	}

	var facts model.GitFacts

	if r, err := runner.Run(ctx, root, "git", "rev-parse", "HEAD"); err == nil && r.ExitCode == 0 {
		facts.CurrentCommit = strings.TrimSpace(string(r.Stdout))
	}
	if r, err := runner.Run(ctx, root, "git", "rev-parse", "--abbrev-ref", "HEAD"); err == nil && r.ExitCode == 0 {
		facts.CurrentBranch = strings.TrimSpace(string(r.Stdout))
	}
	if r, err := runner.Run(ctx, root, "git", "rev-list", "--count", "HEAD"); err == nil && r.ExitCode == 0 {
		if n, perr := strconv.Atoi(strings.TrimSpace(string(r.Stdout))); perr == nil {
			facts.CommitCount = n
		}
	}

	// %H<TAB>%s — single-line per commit, tab-separated. Stable, machine-readable.
	r, err := runner.Run(ctx, root, "git", "log",
		"--max-count="+strconv.Itoa(MaxRecentCommits),
		"--pretty=format:%H\t%s")
	if err == nil && r.ExitCode == 0 {
		for _, line := range strings.Split(strings.TrimRight(string(r.Stdout), "\n"), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) != 2 {
				continue
			}
			facts.RecentCommits = append(facts.RecentCommits, model.Commit{
				Hash:    parts[0],
				Subject: parts[1],
			})
		}
	}

	return facts, nil
}
