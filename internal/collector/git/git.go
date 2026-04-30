// Package git collects git facts by shelling out through internal/adapter/exec.
//
// It is intentionally lightweight: commit count, recent subjects, current ref.
// Anything that needs PR-size distribution or churn lives in Phase 2 behind
// another collector that takes this one's output as input.
package git

import (
	"context"
	"errors"
	"path/filepath"
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

// Collect gathers git facts from the working tree at root using runner.
// If root is a subdirectory of a different git repo (e.g., a fixture inside
// the archfit repo), git facts are treated as unavailable — the enclosing
// repo's history is not relevant to the scan target.
func Collect(ctx context.Context, runner exec.Runner, root string) (model.GitFacts, error) {
	if _, err := runner.Run(ctx, root, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
		return model.GitFacts{}, ErrNoGit
	}

	// Verify the scan root is at or near the git root. If we are deep inside
	// another repo, the git history belongs to that repo, not to the target.
	topR, topErr := runner.Run(ctx, root, "git", "rev-parse", "--show-toplevel")
	if topErr == nil && topR.ExitCode == 0 {
		gitRoot := strings.TrimSpace(string(topR.Stdout))
		absRoot := root
		if !strings.HasPrefix(absRoot, "/") {
			if abs, err := filepath.Abs(absRoot); err == nil {
				absRoot = abs
			}
		}
		// Allow the scan root to be the git root itself, or a shallow child
		// (e.g., monorepo service). Reject deep subdirectories like
		// testdata/e2e/fixture/input/ which are clearly not the project root.
		rel := strings.TrimPrefix(absRoot, gitRoot)
		rel = strings.Trim(rel, "/")
		if rel != "" && strings.Count(rel, "/") >= 3 {
			return model.GitFacts{}, ErrNoGit
		}
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

	// Populate FilesChanged from a separate --numstat invocation.
	// Format: "commit <hash>\n" followed by numstat lines, one per file.
	// Binary files show "-\t-\tpath" and are counted as 1 file.
	populateFilesChanged(ctx, runner, root, facts.RecentCommits)

	return facts, nil
}

// populateFilesChanged issues `git log --numstat` and fills in each commit's
// FilesChanged count. It modifies commits in place. Errors are silently ignored
// — FilesChanged stays 0 (documented as "unknown").
func populateFilesChanged(ctx context.Context, runner exec.Runner, root string, commits []model.Commit) {
	if len(commits) == 0 {
		return
	}

	r, err := runner.Run(ctx, root, "git", "log",
		"--max-count="+strconv.Itoa(len(commits)),
		"--numstat",
		"--pretty=format:commit %H")
	if err != nil || r.ExitCode != 0 {
		return
	}

	parseNumstat(string(r.Stdout), commits)
}

// parseNumstat parses the combined output of `git log --numstat --pretty=format:commit %H`.
// Each commit block looks like:
//
//	commit abc123
//	10	5	file1.go
//	3	1	file2.go
//	-	-	binary.png
//
// Merge commits may have no numstat lines at all. Binary files show "-\t-" for
// added/deleted counts — they are still counted as one file changed.
func parseNumstat(output string, commits []model.Commit) {
	// Build a hash→index map for O(1) lookup.
	byHash := make(map[string]int, len(commits))
	for i, c := range commits {
		byHash[c.Hash] = i
	}

	currentIdx := -1
	fileCount := 0

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")

		if strings.HasPrefix(line, "commit ") {
			// Flush previous commit.
			if currentIdx >= 0 {
				commits[currentIdx].FilesChanged = fileCount
			}
			hash := strings.TrimPrefix(line, "commit ")
			if idx, ok := byHash[hash]; ok {
				currentIdx = idx
			} else {
				currentIdx = -1
			}
			fileCount = 0
			continue
		}

		// Numstat lines: "<added>\t<deleted>\t<path>" or "-\t-\t<path>" for binaries.
		// Empty lines separate commits — skip them.
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "\t", 3)
		if len(parts) == 3 {
			fileCount++
		}
	}

	// Flush last commit.
	if currentIdx >= 0 {
		commits[currentIdx].FilesChanged = fileCount
	}
}
