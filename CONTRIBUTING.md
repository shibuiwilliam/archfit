# Contributing to archfit

archfit is a tool for measuring how well a repository is shaped for coding
agents. The contribution workflow is deliberately structured to *live up to
archfit's own principles* — if changes are easy to make, review, and verify,
archfit's verdict on other repos is credible.

Before your first PR, read:

1. [`CLAUDE.md`](./CLAUDE.md) — the primary contract between contributors and
   the repository. Every section is load-bearing.
2. [`PROJECT.md`](./PROJECT.md) — long-form rationale for why archfit exists
   and what it is not.
3. [`PROJECT.md`](./PROJECT.md) §6 — the phase boundaries and roadmap. If
   a change feels outside the current phase, raise it in an issue first.

## The happy path for a change

1. **Open an issue (or comment on one) before writing non-trivial code.**
   A concise question with two concrete alternatives is almost always better
   than a speculative PR.
2. **Scope the PR tightly.** Hard budget: **≤ 500 changed lines, ≤ 5 packages
   touched**. Pure renames and codegen updates are exempt if labeled
   `refactor: rename` or `chore: codegen`.
3. **Use Conventional Commits.** Prefixes archfit recognizes:
   `feat` `fix` `refactor` `docs` `chore` `test` `pack`.
4. **One logical change per PR.** Do not mix a new rule with an engine refactor.
5. **Run `make lint test self-scan` before pushing.** If `self-scan` flags
   your change, fix the self-violation or add an `ignore` entry in
   `.archfit.yaml` with a `reason` and a short-lived `expires`.

## Adding a new rule

Follow `CLAUDE.md` §6 literally. The fixture-first loop is not a suggestion:

1. Pick the rule ID (`P<n>.<DIM>.<nnn>`) in the right pack.
2. Write `fixtures/<rule-id>/input/` and `fixtures/<rule-id>/expected.json`.
3. Declare the rule in `rules/<rule-id>.yaml` (schema: `schemas/rule.schema.json`).
4. Implement the resolver in `resolvers/`. Pure function of `FactStore`.
5. Add `skills/archfit/reference/remediation/<rule-id>.md`.
6. Add `docs/rules/<rule-id>.md`.
7. `make lint test self-scan` — all green.

New rules ship at `stability: experimental`. Promotion to `stable` requires at
least one release cycle at `experimental` and a review.

## Adding a new collector

Collectors live under `internal/collector/<topic>/`. Requirements:

- Deterministic given identical inputs.
- A fake implementation available for tests.
- No direct use from packs — exposed only via `FactStore`.
- Documented in `internal/collector/README.md` (added with the first entry).

## What requires an ADR

- Any change under `internal/model/` or `schemas/`.
- Changes to exit codes, CLI flag names, or JSON output schema.
- Introducing a new external dependency (also update `docs/dependencies.md`).

ADRs live in `docs/adr/` with YAML frontmatter
(`id`, `title`, `status`, `date`, `tags`). See ADR 0001 as a template.

## Testing

Three layers (see `CLAUDE.md` §7):

- Unit (target: < 5s total).
- Pack tests (target: < 20s).
- End-to-end golden tests under `testdata/e2e/` (target: < 60s).

Commands:

```bash
make test           # unit + pack
make e2e            # end-to-end
make update-golden  # regenerate testdata/e2e/*/expected.json — review the diff!
```

No test may depend on network, real git remotes, or the host's installed
toolchains. Shell out only through `internal/adapter/exec`.

## Reviewing a PR

Reviewers should check, in order:

1. Does `make lint test e2e self-scan` pass?
2. Is the PR size within budget?
3. Are changes to `internal/model/` or `schemas/` accompanied by an ADR?
4. Does the JSON output still canonicalize byte-for-byte (e2e golden)?
5. Is the rule's evidence strong enough for its severity? `weak + error` is never allowed.
6. Is the remediation doc actionable without the human asking archfit for more info?

## Code of Conduct

Contributors agree to the [Code of Conduct](./CODE_OF_CONDUCT.md).

## Licensing

All contributions are licensed under Apache 2.0 (see [`LICENSE`](./LICENSE)).
