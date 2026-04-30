# Self-scan history

Each JSON file records the output of `archfit scan --json .` at the
corresponding release version. The PR-time CI gate (`score-gate` job)
uses the latest snapshot from `main` as the baseline — if a PR drops
the overall score by more than 1.0 point, the check fails.

To record a new snapshot locally:

```bash
make self-scan-record
```

## Scores

| Version | Overall | P1 | P2 | P3 | P4 | P5 | P6 | P7 | Findings |
|---|---|---|---|---|---|---|---|---|---|
| 0.1.0 | 100.0 | 100.0 | 100.0 | 100.0 | 100.0 | 100.0 | 100.0 | 100.0 | 0 |

> This table is updated manually or by the release workflow. Each row
> corresponds to a JSON file in this directory.
