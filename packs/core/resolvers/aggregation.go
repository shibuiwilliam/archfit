package resolvers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// sensitiveCategories maps a human-readable category to path-segment keywords
// that signal security-sensitive or dangerous operations. A file matches a
// category when any segment of its path contains the keyword.
var sensitiveCategories = map[string][]string{
	"auth":      {"auth", "authentication", "authorization", "login", "oauth", "session", "jwt", "rbac", "acl", "permission"},
	"secret":    {"secret", "credential", "token", "apikey", "encrypt", "decrypt", "cipher", "crypto"},
	"migration": {"migration", "migrate", "schema_change"},
	"deploy":    {"deploy", "infra", "terraform", "cloudformation", "pulumi", "ansible"},
}

// maxTopLevelDirs is the threshold. If sensitive files for a category span
// more than this many top-level directories, the capability is too scattered.
const maxTopLevelDirs = 2

// maxSecondLevelDirs is the threshold for depth-2 scatter detection.
// When sensitive files are concentrated in ≤2 top-level dirs (e.g., Java's
// src/), we check whether they scatter across too many second-level dirs
// within a single top-level dir. A higher threshold (3) is used because
// some internal scatter is expected.
const maxSecondLevelDirs = 3

// AggP5AGG001 fires when security-sensitive files are scattered across more
// than 2 top-level directories per category, indicating that dangerous
// capabilities are not aggregated.
//
// For repos with a flat top-level structure (e.g., Java projects where
// everything lives under src/), a secondary depth-2 check detects scatter
// across second-level directories within a single top-level dir.
func AggP5AGG001(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	var findings []model.Finding

	for category, keywords := range sensitiveCategories {
		topDirs := map[string]bool{}
		var matchedFiles []string

		for _, f := range repo.Files {
			if isFixtureOrTestdata(f.Path) {
				continue
			}
			if matchesCategory(f.Path, keywords) {
				topDirs[topLevelDir(f.Path)] = true
				matchedFiles = append(matchedFiles, f.Path)
			}
		}

		if len(matchedFiles) == 0 {
			continue
		}

		// Primary check: top-level directory scatter.
		if len(topDirs) > maxTopLevelDirs {
			dirs := sortedKeys(topDirs)
			findings = append(findings, model.Finding{
				Message: fmt.Sprintf(
					"%s-related files are scattered across %d top-level directories (%s) — dangerous capabilities should be concentrated",
					category, len(dirs), strings.Join(dirs, ", ")),
				Confidence: 0.90,
				Evidence: map[string]any{
					"category":       category,
					"top_level_dirs": dirs,
					"example_files":  truncateSlice(matchedFiles, 10),
					"threshold":      maxTopLevelDirs,
				},
			})
			continue
		}

		// Secondary check: depth-2 scatter within a single top-level dir.
		// This catches Java-style repos where src/ is the only top-level dir
		// but auth code scatters across src/controller/, src/service/, src/filter/, etc.
		if len(topDirs) == 1 {
			secondLevel := map[string]bool{}
			for _, f := range matchedFiles {
				secondLevel[secondLevelDir(f)] = true
			}
			if len(secondLevel) > maxSecondLevelDirs {
				parent := sortedKeys(topDirs)[0]
				dirs := sortedKeys(secondLevel)
				findings = append(findings, model.Finding{
					Message: fmt.Sprintf(
						"%s-related files are scattered across %d directories within %s/ (%s) — consider concentrating them",
						category, len(dirs), parent, strings.Join(dirs, ", ")),
					Confidence: 0.80,
					Evidence: map[string]any{
						"category":          category,
						"parent_dir":        parent,
						"second_level_dirs": dirs,
						"example_files":     truncateSlice(matchedFiles, 10),
						"threshold":         maxSecondLevelDirs,
						"depth":             2,
					},
				})
			}
		}
	}

	// Deterministic ordering.
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Message < findings[j].Message
	})

	return findings, nil, nil
}

// matchesCategory returns true if any segment of path (split by /) contains
// any keyword as a substring. This catches paths like "src/auth_service/handler.go"
// and "pkg/authentication/middleware.go".
func matchesCategory(path string, keywords []string) bool {
	lower := strings.ToLower(path)
	segments := strings.Split(lower, "/")
	for _, seg := range segments {
		for _, kw := range keywords {
			if strings.Contains(seg, kw) {
				return true
			}
		}
	}
	return false
}

// topLevelDir returns the first path segment. Files at the root return ".".
func topLevelDir(path string) string {
	if i := strings.Index(path, "/"); i >= 0 {
		return path[:i]
	}
	return "."
}

// secondLevelDir returns the first two path segments joined (e.g., "src/auth").
// Files at root or with only one segment return the top-level dir.
func secondLevelDir(path string) string {
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		return "."
	}
	if len(parts) < 3 {
		return parts[0]
	}
	return parts[0] + "/" + parts[1]
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func truncateSlice(s []string, limit int) []string {
	if len(s) <= limit {
		return s
	}
	return s[:limit]
}
