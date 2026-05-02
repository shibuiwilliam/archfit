// stability_test.go — CI gates enforcing the rule stability contract
// and the severity ↔ evidence calibration matrix (CLAUDE.md §6 invariant 2).
// Rules default to stable (ADR 0012). Rules listed in experimentalAllowed
// have been explicitly walked back to experimental with an ADR (e.g., ADR 0014).
// New experimental rules must be added to the allowlist with an ADR reference.
package packman_test

import (
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	agenttool "github.com/shibuiwilliam/archfit/packs/agent-tool"
	corepack "github.com/shibuiwilliam/archfit/packs/core"
)

// experimentalAllowed lists rule IDs that are permitted to be experimental.
// Each entry must reference the ADR that authorized the re-tiering.
var experimentalAllowed = map[string]string{
	"P1.LOC.003": "ADR 0014 — threshold uncalibrated",
	"P1.LOC.004": "ADR 0014 — threshold uncalibrated",
	"P1.LOC.005": "Phase 1 new rule — INTENT.md on high-risk paths",
	"P1.LOC.006": "Phase 1 new rule — agent doc bloat check",
	"P1.LOC.009": "Phase 1 new rule — runbook per high-risk slice",
	"P2.SPC.002": "Phase 1 new rule — bidirectional migrations",
	"P2.SPC.004": "Phase 1 new rule — ADR YAML frontmatter",
	"P3.EXP.002": "Phase 1 new rule — no init() cross-package registration",
	"P3.EXP.003": "Phase 1 new rule — reflection density bounded",
	"P3.EXP.005": "Phase 1 new rule — global mutable state minimized",
	"P5.AGG.001": "ADR 0014 — false positives on fixture paths",
	"P5.AGG.003": "Phase 1 new rule — risk-tier file",
	"P5.AGG.004": "Phase 1 new rule — CODEOWNERS on high-risk paths (first error severity)",
}

func TestStability_AllRulesAreStable(t *testing.T) {
	for _, r := range corepack.Rules() {
		if r.Stability == model.StabilityExperimental {
			if _, ok := experimentalAllowed[r.ID]; !ok {
				t.Errorf("core rule %s has stability %q but is not in experimentalAllowed — add an ADR reference or promote to stable",
					r.ID, r.Stability)
			}
			continue
		}
		if r.Stability != model.StabilityStable {
			t.Errorf("core rule %s has unexpected stability %q (want stable or allowed-experimental)",
				r.ID, r.Stability)
		}
	}
	for _, r := range agenttool.Rules() {
		if r.Stability == model.StabilityExperimental {
			if _, ok := experimentalAllowed[r.ID]; !ok {
				t.Errorf("agent-tool rule %s has stability %q but is not in experimentalAllowed — add an ADR reference or promote to stable",
					r.ID, r.Stability)
			}
			continue
		}
		if r.Stability != model.StabilityStable {
			t.Errorf("agent-tool rule %s has unexpected stability %q (want stable or allowed-experimental)",
				r.ID, r.Stability)
		}
	}
}

// TestSeverityCalibration_AllRules walks every registered rule and verifies
// the severity ↔ evidence matrix (CLAUDE.md §6 invariant 2). This is a CI
// gate: Rule.Validate enforces the matrix at construction time, but this test
// catches any rule that was built without going through Validate (e.g., hand-
// wired in tests or generated code that bypasses validation).
//
// Matrix:
//
//	critical → strong only
//	error    → strong only
//	warn     → strong, medium, or sampled (weak rejected)
//	info     �� any
func TestSeverityCalibration_AllRules(t *testing.T) {
	allRules := append(corepack.Rules(), agenttool.Rules()...)
	for _, r := range allRules {
		if err := r.Validate(); err != nil {
			t.Errorf("rule %s fails validation: %v", r.ID, err)
			continue
		}
		// Belt-and-suspenders: explicitly check the matrix even though
		// Validate covers it, so a regression in Validate is caught here.
		switch {
		case r.Severity == model.SeverityCritical && r.EvidenceStrength != model.EvidenceStrong:
			t.Errorf("rule %s: critical severity requires strong evidence, got %s", r.ID, r.EvidenceStrength)
		case r.Severity == model.SeverityError && r.EvidenceStrength != model.EvidenceStrong:
			t.Errorf("rule %s: error severity requires strong evidence, got %s", r.ID, r.EvidenceStrength)
		case r.Severity == model.SeverityWarn && r.EvidenceStrength == model.EvidenceWeak:
			t.Errorf("rule %s: warn severity requires at least medium evidence, got weak", r.ID)
		}
	}
}
