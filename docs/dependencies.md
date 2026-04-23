# Runtime and build dependencies

Per `CLAUDE.md` §3, archfit prefers the standard library. Any external dep
requires a justification at the import site on first use *and* an entry here.

## Phase 1 — zero external dependencies

Phase 1 ships with no non-stdlib dependencies. The `go.mod` is empty of
`require` blocks for that reason. This is a design choice, not an oversight:

- Config is parsed as JSON via `encoding/json` (YAML 1.2 is a superset of JSON).
- All tests use `testing` and `t.TempDir()` for fixtures.
- Subprocess execution goes through `os/exec` behind `internal/adapter/exec`.

## Phase 2 — planned additions (with justification)

These are *planned*, not present. Each will be added with an ADR.

- `gopkg.in/yaml.v3` — proper YAML parsing for `.archfit.yaml` with anchors,
  block scalars, and comments. Justification: consumers expect real YAML.
- `github.com/santhosh-tekuri/jsonschema/v5` — schema validation for rule YAML
  and output JSON in tests. Justification: hand-rolling JSON Schema is not worth the
  maintenance cost for the number of schemas archfit owns.

## Not planned

- Reflection-based YAML or JSON libraries — violate P3 (shallow explicitness).
- CLI frameworks larger than `flag` (`cobra`, `urfave/cli`) — Phase 1's
  commands are small enough that `flag` is enough and the explicit dispatch
  in `main.go` is the feature, not the bug.
