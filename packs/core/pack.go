// Package core wires archfit's core rule pack. Registration is explicit:
// cmd/archfit/main.go calls Register at startup — no init()-time side effects.
package core

import (
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/rule"
	"github.com/shibuiwilliam/archfit/packs/core/resolvers"
)

// PackName matches .archfit.yaml's packs.enabled values.
const PackName = "core"

// Rules returns the core-pack rules. Keep the declarations next to each other
// so adding or removing a rule is a single, localized change.
func Rules() []model.Rule {
	return []model.Rule{
		{
			ID:               "P1.LOC.001",
			Principle:        model.P1Locality,
			Dimension:        "LOC",
			Title:            "Root of the repository has agent-facing documentation",
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "Agents need a consistent entry point. CLAUDE.md / AGENTS.md is the canonical one.",
			Remediation: model.Remediation{
				Summary:  "Add CLAUDE.md (or AGENTS.md) at the repository root.",
				GuideRef: "docs/rules/P1.LOC.001.md",
			},
			Resolver: resolvers.LocP1LOC001,
		},
		{
			ID:               "P1.LOC.002",
			Principle:        model.P1Locality,
			Dimension:        "LOC",
			Title:            "Declared vertical-slice directories carry their own AGENTS.md",
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "Each slice should be independently understandable. An AGENTS.md per slice bounds the context an agent must load.",
			Remediation: model.Remediation{
				Summary:  "Add AGENTS.md at the root of every immediate child of packs/, services/, or modules/.",
				GuideRef: "docs/rules/P1.LOC.002.md",
			},
			Resolver: resolvers.LocP1LOC002,
		},
		{
			ID:               "P4.VER.001",
			Principle:        model.P4Verifiability,
			Dimension:        "VER",
			Title:            "Repository declares a fast verification entrypoint",
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "A single documented command to run the fast checks. Without one, agents guess.",
			Remediation: model.Remediation{
				Summary:  "Add a Makefile (or justfile, Taskfile.yml, package.json, pyproject.toml, Cargo.toml) at the repo root.",
				GuideRef: "docs/rules/P4.VER.001.md",
			},
			Resolver: resolvers.VerP4VER001,
		},
		{
			ID:               "P7.MRD.001",
			Principle:        model.P7MachineReadability,
			Dimension:        "MRD",
			Title:            "Exit codes are documented when the repository ships a CLI",
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "Machine-readable CLIs require an exit-code contract. Agents depend on them for control flow.",
			Remediation: model.Remediation{
				Summary:  "Add docs/exit-codes.md listing every exit code the CLI may emit.",
				GuideRef: "docs/rules/P7.MRD.001.md",
			},
			Resolver: resolvers.MrdP7MRD001,
		},
	}
}

// Register wires the core pack into the given registry.
func Register(reg *rule.Registry) error {
	return reg.Register(PackName, Rules()...)
}
