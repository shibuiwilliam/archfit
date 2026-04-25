package static

import (
	"context"

	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/model"
)

const outputSchemaJSON = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/archfit/schemas/output.schema.json",
  "title": "archfit scan output",
  "description": "Schema for archfit CLI JSON output.",
  "type": "object",
  "required": ["schema_version", "findings"],
  "properties": {
    "schema_version": {
      "type": "string",
      "description": "Semantic version of this output schema."
    },
    "findings": {
      "type": "array",
      "items": {
        "type": "object"
      },
      "description": "List of findings from the scan."
    },
    "score": {
      "type": "number",
      "description": "Overall fitness score (0.0-10.0)."
    }
  }
}
`

// SpcP2SPC010Fixer creates schemas/output.schema.json with a skeleton schema.
type SpcP2SPC010Fixer struct{}

// NewSpcP2SPC010 returns a new fixer for rule P2.SPC.010.
func NewSpcP2SPC010() *SpcP2SPC010Fixer {
	return &SpcP2SPC010Fixer{}
}

// RuleID implements fix.Fixer.
func (f *SpcP2SPC010Fixer) RuleID() string { return "P2.SPC.010" }

// NeedsLLM implements fix.Fixer.
func (f *SpcP2SPC010Fixer) NeedsLLM() bool { return false }

// Plan implements fix.Fixer.
func (f *SpcP2SPC010Fixer) Plan(_ context.Context, _ model.Finding, _ model.FactStore) ([]fix.Change, error) {
	return []fix.Change{
		{
			Path:    "schemas/output.schema.json",
			Action:  fix.ActionCreate,
			Content: []byte(outputSchemaJSON),
			Preview: "Create schemas/output.schema.json with $id and schema_version",
		},
	}, nil
}
