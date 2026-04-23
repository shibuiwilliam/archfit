# Configuration reference

archfit reads `.archfit.yaml` (or `.archfit.yml` / `.archfit.json`) from the
repository root. All fields are optional; omitted fields take the defaults in
`Default()` (see `internal/config/config.go`).

## Phase 1 restriction (important)

Phase 1 parses config as JSON. YAML 1.2 is a strict superset of JSON, so a
JSON document in `.archfit.yaml` is a valid YAML document any YAML-aware
tool round-trips correctly. Full YAML syntax (anchors, block scalars,
unquoted strings) arrives in Phase 2 when `yaml.v3` is introduced.

If you prefer plain JSON for now, write `.archfit.json`.

## Schema

The authoritative schema is `schemas/config.schema.json`.

## Fields

### `version` (required)

Must be `1`. This is the config schema version, not archfit's version.

### `project_type`

A list of tags describing the repo, used in Phase 2 to decide which
project-specific packs to enable by default. Example: `["web-saas"]`.

### `profile`

One of `strict`, `standard`, `permissive`. Default: `standard`.

- `strict` — stricter severity thresholds and lower tolerance for `medium` evidence.
- `standard` — the default.
- `permissive` — treats `error` findings as `warn`. Use only during migrations.

### `risk_tiers`

Map of tier name to list of path globs. Used in Phase 2 by `P5` rules to
weight findings higher in high-risk areas (`src/auth/**`, `migrations/**`, etc.).

### `packs`

```json
{
  "packs": {
    "enabled": ["core"],
    "disabled": []
  }
}
```

If `enabled` is set, only those packs run. Default: all registered packs.

### `overrides`

```json
{
  "overrides": {
    "P4.VER.003": {"timeout_seconds": 60}
  }
}
```

Rule-specific configuration. The set of allowed keys per rule is defined in
each rule's YAML under `config_schema` — not yet implemented in Phase 1.

### `ignore`

```json
{
  "ignore": [
    {
      "rule": "P1.LOC.002",
      "paths": ["packs/legacy-*"],
      "reason": "Legacy slices on a documented deletion path",
      "expires": "2026-12-31"
    }
  ]
}
```

Every ignore entry must carry a `reason` and an `expires` date. Expired
entries surface as warnings on the next scan so suppressions cannot rot
silently. Phase 1 implements rule-wide ignore; `paths` is accepted but
not yet enforced (Phase 2).

## Example

```json
{
  "version": 1,
  "project_type": ["agent-tool"],
  "profile": "standard",
  "packs": { "enabled": ["core"] },
  "ignore": []
}
```
