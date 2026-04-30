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
	"runtime"
	"sort"
	"sync"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// parallelThreshold is the minimum rule count for concurrent evaluation.
// Below this, serial execution is simpler and equally fast.
const parallelThreshold = 8

// Pack describes a rule pack's metadata: its identity, the principles it
// covers, and how many rules it contributes.
type Pack struct {
	Name        string
	Version     string
	Description string
	Principles  []model.Principle
	RuleCount   int
}

// Registry holds the set of rules known to the program. It is not populated
// by import side effects — callers Register explicitly at startup.
type Registry struct {
	mu       sync.RWMutex
	rules    map[string]model.Rule
	packs    map[string][]string // pack -> rule IDs
	packMeta map[string]Pack     // pack name -> metadata
}

// NewRegistry returns an empty rule registry.
func NewRegistry() *Registry {
	return &Registry{
		rules:    map[string]model.Rule{},
		packs:    map[string][]string{},
		packMeta: map[string]Pack{},
	}
}

// Register adds rules to the registry under the given pack name.
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

// Rule returns the rule with the given ID, if registered.
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

// RegisterPack stores pack metadata alongside rule registration.
func (r *Registry) RegisterPack(p Pack) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.packMeta[p.Name] = p
}

// PackInfo returns metadata for a named pack.
func (r *Registry) PackInfo(name string) (Pack, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.packMeta[name]
	return p, ok
}

// AllPacks returns metadata for all registered packs, sorted by name.
func (r *Registry) AllPacks() []Pack {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Pack, 0, len(r.packMeta))
	for _, p := range r.packMeta {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
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

// NewEngine returns a new rule evaluation engine.
func NewEngine() *Engine { return &Engine{} }

// EvalResult is what the scheduler returns to callers.
type EvalResult struct {
	Findings          []model.Finding
	Metrics           []model.Metric
	RulesEvaluated    int
	RulesWithFindings int      // count of rules that produced ≥1 finding
	SkippedRuleIDs    []string // rules skipped due to applies_to mismatch
	Errors            []EvalError
}

// EvalError captures a non-fatal resolver failure. Per CLAUDE.md §13, parse
// failures should be findings — but an unhandled resolver panic/error becomes
// a EvalError so it remains visible.
type EvalError struct {
	RuleID string
	Err    error
}

func (e EvalError) Error() string { return e.RuleID + ": " + e.Err.Error() }

// Evaluate runs all rules against facts and returns the aggregated result.
// Rules whose AppliesTo.Languages don't match the repo's detected languages
// are skipped — they don't count toward RulesEvaluated or scoring weight.
//
// When len(rules) >= parallelThreshold and runtime.NumCPU() > 1, resolvers
// run concurrently. Determinism is preserved: results are collected into
// per-rule slots and merged in rule-ID order before the final sort.
func (e *Engine) Evaluate(ctx context.Context, rules []model.Rule, facts model.FactStore) EvalResult {
	repoLangs := facts.Languages()

	// Partition: determine which rules to evaluate vs skip.
	type indexedRule struct {
		index int
		rule  model.Rule
	}
	var toEval []indexedRule
	var res EvalResult
	for i, r := range rules {
		if shouldSkip(r, repoLangs) {
			res.SkippedRuleIDs = append(res.SkippedRuleIDs, r.ID)
			continue
		}
		toEval = append(toEval, indexedRule{index: i, rule: r})
	}

	// Per-rule result slot — one per toEval entry.
	type ruleResult struct {
		findings []model.Finding
		metrics  []model.Metric
		err      error
	}
	slots := make([]ruleResult, len(toEval))

	if len(toEval) >= parallelThreshold && runtime.NumCPU() > 1 {
		// Parallel path: bounded concurrency via semaphore.
		sem := make(chan struct{}, runtime.NumCPU())
		var wg sync.WaitGroup
		for idx, ir := range toEval {
			if ctx.Err() != nil {
				slots[idx] = ruleResult{err: ctx.Err()}
				continue
			}
			wg.Add(1)
			go func(slot int, r model.Rule) {
				defer wg.Done()
				sem <- struct{}{}        // acquire
				defer func() { <-sem }() // release
				f, m, err := evalOne(ctx, r, facts)
				slots[slot] = ruleResult{findings: f, metrics: m, err: err}
			}(idx, ir.rule)
		}
		wg.Wait()
	} else {
		// Serial path: same behavior as before.
		for idx, ir := range toEval {
			if ctx.Err() != nil {
				slots[idx] = ruleResult{err: ctx.Err()}
				continue
			}
			f, m, err := evalOne(ctx, ir.rule, facts)
			slots[idx] = ruleResult{findings: f, metrics: m, err: err}
		}
	}

	// Merge slots in original rule order → deterministic.
	for idx, ir := range toEval {
		slot := slots[idx]
		res.RulesEvaluated++
		if slot.err != nil {
			res.Errors = append(res.Errors, EvalError{RuleID: ir.rule.ID, Err: slot.err})
			continue
		}
		// Back-fill fields the resolver left blank.
		for i := range slot.findings {
			backfill(&slot.findings[i], ir.rule)
		}
		if len(slot.findings) > 0 {
			res.RulesWithFindings++
		}
		res.Findings = append(res.Findings, slot.findings...)
		res.Metrics = append(res.Metrics, slot.metrics...)
	}

	model.SortFindings(res.Findings)
	sort.SliceStable(res.Metrics, func(i, j int) bool { return res.Metrics[i].Name < res.Metrics[j].Name })
	sort.SliceStable(res.Errors, func(i, j int) bool { return res.Errors[i].RuleID < res.Errors[j].RuleID })
	return res
}

// backfill centralizes field defaults so resolvers stay terse.
func backfill(f *model.Finding, r model.Rule) {
	if f.RuleID == "" {
		f.RuleID = r.ID
	}
	if f.Principle == "" {
		f.Principle = r.Principle
	}
	if f.Severity == "" {
		f.Severity = r.Severity
	}
	if f.EvidenceStrength == "" {
		f.EvidenceStrength = r.EvidenceStrength
	}
	if f.Remediation.Summary == "" {
		f.Remediation = r.Remediation
	}
	if f.Evidence == nil {
		f.Evidence = map[string]any{}
	}
}

// shouldSkip returns true when a rule declares AppliesTo.Languages and none
// of those languages appear in the repo's detected file counts. Rules without
// language constraints always run.
func shouldSkip(r model.Rule, repoLangs map[string]int) bool {
	if len(r.AppliesTo.Languages) == 0 {
		return false // no constraint → runs everywhere
	}
	for _, lang := range r.AppliesTo.Languages {
		if repoLangs[lang] > 0 {
			return false // at least one required language is present
		}
	}
	return true // none of the required languages detected
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
