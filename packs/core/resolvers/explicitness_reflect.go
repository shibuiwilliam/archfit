package resolvers

import (
	"context"
	"fmt"
	"sort"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// reflectFileThreshold is the default maximum number of files that may import
// "reflect" before P3.EXP.003 fires.
const reflectFileThreshold = 3

// ExpP3EXP003 fires when more than reflectFileThreshold files import the
// "reflect" package. A few reflect imports are normal; many suggest
// reflection-based dispatch that obscures concrete types and call paths.
func ExpP3EXP003(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	astFacts, ok := facts.AST()
	if !ok {
		return nil, nil, nil
	}

	// Emit parse failures as findings.
	var findings []model.Finding
	for _, pf := range astFacts.ParseFailures {
		findings = append(findings, model.ParseFailure("P3.EXP.003", pf.Path, pf.Error))
	}

	// Collect files that import "reflect".
	var reflectFiles []string
	for _, gf := range astFacts.GoFiles {
		if gf.ReflectImports {
			reflectFiles = append(reflectFiles, gf.Path)
		}
	}

	if len(reflectFiles) <= reflectFileThreshold {
		return findings, nil, nil
	}

	// Deterministic output.
	sort.Strings(reflectFiles)

	findings = append(findings, model.Finding{
		Confidence: 0.80,
		Path:       ".",
		Message: fmt.Sprintf(
			"repository has %d files importing reflect (threshold: %d)",
			len(reflectFiles), reflectFileThreshold),
		Evidence: map[string]any{
			"reflect_file_count": len(reflectFiles),
			"threshold":          reflectFileThreshold,
			"files":              reflectFiles,
		},
	})

	return findings, nil, nil
}
