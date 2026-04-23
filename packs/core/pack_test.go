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
		ruleID := e.Name()
		t.Run(ruleID, func(t *testing.T) {
			fixtureDir := filepath.Join("fixtures", ruleID)
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

func filterByRule(fs []model.Finding, ruleID string) []model.Finding {
	var out []model.Finding
	for _, f := range fs {
		if f.RuleID == ruleID {
			out = append(out, f)
		}
	}
	return out
}
