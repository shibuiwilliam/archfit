# Metrics and Scoring

## Scoring Algorithm

### Per-Principle Score

```
score(P) = 100 × (1 - penalty(P) / total_weight(P))
```

Where:
- `total_weight(P)` = sum of weights of all rules applicable to principle P
- `penalty(P)` = sum of weights of rules in P that produced at least one finding at or above the profile's severity threshold
- Multiple findings from the same rule count as one penalty (no compounding)
- A rule with zero applicable findings contributes zero penalty

### Overall Score

```
overall = 100 × (1 - Σ penalty(P) / Σ total_weight(P))
```

### Profile Effects

| Profile | Threshold | Effect |
|---|---|---|
| `strict` | `info` | All findings count as penalties |
| `standard` | `warn` | Info findings are informational only |
| `permissive` | `error` | Warn findings are informational only |

### Key Properties

- **Adding rules does not lower scores** for repos that pass the new rules. Scoring is normalized per applicable rule set.
- **Multiple findings per rule do not compound**. A rule either fires or it doesn't — how many times doesn't affect the score.
- **Deterministic**. Same input produces same scores. No randomness.

## Metrics

### `context_span_p50` (P1 Locality)

**Source**: `internal/score/metrics.go:ContextSpanP50()`

Median number of files touched per commit, computed from the last N commits in `GitFacts.RecentCommits`.

```
For each commit: count FilesChanged
Sort the counts
Return the median (p50)
```

**Unit**: files. Lower is better (narrower changes = better locality).

**Requires**: Git collector (`GitFacts` available).

### `verification_latency_s` (P4 Verifiability)

**Source**: `internal/score/metrics.go:VerificationLatency()`

Wall-clock time of the fastest successful verification command, in seconds.

```
From CommandFacts.Results:
  Filter results where ExitCode == 0
  Return min(DurationMS) / 1000.0
  If no successful commands: return 0 with "no_data" marker
```

**Unit**: seconds. Lower is better.

**Requires**: Command collector (`--depth=deep`).

### `invariant_coverage` (P4 Verifiability)

**Source**: `internal/score/metrics.go:InvariantCoverage()`

Fraction of evaluated rules that did NOT produce an error-or-above finding.

```
rules_without_error = count of rules where no finding has severity >= error
total_rules = total rules evaluated
coverage = rules_without_error / total_rules
```

**Unit**: ratio (0.0–1.0). Higher is better.

**Requires**: Rule evaluation results only (no collector dependency).

### `parallel_conflict_rate` (P1 Locality)

**Source**: `internal/score/metrics.go:ParallelConflictRate()`

Fraction of recent commits that are merge commits (heuristic for conflict frequency).

```
merge_commits = count of commits where Subject starts with "Merge"
total_commits = len(RecentCommits)
rate = merge_commits / total_commits
```

**Unit**: ratio (0.0–1.0). Lower is better.

**Requires**: Git collector.

### `rollback_signal` (P6 Reversibility)

**Source**: `internal/score/metrics.go:RollbackSignal()`

Fraction of recent commits that are reverts.

```
revert_commits = count of commits where Subject starts with "Revert"
total_commits = len(RecentCommits)
signal = revert_commits / total_commits
```

**Unit**: ratio (0.0–1.0). Interpretation is nuanced — some reverts are healthy (fast rollback), too many indicate instability.

**Requires**: Git collector.

### `blast_radius_score` (P5 Aggregation)

**Source**: `internal/score/metrics.go:BlastRadius()`

Maximum transitive reach of any package, normalized by total package count.

```
blast_radius = MaxReach / PackageCount
```

**Unit**: ratio (0.0–1.0). Lower is better (no single package affects everything).

**Requires**: Depgraph collector (Go only in Phase 1).

## Adding a New Metric

1. Add a pure function to `internal/score/metrics.go`:
   ```go
   func MyNewMetric(facts SomeFactType) model.Metric {
       return model.Metric{
           Name:      "my_new_metric",
           Value:     computedValue,
           Unit:      "unit_name",
           Principle: "P1",
       }
   }
   ```

2. Wire it into `internal/core/scheduler.go` after rule evaluation:
   ```go
   metrics = append(metrics, score.MyNewMetric(someFacts))
   ```

3. Add property-based tests in `internal/score/metrics_test.go`:
   - Assert value is within expected bounds
   - Assert determinism (same input → same output)
   - Test edge cases (empty input, single element, etc.)

4. Update `schemas/output.schema.json` if the metric name is new (additive change — no version bump needed).

## JSON Output Shape

```json
{
  "schema_version": "0.1.0",
  "scores": {
    "overall": 92.5,
    "by_principle": {
      "P1": 100.0,
      "P2": 85.0,
      "P4": 90.0,
      "P7": 95.0
    }
  },
  "metrics": [
    {"name": "context_span_p50", "value": 3.0, "unit": "files", "principle": "P1"},
    {"name": "invariant_coverage", "value": 0.9, "unit": "ratio", "principle": "P4"}
  ],
  "findings": [...],
  "rules_evaluated": 10
}
```

Scores are rounded to one decimal in output; internal math uses `float64`.
