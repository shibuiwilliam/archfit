package agenttool_test

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
	agenttool "github.com/shibuiwilliam/archfit/packs/agent-tool"
)

type expectedShape struct {
	Findings []struct {
		RuleID          string `json:"rule_id"`
		Severity        string `json:"severity"`
		Path            string `json:"path"`
		MessageContains string `json:"message_contains"`
	} `json:"findings"`
}

func TestAgentToolPack_Fixtures(t *testing.T) {
	reg := rule.NewRegistry()
	if err := agenttool.Register(reg); err != nil {
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
			expectedRaw, err := os.ReadFile(filepath.Join(fixtureDir, "expected.json"))
			if err != nil {
				t.Fatal(err)
			}
			var expected expectedShape
			if err := json.Unmarshal(expectedRaw, &expected); err != nil {
				t.Fatal(err)
			}
			res, err := core.Scan(context.Background(), core.ScanInput{
				Root:  input,
				Rules: reg.Rules(),
			})
			if err != nil {
				t.Fatal(err)
			}
			got := filterByRule(res.Findings, ruleID)
			if len(got) != len(expected.Findings) {
				t.Fatalf("rule %s: got %d findings want %d\ngot: %+v", ruleID, len(got), len(expected.Findings), got)
			}
			for i, want := range expected.Findings {
				if got[i].RuleID != want.RuleID {
					t.Errorf("finding[%d].rule_id: %s want %s", i, got[i].RuleID, want.RuleID)
				}
				if string(got[i].Severity) != want.Severity {
					t.Errorf("finding[%d].severity: %s want %s", i, got[i].Severity, want.Severity)
				}
				if got[i].Path != want.Path {
					t.Errorf("finding[%d].path: %q want %q", i, got[i].Path, want.Path)
				}
				if want.MessageContains != "" && !strings.Contains(got[i].Message, want.MessageContains) {
					t.Errorf("finding[%d].message %q missing %q", i, got[i].Message, want.MessageContains)
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
