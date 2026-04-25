package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnforce(t *testing.T) {
	tests := []struct {
		name         string
		policy       Policy
		scores       map[string]float64
		overall      float64
		enabledPacks []string
		ruleIDs      []string
		repoName     string
		wantTypes    []string // expected violation types in order
	}{
		{
			name: "score below minimum",
			policy: Policy{
				MinScores: map[string]float64{"overall": 80.0, "P1": 70.0},
			},
			scores:    map[string]float64{"P1": 50.0},
			overall:   60.0,
			wantTypes: []string{"min_score", "min_score"},
		},
		{
			name: "missing required pack",
			policy: Policy{
				RequiredPacks: []string{"core", "iac"},
			},
			enabledPacks: []string{"core"},
			overall:      100.0,
			scores:       map[string]float64{},
			wantTypes:    []string{"required_pack"},
		},
		{
			name: "missing required rule",
			policy: Policy{
				RequiredRules: []string{"P1.LOC.001", "P2.SPC.001"},
			},
			ruleIDs:   []string{"P1.LOC.001"},
			overall:   100.0,
			scores:    map[string]float64{},
			wantTypes: []string{"required_rule"},
		},
		{
			name: "exemption suppresses violation",
			policy: Policy{
				RequiredRules: []string{"P1.LOC.001", "P2.SPC.001"},
				Exemptions: []Exemption{
					{Repo: "my-repo", Rules: []string{"P2.SPC.001"}, Reason: "legacy"},
				},
			},
			ruleIDs:   []string{"P1.LOC.001"},
			repoName:  "my-repo",
			overall:   100.0,
			scores:    map[string]float64{},
			wantTypes: nil,
		},
		{
			name: "all requirements met",
			policy: Policy{
				MinScores:     map[string]float64{"overall": 80.0},
				RequiredPacks: []string{"core"},
				RequiredRules: []string{"P1.LOC.001"},
			},
			scores:       map[string]float64{"P1": 90.0},
			overall:      90.0,
			enabledPacks: []string{"core"},
			ruleIDs:      []string{"P1.LOC.001"},
			wantTypes:    nil,
		},
		{
			name:      "empty policy produces no violations",
			policy:    Policy{},
			scores:    map[string]float64{},
			overall:   50.0,
			wantTypes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Enforce(tt.policy, tt.scores, tt.overall, tt.enabledPacks, tt.ruleIDs, tt.repoName)
			if len(got) != len(tt.wantTypes) {
				t.Fatalf("got %d violations, want %d: %+v", len(got), len(tt.wantTypes), got)
			}
			for i, v := range got {
				if v.Type != tt.wantTypes[i] {
					t.Errorf("violation[%d].Type = %q, want %q", i, v.Type, tt.wantTypes[i])
				}
				if v.Detail == "" {
					t.Errorf("violation[%d].Detail is empty", i)
				}
			}
		})
	}
}

func TestLoad(t *testing.T) {
	pol := Policy{
		Version:       1,
		Org:           "test-org",
		MinScores:     map[string]float64{"overall": 75.0},
		RequiredPacks: []string{"core"},
		RequiredRules: []string{"P1.LOC.001"},
		Exemptions: []Exemption{
			{Repo: "legacy", Rules: []string{"P1.LOC.001"}, Reason: "old repo", Expires: "2027-01-01"},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "policy.json")
	data, err := json.Marshal(pol)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Org != "test-org" {
		t.Errorf("Org = %q, want %q", got.Org, "test-org")
	}
	if len(got.RequiredPacks) != 1 || got.RequiredPacks[0] != "core" {
		t.Errorf("RequiredPacks = %v, want [core]", got.RequiredPacks)
	}
}

func TestLoad_missing(t *testing.T) {
	_, err := Load("/nonexistent/policy.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
