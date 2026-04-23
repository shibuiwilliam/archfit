// Package rule provides the rule engine: a Registry (explicit wiring) and an
// Engine that evaluates resolvers against a FactStore.
//
// Deliberately simple, per CLAUDE.md §2: no reflection, no init-time
// registration, no interface-per-struct. Rules are added via explicit Register
// calls from cmd/archfit/main.go.
package rule

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// Registry holds the set of rules known to the program. It is not populated
// by import side effects — callers Register explicitly at startup.
type Registry struct {
	mu    sync.RWMutex
	rules map[string]model.Rule
	packs map[string][]string // pack -> rule IDs
}

func NewRegistry() *Registry {
	return &Registry{
		rules: map[string]model.Rule{},
		packs: map[string][]string{},
	}
}

func (r *Registry) Register(pack string, rules ...model.Rule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rule := range rules {
		if err := rule.Validate(); err != nil {
			return err
		}
		if _, exists := r.rules[rule.ID]; exists {
			return fmt.Errorf("rule %s already registered", rule.ID)
		}
		r.rules[rule.ID] = rule
		r.packs[pack] = append(r.packs[pack], rule.ID)
	}
	return nil
}

// Rules returns all registered rules in ID order.
func (r *Registry) Rules() []model.Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]model.Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		out = append(out, rule)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r *Registry) Rule(id string) (model.Rule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.rules[id]
	return rule, ok
}

// Packs returns pack name -> rule IDs in ID order.
func (r *Registry) Packs() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string][]string, len(r.packs))
	for name, ids := range r.packs {
		cp := append([]string(nil), ids...)
		sort.Strings(cp)
		out[name] = cp
	}
	return out
}

// Filter returns rules whose IDs match includeIDs (or all if empty) and that
// are not suppressed by excludeIDs.
func (r *Registry) Filter(includeIDs, excludeIDs []string) []model.Rule {
	include := toSet(includeIDs)
	exclude := toSet(excludeIDs)
	rules := r.Rules()
	if len(include) == 0 && len(exclude) == 0 {
		return rules
	}
	out := rules[:0:0]
	for _, rule := range rules {
		if len(include) > 0 {
			if _, ok := include[rule.ID]; !ok {
				continue
			}
		}
		if _, blocked := exclude[rule.ID]; blocked {
			continue
		}
		out = append(out, rule)
	}
	return out
}

func toSet(ss []string) map[string]struct{} {
	if len(ss) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return m
}

// Engine executes a set of rules against a FactStore. Resolvers run sequentially;
// the cost of rule evaluation in Phase 1 is negligible, and serial execution
// keeps output deterministic without synchronization effort.
type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

// EvalResult is what the scheduler returns to callers.
type EvalResult struct {
	Findings       []model.Finding
	Metrics        []model.Metric
	RulesEvaluated int
	Errors         []RuleError
}

// RuleError captures a non-fatal resolver failure. Per CLAUDE.md §13, parse
// failures should be findings — but an unhandled resolver panic/error becomes
// a RuleError so it remains visible.
type RuleError struct {
	RuleID string
	Err    error
}

func (e RuleError) Error() string { return e.RuleID + ": " + e.Err.Error() }

func (e *Engine) Evaluate(ctx context.Context, rules []model.Rule, facts model.FactStore) EvalResult {
	var res EvalResult
	for _, rule := range rules {
		if ctx.Err() != nil {
			res.Errors = append(res.Errors, RuleError{RuleID: rule.ID, Err: ctx.Err()})
			continue
		}
		findings, metrics, err := evalOne(ctx, rule, facts)
		res.RulesEvaluated++
		if err != nil {
			res.Errors = append(res.Errors, RuleError{RuleID: rule.ID, Err: err})
			continue
		}
		// Back-fill any fields the resolver left blank — centralizing the
		// contract keeps resolvers terse and consistent.
		for i := range findings {
			if findings[i].RuleID == "" {
				findings[i].RuleID = rule.ID
			}
			if findings[i].Principle == "" {
				findings[i].Principle = rule.Principle
			}
			if findings[i].Severity == "" {
				findings[i].Severity = rule.Severity
			}
			if findings[i].EvidenceStrength == "" {
				findings[i].EvidenceStrength = rule.EvidenceStrength
			}
			if findings[i].Remediation.Summary == "" {
				findings[i].Remediation = rule.Remediation
			}
			if findings[i].Evidence == nil {
				findings[i].Evidence = map[string]any{}
			}
		}
		res.Findings = append(res.Findings, findings...)
		res.Metrics = append(res.Metrics, metrics...)
	}
	model.SortFindings(res.Findings)
	sort.SliceStable(res.Metrics, func(i, j int) bool { return res.Metrics[i].Name < res.Metrics[j].Name })
	sort.SliceStable(res.Errors, func(i, j int) bool { return res.Errors[i].RuleID < res.Errors[j].RuleID })
	return res
}

func evalOne(ctx context.Context, rule model.Rule, facts model.FactStore) (findings []model.Finding, metrics []model.Metric, err error) {
	// Resolver panics must not take the scan down.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("resolver panic: %v", r)
		}
	}()
	return rule.Resolver(ctx, facts)
}
