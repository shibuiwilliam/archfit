// Package score computes the overall and per-principle scores from the set of
// rules applied during a scan and the findings they produced.
//
// Key property (CLAUDE.md §13): scoring is weight-based and normalized per
// applicable rule set. Adding more rules does not make existing repos worse
// mechanically — each principle's score is computed against the weight of the
// rules that ran, not the absolute count.
//
// Score model v2 (Phase 1, CLAUDE.md §7.5):
//   - Evidence factor modulates weight: strong=1.0, medium=0.85, weak=0.7, sampled=0.8.
//   - severity_class pass rates are the primary signal; overall is secondary.
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

// evidenceFactor modulates each rule's weight contribution based on how
// reliably the rule can verify its claims. Strong evidence counts fully;
// weaker evidence is discounted. See CLAUDE.md §7.5.
var evidenceFactor = map[model.EvidenceStrength]float64{
	model.EvidenceStrong:  1.0,
	model.EvidenceMedium:  0.85,
	model.EvidenceWeak:    0.70,
	model.EvidenceSampled: 0.80,
}

// SeverityClassRates holds pass rates per severity tier. A pass rate of 1.0
// means no rules of that severity produced findings. error_pass_rate is the
// primary signal (CLAUDE.md §7.5); overall is secondary.
type SeverityClassRates struct {
	CriticalPassRate float64
	ErrorPassRate    float64
	WarnPassRate     float64
	InfoPassRate     float64
}

// Scores is the aggregate result emitted alongside findings.
type Scores struct {
	Overall         float64
	ByPrinciple     map[model.Principle]float64
	BySeverityClass SeverityClassRates
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

	// Build a map from ruleID to rule for evidence lookup.
	ruleByID := make(map[string]model.Rule, len(rules))
	for _, r := range rules {
		ruleByID[r.ID] = r
	}

	perP := map[model.Principle]*bucket{}
	total := &bucket{}

	// Severity class counters: [total, passed] per severity tier.
	sevTotal := map[model.Severity]int{}
	sevPassed := map[model.Severity]int{}

	for _, r := range rules {
		if skippedSet[r.ID] {
			continue // rule was not evaluated; don't count its weight
		}
		w := r.Weight
		if w == 0 {
			w = 1
		}
		// Apply evidence factor to weight.
		ef := evidenceFactor[r.EvidenceStrength]
		if ef == 0 {
			ef = 1.0 // unknown evidence — full weight (defensive)
		}
		effectiveWeight := w * ef

		total.totalWeight += effectiveWeight
		b := perP[r.Principle]
		if b == nil {
			b = &bucket{}
			perP[r.Principle] = b
		}
		b.totalWeight += effectiveWeight

		// Track severity class pass rates using the rule's declared severity.
		sevTotal[r.Severity]++

		if sev, hit := worstBySev[r.ID]; hit {
			penalty := severityPenalty[sev] * effectiveWeight
			b.penaltyApplied += penalty
			total.penaltyApplied += penalty
		} else {
			sevPassed[r.Severity]++
		}
	}

	out := Scores{ByPrinciple: map[model.Principle]float64{}}
	out.Overall = toScore(total.totalWeight, total.penaltyApplied)
	for _, p := range model.AllPrinciples() {
		b := perP[p]
		if b == nil {
			continue
		}
		out.ByPrinciple[p] = toScore(b.totalWeight, b.penaltyApplied)
	}

	// Compute severity class pass rates.
	out.BySeverityClass = SeverityClassRates{
		CriticalPassRate: passRate(sevTotal[model.SeverityCritical], sevPassed[model.SeverityCritical]),
		ErrorPassRate:    passRate(sevTotal[model.SeverityError], sevPassed[model.SeverityError]),
		WarnPassRate:     passRate(sevTotal[model.SeverityWarn], sevPassed[model.SeverityWarn]),
		InfoPassRate:     passRate(sevTotal[model.SeverityInfo], sevPassed[model.SeverityInfo]),
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

// passRate returns the fraction of rules that passed (produced no findings)
// for a severity tier. Returns 1.0 when no rules exist at the tier.
func passRate(total, passed int) float64 {
	if total == 0 {
		return 1.0
	}
	r := float64(passed) / float64(total)
	return math.Round(r*100) / 100 // two decimal places
}
