package resolvers

import (
	"context"
	"fmt"
	"sort"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// globalMutableThreshold is the default maximum number of mutable package-level
// variables across the entire repository before P3.EXP.005 fires.
const globalMutableThreshold = 10

// ExpP3EXP005 fires when the total count of mutable package-level variables
// across the repository exceeds globalMutableThreshold. Blank identifiers (_)
// are excluded.
func ExpP3EXP005(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	astFacts, ok := facts.AST()
	if !ok {
		return nil, nil, nil
	}

	// Emit parse failures as findings.
	var findings []model.Finding
	for _, pf := range astFacts.ParseFailures {
		findings = append(findings, model.ParseFailure("P3.EXP.005", pf.Path, pf.Error))
	}

	// Collect all mutable package-level vars, excluding blank identifiers.
	type varRef struct {
		Path string
		Name string
		Line int
	}
	var mutableVars []varRef
	for _, gf := range astFacts.GoFiles {
		for _, v := range gf.PkgLevelVars {
			if !v.Mutable {
				continue
			}
			if v.Name == "_" {
				continue
			}
			mutableVars = append(mutableVars, varRef{
				Path: gf.Path,
				Name: v.Name,
				Line: v.Line,
			})
		}
	}

	if len(mutableVars) <= globalMutableThreshold {
		return findings, nil, nil
	}

	// Deterministic sort: path asc, then line asc.
	sort.Slice(mutableVars, func(i, j int) bool {
		if mutableVars[i].Path != mutableVars[j].Path {
			return mutableVars[i].Path < mutableVars[j].Path
		}
		return mutableVars[i].Line < mutableVars[j].Line
	})

	// Build evidence listing each var.
	varList := make([]map[string]any, len(mutableVars))
	for i, v := range mutableVars {
		varList[i] = map[string]any{
			"path": v.Path,
			"name": v.Name,
			"line": v.Line,
		}
	}

	findings = append(findings, model.Finding{
		Confidence: 0.85,
		Path:       ".",
		Message: fmt.Sprintf(
			"repository has %d global mutable package-level variables (threshold: %d)",
			len(mutableVars), globalMutableThreshold),
		Evidence: map[string]any{
			"total_mutable_vars": len(mutableVars),
			"threshold":          globalMutableThreshold,
			"vars":               varList,
		},
	})

	return findings, nil, nil
}
