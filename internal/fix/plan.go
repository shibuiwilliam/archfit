package fix

import (
	"fmt"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// PlannedFix pairs a finding with the changes proposed to resolve it.
type PlannedFix struct {
	// RuleID is the rule that produced the finding.
	RuleID string `json:"rule_id"`
	// Finding is the original finding being addressed.
	Finding model.Finding `json:"finding"`
	// Changes are the proposed file modifications.
	Changes []Change `json:"changes"`
	// NeedsLLM indicates whether the fixer required --with-llm.
	NeedsLLM bool `json:"needs_llm,omitempty"`
}

// Plan is the complete set of proposed fixes before apply.
type Plan struct {
	Fixes []PlannedFix `json:"fixes"`
}

// Summary returns a human-readable summary of the plan for terminal output.
func (p Plan) Summary() string {
	if len(p.Fixes) == 0 {
		return "no fixable findings"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "fix plan: %d finding(s) to address\n", len(p.Fixes))
	for _, f := range p.Fixes {
		fmt.Fprintf(&b, "\n  %s — %s\n", f.RuleID, f.Finding.Message)
		for _, c := range f.Changes {
			fmt.Fprintf(&b, "    %s %s", c.Action, c.Path)
			if c.Preview != "" {
				fmt.Fprintf(&b, " (%s)", c.Preview)
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// FilePaths returns deduplicated list of all file paths that would be touched.
func (p Plan) FilePaths() []string {
	seen := map[string]struct{}{}
	var paths []string
	for _, f := range p.Fixes {
		for _, c := range f.Changes {
			if _, ok := seen[c.Path]; !ok {
				seen[c.Path] = struct{}{}
				paths = append(paths, c.Path)
			}
		}
	}
	return paths
}
