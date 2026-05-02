# Claude Code Agent Skill

archfit ships with a Claude Code agent skill that lets Claude scan repositories, interpret findings, and apply remediations — all driven by the CLI.

## How it works

The skill lives at `.claude/skills/archfit/` in the archfit repo. When Claude Code opens a project that has this directory, it auto-discovers the skill and can use archfit without any setup.

```
.claude/skills/archfit/
├── SKILL.md                    # Entry point (auto-loaded by Claude Code)
├── scripts/                    # Orchestration scripts for multi-step workflows
│   ├── triage.sh               # Classify findings by severity and actionability
│   ├── plan_remediation.sh     # Build a remediation plan from scan results
│   ├── apply_safe_fixes.sh     # Apply deterministic fixes with rollback
│   └── verify_loop.sh          # Re-scan and confirm findings are resolved
├── reference/
│   └── remediation/            # Per-rule decision trees and fix snippets
│       ├── P1.LOC.001.md
│       ├── P1.LOC.002.md
│       ├── ...
│       └── P7.MRD.003.md
└── templates/
```

### Progressive disclosure

The skill follows a three-level loading model:

1. **Level 1** — `SKILL.md` frontmatter (`name` + `description`): tells Claude Code *when* to activate the skill
2. **Level 2** — `SKILL.md` body: the core scan-propose-verify loop, commands, output schema
3. **Level 3** — `reference/remediation/<rule-id>.md`: loaded on demand when a specific finding needs remediation

This keeps context-window usage minimal. Claude only loads what it needs.

## Using the skill in your repo

### Option 1: Copy into your project

```bash
# From your project root
mkdir -p .claude/skills
cp -r /path/to/archfit/.claude/skills/archfit .claude/skills/
```

Claude Code will auto-discover the skill when working in your project.

### Option 2: Personal skill (all repos)

```bash
cp -r /path/to/archfit/.claude/skills/archfit ~/.claude/skills/
```

This makes the skill available in every repository you open.

### Option 3: Run archfit directly

Even without the skill, Claude Code can run archfit if the binary is on `$PATH`:

```
> Run archfit scan on this repo and fix any findings
```

The skill just makes Claude better at interpreting results and following remediation decision trees.

## What the skill does

When triggered, the skill follows this loop:

```
1. Run    →  archfit scan --json .
2. Read   →  Parse findings[] from JSON output
3. Propose →  For each finding, load reference/remediation/<rule-id>.md
              and follow its decision tree
4. Verify  →  Re-run archfit scan after applying fixes
              Only report success if the finding disappeared
```

### Decision trees

Each remediation guide contains a decision tree that tells Claude when to proceed autonomously and when to ask the user:

```markdown
# Example: P1.LOC.001 remediation

1. Does CLAUDE.md already exist with a different name?
   - Yes → rename it. Proceed without asking.
   - No → continue.

2. Is this a documentation-only repo?
   - Yes → suggest suppression. Ask the user.
   - No → create CLAUDE.md from template. Proceed.
```

This prevents the agent from making inappropriate changes (like adding CLAUDE.md to a repo that doesn't need one).

### Scope limits

The skill enforces these boundaries:

- **Never mass-fix silently** — each finding goes through its decision tree
- **Never suppress without documentation** — ignores require a `reason` and `expires` date in `.archfit.yaml`
- **Never skip re-scan** — the re-scan is the proof that the fix worked
- **Never change rule severities** — that's an archfit-internal change

## Skill Scripts

The `scripts/` directory contains shell scripts that orchestrate multi-step workflows. Claude Code invokes these as part of the scan-fix-verify loop:

| Script | Purpose |
|--------|---------|
| `triage.sh` | Classify findings by severity and actionability; prioritize what to fix first |
| `plan_remediation.sh` | Build an ordered remediation plan from scan JSON output |
| `apply_safe_fixes.sh` | Apply deterministic (static) fixes with automatic rollback on failure |
| `verify_loop.sh` | Re-scan after fixes and confirm each targeted finding is resolved |

The scripts are designed to be composable. A typical agent session runs them in sequence: triage, plan, apply, verify.

## Contract-aware workflow

When `.archfit-contract.yaml` exists in the repo, the skill reads it before starting work:

```
1. Load the contract
2. Check which area budgets are affected by planned changes
3. Follow agent directives (e.g., "stop and ask on severity >= error")
4. After changes, verify no hard constraint is violated
5. Report contract status in the summary
```

See [Fitness Contract](contract.md) for contract file format.

## Commands the skill uses

| Command | When used |
|---------|-----------|
| `archfit scan --json .` | Every scan (primary workflow) |
| `archfit fix <rule-id> .` | When a finding has an auto-fixer |
| `archfit fix --plan <rule-id> .` | Preview fix before applying |
| `archfit explain <rule-id>` | When user asks about a specific rule |
| `archfit contract check --json .` | When `.archfit-contract.yaml` exists |
| `archfit scan --explain-coverage .` | When user asks why score is high with few findings |

## Writing your own agent skill for archfit

If you're building a skill for a different agent (Copilot, Cursor, etc.), the key contract is the JSON output:

1. Run `archfit scan --json .` — the output schema is stable and versioned
2. Parse `findings[]` — sorted deterministically (severity desc, rule_id asc, path asc)
3. Each finding has `evidence` sufficient to verify the claim without re-running
4. Each finding has `remediation.summary` for a quick fix description
5. Re-scan after applying fixes — exit code 0 means the finding is gone

The JSON schema is at `schemas/output.schema.json` with `schema_version` for forward compatibility.

## Remediation guide format

Each guide in `reference/remediation/` follows this structure:

```markdown
# Remediation: <rule-id> — <short description>

**Summary**: One sentence explaining what the rule checks.

## Decision tree

1. **First question?**
   - Yes → action. Proceed / Ask the user.
   - No → continue.

2. **Second question?**
   ...

## Snippet

<minimal code/config to fix the finding>

## See also

- `docs/rules/<rule-id>.md`
```

Keep remediation files under 100 lines. For deeper context, link to the full rule documentation.
