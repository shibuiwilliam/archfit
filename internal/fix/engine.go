package fix

import (
	"context"
	"errors"
	"fmt"
	iofs "io/fs"
	"path/filepath"
	"time"

	adaptfs "github.com/shibuiwilliam/archfit/internal/adapter/fs"
	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// Engine orchestrates the plan-apply-verify loop for auto-fixes.
// It holds a registry of Fixer implementations keyed by rule ID.
// All filesystem access goes through the injected adaptfs.FS adapter
// (CLAUDE.md §3 P5 — no direct os calls).
type Engine struct {
	fixers map[string]Fixer
	fs     adaptfs.FS
}

// NewEngine returns an Engine backed by the real filesystem.
func NewEngine() *Engine {
	return &Engine{fixers: map[string]Fixer{}, fs: adaptfs.NewReal()}
}

// NewEngineWithFS returns an Engine backed by the given filesystem adapter.
// Use adaptfs.NewMemory() in tests.
func NewEngineWithFS(fsys adaptfs.FS) *Engine {
	return &Engine{fixers: map[string]Fixer{}, fs: fsys}
}

// Register adds a fixer for a specific rule. If a fixer for the same rule ID
// is already registered, the new one replaces it.
func (e *Engine) Register(f Fixer) {
	e.fixers[f.RuleID()] = f
}

// Input is what the CLI passes to the fix engine.
type Input struct {
	// Root is the repository root directory.
	Root string
	// RuleIDs limits fixing to these rules. Empty means all fixable rules.
	RuleIDs []string
	// DryRun causes the engine to return the plan without applying changes.
	DryRun bool
	// Facts is the read-only view of collected repo facts.
	Facts model.FactStore
	// Findings is the set of findings from the most recent scan.
	Findings []model.Finding
	// Scanner re-scans the repo after fixes are applied. It is injected so the
	// engine is decoupled from collectors and adapters.
	Scanner func(ctx context.Context) (core.ScanResult, error)
	// LogPath is the path for the audit log. Empty uses DefaultLogPath relative
	// to Root.
	LogPath string
}

// AppliedFix records a single fix that was applied and its verification status.
type AppliedFix struct {
	RuleID   string   `json:"rule_id"`
	Files    []string `json:"files"`
	Verified bool     `json:"verified"`
}

// Result is returned by Fix with the plan, applied changes, and
// verification outcome.
type Result struct {
	Plan      Plan            `json:"plan"`
	Applied   []AppliedFix    `json:"applied,omitempty"`
	Verified  bool            `json:"verified"`
	NewIssues []model.Finding `json:"new_issues,omitempty"`
}

// Fix runs the full loop: plan, apply, verify, and rollback on failure.
func (e *Engine) Fix(ctx context.Context, input Input) (Result, error) {
	// 1. Build the plan: match findings to registered fixers.
	plan, err := e.buildPlan(ctx, input)
	if err != nil {
		return Result{}, fmt.Errorf("building fix plan: %w", err)
	}

	result := Result{Plan: plan}

	if len(plan.Fixes) == 0 {
		result.Verified = true
		return result, nil
	}

	// 2. If dry-run, return the plan without applying.
	if input.DryRun {
		return result, nil
	}

	// 3. Snapshot existing files for rollback.
	snapshots, err := e.snapshot(input.Root, plan)
	if err != nil {
		return result, fmt.Errorf("snapshotting files: %w", err)
	}

	// 4. Apply changes.
	applied, err := e.apply(input.Root, plan)
	if err != nil {
		_ = e.rollback(input.Root, snapshots)
		return result, fmt.Errorf("applying fixes: %w", err)
	}
	result.Applied = applied

	// 5. Re-scan to verify fixes.
	if input.Scanner == nil {
		// No scanner provided; skip verification.
		result.Verified = false
		e.logFixes(input, applied, false, "")
		return result, nil
	}

	scanResult, err := input.Scanner(ctx)
	if err != nil {
		_ = e.rollback(input.Root, snapshots)
		return result, fmt.Errorf("verification scan: %w", err)
	}

	// 6. Check that fixed findings disappeared and no new ones appeared.
	fixedRuleIDs := make(map[string]bool)
	for _, pf := range plan.Fixes {
		fixedRuleIDs[pf.RuleID] = true
	}

	originalRuleIDs := make(map[string]bool)
	for _, f := range input.Findings {
		originalRuleIDs[f.RuleID] = true
	}

	verified := true
	var newIssues []model.Finding

	for _, f := range scanResult.Findings {
		if fixedRuleIDs[f.RuleID] {
			// A finding for a rule we tried to fix still exists — fix failed.
			verified = false
		}
		if !originalRuleIDs[f.RuleID] {
			// A finding for a rule that was clean before — regression.
			newIssues = append(newIssues, f)
		}
	}

	if len(newIssues) > 0 {
		verified = false
	}

	result.Verified = verified
	result.NewIssues = newIssues

	// 7. Rollback if verification failed.
	if !verified {
		_ = e.rollback(input.Root, snapshots)
		// Mark applied fixes as unverified.
		for i := range result.Applied {
			result.Applied[i].Verified = false
		}
		e.logFixes(input, result.Applied, false, "verification failed; rolled back")
		return result, nil
	}

	// Mark all applied fixes as verified.
	for i := range result.Applied {
		result.Applied[i].Verified = true
	}

	// 8. Log successful fixes.
	e.logFixes(input, result.Applied, true, "")

	return result, nil
}

// buildPlan matches findings to fixers and calls Plan on each.
func (e *Engine) buildPlan(ctx context.Context, input Input) (Plan, error) {
	wantedRules := make(map[string]bool)
	for _, id := range input.RuleIDs {
		wantedRules[id] = true
	}

	var fixes []PlannedFix
	for _, finding := range input.Findings {
		// Filter by requested rule IDs.
		if len(wantedRules) > 0 && !wantedRules[finding.RuleID] {
			continue
		}

		fixer, ok := e.fixers[finding.RuleID]
		if !ok {
			continue
		}

		changes, err := fixer.Plan(ctx, finding, input.Facts)
		if err != nil {
			return Plan{}, fmt.Errorf("planning fix for %s: %w", finding.RuleID, err)
		}

		fixes = append(fixes, PlannedFix{
			RuleID:   finding.RuleID,
			Finding:  finding,
			Changes:  changes,
			NeedsLLM: fixer.NeedsLLM(),
		})
	}

	return Plan{Fixes: fixes}, nil
}

// fileSnapshot records the original state of a file for rollback.
type fileSnapshot struct {
	path    string
	existed bool
	content []byte
}

// snapshot reads the current content of all files that will be changed.
func (e *Engine) snapshot(root string, plan Plan) ([]fileSnapshot, error) {
	seen := make(map[string]bool)
	var snapshots []fileSnapshot

	for _, pf := range plan.Fixes {
		for _, c := range pf.Changes {
			absPath := filepath.Join(root, c.Path)
			if seen[absPath] {
				continue
			}
			seen[absPath] = true

			data, err := e.fs.ReadFile(absPath)
			if isNotExist(err) {
				snapshots = append(snapshots, fileSnapshot{path: absPath, existed: false})
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", c.Path, err)
			}
			snapshots = append(snapshots, fileSnapshot{path: absPath, existed: true, content: data})
		}
	}
	return snapshots, nil
}

// apply writes all changes through the adapter.
func (e *Engine) apply(root string, plan Plan) ([]AppliedFix, error) {
	var applied []AppliedFix

	for _, pf := range plan.Fixes {
		var files []string
		for _, c := range pf.Changes {
			absPath := filepath.Join(root, c.Path)
			if err := e.fs.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
				return applied, fmt.Errorf("creating directory for %s: %w", c.Path, err)
			}

			switch c.Action {
			case ActionCreate:
				if err := e.fs.WriteFile(absPath, c.Content, 0o644); err != nil {
					return applied, fmt.Errorf("creating %s: %w", c.Path, err)
				}
			case ActionModify:
				if err := e.fs.WriteFile(absPath, c.Content, 0o644); err != nil {
					return applied, fmt.Errorf("modifying %s: %w", c.Path, err)
				}
			case ActionAppend:
				existing, err := e.fs.ReadFile(absPath)
				if err != nil && !isNotExist(err) {
					return applied, fmt.Errorf("reading %s for append: %w", c.Path, err)
				}
				combined := make([]byte, len(existing)+len(c.Content))
				copy(combined, existing)
				copy(combined[len(existing):], c.Content)
				if err := e.fs.WriteFile(absPath, combined, 0o644); err != nil {
					return applied, fmt.Errorf("appending to %s: %w", c.Path, err)
				}
			default:
				return applied, fmt.Errorf("unknown action %q for %s", c.Action, c.Path)
			}
			files = append(files, c.Path)
		}
		applied = append(applied, AppliedFix{RuleID: pf.RuleID, Files: files})
	}
	return applied, nil
}

