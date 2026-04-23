# Pack: agent-tool — agent guide

Rules here apply to repositories that ship a tool meant to be driven by coding agents. Before editing:

- Read `INTENT.md` in this directory.
- Confirm the rule you are adding is specific to agent-driven tools. If it applies to every repository, it belongs in `core`, not here.

Structure mirrors `packs/core`:

- `rules/` — YAML declarations.
- `resolvers/` — pure functions of `FactStore`.
- `fixtures/<rule-id>/` — minimal repo + `expected.json`.
- `pack_test.go` — table test wiring fixtures through the engine.

**Opt-in by design.** The pack is added to `.archfit.yaml` by the consumer. Do not make rules here run by default across all repos — they will produce noise on repos that never intended to ship an agent-tool.

Do not:

- Duplicate rules already in `core`.
- Assume the consumer repo is written in Go. Language-specific checks go in language-specific packs.
- Shell out. Import `internal/adapter/*` is forbidden.
