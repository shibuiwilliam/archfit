# LLM-assisted explanation (`--with-llm`)

Phase 3a adds an **opt-in** LLM path: archfit can call Google Gemini to produce a finding-specific explanation on top of the static rule docs. This document is the contract between archfit and its users for that feature.

The full design rationale is in [ADR 0003](./adr/0003-llm-explanation.md).

## TL;DR

```bash
export GOOGLE_API_KEY=...           # or GEMINI_API_KEY
./bin/archfit scan --with-llm .     # at most 5 findings are enriched
./bin/archfit explain --with-llm P1.LOC.001
./bin/archfit check --with-llm P7.MRD.001 .
./bin/archfit scan --with-llm --llm-budget=20 --json . | jq '.findings[].llm_suggestion'
```

Omit `--with-llm` and nothing changes — the base scan path is byte-identical with or without an API key set.

## What it does

When `--with-llm` is set, archfit:

1. Runs the normal scan (the scan path does not call the LLM).
2. For up to `--llm-budget` findings (default **5**), issues one Gemini call with:
   - the rule ID, title, rationale, severity, and evidence strength,
   - the finding's path, message, and evidence map,
   - the repo's `project_type` from `.archfit.yaml`.
3. Attaches an `llm_suggestion` object to each enriched finding. The terminal renderer prints it below the finding; the JSON renderer emits it as a nested field; the SARIF renderer places it in `results[].properties.llm_suggestion`.

The LLM is instructed to produce ≤200 words in three short sections: *why it matters here*, *concrete fix*, and *when to suppress*.

## What it does NOT do

- **It never fails the scan.** If the API is down, the key is rotated, or the call times out, archfit logs a single stderr line per skipped finding and keeps the static remediation. Exit code behavior is identical to a run without `--with-llm`.
- **It never runs without `--with-llm`.** The base `archfit scan .` makes zero LLM calls, regardless of whether `GOOGLE_API_KEY` is set.
- **It is not deterministic.** Golden tests under `testdata/e2e/` explicitly run without the flag. Do not pin LLM output as a golden.
- **It does not auto-fix anything.** `archfit fix` is a separate feature in Phase 3c. The suggestion is advisory — the agent or human still performs the change.

## Configuration

| Setting | Where | Default |
|---|---|---|
| API key | `GOOGLE_API_KEY` env, or `GEMINI_API_KEY` | **required** — command exits `4` if missing |
| Model | `LLM_MODEL` env | `gemini-2.5-flash` |
| Per-run budget | `--llm-budget N` | `5` |
| Per-call timeout | (not configurable) | `30s` |

API keys are never written to logs, never read from `.archfit.yaml`, and never embedded in the binary. See [`SECURITY.md`](../SECURITY.md) for data-handling guidance.

## Cost safety

Two layers guard against runaway cost:

- **Budget**: `--llm-budget N` caps the number of calls per run. The default of 5 covers the typical "1–3 new findings per PR" case without surprises.
- **Cache**: identical prompts within one run are served from an in-memory cache for free. Useful when `--llm-budget` is large and multiple findings share evidence.

A disk-backed cache and a daily spend cap are planned for Phase 3b.

## Data sent to Gemini

When you set `--with-llm`, archfit sends Gemini:

- The rule's ID, title, severity, rationale, and static remediation.
- The finding's path (repo-relative), message, and evidence map.
- The repo's declared `project_type` (if any).

Archfit does **not** send:

- The repository's source code.
- Environment variables other than the API key (which goes in the Authorization header, not the prompt).
- Git history, commit metadata, or author information.
- The contents of files flagged by a rule. Only the rule's evidence (a small structured map) is transmitted.

If the evidence map contains values longer than 8 KiB total, the prompt is truncated at the boundary with a `[truncated]` marker.

## When to use it

- **CI**: use it on PRs with a small budget. The diff between `main.json` and the PR's scan is usually only 1–3 findings, well under the default budget.
- **Local development**: use it when an unfamiliar rule fires and the static doc does not quite fit your case.
- **Triage**: do not use it to mass-audit a large repo. Budget-wise, a fresh scan with many findings is better served by fixing the top-severity ones by hand first and re-running.

## When NOT to use it

- On proprietary or regulated code where sending evidence to a third party is out of policy.
- When a deterministic, auditable output is required — SARIF's core fields are deterministic without `--with-llm`, but the `llm_suggestion` property is not.
- In CI gates that compare output byte-for-byte. Use `archfit scan` (no LLM) for the gate, then `archfit scan --with-llm` for the PR comment.

## Extension points (Phase 3b+)

The adapter interface (`internal/adapter/llm/Client`) is provider-agnostic. Adding OpenAI, Anthropic, or a local Ollama backend is a single-file implementation plus a `--llm-provider` flag. See `DEVELOPMENT_PLAN.md`.
