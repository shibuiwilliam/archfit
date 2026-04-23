# Pack: core — intent

These rules apply to **every** repository archfit scans. They check the smallest set of invariants the rest of archfit's value depends on:

- Is there a place for an agent to start reading? (P1.LOC.001)
- Is the repo structured as vertical slices? (P1.LOC.002)
- Is there a named fast verification loop? (P4.VER.001)
- Are exit codes documented when a CLI ships? (P7.MRD.001)

Each rule is `strong`-evidence — we either see the file or we don't. Severities are `warn`. The core pack is not where opinion lives; opinionated rules live in `web-saas`, `iac`, `mobile`, etc.

This pack is the entry for new contributors: if adding a new rule here feels forced, it probably belongs elsewhere.
