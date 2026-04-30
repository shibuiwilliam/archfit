package core_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/rule"
	corepack "github.com/shibuiwilliam/archfit/packs/core"
)

// expectedShape is a forgiving comparison schema: we don't diff the full JSON
// output (findings carry confidence floats, evidence blobs, etc). Instead we
// assert the set of (rule_id, severity, path) triples and a message substring.
// The full golden-JSON diff is a Phase 2 feature once the JSON contract freezes.
type expectedShape struct {
	Findings []struct {
		RuleID          string `json:"rule_id"`
		Severity        string `json:"severity"`
		Path            string `json:"path"`
		MessageContains string `json:"message_contains"`
	} `json:"findings"`
}

func TestCorePack_Fixtures(t *testing.T) {
	reg := rule.NewRegistry()
	if err := corepack.Register(reg); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir("fixtures")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		fixtureName := e.Name()
		// Support variant fixtures like "P1.LOC.002-packages" where the
		// rule ID is the prefix before the first hyphen-separated suffix
		// that doesn't match the P<n>.<DIM>.<nnn> pattern.
		ruleID := fixtureName
		if idx := strings.LastIndex(fixtureName, "-"); idx > 0 {
			candidate := fixtureName[:idx]
			// Only trim the suffix if the prefix looks like a valid rule ID.
			if len(candidate) >= 10 && candidate[0] == 'P' {
				ruleID = candidate
			}
		}
		t.Run(fixtureName, func(t *testing.T) {
			fixtureDir := filepath.Join("fixtures", fixtureName)
			input := filepath.Join(fixtureDir, "input")
			expectedPath := filepath.Join(fixtureDir, "expected.json")

			expectedRaw, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatal(err)
			}
			var expected expectedShape
			if err := json.Unmarshal(expectedRaw, &expected); err != nil {
				t.Fatalf("expected.json invalid: %v", err)
			}

			res, err := core.Scan(context.Background(), core.ScanInput{
				Root:  input,
				Rules: reg.Rules(),
				// No runner: git facts are unavailable for fixture scans.
			})
			if err != nil {
				t.Fatal(err)
			}

			// Only check findings for the rule under test. Other findings may
			// legitimately fire in a fixture (e.g. P1.LOC.001 on the P4 fixture
			// if the fixture author forgets CLAUDE.md) — those are separate bugs.
			gotForRule := filterByRule(res.Findings, ruleID)
			if len(gotForRule) != len(expected.Findings) {
				t.Fatalf("rule %s: got %d findings, want %d.\ngot: %+v",
					ruleID, len(gotForRule), len(expected.Findings), gotForRule)
			}
			for i, want := range expected.Findings {
				got := gotForRule[i]
				if got.RuleID != want.RuleID {
					t.Errorf("finding[%d].rule_id: got %s want %s", i, got.RuleID, want.RuleID)
				}
				if string(got.Severity) != want.Severity {
					t.Errorf("finding[%d].severity: got %s want %s", i, got.Severity, want.Severity)
				}
				if got.Path != want.Path {
					t.Errorf("finding[%d].path: got %q want %q", i, got.Path, want.Path)
				}
				if want.MessageContains != "" && !strings.Contains(got.Message, want.MessageContains) {
					t.Errorf("finding[%d].message: %q does not contain %q", i, got.Message, want.MessageContains)
				}
			}
		})
	}
}

// TestCorePack_PairFixtures ensures every registered core rule has both a
// positive fixture (rule fires) and a negative fixture (rule does not fire).
// CLAUDE.md §17 requires this. Without both, a rule cannot exit experimental.
func TestCorePack_PairFixtures(t *testing.T) {
	rules := corepack.Rules()
	entries, err := os.ReadDir("fixtures")
	if err != nil {
		t.Fatal(err)
	}

	fixtureNames := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			fixtureNames[e.Name()] = true
		}
	}

	// Rules that require runtime data (git history, depgraph) cannot produce
	// findings in fixture mode because core.Scan runs without a runner.
	// These are tested via metrics_test.go and CLI integration tests instead.
	runtimeOnly := map[string]bool{
		"P1.LOC.003": true, // requires DepGraph facts
		"P1.LOC.004": true, // requires Git facts
	}

	seen := map[string]bool{}
	for _, r := range rules {
		if seen[r.ID] {
			continue
		}
		seen[r.ID] = true

		hasPositive := false
		for name := range fixtureNames {
			if extractRuleID(name) == r.ID && !strings.HasSuffix(name, "-negative") {
				hasPositive = true
				break
			}
		}
		if !hasPositive && !runtimeOnly[r.ID] {
			t.Errorf("rule %s: missing positive fixture (fixtures/%s/)", r.ID, r.ID)
		}

		hasNegative := false
		for name := range fixtureNames {
			if extractRuleID(name) == r.ID && strings.HasSuffix(name, "-negative") {
				hasNegative = true
				break
			}
		}
		if !hasNegative {
			t.Errorf("rule %s: missing negative fixture (fixtures/%s-negative/)", r.ID, r.ID)
		}
	}
}

// extractRuleID derives the rule ID from a fixture directory name.
// "P3.EXP.001-spring" → "P3.EXP.001", "P3.EXP.001-negative" → "P3.EXP.001".
func extractRuleID(fixtureName string) string {
	id := fixtureName
	for {
		idx := strings.LastIndex(id, "-")
		if idx <= 0 {
			break
		}
		candidate := id[:idx]
		if len(candidate) >= 10 && candidate[0] == 'P' {
			id = candidate
			continue
		}
		break
	}
	return id
}

func filterByRule(fs []model.Finding, ruleID string) []model.Finding {
	var out []model.Finding
	for _, f := range fs {
		if f.RuleID == ruleID {
			out = append(out, f)
		}
	}
	return out
}
