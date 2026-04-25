# Agent Behavior Observatory

> Element 1 of the three strategic elements. Builds after the fitness contract.

## Overview

The observatory watches how coding agents actually interact with a repository and produces behavioral metrics that static analysis cannot capture. This is archfit's unique differentiator: no other tool observes the *agent* as it works.

## Key Insight

archfit currently asks "Is this repo shaped for agents?" by inspecting static structure. The observatory answers the transformative question: **"When an agent actually works on this repo, what happens?"**

## Behavioral Metrics

| Metric | What it measures | Principle |
|---|---|---|
| `agent_context_efficiency` | Files read vs. files needed for the change | P1 |
| `agent_retry_rate` | Failed commands / total commands | P4 |
| `agent_time_to_first_verify` | Seconds from first edit to first passing test | P4 |
| `agent_cross_boundary_reads` | Reads outside the task's vertical slice | P1 |
| `agent_dangerous_touches` | Edits in P5-flagged dangerous areas | P5 |
| `agent_rollback_frequency` | Self-reverts / total edits | P6 |

## Trace Schema

Traces are JSON Lines (one event per line) or a single JSON array:

```json
{
  "schema_version": "0.1.0",
  "agent": "claude-code",
  "session_id": "abc123",
  "repo_commit": "7563385",
  "events": [
    {"type": "file_read", "path": "internal/model/model.go", "ts": "..."},
    {"type": "file_write", "path": "internal/score/metrics.go", "ts": "..."},
    {"type": "command_run", "command": "make test", "exit_code": 0, "duration_ms": 4200}
  ]
}
```

Event types: `file_read`, `file_write`, `command_run`, `command_fail`, `tool_call`, `error`, `context_load`.

## Package Structure

```
internal/observer/
├── trace.go            # Trace, Event, EventType types
├── trace_test.go
├── ingest.go           # Parse trace files (JSON Lines format)
├── ingest_test.go
├── metrics.go          # Behavioral metric computation
├── metrics_test.go
├── hotspot.go          # Cross-reference with static findings
├── hotspot_test.go
└── testdata/           # Sample trace files
schemas/
├── trace.schema.json
└── observe-output.schema.json
```

## Implementation Steps

| Step | Description | Status | Effort |
|------|-------------|--------|--------|
| 1.1 | Trace schema and types | Not started | ~150 lines |
| 1.2 | Behavioral metrics computation | Not started | ~200 lines |
| 1.3 | Hotspot analysis | Not started | ~150 lines |
| 1.4 | CLI wiring (`archfit observe`) | Not started | ~200 lines |

## Architecture Rules

- The observer reads trace files (collected data). It never instruments or modifies the agent.
- All metric functions are pure (no I/O). They receive parsed traces.
- Hotspot analysis cross-references traces with static findings — no new I/O.
- The observer does NOT import from `internal/adapter/`.
- ADR required: `docs/adr/0007-agent-behavior-observatory.md`.

## CLI Commands

```bash
archfit observe --trace-dir .agent-traces/ .    # analyze traces
archfit observe --report .                       # observatory report
archfit observe --json .                         # JSON output
```

The command is read-only and informational (exit code always 0).

## Hotspot Analysis

A hotspot is a directory prefix where agents struggle:
- `read_fan_out > median * 2` (agent reads too many files for changes in this area)
- `retry_count > 2` (agent fails repeatedly in this area)

Hotspots are cross-referenced with static scan findings to produce actionable recommendations.

## Related Files

- `development/fitness-contract.md` — contract can consume observatory metrics as soft targets
- `development/metrics-and-scoring.md` — behavioral metrics complement static metrics
- `internal/score/metrics.go` — static metric computation (behavioral metrics follow the same pattern)
