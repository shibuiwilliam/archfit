---
title: "ADR 0008: Contract exit codes"
date: 2026-04-26
status: accepted
---

# ADR 0008 — Contract Exit Codes

## Context

The fitness contract system (`archfit contract check`) needs a way to
distinguish between three outcomes in CI:

1. All hard constraints met, all soft targets met → success
2. All hard constraints met, some soft targets missed → advisory
3. Hard constraint violated → failure

The existing exit codes (0–4) cover scan-level outcomes. Contract checking
introduces a new advisory state that is neither "success" nor "failure."

## Decision

Add exit code **5** for `archfit contract check`:

| Code | Meaning |
|------|---------|
| 0 | All hard constraints AND soft targets met |
| 1 | Hard constraint violated (blocks CI) |
| 5 | Soft target missed, no hard violations (advisory) |

Exit codes 2 (usage), 3 (runtime), and 4 (config) retain their existing
meanings for the contract commands.

## Rationale

- Exit 0 vs 5 lets CI pipelines distinguish "everything is good" from
  "everything required is good, but aspirational targets are not met."
- Teams can configure CI to allow exit 5 (`|| [ $? -eq 5 ]`) while blocking
  exit 1, giving them visibility into soft targets without breaking builds.
- Exit 5 was chosen because it is the next unused code and does not conflict
  with common Unix conventions (1 = general error, 2 = usage).

## Consequences

- `docs/exit-codes.md` must be updated to document exit code 5.
- The `archfit contract check` command is the only command that emits exit 5.
  `archfit scan` continues to use only 0–4.
