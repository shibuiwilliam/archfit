package resolvers

import (
	"context"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// migrationDirNames are directory names that conventionally hold DB migrations.
var migrationDirNames = []string{
	"migrations", "migration", "db/migrations", "db/migrate",
}

// SpcP2SPC002 fires when a migrations directory contains SQL files that are
// not bidirectional. It checks for paired up/down files using common naming
// conventions: *up*.sql / *down*.sql, or *.up.sql / *.down.sql.
func SpcP2SPC002(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	// Find migration directories that actually contain .sql files.
	type migDir struct {
		path    string
		upCount int
		dnCount int
	}
	dirs := map[string]*migDir{}

	for _, f := range repo.Files {
		if isFixtureOrTestdata(f.Path) {
			continue
		}
		if f.Ext != ".sql" {
			continue
		}
		lowerPath := strings.ToLower(f.Path)
		for _, prefix := range migrationDirNames {
			if !strings.HasPrefix(lowerPath, prefix+"/") && !strings.Contains(lowerPath, "/"+prefix+"/") {
				continue
			}
			dirKey := prefix
			// Use the actual path prefix up to the migration dir.
			idx := strings.Index(lowerPath, prefix)
			if idx >= 0 {
				dirKey = f.Path[:idx+len(prefix)]
			}
			md, ok := dirs[dirKey]
			if !ok {
				md = &migDir{path: dirKey}
				dirs[dirKey] = md
			}
			base := strings.ToLower(fileBase(f.Path))
			if strings.Contains(base, "up") {
				md.upCount++
			}
			if strings.Contains(base, "down") {
				md.dnCount++
			}
			break
		}
	}

	if len(dirs) == 0 {
		return nil, nil, nil
	}

	var findings []model.Finding
	for _, md := range dirs {
		if md.upCount == 0 && md.dnCount == 0 {
			// No up/down convention detected; can't assess directionality.
			continue
		}
		if md.upCount > 0 && md.dnCount > 0 {
			// Both directions present — OK.
			continue
		}
		direction := "up"
		if md.dnCount > 0 {
			direction = "down"
		}
		findings = append(findings, model.Finding{
			Confidence: 0.90,
			Path:       md.path,
			Message:    md.path + " has only " + direction + " migrations — missing bidirectional pair",
			Evidence: map[string]any{
				"directory":  md.path,
				"up_count":   md.upCount,
				"down_count": md.dnCount,
			},
		})
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].Path < findings[j].Path })
	return findings, nil, nil
}
