// Package resolvers holds the pure ResolverFunc implementations for the
// agent-tool pack. Pure functions of model.FactStore: no I/O, no imports from
// internal/adapter or internal/collector. Schema content comes from the
// schema collector via FactStore.Schemas().
package resolvers

import (
	"context"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// specFirstFiles are basenames/extensions that indicate spec-first practices
// beyond JSON Schema. If any of these exist, the repo has some spec-first
// coverage and the "no schemas" finding should not fire.
var specFirstBasenames = []string{
	"openapi.yaml", "openapi.yml", "openapi.json",
	"swagger.yaml", "swagger.yml", "swagger.json",
}

var specFirstExtensions = []string{
	".proto",    // Protocol Buffers
	".graphql",  // GraphQL SDL
	".gql",      // GraphQL SDL (short form)
	".avsc",     // Apache Avro
	".asyncapi", // AsyncAPI
}

// P2SPC010 fires when the repo lacks a versioned JSON Schema — either there
// are no schemas/*.schema.json files at all, or none of them declare a
// top-level "$id". Parse errors on schema files are surfaced as separate
// ParseFailure findings per CLAUDE.md §13.
//
// If no JSON Schema files exist but the repo contains OpenAPI, Protobuf,
// GraphQL, or Avro spec files, the "no schemas" finding is suppressed —
// the repo has spec-first coverage via a different mechanism.
func P2SPC010(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	sc := facts.Schemas()
	var findings []model.Finding

	if len(sc.Files) == 0 {
		// Before firing, check for alternative spec-first formats.
		if altSpecs := findSpecFirstFiles(facts.Repo()); len(altSpecs) > 0 {
			return nil, nil, nil
		}
		return []model.Finding{{
			Path:       "schemas/",
			Message:    "no spec-first artifacts found (checked JSON Schema under schemas/, OpenAPI, Protobuf, GraphQL) — agent consumers cannot discover the output contract",
			Confidence: 0.97,
			Evidence: map[string]any{
				"looked_for_pattern":    "schemas/*.schema.json",
				"also_checked_formats":  []string{"OpenAPI", "Protobuf", "GraphQL", "Avro"},
				"also_checked_basename": specFirstBasenames,
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

// findSpecFirstFiles returns paths of alternative spec-first files in the repo.
func findSpecFirstFiles(repo model.RepoFacts) []string {
	var found []string
	for _, f := range repo.Files {
		base := f.Path
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		baseLower := strings.ToLower(base)
		for _, name := range specFirstBasenames {
			if baseLower == name {
				found = append(found, f.Path)
				break
			}
		}
		if len(found) > 0 && found[len(found)-1] == f.Path {
			continue // already matched by basename
		}
		for _, ext := range specFirstExtensions {
			if strings.HasSuffix(baseLower, ext) {
				found = append(found, f.Path)
				break
			}
		}
	}
	return found
}
