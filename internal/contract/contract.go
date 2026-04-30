// Package contract implements fitness contracts — machine-executable
// declarations of what architectural fitness means for a specific repository.
//
// A contract defines hard constraints (must satisfy), soft targets
// (should work toward), area budgets (SRE-style finding budgets per path),
// and agent directives (instructions for coding agents).
//
// Parsing uses sigs.k8s.io/yaml which accepts both YAML 1.2 and JSON.
// Checking is a pure function: no I/O, no imports from adapter/collector.
package contract

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	// sigs.k8s.io/yaml preserves `json:"..."` tag semantics on Go structs.
	// Justified in docs/dependencies.md.
	"sigs.k8s.io/yaml"
)

// Contract is the deserialized .archfit-contract.yaml.
type Contract struct {
	Version         int              `json:"version"`
	HardConstraints []Constraint     `json:"hard_constraints,omitempty"`
	SoftTargets     []Target         `json:"soft_targets,omitempty"`
	AreaBudgets     []AreaBudget     `json:"area_budgets,omitempty"`
	AgentDirectives []AgentDirective `json:"agent_directives,omitempty"`
}

// Constraint defines a hard requirement that must be satisfied.
// Exactly one of Principle or Rule should be set.
type Constraint struct {
	Principle   string  `json:"principle,omitempty"` // "P1", "P4", "overall"
	Rule        string  `json:"rule,omitempty"`      // "P5.AGG.001"
	MinScore    float64 `json:"min_score,omitempty"` // min acceptable score
	MaxFindings int     `json:"max_findings"`        // max acceptable findings count
	Scope       string  `json:"scope"`               // glob pattern ("**" = all)
	Rationale   string  `json:"rationale,omitempty"`
}

// Target defines an aspirational goal the team is working toward.
type Target struct {
	Principle   string  `json:"principle,omitempty"`
	Metric      string  `json:"metric,omitempty"`
	TargetScore float64 `json:"target_score,omitempty"`
	TargetValue float64 `json:"target,omitempty"`
	Current     float64 `json:"current,omitempty"`
	Deadline    string  `json:"deadline,omitempty"`
}

// AreaBudget is an SRE-style finding budget for a specific path area.
type AreaBudget struct {
	Path                string   `json:"path"` // glob pattern
	MaxFindings         int      `json:"max_findings"`
	MaxNewFindingsPerPR int      `json:"max_new_findings_per_pr"`
	Principles          []string `json:"principles,omitempty"` // empty = all
	Owner               string   `json:"owner,omitempty"`
}

// AgentDirective is a machine-readable instruction for coding agents.
type AgentDirective struct {
	When   string `json:"when"`   // condition expression
	Action string `json:"action"` // instruction text
}

// candidates are the filenames searched for, in priority order.
var candidates = []string{".archfit-contract.yaml", ".archfit-contract.yml", ".archfit-contract.json"}

// Load reads a contract from root, returning an empty contract if none exists.
func Load(root string) (c Contract, path string, found bool, err error) {
	for _, name := range candidates {
		p := filepath.Join(root, name)
		data, err := os.ReadFile(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return Contract{}, p, true, err
		}
		c, err := parse(data)
		if err != nil {
			return Contract{}, p, true, fmt.Errorf("%s: %w\nhint: archfit reads YAML 1.2; check indentation and quoting", p, err)
		}
		return c, p, true, nil
	}
	return Contract{Version: 1}, "", false, nil
}

// LoadFile reads a contract from an explicit path.
func LoadFile(path string) (Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, err
	}
	c, err := parse(data)
	if err != nil {
		return Contract{}, fmt.Errorf("%s: %w\nhint: archfit reads YAML 1.2; check indentation and quoting", path, err)
	}
	return c, nil
}

func parse(data []byte) (Contract, error) {
	var c Contract
	if err := yaml.UnmarshalStrict(data, &c); err != nil {
		return Contract{}, err
	}
	if err := c.Validate(); err != nil {
		return Contract{}, err
	}
	return c, nil
}

// Validate checks that all contract fields are well-formed.
func (c Contract) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported contract version %d (want 1)", c.Version)
	}
	for i, hc := range c.HardConstraints {
		if hc.Principle == "" && hc.Rule == "" {
			return fmt.Errorf("hard_constraints[%d]: one of principle or rule is required", i)
		}
		if hc.Scope == "" {
			return fmt.Errorf("hard_constraints[%d]: scope is required", i)
		}
	}
	for i, ab := range c.AreaBudgets {
		if ab.Path == "" {
			return fmt.Errorf("area_budgets[%d]: path is required", i)
		}
	}
	for i, d := range c.AgentDirectives {
		if d.When == "" || d.Action == "" {
			return fmt.Errorf("agent_directives[%d]: when and action are required", i)
		}
	}
	return nil
}
