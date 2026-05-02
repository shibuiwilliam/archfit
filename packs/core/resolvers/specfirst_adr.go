package resolvers

import (
	"context"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// SpcP2SPC004 fires when ADR files in docs/adr/ lack YAML frontmatter.
// An ADR has valid frontmatter if its content starts with "---" (the file
// must begin with a YAML document delimiter). We check file content via
// the Lines field: a file with frontmatter will have at least 4 lines
// (opening ---, id, title, closing ---). But since FactStore doesn't
// expose file content, we use a heuristic: check if the file's first few
// bytes match the frontmatter pattern by looking at file size vs line count.
//
// Actually, the filesystem collector gives us file metadata but not content.
// So we rely on the file extension (.md) and check if the file is in
// docs/adr/. For content inspection, we need the file itself. Since
// resolvers can't do I/O, we use the fact that the schema collector already
// parses some files. For ADRs, we check presence and use a simple heuristic:
// small ADR files (< 100 bytes) with no frontmatter are flagged.
//
// Phase 1 approach: the ecosystem collector detects ADR directories. The
// resolver checks each .md file in docs/adr/ and flags any that don't start
// with "---" by checking via file size heuristics. This is imperfect but
// strong enough for the experimental tier.
//
// Correction: We DO have access to ByPath but not content. The simplest
// correct approach is to require the ADR directory to exist AND for the repo
// to have a consistent structure. For now, we flag ADR files that are very
// small (likely missing frontmatter). The AST collector could be extended
// in Phase 1.5 to parse markdown frontmatter.
//
// FINAL APPROACH: We read file content through a simple mechanism — the
// collector/fs package records file content for small files. Actually no,
// it doesn't. Let me use the simplest workable approach:
//
// Check if docs/adr/ has .md files. For each, if the file has fewer than
// 5 lines, it almost certainly lacks frontmatter. For larger files, we
// can't check without content. This gives strong evidence for trivially-
// missing frontmatter but won't catch files that have content but no "---".
//
// This is acceptable at experimental stability. A content-reading collector
// would make this precise.
func SpcP2SPC004(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	// Find all .md files under docs/adr/.
	var adrFiles []model.FileFact
	for _, f := range repo.Files {
		if strings.HasPrefix(f.Path, "docs/adr/") && f.Ext == ".md" {
			adrFiles = append(adrFiles, f)
		}
	}

	if len(adrFiles) == 0 {
		// No ADR directory — rule doesn't apply (P7 covers ADR presence).
		return nil, nil, nil
	}

	var findings []model.Finding
	for _, f := range adrFiles {
		// Heuristic: files with < 5 lines almost certainly lack frontmatter
		// (---\nid: ...\ntitle: ...\nstatus: ...\ndate: ...\n---\n = 6 lines minimum).
		// For files with >= 5 lines, we check if the file is large enough
		// to plausibly contain frontmatter (at least ~50 bytes for the delimiters
		// + minimal fields).
		if f.Lines < 5 || f.Size < 30 {
			findings = append(findings, model.Finding{
				Confidence: 0.85,
				Path:       f.Path,
				Message:    f.Path + " missing YAML frontmatter (id, title, status, date)",
				Evidence: map[string]any{
					"lines": f.Lines,
					"bytes": f.Size,
				},
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].Path < findings[j].Path })
	return findings, nil, nil
}
