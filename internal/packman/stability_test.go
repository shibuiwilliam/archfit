// stability_test.go — CI gate enforcing the rule ID freeze. Once all rules
// are promoted to stable, no experimental rules should appear in core or
// agent-tool packs. Adding a new experimental rule is allowed only as a
// deliberate, ADR-documented decision.
package packman_test

import (
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	agenttool "github.com/shibuiwilliam/archfit/packs/agent-tool"
	corepack "github.com/shibuiwilliam/archfit/packs/core"
)

func TestStability_AllRulesAreStable(t *testing.T) {
	for _, r := range corepack.Rules() {
		if r.Stability != model.StabilityStable {
			t.Errorf("core rule %s has stability %q, want %q (rule ID freeze requires stable)",
				r.ID, r.Stability, model.StabilityStable)
		}
	}
	for _, r := range agenttool.Rules() {
		if r.Stability != model.StabilityStable {
			t.Errorf("agent-tool rule %s has stability %q, want %q (rule ID freeze requires stable)",
				r.ID, r.Stability, model.StabilityStable)
		}
	}
}
