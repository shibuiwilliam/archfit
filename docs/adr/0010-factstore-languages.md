---
id: 0010
title: Add Languages() to FactStore interface for applies_to activation
status: accepted
date: 2026-04-30
---

## Context

The `model.Rule` struct has an `AppliesTo.Languages` field (declared in the
rule YAML schema since day one), but the rule engine ignores it — every rule
runs against every repo. This causes false signals: a pure Go repo gets
penalized by P3.EXP.001 for not having `.env.example`, even though `.env`
is not a Go convention.

The data needed to implement filtering already exists: `RepoFacts.Languages`
is a `map[string]int` populated by the filesystem collector. It just isn't
exposed on the `FactStore` interface.

## Decision

1. Add `Languages() map[string]int` to `model.FactStore`. This is a
   public-interface change, requiring this ADR per CLAUDE.md §9.

2. The rule engine (`rule.Engine.Evaluate`) skips a rule when its
   `AppliesTo.Languages` is non-empty and none of the listed languages
   appear in `facts.Languages()`. Skipped rules are tracked in
   `EvalResult.SkippedRuleIDs`.

3. `score.Compute` accepts an optional `skippedRuleIDs` variadic. Skipped
   rules' weights are excluded from the denominator so they cannot penalize
   the score for ecosystems they don't target.

4. The code-generation tool (`genrules`) emits `AppliesTo` when the YAML
   declares `applies_to.languages`.

## Consequences

- All `FactStore` implementations (real, test fakes) must add the
  `Languages()` method. Test fakes return `nil`.
- Rules without `applies_to` continue to run everywhere (backward compatible).
- Adding language tags to a rule is a YAML-only change + `make generate`.
- A single-language repo no longer gets penalized by rules targeting other
  ecosystems.
