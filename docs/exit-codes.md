# archfit — exit codes

These codes are **part of the stability contract**. Per `CLAUDE.md` §8 and §11,
renumbering or repurposing any code requires an ADR and a major-version bump.

| Code | Meaning |
|:---:|---|
| `0` | Success. For `scan`, also returned when all findings are below the `--fail-on` threshold. |
| `1` | `scan` found at least one finding at or above the `--fail-on` threshold (default: `error`). |
| `2` | Usage error — unknown subcommand, bad flag, missing required argument. |
| `3` | Runtime error — an unexpected failure while scanning (e.g., unreadable target path, resolver panic that escaped recovery). |
| `4` | Configuration error — `.archfit.yaml` failed to parse or validate. |

## Notes for agents

- Treat `1` as a **contract**, not a crash. Re-reading the JSON output is the correct response.
- Never interpret `2`–`4` as "the repo has problems." They mean the run itself did not complete.
- Exit codes above `4` are reserved. If you see one, treat it as `3` and open an issue.

## Scope of this document

This document covers only the exit codes emitted by the `archfit` binary.
It does not cover:

- Exit codes from subprocesses archfit spawns (`git`, etc.) — those are
  observed inside the run and never propagate.
- Exit codes from a CI action wrapper around archfit — consult that
  wrapper's own documentation.
