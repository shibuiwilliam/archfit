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
				Summary:  "Add AGENTS.md at the root of every immediate child of packs/, services/, modules/, packages/, apps/, libs/, or other recognized slice containers.",
				GuideRef: "docs/rules/P1.LOC.002.md",
			},
			Resolver: resolvers.LocP1LOC002,
		},
		{
			ID:               "P3.EXP.001",
			Principle:        model.P3ShallowExplicitness,
			Dimension:        "EXP",
			Title:            "Environment and application configuration is documented explicitly",
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "Hidden configuration is the opposite of shallow explicitness. .env files, Spring Boot profiles, Terraform tfvars, and Rails environments must be documented so agents can discover required settings.",
			Remediation: model.Remediation{
				Summary:  "Add .env.example, config/README.md, terraform.tfvars.example, or equivalent documentation for your stack's configuration mechanism.",
				GuideRef: "docs/rules/P3.EXP.001.md",
			},
			Resolver: resolvers.ExpP3EXP001,
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
				Summary:  "Add a Makefile (or justfile, Taskfile, package.json, pyproject.toml, Cargo.toml, pom.xml, build.gradle, Gemfile, Rakefile, composer.json, etc.) at the repo root.",
				GuideRef: "docs/rules/P4.VER.001.md",
			},
			Resolver: resolvers.VerP4VER001,
		},
		{
			ID:               "P5.AGG.001",
			Principle:        model.P5Aggregation,
			Dimension:        "AGG",
			Title:            "Security-sensitive files are concentrated in few directories",
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "Dangerous capabilities (auth, secrets, migrations, deploy) should be concentrated so they can be audited and guarded, not scattered across the tree.",
			Remediation: model.Remediation{
				Summary:  "Consolidate security-sensitive files under at most 2 top-level directories per category.",
				GuideRef: "docs/rules/P5.AGG.001.md",
			},
			Resolver: resolvers.AggP5AGG001,
		},
		{
			ID:               "P6.REV.001",
			Principle:        model.P6Reversibility,
			Dimension:        "REV",
			Title:            "Deployment has rollback documentation",
			Severity:         model.SeverityWarn,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "If a repo deploys, it must document how to roll back. Reversibility requires knowing the undo procedure before things go wrong.",
			Remediation: model.Remediation{
				Summary:  "Add docs/deployment.md or RUNBOOK.md documenting deploy, verify, and rollback procedures.",
				GuideRef: "docs/rules/P6.REV.001.md",
			},
			Resolver: resolvers.RevP6REV001,
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
		{
			ID:               "P4.VER.002",
			Principle:        model.P4Verifiability,
			Dimension:        "VER",
			Title:            "Source directories have test coverage",
			Severity:         model.SeverityInfo,
			EvidenceStrength: model.EvidenceMedium,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "Agents need test coverage to verify their changes. Directories without test files are unverifiable areas where agents work blind.",
			Remediation: model.Remediation{
				Summary:  "Add test files alongside source files in untested directories.",
				GuideRef: "docs/rules/P4.VER.002.md",
			},
			Resolver: resolvers.VerP4VER002,
		},
		{
			ID:               "P4.VER.003",
			Principle:        model.P4Verifiability,
			Dimension:        "VER",
			Title:            "Repository has CI configuration",
			Severity:         model.SeverityInfo,
			EvidenceStrength: model.EvidenceStrong,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "A repo without CI is locally verifiable but not continuously verified. Agents need CI to confirm changes pass on a clean environment.",
			Remediation: model.Remediation{
				Summary:  "Add a CI configuration (.github/workflows/, .gitlab-ci.yml, Jenkinsfile, or similar).",
				GuideRef: "docs/rules/P4.VER.003.md",
			},
			Resolver: resolvers.VerP4VER003,
		},
		{
			ID:               "P1.LOC.003",
			Principle:        model.P1Locality,
			Dimension:        "LOC",
			Title:            "Dependency graph coupling is bounded",
			Severity:         model.SeverityInfo,
			EvidenceStrength: model.EvidenceMedium,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "Packages with high transitive reach force agents to load many neighbors. Bounded coupling keeps context manageable.",
			Remediation: model.Remediation{
				Summary:  "Split high-reach packages into smaller modules with narrower interfaces.",
				GuideRef: "docs/rules/P1.LOC.003.md",
			},
			Resolver: resolvers.LocP1LOC003,
		},
		{
			ID:               "P1.LOC.004",
			Principle:        model.P1Locality,
			Dimension:        "LOC",
			Title:            "Commits touch a bounded number of files",
			Severity:         model.SeverityInfo,
			EvidenceStrength: model.EvidenceSampled,
			Stability:        model.StabilityExperimental,
			Weight:           1,
			Rationale:        "Repos where typical commits touch many files have poor locality — agents must hold wide context for routine changes.",
			Remediation: model.Remediation{
				Summary:  "Refactor to reduce cross-cutting changes; ensure each commit stays within a narrow slice.",
				GuideRef: "docs/rules/P1.LOC.004.md",
			},
			Resolver: resolvers.LocP1LOC004,
		},
	}
}

// Register wires the core pack into the given registry.
func Register(reg *rule.Registry) error {
	if err := reg.Register(PackName, Rules()...); err != nil {
		return err
	}
	reg.RegisterPack(rule.Pack{
		Name:        PackName,
		Version:     "0.3.0",
		Description: "Universal rules that apply to every repository",
		Principles:  []model.Principle{model.P1Locality, model.P4Verifiability, model.P7MachineReadability},
		RuleCount:   len(Rules()),
	})
	return nil
}
