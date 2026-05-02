# Ground truth annotations

Each subdirectory corresponds to a repo in `corpus.yaml` and contains
`expected_findings.yaml` — hand-annotated expected findings.

Format per entry:

```yaml
- rule_id: P1.LOC.001
  expected: true        # should the rule fire?
  path: ""              # expected path (empty = repo-level)
  notes: "No CLAUDE.md"
```

A rule's precision on the corpus is:
  true_positives / (true_positives + false_positives)

Promotion to `stable` requires precision >= 0.85 across the corpus.
See CLAUDE.md §7.3.
