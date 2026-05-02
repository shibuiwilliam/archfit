package resolvers

import (
	"context"
	"fmt"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// agentDocMaxLines is the line limit for agent-facing docs (CLAUDE.md §13).
const agentDocMaxLines = 400

// agentDocMaxBytes is the byte limit (10 KB).
const agentDocMaxBytes int64 = 10 * 1024

// agentDocFiles are the files checked for bloat.
var agentDocFiles = []string{"CLAUDE.md", "AGENTS.md"}

// LocP1LOC006 fires when CLAUDE.md or AGENTS.md exceeds 400 lines or 10 KB.
func LocP1LOC006(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()
	var findings []model.Finding

	for _, name := range agentDocFiles {
		f, ok := repo.ByPath[name]
		if !ok {
			continue
		}
		if f.Lines > agentDocMaxLines {
			findings = append(findings, model.Finding{
				Confidence: 0.95,
				Path:       name,
				Message: fmt.Sprintf(
					"%s exceeds %d lines (%d lines) — agents may not fit it in context",
					name, agentDocMaxLines, f.Lines),
				Evidence: map[string]any{
					"lines":     f.Lines,
					"max_lines": agentDocMaxLines,
				},
			})
		} else if f.Size > agentDocMaxBytes {
			findings = append(findings, model.Finding{
				Confidence: 0.95,
				Path:       name,
				Message: fmt.Sprintf(
					"%s exceeds %d bytes (%d bytes) — agents may not fit it in context",
					name, agentDocMaxBytes, f.Size),
				Evidence: map[string]any{
					"bytes":     f.Size,
					"max_bytes": agentDocMaxBytes,
				},
			})
		}
	}

	return findings, nil, nil
}
