# archfit GitHub Action

Composite action that runs an archfit architecture fitness scan on pull requests.

## Usage

```yaml
name: archfit
on:
  pull_request:

permissions:
  contents: read
  pull-requests: write

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: ./.github/archfit-action
        with:
          fail-on: error
          comment: true
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Inputs

| Input             | Default  | Description                                    |
|-------------------|----------|------------------------------------------------|
| `fail-on`         | `error`  | Severity threshold (`info\|warn\|error\|critical`) |
| `baseline-branch` | `main`   | Branch to compare against for diff             |
| `comment`         | `true`   | Post a PR comment with scan results            |
| `sarif`           | `false`  | Generate `archfit.sarif` for Code Scanning     |
| `version`         | `latest` | archfit version to install                     |

## Outputs

- Exit code 0 when no findings at or above the `fail-on` threshold.
- Exit code 1 when findings exceed the threshold.
- `archfit.sarif` written to the workspace root when `sarif: true`.

## Notes

- The action installs archfit via `go install`; a Go toolchain must be available.
- PR comments require `GITHUB_TOKEN` and `pull-requests: write` permission.
- Fork PRs may not have comment permission; the action logs a warning and continues.
- If the baseline branch does not exist on the remote, the diff step is skipped.
