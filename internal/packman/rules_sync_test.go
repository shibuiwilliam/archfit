// rules_sync_test.go — regression gate ensuring YAML rule files and Go rule
// declarations stay in sync. If anyone adds a rule in Go without a YAML file
// (or vice versa), this test fails.
//
// This is the CI enforcement mechanism for CLAUDE.md §3 P2 (spec-first):
// YAML is the source of truth; Go retains only resolver wiring.
package packman_test

import (
	"sort"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	agenttool "github.com/shibuiwilliam/archfit/packs/agent-tool"
	corepack "github.com/shibuiwilliam/archfit/packs/core"
)

func TestRulesSync_Core(t *testing.T) {
	goIDs := ruleIDs(corepack.Rules())
	yamlIDs := ruleIDs(corepack.GeneratedRules())
	assertIDSetsEqual(t, "core", goIDs, yamlIDs)
}

func TestRulesSync_AgentTool(t *testing.T) {
	goIDs := ruleIDs(agenttool.Rules())
	yamlIDs := ruleIDs(agenttool.GeneratedRules())
	assertIDSetsEqual(t, "agent-tool", goIDs, yamlIDs)
}

func ruleIDs(rules []model.Rule) []string {
	ids := make([]string, len(rules))
	for i, r := range rules {
		ids[i] = r.ID
	}
	sort.Strings(ids)
	return ids
}

func assertIDSetsEqual(t *testing.T, pack string, goIDs, yamlIDs []string) {
	t.Helper()
	goSet := toSet(goIDs)
	yamlSet := toSet(yamlIDs)

	for _, id := range goIDs {
		if !yamlSet[id] {
			t.Errorf("pack %s: rule %s exists in Go (pack.go) but has no YAML file under rules/", pack, id)
		}
	}
	for _, id := range yamlIDs {
		if !goSet[id] {
			t.Errorf("pack %s: rule %s has a YAML file but is not declared in Go (pack.go)", pack, id)
		}
	}
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}
