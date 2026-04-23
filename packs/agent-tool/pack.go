// Package agenttool wires archfit's agent-tool rule pack. Registration is
// explicit (cmd/archfit/main.go) — no init() side effects.
package agenttool

import (
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/rule"
	"github.com/shibuiwilliam/archfit/packs/agent-tool/resolvers"
)

const PackName = "agent-tool"

func Rules() []model.Rule {
	return []model.Rule{
		{
			ID:               "P2.SPC.010",
			Principle:        model.P2SpecFirst,
			Dimension:        "SPC",
			Title:            "Tool ships a versioned JSON output schema",
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "Agent consumers need a pinnable contract for machine-readable output; a versioned JSON Schema with $id is the canonical signal.",
			Remediation: model.Remediation{
				Summary:  "Place a JSON Schema at schemas/output.schema.json with a $id, and emit a schema_version in the tool's output.",
				GuideRef: "docs/rules/P2.SPC.010.md",
			},
			Resolver: resolvers.P2SPC010,
		},
		{
			ID:               "P7.MRD.002",
			Principle:        model.P7MachineReadability,
			Dimension:        "MRD",
			Title:            "Agent-tool repository has a CHANGELOG.md at the root",
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "Agents need a machine-readable record of what changed between tool versions.",
			Remediation: model.Remediation{
				Summary:  "Add CHANGELOG.md at the repository root following the Keep a Changelog format.",
				GuideRef: "docs/rules/P7.MRD.002.md",
			},
			Resolver: resolvers.P7MRD002,
		},
		{
			ID:               "P7.MRD.003",
			Principle:        model.P7MachineReadability,
			Dimension:        "MRD",
			Title:            "Agent-tool repository records ADRs under docs/adr/",
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "ADRs surface irreversible design decisions to agents; their absence hides load-bearing choices.",
			Remediation: model.Remediation{
				Summary:  "Create docs/adr/ and seed it with an ADR documenting the architecture overview.",
				GuideRef: "docs/rules/P7.MRD.003.md",
			},
			Resolver: resolvers.P7MRD003,
		},
	}
}

func Register(reg *rule.Registry) error {
	return reg.Register(PackName, Rules()...)
}
