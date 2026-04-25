package policy

import "fmt"

// Enforce checks scan results against the policy and returns violations.
// repoName is used for exemption matching. Scores maps principle names
// (e.g. "P1") to their scores. overall is the aggregate score. enabledPacks
// and ruleIDs describe what was active during the scan.
func Enforce(pol Policy, scores map[string]float64, overall float64,
	enabledPacks []string, ruleIDs []string, repoName string) []Violation {

	// Build exemption index for this repo.
	exemptRules := map[string]bool{}
	for _, ex := range pol.Exemptions {
		if ex.Repo == repoName {
			for _, r := range ex.Rules {
				exemptRules[r] = true
			}
		}
	}

	var violations []Violation

	// 1. Check minimum scores.
	for key, minScore := range pol.MinScores {
		var actual float64
		if key == "overall" {
			actual = overall
		} else {
			actual = scores[key]
		}
		if actual < minScore {
			violations = append(violations, Violation{
				Type:   "min_score",
				Detail: fmt.Sprintf("%s score %.1f is below minimum %.1f", key, actual, minScore),
			})
		}
	}

	// 2. Check required packs.
	packSet := toSet(enabledPacks)
	for _, rp := range pol.RequiredPacks {
		if !packSet[rp] {
			violations = append(violations, Violation{
				Type:   "required_pack",
				Detail: fmt.Sprintf("required pack %q is not enabled", rp),
			})
		}
	}

	// 3. Check required rules (with exemption support).
	ruleSet := toSet(ruleIDs)
	for _, rr := range pol.RequiredRules {
		if exemptRules[rr] {
			continue
		}
		if !ruleSet[rr] {
			violations = append(violations, Violation{
				Type:   "required_rule",
				Detail: fmt.Sprintf("required rule %q was not evaluated", rr),
			})
		}
	}

	return violations
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}
