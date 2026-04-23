# Security policy

## Supported versions

archfit is pre-1.0. The most recent minor release receives security
patches; earlier pre-1.0 lines do not. Once 1.0 ships, the supported
window will follow SemVer (the current major + one minor back).

| Version | Supported |
|---|---|
| `0.2.x` | yes (current) |
| `0.1.x` | no — upgrade to 0.2 |

## Reporting a vulnerability

Please report privately. Do **not** open a public issue.

- Preferred: GitHub's private vulnerability reporting for this repository
  (Security → Report a vulnerability).
- Alternative: email the maintainers at `security@example.invalid` (placeholder;
  replace with the real address before a public release).

Include, if possible:

- A minimal reproducer.
- The archfit version (`archfit version`).
- The impact you observed (e.g., arbitrary file read, command injection, data leak).

### Response timeline

- **Acknowledgment** within 72 hours.
- **Initial triage** within 7 days.
- **Fix or public advisory** within 90 days, or sooner for high-severity issues.

## Safe use of archfit

archfit reads files from, and (when available) runs `git` against, the
repository you point it at. Some modes can invoke build tooling. Treat
archfit like any other program that executes code from a target:

- **Scan only repositories you trust**, or run archfit inside a sandbox.
- The `--depth=shallow` mode (planned for Phase 3) will guarantee no
  command execution and is the safe choice for untrusted input.
- The `internal/adapter/exec` boundary is the only place archfit launches
  subprocesses. Every call site is auditable from one file.

## Hardening guidance for downstream integrations

- Pin archfit's version in CI, ideally by SHA.
- Use `archfit scan --json` and parse the output; do not `eval` the terminal
  format.
- Treat exit code `1` as "findings present" (a contract), not as "archfit crashed."
- Store SARIF uploads only from the same job that produced them; do not
  re-upload artifacts from untrusted sources.
