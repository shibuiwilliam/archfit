# Cross-Stack Detection Improvements

archfit was built in Go and its detection patterns reflect Go conventions. This document catalogs the gaps and provides concrete implementation tasks to make archfit credible across all major technology stacks.

Source analysis: `REPORT_CROSS_STACK_USEFULNESS.md`

## Current Stack Coverage

| Stack | Score | Key Gaps |
|---|---|---|
| Go | 8/10 | Narrow slice container list (P1.LOC.002) |
| Node/TypeScript | 6/10 | Missing `packages/`, `apps/`, `libs/` for monorepos |
| Python | 5/10 | Django `apps/` invisible, CLI detection misses `console_scripts` |
| Java/Spring | 2/10 | `pom.xml`/`build.gradle` not recognized, `application.yml` invisible |
| Ruby/Rails | 2/10 | `Gemfile`/`Rakefile` not recognized, `engines/` not detected |
| Terraform/IaC | 3/10 | `terraform.tfvars` invisible, no module structure detection |
| Frontend | 2/10 | Only `package.json` and `.env` work |

## Priority 0 — Ship Before Anything Else (5-15 min each)

### Task 1: Add build tool detection to P4.VER.001

**File**: `packs/core/resolvers/p4_ver_001.go`

Add to the verification entrypoint list:
```
pom.xml, build.gradle, build.gradle.kts, settings.gradle, settings.gradle.kts,
Gemfile, Rakefile, rakefile, composer.json, mix.exs, build.sbt,
CMakeLists.txt, deno.json, deno.jsonc, Earthfile, BUILD.bazel
```

**Impact**: Eliminates false negatives for Java, Ruby, PHP, Elixir, Scala, C/C++, Deno, Bazel.

### Task 2: Expand slice containers in P1.LOC.002

**File**: `packs/core/resolvers/p1_loc_002.go`

Expand the `sliceContainers` list:
```go
var sliceContainers = []string{
    "packs", "services", "modules",
    // Monorepo conventions
    "packages", "apps", "libs",
    // Domain-driven design
    "domains", "bounded-contexts", "features",
    // Language-specific
    "plugins", "engines", "components",
    // Terraform
    "stacks", "envs", "environments",
}
```

**Impact**: Covers NX/Turborepo, Django, Rails engines, Lerna monorepos, Terraform stacks.

### Task 3: Widen CLI detection in P7.MRD.001

**File**: `packs/core/resolvers/p7_mrd_001.go`

In addition to `cmd/` and `bin/` with source files, also check:
- `package.json` with a `"bin"` field
- `pyproject.toml` with `[project.scripts]`
- `exe/` directory (Ruby gem convention)

**Impact**: Covers Node CLI tools, Python CLI tools, Ruby gems.

## Priority 1 — Config-Aware Detection (1-2 hours each)

### Task 4: Spring Boot config detection for P3.EXP.001

Extend P3.EXP.001 or create P3.EXP.002:
- If `application.yml` or `application.properties` exists
- And `application-*.yml` profile variants exist (dev, staging, prod)
- Then require a `config/README.md` or documented configuration reference

**Why**: Spring Boot profiles are the JVM equivalent of undocumented `.env` files.

### Task 5: Terraform variable documentation

Create P3.EXP.003 or extend P3.EXP.001:
- If `*.tf` files exist with `variable` blocks
- Then require `terraform.tfvars.example` or `variables.md`

### Task 6: Make slice containers configurable

Add to `.archfit.yaml`:
```json
{
  "slice_containers": ["api", "core", "domain", "infrastructure"]
}
```

When set, P1.LOC.002 uses this list instead of the hardcoded defaults.

**File**: `internal/config/config.go` (add field), `packs/core/resolvers/p1_loc_002.go` (read from config via FactStore).

**Note**: this requires exposing config values through FactStore, which is a design decision. Alternative: embed the list in `.archfit.yaml` `overrides` for P1.LOC.002.

## Priority 2 — Coverage Transparency

### Task 7: Add rule applicability reporting

When a rule's resolver finds nothing to check (no `.env` files, no `cmd/` directory, no schemas), it should report this in the output:

```json
{
  "coverage": {
    "rules_applicable": 6,
    "rules_evaluated": 10,
    "rules_not_applicable": ["P3.EXP.001", "P7.MRD.001", "P2.SPC.010", "P7.MRD.003"]
  }
}
```

This prevents false confidence from silent pass-throughs.

**Files**: `internal/model/model.go` (add `NotApplicable` to resolver return), `internal/core/scheduler.go`, `schemas/output.schema.json`.

### Task 8: Auto-detect project type

Use `RepoFacts.Languages` to auto-detect when `project_type` is empty:
- `.java` → Java, check for `pom.xml`/`build.gradle`
- `.rb` → Ruby, check for `Gemfile`/`Rakefile`
- `.tf` → Terraform, check for `terraform.tfvars`
- `.py` with `manage.py` → Django
- `.ts` with `angular.json` → Angular

**File**: `internal/config/config.go` or a new `internal/collector/projecttype/` collector.

## Priority 3 — Spec Detection Expansion

### Task 9: OpenAPI/Protobuf/GraphQL for P2.SPC.010

Currently checks only `schemas/*.schema.json`. Also detect:
- `openapi.yaml` / `openapi.json` / `swagger.yaml`
- `*.proto` files (Protobuf)
- `*.graphql` / `schema.graphql`
- `*.avsc` (Avro)
- `asyncapi.yaml`

**File**: `packs/agent-tool/resolvers/p2_spc_010.go`, `internal/collector/schema/schema.go`.

## Implementation Notes

- P0 tasks are pure resolver changes — no new packages, no new deps, no ADR needed
- P1 tasks may need new rules (YAML + resolver + fixture + expected.json + remediation doc + docs/rules/)
- P2 tasks require schema changes (ADR needed for output schema)
- All tasks must include fixture updates and pass `make self-scan`
- P0 tasks should be shipped as a single PR: "fix: expand cross-stack detection patterns"
