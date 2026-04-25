# Auto-Fix (`archfit fix`)

`archfit fix` closes the scan-fix-verify loop. It automatically remediates findings and verifies the fix by re-scanning.

## Usage

```bash
# Fix a specific rule
archfit fix P1.LOC.001 .

# Fix all fixable findings
archfit fix --all .

# Preview what would change
archfit fix --dry-run --all .

# Show the plan without applying
archfit fix --plan --all .

# JSON output for automation
archfit fix --json --all .
```

## How It Works

1. **Scan**: run all enabled rules to find current findings
2. **Plan**: for each finding with a registered fixer, propose changes
3. **Apply**: write changes to disk
4. **Verify**: re-scan to confirm the finding is gone
5. **Rollback**: if the finding persists or new findings appear, undo all changes

## Available Fixers

| Rule | What it creates | Type |
|---|---|---|
| P1.LOC.001 | `CLAUDE.md` from template | Static |
| P1.LOC.002 | `AGENTS.md` in each slice missing one | Static |
| P4.VER.001 | `Makefile` with `test` target | Static |
| P7.MRD.001 | `docs/exit-codes.md` from template | Static |
| P7.MRD.002 | `CHANGELOG.md` from template | Static |
| P7.MRD.003 | `docs/adr/` directory with template ADR | Static |
| P2.SPC.010 | `schemas/output.schema.json` skeleton | Static |

## Static vs LLM-Assisted Fixers

**Static fixers** produce deterministic output from embedded templates. They are safe to run unattended and are the default for `--all`.

**LLM-assisted fixers** use the LLM adapter to generate contextual content. They require `--with-llm` and fall back to the static template if the LLM call fails.

```bash
# LLM-enriched fix
archfit fix --with-llm P1.LOC.001 .
```

## Flags

| Flag | Description |
|---|---|
| `--all` | Fix all fixable findings |
| `--dry-run` | Show what would change without applying |
| `--plan` | Show fix plan and exit |
| `--json` | Emit result as JSON |
| `--with-llm` | Use LLM for contextual content |
| `-C <dir>` | Change directory before running |

## Safety

- Every fix is verified by automatic re-scan
- If verification fails, all changes are rolled back
- Fix actions are logged to `.archfit-fix-log.json` for audit
- LLM fixers never fail the fix — they fall back to static templates
- No fix modifies existing file content unless the fixer explicitly uses `ActionModify`

## In CI

```yaml
- name: Auto-fix
  run: |
    archfit fix --all .
    git diff --quiet || git commit -am "chore: archfit auto-fix"
```
