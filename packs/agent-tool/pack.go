// Package agenttool wires archfit's agent-tool rule pack. Registration is
// explicit (cmd/archfit/main.go) — no init() side effects.
//
// Rule metadata comes from GeneratedRules() produced by `make generate`.
// This file provides only the resolver wiring.
package agenttool

import (
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/rule"
	"github.com/shibuiwilliam/archfit/packs/agent-tool/resolvers"
)

// PackName is the identifier for the agent-tool rule pack.
const PackName = "agent-tool"

// resolverMap wires rule IDs to resolver functions.
var resolverMap = map[string]model.ResolverFunc{
	"P2.SPC.010": resolvers.P2SPC010,
	"P7.MRD.002": resolvers.P7MRD002,
	"P7.MRD.003": resolvers.P7MRD003,
}

// Rules returns all rules with resolvers wired in.
func Rules() []model.Rule {
	generated := GeneratedRules()
	rules := make([]model.Rule, len(generated))
	for i, r := range generated {
		r.Resolver = resolverMap[r.ID]
		rules[i] = r
	}
	return rules
}

// Register adds the agent-tool rules to reg.
func Register(reg *rule.Registry) error {
	if err := reg.Register(PackName, Rules()...); err != nil {
		return err
	}
	reg.RegisterPack(rule.Pack{
		Name:        PackName,
		Version:     "0.3.0",
		Description: "Rules for repositories whose consumers are coding agents",
		Principles:  []model.Principle{model.P2SpecFirst, model.P7MachineReadability},
		RuleCount:   len(Rules()),
	})
	return nil
}
