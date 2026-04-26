package resolvers

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// testSuffixes maps source extensions to the test file suffixes that indicate
// the directory has test coverage. Only the filename suffix is checked (not
// the full path), so "app_test.go" matches "_test.go" for .go sources.
var testSuffixes = map[string][]string{
	".go":   {"_test.go"},
	".py":   {"_test.py", "test_"},
	".ts":   {".test.ts", ".spec.ts"},
	".tsx":  {".test.tsx", ".spec.tsx"},
	".js":   {".test.js", ".spec.js"},
	".jsx":  {".test.jsx", ".spec.jsx"},
	".rb":   {"_spec.rb", "_test.rb"},
	".java": {"Test", "Tests"},
}

// minSourceDirsForRule is the minimum number of source directories before
// P4.VER.002 fires. Repos with very few directories are too small to benefit.
const minSourceDirsForRule = 3

// untestedDirThreshold is the fraction of source directories that must have
// tests. Directories without any test files below this threshold trigger a finding.
const untestedDirThreshold = 0.7

// VerP4VER002 fires when more than 30% of source directories have no test
// files. This measures verification depth beyond just "does a Makefile exist."
func VerP4VER002(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	// Group source files by directory, tracking which have tests.
	dirHasSource := map[string]bool{}
	dirHasTest := map[string]bool{}

	for _, f := range repo.Files {
		if isFixtureOrTestdata(f.Path) {
			continue
		}
		ext := f.Ext
		patterns, isSource := testSuffixes[ext]
		if !isSource {
			continue
		}
		dir := path.Dir(f.Path)
		dirHasSource[dir] = true

		base := fileBase(f.Path)
		for _, pat := range patterns {
			if strings.HasSuffix(base, pat) || strings.HasPrefix(base, pat) {
				dirHasTest[dir] = true
				break
			}
		}
	}

	totalDirs := len(dirHasSource)
	if totalDirs < minSourceDirsForRule {
		return nil, nil, nil
	}

	testedDirs := len(dirHasTest)
	ratio := float64(testedDirs) / float64(totalDirs)

	if ratio >= untestedDirThreshold {
		return nil, nil, nil
	}

	// Collect untested directories for evidence.
	var untested []string
	for dir := range dirHasSource {
		if !dirHasTest[dir] {
			untested = append(untested, dir)
		}
	}
	sort.Strings(untested)

	return []model.Finding{{
		Message: fmt.Sprintf(
			"%.0f%% of source directories have test files (%d/%d) — agents cannot verify changes in untested areas",
			ratio*100, testedDirs, totalDirs),
		Confidence: 0.80,
		Evidence: map[string]any{
			"tested_dirs":   testedDirs,
			"total_dirs":    totalDirs,
			"ratio":         ratio,
			"threshold":     untestedDirThreshold,
			"untested_dirs": truncateSlice(untested, 10),
		},
	}}, nil, nil
}
