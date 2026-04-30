// Package score computes the overall and per-principle scores from the set of
// rules applied during a scan and the findings they produced.
//
// Key property (CLAUDE.md §13): scoring is weight-based and normalized per
// applicable rule set. Adding more rules does not make existing repos worse
// mechanically — each principle's score is computed against the weight of the
// rules that ran, not the absolute count.
package score

import (
	"math"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// Severity penalty, as a fraction of a rule's weight, applied when a rule
// produces at least one finding. Multiple findings do not compound — the rule's
// weight is fully spent at the first failure. This keeps scoring stable against
// rules that happen to be "noisy".
var severityPenalty = map[model.Severity]float64{
	model.SeverityInfo:     0.10,
	model.SeverityWarn:     0.40,
	model.SeverityError:    0.80,
	model.SeverityCritical: 1.00,
}

// Scores is the aggregate result emitted alongside findings.
type Scores struct {
	Overall     float64
	ByPrinciple map[model.Principle]float64
}

// Compute returns scores in [0, 100], rounded to one decimal place per CLAUDE.md §9.
// Rules whose IDs appear in skippedRuleIDs are excluded from weight calculation —
// they were never evaluated (e.g. applies_to mismatch) and should not affect the score.
func Compute(rules []model.Rule, findings []model.Finding, skippedRuleIDs ...string) Scores {
	skippedSet := make(map[string]bool, len(skippedRuleIDs))
	for _, id := range skippedRuleIDs {
		skippedSet[id] = true
	}
	type bucket struct {
		totalWeight    float64
		penaltyApplied float64
	}
	// worstBySev[ruleID] is the highest-severity finding observed for that rule.
	worstBySev := map[string]model.Severity{}
	for _, f := range findings {
		prev := worstBySev[f.RuleID]
		if prev == "" || f.Severity.Rank() > prev.Rank() {
			worstBySev[f.RuleID] = f.Severity
		}
	}

	perP := map[model.Principle]*bucket{}
	total := &bucket{}

	for _, r := range rules {
		if skippedSet[r.ID] {
			continue // rule was not evaluated; don't count its weight
		}
		w := r.Weight
		if w == 0 {
			w = 1
		}
		total.totalWeight += w
		b := perP[r.Principle]
		if b == nil {
			b = &bucket{}
			perP[r.Principle] = b
		}
		b.totalWeight += w

		if sev, hit := worstBySev[r.ID]; hit {
			penalty := severityPenalty[sev] * w
			b.penaltyApplied += penalty
			total.penaltyApplied += penalty
		}
	}

	out := Scores{ByPrinciple: map[model.Principle]float64{}}
	out.Overall = toScore(total.totalWeight, total.penaltyApplied)
	for _, p := range model.AllPrinciples() {
		b := perP[p]
		if b == nil {
			// No rules ran for this principle — don't fabricate 100; omit it.
			continue
		}
		out.ByPrinciple[p] = toScore(b.totalWeight, b.penaltyApplied)
	}
	return out
}

func toScore(totalW, penalty float64) float64 {
	if totalW <= 0 {
		return 100
	}
	s := 100.0 * (1.0 - penalty/totalW)
	if s < 0 {
		s = 0
	}
	if s > 100 {
		s = 100
	}
	return math.Round(s*10) / 10
}
