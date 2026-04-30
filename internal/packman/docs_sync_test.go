// docs_sync_test.go — regression gate ensuring every registered rule has both:
//   - docs/rules/<id>.md
//   - .claude/skills/archfit/reference/remediation/<id>.md
//
// If either file is missing for any rule, this test fails with a single-line
// diagnostic. This is the CI enforcement for CLAUDE.md §13 and §17:
// "every new rule ships with a remediation guide."
package packman_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
	agenttool "github.com/shibuiwilliam/archfit/packs/agent-tool"
	corepack "github.com/shibuiwilliam/archfit/packs/core"
)

func TestDocsSync_AllRulesHaveDocumentation(t *testing.T) {
	repoRoot := filepath.Join("..", "..")

	var allRules []model.Rule
	allRules = append(allRules, corepack.Rules()...)
	allRules = append(allRules, agenttool.Rules()...)

	if len(allRules) == 0 {
		t.Fatal("no rules registered — registry is empty")
	}

	for _, r := range allRules {
		// docs/rules/<id>.md
		docsPath := filepath.Join(repoRoot, "docs", "rules", r.ID+".md")
		if _, err := os.Stat(docsPath); err != nil {
			t.Errorf("missing docs/rules/%s.md — every rule must have a rule doc (CLAUDE.md §17)", r.ID)
		}

		// .claude/skills/archfit/reference/remediation/<id>.md
		remPath := filepath.Join(repoRoot, ".claude", "skills", "archfit", "reference", "remediation", r.ID+".md")
		if _, err := os.Stat(remPath); err != nil {
			t.Errorf("missing .claude/skills/archfit/reference/remediation/%s.md — every rule must have a remediation guide (CLAUDE.md §13)", r.ID)
		}
	}
}
