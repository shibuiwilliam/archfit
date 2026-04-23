# Fixture: triggers P1.LOC.002 (not P1.LOC.001)

The root CLAUDE.md is present, so P1.LOC.001 should not fire.
The packs/ directory contains a slice (`packs/alpha`) without an AGENTS.md, so P1.LOC.002 should fire for `packs/alpha`.
