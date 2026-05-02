// Package core wires archfit's core rule pack. Registration is explicit:
// cmd/archfit/main.go calls Register at startup — no init()-time side effects.
//
// Rule metadata (ID, title, severity, etc.) comes from GeneratedRules()
// which is produced from packs/core/rules/*.yaml by `make generate`.
// This file provides only the resolver wiring — a map from rule ID to the
// Go function that implements the detection logic.
package core

import (
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/rule"
	"github.com/shibuiwilliam/archfit/packs/core/resolvers"
)

// PackName matches .archfit.yaml's packs.enabled values.
const PackName = "core"

// resolverMap wires rule IDs to their resolver functions. This is the only
// thing humans edit when adding a rule — the metadata comes from YAML.
var resolverMap = map[string]model.ResolverFunc{
	"P1.LOC.001": resolvers.LocP1LOC001,
	"P1.LOC.002": resolvers.LocP1LOC002,
	"P1.LOC.003": resolvers.LocP1LOC003,
	"P1.LOC.004": resolvers.LocP1LOC004,
	"P1.LOC.005": resolvers.LocP1LOC005,
	"P1.LOC.006": resolvers.LocP1LOC006,
	"P1.LOC.009": resolvers.LocP1LOC009,
	"P2.SPC.001": resolvers.SpcP2SPC001,
	"P2.SPC.002": resolvers.SpcP2SPC002,
	"P2.SPC.004": resolvers.SpcP2SPC004,
	"P3.EXP.001": resolvers.ExpP3EXP001,
	"P3.EXP.002": resolvers.ExpP3EXP002,
	"P3.EXP.003": resolvers.ExpP3EXP003,
	"P3.EXP.005": resolvers.ExpP3EXP005,
	"P4.VER.001": resolvers.VerP4VER001,
	"P4.VER.002": resolvers.VerP4VER002,
	"P4.VER.003": resolvers.VerP4VER003,
	"P5.AGG.001": resolvers.AggP5AGG001,
	"P5.AGG.002": resolvers.AggP5AGG002,
	"P5.AGG.003": resolvers.AggP5AGG003,
	"P5.AGG.004": resolvers.AggP5AGG004,
	"P6.REV.001": resolvers.RevP6REV001,
	"P6.REV.002": resolvers.RevP6REV002,
	"P7.MRD.001": resolvers.MrdP7MRD001,
}

// Rules returns the core-pack rules with resolvers wired in.
// Metadata comes from GeneratedRules() (YAML source of truth);
// resolvers come from resolverMap (Go source of truth).
func Rules() []model.Rule {
	generated := GeneratedRules()
	rules := make([]model.Rule, len(generated))
	for i, r := range generated {
		r.Resolver = resolverMap[r.ID]
		rules[i] = r
	}
	return rules
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
