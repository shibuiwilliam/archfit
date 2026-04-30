---
id: 0011
title: Centralize ecosystem detection in internal/collector/ecosystem
status: accepted
date: 2026-04-30
---

## Context

Ecosystem detection logic (CI platforms, deployment tools, application
frameworks) was scattered across 4+ resolvers in private keyword tables.
Each resolver re-walked the file list independently, and CI platform
detection was duplicated between `verifiability_ci.go` and
`aggregation_secrets.go`.

## Decision

1. Create `internal/collector/ecosystem/` with typed detection rules for
   CI platforms (9), deployment tools (14), and application frameworks (2).
   Detection runs once per scan in a single pass over the file list.

2. Add `EcosystemFacts` type to `model` with `Has(name)`, `HasCI()`,
   `CIFiles()`, and `HasDeployment()` query methods.

3. Add `Ecosystems() EcosystemFacts` to `model.FactStore` interface.
   This is a public interface change (same category as ADR 0010).

4. Migrate resolvers:
   - `verifiability_ci.go`: replaced private `ciConfigFiles` +
     `ciConfigDirPrefixes` with `facts.Ecosystems().HasCI()`
   - `aggregation_secrets.go`: replaced duplicate CI detection with
     `facts.Ecosystems().CIFiles()`
   - `explicitness.go`: uses `facts.Ecosystems().Has("spring")` and
     `Has("rails")` as fast-path skip before file walks

5. `reversibility.go` retains its private `hasDeploymentArtifacts`
   function for now — the deployment artifact list includes directory
   prefixes (`deploy/`, `deployment/`) that don't map 1:1 to ecosystem
   names. Full migration deferred to avoid scope creep.

## Consequences

- Single round of ecosystem detection per scan (was 2-4 independent walks).
- Adding a new CI platform or deployment tool is a one-line change in the
  collector, not a hunt through multiple resolvers.
- All `FactStore` implementations must add `Ecosystems()`.
- The `secretScannerKeywords` list in `aggregation_secrets.go` remains
  resolver-private — it's tool-specific, not ecosystem detection.
