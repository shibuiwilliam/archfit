// Package policy implements organization-level fitness policies.
//
// A policy file (JSON) lets an organization declare minimum scores, required
// packs, required rules, and per-repo exemptions. The Enforce function checks
// scan results against these requirements and returns any violations.
package policy

import (
	"encoding/json"
	"fmt"
	"os"
)

// Policy represents an organization-level fitness policy.
type Policy struct {
	Version       int                `json:"version"`
	Org           string             `json:"org"`
	MinScores     map[string]float64 `json:"minimum_scores"` // "overall", "P1", etc.
	RequiredPacks []string           `json:"required_packs"`
	RequiredRules []string           `json:"required_rules"`
	Severities    map[string]string  `json:"custom_severities"` // rule ID -> severity
	Exemptions    []Exemption        `json:"exemptions"`
}

// Exemption allows a specific repo to bypass certain rules.
type Exemption struct {
	Repo    string   `json:"repo"`
	Rules   []string `json:"rules"`
	Reason  string   `json:"reason"`
	Expires string   `json:"expires"`
}

// Violation is a policy requirement that was not met.
type Violation struct {
	Type   string `json:"type"` // "min_score", "required_pack", "required_rule"
	Detail string `json:"detail"`
}

// Load reads and parses a policy file (JSON format).
func Load(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("policy: %w", err)
	}
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return Policy{}, fmt.Errorf("policy: %w", err)
	}
	return p, nil
}
