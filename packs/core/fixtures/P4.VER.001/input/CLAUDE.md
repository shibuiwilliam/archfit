# Fixture: triggers P4.VER.001

CLAUDE.md is present (so P1.LOC.001 does not fire), but there is no Makefile,
justfile, Taskfile, package.json, pyproject.toml, Cargo.toml, or go.mod at the
root. P4.VER.001 should fire.
