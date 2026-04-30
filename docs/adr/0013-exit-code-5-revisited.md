---
id: 0013
title: Exit code 5 revisited — soft-target miss signaling
status: proposed
date: 2026-04-30
supersedes: null
---

## Context

ADR 0008 introduced exit code 5 for `archfit contract check` to distinguish
"soft target missed" from "hard constraint violated" (exit 1) and "all good"
(exit 0). PROJECT.md §5.1 flagged it as "under review" because:

1. **CI friction.** Most CI systems treat any non-zero exit as failure. Teams
   that want advisory-only soft targets must add `|| [ $? -eq 5 ]` to their
   pipeline. This is fragile — a typo breaks the pipeline, and the intent
   (advisory vs blocking) is expressed in shell, not in archfit config.

2. **Agent confusion.** Coding agents (Claude Code, Copilot) typically treat
   non-zero exit as "something went wrong." Exit 5 requires agents to know
   archfit-specific semantics. The SKILL.md already warns about this, but
   it's an ongoing friction point.

3. **ADR 0012 froze exit codes 0–5.** Removing exit 5 after 1.0 would be a
   breaking change. This ADR is the last chance to decide before the freeze
   takes full effect.

## Options

### Option A: Keep exit 5 as-is

**How it works:** No change. `contract check` returns 0 (all met), 1 (hard
fail), or 5 (soft miss). CI must handle the three-way exit.

**Pros:**
- Already implemented and documented.
- Gives CI the most granular signal without parsing JSON.
- Teams that want to block on soft targets can treat 5 as failure.

**Cons:**
- Requires `|| [ $? -eq 5 ]` in CI for advisory-only mode.
- Non-standard: most tools use 0/1 only. Exit 5 is archfit-specific.
- Agents must special-case exit 5.

**Risk:** Low. It's working today. The CI workaround is well-documented.

---

### Option B: Replace exit 5 with a JSON field (exit 0)

**How it works:** `contract check` always exits 0 when hard constraints pass,
regardless of soft targets. Soft misses are reported in the JSON output via a
`contract_result.soft_misses` array and a top-level `advisory: true` field.
CI consumers parse the JSON to detect soft misses.

**Implementation:**
- `contract check` returns only 0 (pass) or 1 (hard fail).
- JSON output gains `"advisory": true` when soft targets are missed.
- Exit code 5 is removed from the code and reserved (never emitted).

**Pros:**
- CI pipelines "just work" — exit 0 = pass, exit 1 = fail.
- Agents see exit 0 and proceed. Advisory state is visible in output.
- Simpler mental model: non-zero = something broke.

**Cons:**
- **Breaking change** if done after 1.0 (ADR 0012 froze exit codes).
  Must land before the freeze or require a 2.0 bump.
- Teams that want CI to warn on soft misses must parse JSON or use
  `--fail-on-soft` (new flag, see Option C).
- Less granular than exit 5 for shell-only consumers.

**Risk:** Medium. Removes a signal some teams may already depend on.
Mitigated by landing before 1.0 freeze.

---

### Option C: Add a config flag controlling behavior

**How it works:** `.archfit.yaml` gains:
```yaml
contract:
  soft_target_exit: advisory  # "advisory" (exit 0) or "fail" (exit 1)
```

Default: `advisory` (exit 0 on soft miss). Teams that want CI to block on
soft targets set `soft_target_exit: fail` (exit 1, same as hard violation).
Exit 5 is removed entirely.

**Implementation:**
- New config field `contract.soft_target_exit` (default: `advisory`).
- `contract check` returns 0 (advisory mode + soft miss) or 1 (fail mode +
  soft miss, or hard violation).
- JSON output includes `"soft_misses"` array regardless of mode.
- Exit code 5 is retired.

**Pros:**
- Policy is in the config, not in CI shell scripts.
- Teams choose their own strictness without archfit-specific shell tricks.
- Simpler exit code contract: 0 = pass, 1 = fail, 2–4 = errors.
- Agents always see 0 or 1 — no special cases.

**Cons:**
- New config surface to maintain.
- Two modes means two code paths to test.
- Teams that already use exit 5 must migrate their CI config.

**Risk:** Low-medium. The config field is simple and well-scoped.

---

## Recommendation

**Option C** — config flag with `advisory` as default.

Rationale:
- It moves policy from CI shell into archfit config (P3: shallow explicitness).
- It eliminates the non-standard exit code (simplifies agents and CI).
- The default (`advisory`) matches what most teams want: don't break the
  build on aspirational goals.
- Teams that want strict enforcement set `soft_target_exit: fail` — clearer
  intent than `|| [ $? -eq 5 ]`.

## Decision

**Pending.** Waiting for maintainer decision before implementation.

## Migration path (if Option B or C is chosen)

1. Deprecate exit 5 in a minor release with a warning message.
2. Add the new behavior behind a flag (C) or unconditionally (B).
3. Remove exit 5 emission in the next minor release.
4. Reserve exit 5 (never reuse for a different meaning).