// rollback restores all snapshotted files to their original state.
func (e *Engine) rollback(root string, snapshots []fileSnapshot) error {
	var firstErr error
	for _, s := range snapshots {
		if !s.existed {
			if err := e.fs.Remove(s.path); err != nil && !isNotExist(err) {
				if firstErr == nil {
					firstErr = err
				}
			}
			continue
		}
		if err := e.fs.WriteFile(s.path, s.content, 0o644); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// logFixes appends one LogEntry per applied fix.
func (e *Engine) logFixes(input Input, applied []AppliedFix, verified bool, errMsg string) {
	logPath := input.LogPath
	if logPath == "" {
		logPath = filepath.Join(input.Root, DefaultLogPath)
	}

	ts := time.Now().UTC().Format(time.RFC3339)
	for _, a := range applied {
		action := "applied"
		if !verified {
			action = "rolled_back"
		}
		_ = AppendLogFS(e.fs, logPath, LogEntry{
			Timestamp: ts,
			RuleID:    a.RuleID,
			Action:    action,
			Files:     a.Files,
			Verified:  verified,
			Error:     errMsg,
		})
	}
}

// isNotExist checks whether an error indicates a file does not exist.
func isNotExist(err error) bool {
	return err != nil && errors.Is(err, iofs.ErrNotExist)
}
