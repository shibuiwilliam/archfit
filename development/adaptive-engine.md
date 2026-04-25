# Adaptive Rule Engine

> Element 3 of the three strategic elements. Builds last — needs historical data from fixes and scans.

## Overview

The adaptive engine is a post-processing layer that adjusts finding confidence and rule thresholds based on fix outcomes, suppress history, and repo characteristics. Resolvers remain pure functions of FactStore. Adaptation happens **after** rule evaluation.

## Key Insight

archfit's rules currently have fixed severity, fixed confidence, and fixed applicability. A rule that matters critically for a payment service may be irrelevant for a documentation repo. When a fix sticks (verified by re-scan), that is positive training data. When it's rolled back or suppressed, that is negative training data.

## How It Works

### Fix Outcome Tracking

Extend the existing fix log (`internal/fix/log.go`) with repo signals:

```go
type RepoSignals struct {
    FileCount    int            `json:"file_count"`
    Languages    map[string]int `json:"languages"`
    ProjectTypes []string       `json:"project_types"`
    TopLevelDirs int            `json:"top_level_dirs"`
}
```

### Adaptive Confidence

Adjusted after resolvers run:

```
adjusted = base × (0.5 + 0.5 × successRate) × (1.0 - 0.3 × suppressRate)
```

Where:
- `successRate` = verified fixes / total fixes for this rule in similar repos
- `suppressRate` = suppressed findings / total findings for this rule

Clamped to [0.1, 1.0]. When adjusted confidence drops below 0.5, evidence gains an `"adaptive_note"` field.

### Threshold Adaptation

For rules with numeric thresholds (e.g., P5.AGG.001's `maxTopLevelDirs`):

```go
func AdaptiveThreshold(ruleID string, repo model.RepoFacts) int {
    topDirCount := countTopLevelDirs(repo)
    if topDirCount <= 5  { return 2 }  // small repos: strict
    if topDirCount <= 15 { return 3 }  // medium: slightly relaxed
    return max(4, topDirCount/5)       // large: proportional
}
```

## Package Structure

```
internal/adaptive/
├── engine.go           # AdaptiveEngine: adjusts findings post-resolver
├── engine_test.go
├── confidence.go       # Confidence adjustment logic
├── confidence_test.go
├── threshold.go        # Context-aware threshold computation
├── threshold_test.go
├── history.go          # Read fix log and suppress history
└── history_test.go
```

## Implementation Steps

| Step | Description | Status | Effort |
|------|-------------|--------|--------|
| 3.1 | Fix outcome tracking (extend log) | Not started | ~100 lines |
| 3.2 | Adaptive confidence engine | Not started | ~200 lines |
| 3.3 | Threshold adaptation | Not started | ~150 lines |
| 3.4 | CLI wiring (`--adaptive` flag) | Not started | ~100 lines |
| 3.5 | Telemetry design (stub only) | Not started | ~50 lines |

## Architecture Rules

- The adaptive engine is **opt-in** via `--adaptive` flag or `adaptive: true` in `.archfit.yaml`.
- It runs AFTER rule evaluation and BEFORE rendering.
- Resolvers remain pure. The adaptive layer never modifies resolver functions.
- All functions are pure once constructed (history is read at init, `Adjust()` is pure).
- Threshold adaptation may need a new optional method on `FactStore`: `Adaptive() (AdaptiveContext, bool)`. This requires an ADR.
- Do NOT introduce a database. File-based storage (fix log) is sufficient.

## Telemetry (Future)

Opt-in anonymized telemetry (design only, not implemented in Phase 1):

```go
type TelemetryEvent struct {
    RuleID      string      `json:"rule_id"`
    Outcome     string      `json:"outcome"`      // "fired", "suppressed", "fixed_verified"
    RepoSignals RepoSignals `json:"repo_signals"`
    Timestamp   string      `json:"ts"`
}
```

What is sent: rule ID, outcome, repo signals (file count, language distribution).
What is **never** sent: source code, file names, file contents, git history, API keys.

ADR required: `docs/adr/0010-telemetry.md`.

## CLI Integration

```bash
archfit scan --adaptive .           # enable adaptive confidence
archfit scan --adaptive --json .    # adaptive adjustments visible in evidence
```

Adaptive adjustments appear as additional evidence fields in findings:

```json
{
  "evidence": {
    "adaptive_confidence": 0.72,
    "adaptive_note": "confidence reduced: 45% suppress rate in repos with >50 files"
  }
}
```

## Related Files

- `internal/fix/log.go` — fix audit log (extended with repo signals in Step 3.1)
- `development/metrics-and-scoring.md` — scoring algorithm (adaptive engine does not change scoring)
- `development/fitness-contract.md` — contract can reference adaptive metrics
