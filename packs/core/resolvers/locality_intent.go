package resolvers

import (
	"context"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// LocP1LOC005 fires when directories matching high-risk keywords lack an INTENT.md.
// Reuses highRiskKeywords from aggregation_codeowners.go.
func LocP1LOC005(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	// Collect all directories that contain at least one file and whose name
	// matches a high-risk keyword.
	highRiskDirs := map[string]bool{}
	for _, f := range repo.Files {
		if isFixtureOrTestdata(f.Path) {
			continue
		}
		parts := strings.Split(f.Path, "/")
		// Check each directory component (not the file basename).
		for i := 0; i < len(parts)-1; i++ {
			seg := strings.ToLower(parts[i])
			for _, keywords := range highRiskKeywords {
				for _, kw := range keywords {
					if strings.Contains(seg, kw) {
						dirPath := strings.Join(parts[:i+1], "/")
						highRiskDirs[dirPath] = true
					}
				}
			}
		}
	}

	if len(highRiskDirs) == 0 {
		return nil, nil, nil
	}

	var findings []model.Finding
	for dir := range highRiskDirs {
		intentPath := dir + "/INTENT.md"
		if _, ok := repo.ByPath[intentPath]; ok {
			continue
		}
		findings = append(findings, model.Finding{
			Confidence: 0.90,
			Path:       dir,
			Message:    dir + " is a high-risk directory but has no INTENT.md",
			Evidence: map[string]any{
				"directory":  dir,
				"looked_for": intentPath,
			},
		})
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].Path < findings[j].Path })
	return findings, nil, nil
}
