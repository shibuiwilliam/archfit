package contract

import (
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/score"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name           string
		contract       Contract
		scores         score.Scores
		findings       []model.Finding
		wantPassed     bool
		wantViolations int
		wantMisses     int
		wantBudgets    int
	}{
		{
			name:       "empty contract passes",
			contract:   Contract{Version: 1},
			scores:     score.Scores{Overall: 100},
			wantPassed: true,
		},
		{
			name: "all hard constraints met",
			contract: Contract{
				Version: 1,
				HardConstraints: []Constraint{
					{Principle: "P1", MinScore: 80, Scope: "**"},
					{Principle: "overall", MinScore: 70, Scope: "**"},
				},
			},
			scores: score.Scores{
				Overall:     90,
				ByPrinciple: map[model.Principle]float64{"P1": 85},
			},
			wantPassed:     true,
			wantViolations: 0,
		},
		{
			name: "hard constraint violated by min_score",
			contract: Contract{
				Version: 1,
				HardConstraints: []Constraint{
					{Principle: "P4", MinScore: 90, Scope: "**"},
				},
			},
			scores: score.Scores{
				Overall:     80,
				ByPrinciple: map[model.Principle]float64{"P4": 60},
			},
			wantPassed:     false,
			wantViolations: 1,
		},
		{
			name: "hard constraint violated by max_findings",
			contract: Contract{
				Version: 1,
				HardConstraints: []Constraint{
					{Rule: "P5.AGG.001", MaxFindings: 0, Scope: "**"},
				},
			},
			scores: score.Scores{Overall: 80},
			findings: []model.Finding{
				{RuleID: "P5.AGG.001", Path: "src/auth/", Severity: model.SeverityWarn},
				{RuleID: "P5.AGG.001", Path: "lib/auth/", Severity: model.SeverityWarn},
			},
			wantPassed:     false,
			wantViolations: 1,
		},
		{
			name: "area budget exhausted",
			contract: Contract{
				Version: 1,
				AreaBudgets: []AreaBudget{
					{Path: "**", MaxFindings: 2},
				},
			},
			scores: score.Scores{Overall: 60},
			findings: []model.Finding{
				{RuleID: "P1.LOC.001", Path: "src/app.go", Principle: "P1"},
				{RuleID: "P3.EXP.001", Path: "src/config.go", Principle: "P3"},
				{RuleID: "P4.VER.001", Path: "src/test.go", Principle: "P4"},
			},
			wantPassed:  true, // budgets don't cause hard failure
			wantBudgets: 1,
		},
		{
			name: "soft target missed passes but records miss",
			contract: Contract{
				Version: 1,
				SoftTargets: []Target{
					{Principle: "P1", TargetScore: 95},
				},
			},
			scores: score.Scores{
				Overall:     80,
				ByPrinciple: map[model.Principle]float64{"P1": 75},
			},
			wantPassed: true,
			wantMisses: 1,
		},
		{
			name: "scoped finding constraint only counts matching rule",
			contract: Contract{
				Version: 1,
				HardConstraints: []Constraint{
					{Rule: "P1.LOC.001", MaxFindings: 0, Scope: "**"},
				},
			},
			scores: score.Scores{Overall: 80},
			findings: []model.Finding{
				{RuleID: "P3.EXP.001", Path: "src/config.go"}, // different rule
			},
			wantPassed:     true,
			wantViolations: 0,
		},
		{
			name: "budget with principle filter",
			contract: Contract{
				Version: 1,
				AreaBudgets: []AreaBudget{
					{Path: "**", MaxFindings: 1, Principles: []string{"P5"}},
				},
			},
			scores: score.Scores{Overall: 80},
			findings: []model.Finding{
				{RuleID: "P1.LOC.001", Path: "src/app.go", Principle: "P1"},  // excluded by principle filter
				{RuleID: "P5.AGG.001", Path: "src/auth.go", Principle: "P5"}, // counted
			},
			wantPassed:  true,
			wantBudgets: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Check(tt.contract, tt.scores, tt.findings)
			if res.Passed != tt.wantPassed {
				t.Errorf("Passed = %v, want %v", res.Passed, tt.wantPassed)
			}
			if tt.wantViolations > 0 && len(res.HardViolations) != tt.wantViolations {
				t.Errorf("violations = %d, want %d: %+v", len(res.HardViolations), tt.wantViolations, res.HardViolations)
			}
			if tt.wantMisses > 0 && len(res.SoftMisses) != tt.wantMisses {
				t.Errorf("soft misses = %d, want %d", len(res.SoftMisses), tt.wantMisses)
			}
			if tt.wantBudgets > 0 && len(res.BudgetStatus) != tt.wantBudgets {
				t.Errorf("budget entries = %d, want %d", len(res.BudgetStatus), tt.wantBudgets)
			}
			// Verify budget exhaustion for the budget test case.
			if tt.name == "area budget exhausted" && len(res.BudgetStatus) > 0 {
				bs := res.BudgetStatus[0]
				if !bs.Exhausted {
					t.Errorf("budget should be exhausted (current=%d, max=%d)", bs.Current, bs.Budget.MaxFindings)
				}
				if bs.Current != 3 {
					t.Errorf("budget current = %d, want 3", bs.Current)
				}
			}
			// Verify principle-filtered budget only counts P5 findings.
			if tt.name == "budget with principle filter" && len(res.BudgetStatus) > 0 {
				bs := res.BudgetStatus[0]
				if bs.Current != 1 {
					t.Errorf("budget current = %d, want 1 (only P5 findings)", bs.Current)
				}
			}
		})
	}
}

