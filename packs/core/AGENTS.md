# Pack: core — agent guide

You are working on archfit's `core` pack. This pack holds rules that apply universally.

Before editing:

- Read `INTENT.md` in this directory.
- Read `../../CLAUDE.md` §6 (rule authoring) and §13 (what not to do).
- Confirm the rule you are about to add could not live in a more specific pack.

Structure:

- `rules/` — YAML declarations, one file per rule. Schema: `schemas/rule.schema.json`.
- `resolvers/` — Go resolver functions, one per rule. Pure functions of `FactStore`. No I/O, no filesystem calls, no git calls.
- `fixtures/<rule-id>/` — each rule has `input/` (a minimal repo that should trigger the rule) and `expected.json` (the finding shape). Table tests in `pack_test.go` diff the two.

When adding a rule:

1. Write the fixture first — the `input/` directory plus `expected.json`.
2. Write the YAML declaration.
3. Implement the resolver until the fixture test passes.
4. Add `.claude/skills/archfit/reference/remediation/<rule-id>.md`.
5. Add `docs/rules/<rule-id>.md`.
6. Run `make lint test self-scan`.

Do not:

- Import `internal/adapter/*` or `internal/collector/*` directly. Resolvers consume `FactStore` only.
- Reach into `os` / `io/fs` from a resolver. If you need a new fact, extend a collector.
- Add a rule whose evidence is `weak` and severity is `error` (see CLAUDE.md §13).
