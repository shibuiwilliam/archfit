---
id: 0014
title: Stability re-tiering — walk back blanket-stable freeze for uncalibrated rules
status: accepted
date: 2026-05-02
supersedes: null
---

## Context

ADR 0012 promoted all 17 rules to `stability: stable` and froze rule IDs.
This was appropriate for the ID freeze, but it also froze *behavioral
promises* for rules whose thresholds and detection logic have never been
validated against a calibration corpus.

PROJECT.md §3.7 identifies three rules with known calibration gaps:

- **P1.LOC.003** (dependency graph coupling bounded) — the threshold
  (`max_reach ≤ 10 packages`) was chosen by intuition, not data.
- **P1.LOC.004** (commits touch bounded files) — the threshold
  (`fan-out ≤ 8 files`) was chosen by intuition, not data.
- **P5.AGG.001** (security-sensitive files concentrated) — known false
  positives on test fixture paths (testdata/, packs/*/fixtures/).

Phase 1 introduces a calibration corpus (§6.1.5). Threshold and detection
logic changes must be permitted on these three rules so that calibration
data can drive them. The `stable` tier prevents this without an ADR per
change.

## Decision

Walk back the stability tier for three rules from `stable` to
`experimental`:

| Rule ID | Current | New | Reason |
|---|---|---|---|
| P1.LOC.003 | stable | experimental | Threshold uncalibrated |
| P1.LOC.004 | stable | experimental | Threshold uncalibrated |
| P5.AGG.001 | stable | experimental | False positives on fixture paths |

**What stays frozen (per ADR 0012):**

- Rule IDs — no renumbering or repurposing.
- Severity — changes still require an ADR.
- Output schema fields — unchanged.

**What becomes flexible:**

- Detection logic (resolver implementation, keyword lists, thresholds).
- Evidence interpretation (e.g., excluding fixture paths from scatter count).
- Confidence values.

These rules will be re-promoted to `stable` only after:

1. A calibration corpus run shows precision ≥ 0.85.
2. At least one full release cycle as `experimental`.

## Consequences

- `TestStability_AllRulesAreStable` is updated to allow these three rules
  as `experimental`, documented by this ADR.
- Phase 1 rule additions will also enter as `experimental` — the test is
  relaxed to use an allowlist rather than a blanket check.
- Consumers who depend on the *behavior* (not just the ID) of these rules
  should pin to a specific archfit version until the rules return to
  `stable`.