func TestParse_YAML_WithComments(t *testing.T) {
	data := []byte(`# Fitness contract for the auth service
version: 1

# The overall score must stay above 80
hard_constraints:
  - principle: overall
    min_score: 80.0
    scope: "**"

soft_targets:
  - principle: P4
    target_score: 95.0
    deadline: "2030-06-30"
`)
	c, err := parse(data)
	if err != nil {
		t.Fatalf("YAML with comments should parse: %v", err)
	}
	if c.Version != 1 {
		t.Errorf("version = %d, want 1", c.Version)
	}
	if len(c.HardConstraints) != 1 {
		t.Errorf("hard_constraints = %d, want 1", len(c.HardConstraints))
	}
	if len(c.SoftTargets) != 1 || c.SoftTargets[0].TargetScore != 95.0 {
		t.Errorf("soft_targets = %+v, want 1 entry with target 95.0", c.SoftTargets)
	}
}

func TestParse_JSON_BackwardCompat(t *testing.T) {
	data := []byte(`{"version":1,"hard_constraints":[{"principle":"P1","min_score":70,"scope":"**"}]}`)
	c, err := parse(data)
	if err != nil {
		t.Fatalf("JSON should still parse: %v", err)
	}
	if len(c.HardConstraints) != 1 {
		t.Errorf("hard_constraints = %d, want 1", len(c.HardConstraints))
	}
}

func TestParse_YAML_UnknownFieldRejected(t *testing.T) {
	data := []byte("version: 1\nbogus: true\n")
	_, err := parse(data)
	if err == nil {
		t.Fatal("YAML with unknown field should be rejected by strict parsing")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		c       Contract
		wantErr bool
	}{
		{
			name:    "valid empty contract",
			c:       Contract{Version: 1},
			wantErr: false,
		},
		{
			name:    "wrong version",
			c:       Contract{Version: 2},
			wantErr: true,
		},
		{
			name: "constraint missing scope",
			c: Contract{
				Version:         1,
				HardConstraints: []Constraint{{Principle: "P1", MinScore: 80}},
			},
			wantErr: true,
		},
		{
			name: "constraint missing principle and rule",
			c: Contract{
				Version:         1,
				HardConstraints: []Constraint{{Scope: "**", MinScore: 80}},
			},
			wantErr: true,
		},
		{
			name: "area budget missing path",
			c: Contract{
				Version:     1,
				AreaBudgets: []AreaBudget{{MaxFindings: 5}},
			},
			wantErr: true,
		},
		{
			name: "directive missing when",
			c: Contract{
				Version:         1,
				AgentDirectives: []AgentDirective{{Action: "stop"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
