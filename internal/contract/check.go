package contract

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/score"
)

// CheckResult is the outcome of evaluating scan results against a contract.
type CheckResult struct {
	HardViolations []Violation   `json:"hard_violations"`
	SoftMisses     []SoftMiss    `json:"soft_misses"`
	BudgetStatus   []BudgetState `json:"budget_status"`
	Passed         bool          `json:"passed"` // true when all hard constraints met
}

// Violation describes a hard constraint that was not satisfied.
type Violation struct {
	Constraint Constraint `json:"constraint"`
	Actual     float64    `json:"actual"`
	Detail     string     `json:"detail"`
}

// SoftMiss describes a soft target that has not yet been reached.
type SoftMiss struct {
	Target Target  `json:"target"`
	Actual float64 `json:"actual"`
	Detail string  `json:"detail"`
}

// BudgetState tracks the current finding count vs. budget for an area.
type BudgetState struct {
	Budget    AreaBudget `json:"budget"`
	Current   int        `json:"current_findings"`
	Remaining int        `json:"remaining"`
	Exhausted bool       `json:"exhausted"`
}

// Check evaluates scan results against the contract. It is a pure function:
// no I/O, no side effects. Receives pre-computed scan results.
func Check(c Contract, scores score.Scores, findings []model.Finding) CheckResult {
	var res CheckResult

	// 1. Check hard constraints.
	for _, hc := range c.HardConstraints {
		if v := checkHardConstraint(hc, scores, findings); v != nil {
			res.HardViolations = append(res.HardViolations, *v)
		}
	}

	// 2. Check soft targets.
	for _, t := range c.SoftTargets {
		if m := checkSoftTarget(t, scores); m != nil {
			res.SoftMisses = append(res.SoftMisses, *m)
		}
	}

	// 3. Compute area budget status.
	for _, ab := range c.AreaBudgets {
		res.BudgetStatus = append(res.BudgetStatus, computeBudgetState(ab, findings))
	}

	// Deterministic ordering.
	sort.Slice(res.HardViolations, func(i, j int) bool {
		return res.HardViolations[i].Detail < res.HardViolations[j].Detail
	})
	sort.Slice(res.SoftMisses, func(i, j int) bool {
		return res.SoftMisses[i].Detail < res.SoftMisses[j].Detail
	})

	res.Passed = len(res.HardViolations) == 0
	return res
}

func checkHardConstraint(hc Constraint, scores score.Scores, findings []model.Finding) *Violation {
	// Score-based constraint.
	if hc.MinScore > 0 {
		actual := resolveScore(hc.Principle, scores)
		if actual < hc.MinScore {
			return &Violation{
				Constraint: hc,
				Actual:     actual,
				Detail: fmt.Sprintf("%s score %.1f is below minimum %.1f",
					principleLabel(hc.Principle), actual, hc.MinScore),
			}
		}
		return nil
	}

	// Finding-count constraint (rule-based).
	if hc.Rule != "" {
		count := countFindings(hc.Rule, hc.Scope, findings)
		if count > hc.MaxFindings {
			return &Violation{
				Constraint: hc,
				Actual:     float64(count),
				Detail: fmt.Sprintf("rule %s has %d findings (max %d) in scope %s",
					hc.Rule, count, hc.MaxFindings, hc.Scope),
			}
		}
	}
	return nil
}

func checkSoftTarget(t Target, scores score.Scores) *SoftMiss {
	if t.Principle != "" && t.TargetScore > 0 {
		actual := resolveScore(t.Principle, scores)
		if actual < t.TargetScore {
			return &SoftMiss{
				Target: t,
				Actual: actual,
				Detail: fmt.Sprintf("%s score %.1f has not reached target %.1f",
					principleLabel(t.Principle), actual, t.TargetScore),
			}
		}
	}
	return nil
}

func computeBudgetState(ab AreaBudget, findings []model.Finding) BudgetState {
	current := 0
	for _, f := range findings {
		if !matchesScope(f.Path, ab.Path) {
			continue
		}
		if len(ab.Principles) > 0 && !containsString(ab.Principles, string(f.Principle)) {
			continue
		}
		current++
	}
	remaining := ab.MaxFindings - current
	if remaining < 0 {
		remaining = 0
	}
	return BudgetState{
		Budget:    ab,
		Current:   current,
		Remaining: remaining,
		Exhausted: current >= ab.MaxFindings,
	}
}

// resolveScore returns the score for a principle key or "overall".
func resolveScore(key string, scores score.Scores) float64 {
	if key == "overall" || key == "" {
		return scores.Overall
	}
	return scores.ByPrinciple[model.Principle(key)]
}

func principleLabel(p string) string {
	if p == "" || p == "overall" {
		return "overall"
	}
	return p
}

// countFindings counts findings matching ruleID within the given scope glob.
func countFindings(ruleID, scope string, findings []model.Finding) int {
	count := 0
	for _, f := range findings {
		if f.RuleID != ruleID {
			continue
		}
		if matchesScope(f.Path, scope) {
			count++
		}
	}
	return count
}

// matchesScope checks whether a finding path matches a scope glob pattern.
// "**" matches everything. Empty path matches "**" only.
func matchesScope(findingPath, scopePattern string) bool {
	if scopePattern == "**" {
		return true
	}
	if findingPath == "" {
		return false
	}
	matched, _ := filepath.Match(scopePattern, findingPath)
	return matched
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
