// Package resolvers holds the pure ResolverFunc implementations for the core pack.
// Every function in this package must be a pure function of model.FactStore.
// No imports from internal/adapter, internal/collector, os, or io/fs.
package resolvers

import (
	"context"
	"path"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// sliceContainers are well-known directory names that, if present at the repo
// root, strongly imply a vertical-slice layout. A repo that doesn't use any of
// these is not penalized by P1.LOC.002.
var sliceContainers = []string{
	"packs", "services", "modules",
	// Monorepo conventions (Lerna, Turborepo, NX, Yarn workspaces)
	"packages", "apps", "libs",
	// Plugin / extension architectures (Rails engines, WordPress, etc.)
	"plugins", "engines", "components",
	// Domain-driven design
	"domains", "features",
}

// LocP1LOC001 fires when neither CLAUDE.md nor AGENTS.md exists at the repo root.
func LocP1LOC001(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()
	if _, ok := repo.ByPath["CLAUDE.md"]; ok {
		return nil, nil, nil
	}
	if _, ok := repo.ByPath["AGENTS.md"]; ok {
		return nil, nil, nil
	}
	// Case-insensitive fallback — some repos ship `Claude.md`, `agents.md`, etc.
	for _, p := range append(append([]string(nil), repo.ByBase["claude.md"]...), repo.ByBase["agents.md"]...) {
		if !strings.Contains(p, "/") {
			return nil, nil, nil
		}
	}
	return []model.Finding{{
		Message:    "no CLAUDE.md or AGENTS.md at repo root — agents have no canonical entry point",
		Confidence: 0.99,
		Evidence: map[string]any{
			"checked_paths": []string{"CLAUDE.md", "AGENTS.md"},
		},
	}}, nil, nil
}

// LocP1LOC002 fires for each vertical slice container child that lacks an AGENTS.md.
// The rule only applies when at least one known container directory exists at the root.
func LocP1LOC002(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()
	children := sliceChildren(repo)
	if len(children) == 0 {
		return nil, nil, nil
	}
	var findings []model.Finding
	for _, child := range children {
		hasAgents := false
		for _, candidate := range []string{child + "/AGENTS.md", child + "/CLAUDE.md"} {
			if _, ok := repo.ByPath[candidate]; ok {
				hasAgents = true
				break
			}
		}
		if hasAgents {
			continue
		}
		findings = append(findings, model.Finding{
			Path:       child,
			Message:    "vertical slice is missing an AGENTS.md",
			Confidence: 0.95,
			Evidence: map[string]any{
				"slice": child,
				"looked_for": []string{
					child + "/AGENTS.md",
					child + "/CLAUDE.md",
				},
			},
		})
	}
	return findings, nil, nil
}

// sliceChildren returns sorted, repo-relative paths like "packs/core" for every
// immediate child of a known slice-container directory that contains source files.
func sliceChildren(repo model.RepoFacts) []string {
	// Which container dirs exist at root?
	present := map[string]bool{}
	for _, container := range sliceContainers {
		for _, f := range repo.Files {
			if strings.HasPrefix(f.Path, container+"/") {
				present[container] = true
				break
			}
		}
	}
	if len(present) == 0 {
		return nil
	}
	seen := map[string]bool{}
	for _, f := range repo.Files {
		parts := strings.SplitN(f.Path, "/", 3)
		if len(parts) < 3 {
			continue
		}
		if !present[parts[0]] {
			continue
		}
		// Skip slice-container files that live at the container level (e.g. packs/README.md).
		seen[path.Join(parts[0], parts[1])] = true
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
