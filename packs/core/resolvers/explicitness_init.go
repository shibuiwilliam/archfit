package resolvers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// ExpP3EXP002 fires when Go init() functions register handlers, drivers, or
// other cross-package side effects. These implicit registrations are invisible
// to agents searching for where behavior is wired.
//
// Detection: reads AST facts (GoFileFacts.InitFunctions). At standard depth,
// init() functions are detected but cross-package calls are not extracted, so
// we report any init() that exists. At deep depth, we only report init()
// functions that contain cross-package calls.
func ExpP3EXP002(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	astFacts, ok := facts.AST()
	if !ok {
		// AST collector didn't run (shallow depth or no Go files).
		// Rule is skipped via applies_to.languages — no findings.
		return nil, nil, nil
	}

	// Emit parse failures as findings so they're not silently lost.
	var findings []model.Finding
	for _, pf := range astFacts.ParseFailures {
		findings = append(findings, model.ParseFailure("P3.EXP.002", pf.Path, pf.Error))
	}

	for _, gf := range astFacts.GoFiles {
		for _, initFn := range gf.InitFunctions {
			// At deep depth, only flag init() with actual cross-pkg calls.
			// At standard depth, CrossPkgCalls is empty (not extracted),
			// so we flag any init() function as potentially problematic.
			if len(initFn.CrossPkgCalls) > 0 {
				calls := initFn.CrossPkgCalls
				findings = append(findings, model.Finding{
					Confidence: 0.95,
					Path:       gf.Path,
					Message: fmt.Sprintf(
						"init() registers across package boundaries: %s",
						strings.Join(calls, ", ")),
					Evidence: map[string]any{
						"line":            initFn.Line,
						"cross_pkg_calls": calls,
						"package":         gf.Package,
					},
				})
			} else if len(initFn.CrossPkgCalls) == 0 {
				// Standard depth: cross-pkg calls not extracted.
				// Only flag if we're in standard mode (no call data).
				// At deep depth, an init() with zero cross-pkg calls is fine.
				// We detect "standard mode" by checking that the init body
				// was not analyzed — CrossPkgCalls is nil (not []string{}).
				if initFn.CrossPkgCalls == nil {
					findings = append(findings, model.Finding{
						Confidence: 0.80,
						Path:       gf.Path,
						Message: fmt.Sprintf(
							"init() registers across package boundaries: init() at line %d (run with --depth=deep for call details)",
							initFn.Line),
						Evidence: map[string]any{
							"line":    initFn.Line,
							"package": gf.Package,
							"note":    "cross-package calls not extracted at standard depth",
						},
					})
				}
			}
		}
	}

	// Deterministic output: sort by path, then line.
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		li, _ := findings[i].Evidence["line"].(int)
		lj, _ := findings[j].Evidence["line"].(int)
		return li < lj
	})

	return findings, nil, nil
}
