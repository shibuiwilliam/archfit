package resolvers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// codeownersLocations are the conventional paths for CODEOWNERS files.
var codeownersLocations = []string{
	".github/CODEOWNERS",
	"CODEOWNERS",
	"docs/CODEOWNERS",
}

// highRiskKeywords maps categories to path-segment keywords that signal
// high-risk directories. A directory matches when any segment contains a
// keyword. Reuses the same concept as sensitiveCategories in aggregation.go
// but is intentionally a separate list so the two rules evolve independently.
var highRiskKeywords = map[string][]string{
	"auth":      {"auth", "authentication", "authorization", "login", "oauth", "session", "jwt", "rbac", "acl", "permission"},
	"secret":    {"secret", "credential", "token", "apikey", "encrypt", "decrypt", "cipher", "crypto"},
	"migration": {"migration", "migrate", "schema_change"},
	"deploy":    {"deploy", "infra", "terraform", "cloudformation", "pulumi", "ansible"},
}

// AggP5AGG004 fires when the repository contains high-risk paths (auth,
// secrets, migrations, deploy) but has no CODEOWNERS file, meaning those
// paths lack mandatory review protection.
//
// This is archfit's first error-severity rule (Phase 1, PROJECT.md §6.1.2).
func AggP5AGG004(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	// Step 1: detect high-risk directories.
	highRiskDirs := detectHighRiskDirs(repo)
	if len(highRiskDirs) == 0 {
		// No high-risk paths found — rule does not apply.
		return nil, nil, nil
	}

	// Step 2: check for CODEOWNERS file.
	for _, loc := range codeownersLocations {
		if _, ok := repo.ByPath[loc]; ok {
			// CODEOWNERS exists. For now, its mere presence is sufficient.
			// Future enhancement: parse entries and verify high-risk dirs
			// are actually covered (cross-reference).
			return nil, nil, nil
		}
	}

	// Step 3: high-risk paths exist but no CODEOWNERS.
	categories := make([]string, 0, len(highRiskDirs))
	for cat := range highRiskDirs {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	exampleDirs := make([]string, 0)
	for _, cat := range categories {
		dirs := highRiskDirs[cat]
		sort.Strings(dirs)
		if len(dirs) > 3 {
			dirs = dirs[:3]
		}
		exampleDirs = append(exampleDirs, dirs...)
	}

	return []model.Finding{
		{
			Confidence: 0.95,
			Message: fmt.Sprintf(
				"high-risk paths detected but no CODEOWNERS file — categories: %s",
				strings.Join(categories, ", ")),
			Evidence: map[string]any{
				"categories":          categories,
				"example_directories": exampleDirs,
				"looked_for":          codeownersLocations,
			},
		},
	}, nil, nil
}

// detectHighRiskDirs returns a map of category → unique top-level directories
// that contain files matching high-risk keywords. Fixture and testdata paths
// are excluded.
func detectHighRiskDirs(repo model.RepoFacts) map[string][]string {
	result := map[string]map[string]bool{}

	for _, f := range repo.Files {
		if isFixtureOrTestdata(f.Path) {
			continue
		}
		lower := strings.ToLower(f.Path)
		segments := strings.Split(lower, "/")
		for cat, keywords := range highRiskKeywords {
			for _, seg := range segments {
				for _, kw := range keywords {
					if strings.Contains(seg, kw) {
						dir := dirOfPath(f.Path)
						if result[cat] == nil {
							result[cat] = map[string]bool{}
						}
						result[cat][dir] = true
						goto nextFile
					}
				}
			}
		nextFile:
		}
	}

	// Convert to sorted slices.
	out := map[string][]string{}
	for cat, dirs := range result {
		list := make([]string, 0, len(dirs))
		for d := range dirs {
			list = append(list, d)
		}
		sort.Strings(list)
		out[cat] = list
	}
	return out
}

// dirOfPath returns the directory portion of a path. For root-level files
// returns ".".
func dirOfPath(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	return "."
}
