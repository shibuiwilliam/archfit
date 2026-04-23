---
name: archfit
description: Scan a repository with the archfit CLI, read the JSON output, and propose remediations that archfit verifies on re-scan. Use when the user asks to run archfit, score a repo for coding agents, remediate a rule ID (e.g. P1.LOC.001), or asks what archfit would flag.
---

# archfit — Claude Code agent skill

archfit evaluates whether a repository is shaped for coding agents to work
on **safely** and **quickly** along seven principles: locality, spec-first,
shallow explicitness, verifiability, aggregation of danger, reversibility,
and machine-readability.

**This skill is deliberately short.** Deep material lives in
`reference/`. Load it on demand.

## When to use this skill

Use it whenever the user asks you to:

- "run archfit" / "scan this repo"
- "score this repo for agents"
- "what would archfit flag about…"
- fix a specific rule ID (e.g. "remediate P1.LOC.001")

Do **not** use it for:

- linting, formatting, SAST — those are separate tools (archfit sits above them).
- grading the repo against standards (SLSA, NIST, etc.) — archfit reports
  on agent-fitness, not compliance.

## The core loop

1. **Run**: `archfit scan --json . > /tmp/archfit.json`
2. **Read**: the `findings[]` array in the JSON. Sort order is fixed
   (severity desc, rule_id asc, path asc), so index-based references
   are stable within a single run.
3. **Propose**: for each finding, read `reference/remediation/<rule-id>.md`
   on demand. Follow its decision tree — it tells you when to ask the user
   and when to proceed.
4. **Verify**: re-run `archfit scan` after applying the remediation. If
   the finding disappears and no new ones appeared, report success.

Never claim a remediation is done without the re-scan. The re-scan is the
contract.

## Commands

```
archfit scan [path]                # full scan
archfit scan --json [path]         # JSON to stdout (the contract)
archfit score [path]               # numbers only, no finding list
archfit explain <rule-id>          # rule rationale + remediation
archfit list-rules                 # all registered rules
archfit list-packs                 # packs and their rule IDs
archfit validate-config [path]     # check .archfit.yaml
```

Global flags:

- `--fail-on {info|warn|error|critical}` — exit code 1 when any finding
  meets this severity. Default: `error`.
- `--format {terminal|json|md}` — output format. `--json` is shorthand
  for `--format=json`.
- `-C <dir>` — change to directory before running.

## Output schema

The JSON output follows `schemas/output.schema.json`. Minimum shape:

```json
{
  "schema_version": "0.1.0",
  "tool": {"name": "archfit", "version": "..."},
  "target": {"path": "...", "profile": "standard"},
  "summary": {
    "rules_evaluated": 4,
    "findings_total": 1,
    "by_severity": {"info": 0, "warn": 1, "error": 0, "critical": 0}
  },
  "scores": {"overall": 90.0, "by_principle": {"P1": 100.0}},
  "findings": [
    {
      "rule_id": "P7.MRD.001",
      "principle": "P7",
      "severity": "warn",
      "evidence_strength": "strong",
      "confidence": 0.95,
      "path": "docs/",
      "message": "…",
      "evidence": {"looked_for": ["docs/exit-codes.md"]},
      "remediation": {"summary": "…", "guide_ref": "docs/rules/P7.MRD.001.md"}
    }
  ],
  "metrics": []
}
```

## Remediation guides (progressive disclosure)

When a finding appears, load the matching guide:

- `reference/remediation/P1.LOC.001.md`
- `reference/remediation/P1.LOC.002.md`
- `reference/remediation/P4.VER.001.md`
- `reference/remediation/P7.MRD.001.md`

Guides are kept under 100 lines each. For deeper context they link to
`docs/rules/<rule-id>.md` in the main repo.

## Scope limits

- **Do not** mass-fix all findings silently. For each finding, follow the
  guide's decision tree — some findings require asking the user first.
- **Do not** suppress a finding without writing a time-limited `ignore`
  entry in `.archfit.yaml` and a `reason`.
- **Do not** change rule severities. That's an archfit-repo change, not a
  consumer-repo change.

## Exit codes

See `../../../docs/exit-codes.md`. Agents should treat exit code `1` as a
signal to re-read the JSON, not as a failure.
