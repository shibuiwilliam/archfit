# Deployment — archfit

archfit is a CLI tool distributed as pre-built binaries. There is no
long-running service to deploy.

## Release process

1. Tag a release: `git tag v0.x.y`
2. CI builds cross-platform binaries (`linux/amd64`, `linux/arm64`,
   `darwin/amd64`, `darwin/arm64`, `windows/amd64`)
3. CI publishes binaries to GitHub Releases

## How to verify

- Download the binary for your platform from the release page.
- Run `archfit version` to confirm the version matches the tag.
- Run `archfit scan .` on the archfit repo itself — it must exit 0.

## How to roll back

Revert to a previous release:

1. Users: download the previous version from GitHub Releases.
2. Maintainers: `git revert <commit>` and cut a patch release.

There is no server-side state to migrate or roll back.

## CI workflows

- `.github/workflows/ci.yml`: lint, test, self-scan, cross-build on every push.
- Release workflow: triggered on tag push (planned, not yet implemented).
