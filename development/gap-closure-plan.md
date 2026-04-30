# Gap closure plan — PROJECT.md & CLAUDE.md vs codebase

**Date**: 2026-04-30
**Audit method**: Full read of PROJECT.md, CLAUDE.md, and systematic
verification against the live codebase.

## Executive summary

The codebase is functionally sound — all 17 rules work, tests pass, self-scan
exits 0. The gaps are **documentation drift**: the governing documents
(CLAUDE.md, PROJECT.md) were not updated as features shipped. There is also
one **build bug** in the Dockerfile.

## Prioritized gaps

### P0 — Must fix (blocks 1.0 release)

| # | Gap | File | Fix |
|---|---|---|---|
| 1 | Dockerfile uses wrong ldflags variable (`version.Version` → `version.linkerVersion`) | `Dockerfile` | Change ldflags path |
| 2 | CLAUDE.md §2 says 14 rules/experimental; reality is 17/stable | `CLAUDE.md:34-38` | Update table |
| 3 | CLAUDE.md §2 says schema version 0.1.0; reality is 1.0.0 | `CLAUDE.md:38` | Update value |
| 4 | CLAUDE.md §5 layout says core has 11 rules; reality is 14 | `CLAUDE.md:113-114` | Update counts |
| 5 | Makefile missing `make dev` target referenced in CLAUDE.md §18 | `Makefile` | Add target |
| 6 | PROJECT.md §2.2 rule table shows 14 rules; reality is 17 | `PROJECT.md:49-66` | Add 3 missing rows |
| 7 | SECURITY.md has placeholder email | `SECURITY.md` | Update or note |

### P1 — Should fix (consistency)

| # | Gap | File | Fix |
|---|---|---|---|
| 8 | CONTRIBUTING.md references DEVELOPMENT_PLAN.md instead of PROJECT.md | `CONTRIBUTING.md` | Update reference |
| 9 | SECURITY.md says 0.2.x is current; VERSION says 0.1.0 | `SECURITY.md` | Align version |
| 10 | CLAUDE.md §8 still says "Until §7.1 lands" — §7.1 is closed | `CLAUDE.md` | Update wording |
| 11 | CLAUDE.md §17 DoD says `stability: experimental` — rules are now stable | `CLAUDE.md` | Remove or update |

## Implementation order

1. Fix Dockerfile (build blocker)
2. Update CLAUDE.md (single sweep — all stale claims)
3. Update PROJECT.md §2.2 (rule table)
4. Add `make dev` to Makefile
5. Update SECURITY.md
6. Update CONTRIBUTING.md
7. Verify: `make lint test self-scan`
