// Package resolvers holds the pure ResolverFunc implementations for the
// agent-tool pack. Pure functions of model.FactStore: no I/O, no imports from
// internal/adapter or internal/collector. Schema content comes from the
// schema collector via FactStore.Schemas().
package resolvers

import (
	"context"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// P2SPC010 fires when the repo lacks a versioned JSON Schema — either there
// are no schemas/*.schema.json files at all, or none of them declare a
// top-level "$id". Parse errors on schema files are surfaced as separate
// ParseFailure findings per CLAUDE.md §13.
func P2SPC010(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	sc := facts.Schemas()
	var findings []model.Finding

	if len(sc.Files) == 0 {
		return []model.Finding{{
			Path:       "schemas/",
			Message:    "no JSON Schema files found under schemas/ — agent consumers cannot discover the output contract",
			Confidence: 0.97,
			Evidence: map[string]any{
				"looked_for_pattern": "schemas/*.schema.json",
			},
		}}, nil, nil
	}

	parseable := 0
	anyWithID := false
	for _, f := range sc.Files {
		if f.ParseError != "" {
			findings = append(findings, model.ParseFailure("P2.SPC.010", f.Path, f.ParseError))
			continue
		}
		parseable++
		if f.ID != "" {
			anyWithID = true
		}
	}

	if parseable > 0 && !anyWithID {
		paths := make([]string, 0, len(sc.Files))
		for _, f := range sc.Files {
			if f.ParseError == "" {
				paths = append(paths, f.Path)
			}
		}
		findings = append(findings, model.Finding{
			Path:       "schemas/",
			Message:    "no JSON Schema under schemas/ declares a $id — consumers cannot pin to a specific contract",
			Confidence: 0.95,
			Evidence: map[string]any{
				"schemas_found": paths,
				"missing_field": "$id",
			},
		})
	}
	return findings, nil, nil
}
