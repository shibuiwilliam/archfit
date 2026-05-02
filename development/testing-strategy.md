# Testing Strategy

## Three-Layer Testing

### Layer 1: Unit Tests (< 5s total)

Pure logic tests. No filesystem, no network, no subprocess.

**What to test**:
- `internal/model/`: type validation, sorting, ParseFailure helper
- `internal/score/`: scoring algorithm with various rule/finding combinations
- `internal/score/metrics.go`: metric computations with property-based tests
- `internal/config/`: config loading and validation
- `internal/rule/`: registry operations, engine evaluation with fake FactStore
- `internal/fix/`: plan generation, change application, rollback logic
- `internal/policy/`: policy enforcement with table tests
- Individual resolvers against in-memory FactStore

**Conventions**:
- Table-driven tests with ≥ 3 cases per function
- Subtests named descriptively: `t.Run("empty_repo_produces_no_findings", ...)`
- No `testify` or assertion libraries — standard `testing` package only
- Property-based tests for scoring and metrics (generate random inputs, assert invariants)

### Layer 2: Pack Tests (< 20s total)

Each rule runs against its fixture repo. Output is diffed against `expected.json`.

**Structure**:
```
packs/core/
├── fixtures/
│   ├── P1.LOC.001/
│   │   ├── input/          # minimal repo that triggers the rule
│   │   │   └── (no CLAUDE.md or AGENTS.md)
│   │   └── expected.json   # expected finding shape
│   ├── P1.LOC.002/
│   │   ├── input/
│   │   │   ├── services/
│   │   │   │   └── billing/
│   │   │   │       └── main.go
│   │   │   └── CLAUDE.md
│   │   └── expected.json
│   └── ...
└── pack_test.go            # table tests iterating fixtures
```

**How pack tests work**:
1. Build a FactStore from the fixture's `input/` directory
2. Run the rule's resolver against that FactStore
3. Canonicalize findings (sort, strip non-deterministic fields)
4. Compare against `expected.json`

**Golden test updates**:
```bash
make update-golden    # regenerate expected.json files
git diff              # review EVERY change carefully
```

Never commit golden updates and code changes in the same commit without a review note.

### Layer 3: End-to-End (< 60s, CI only by default)

Full `archfit scan` on controlled repos in `testdata/e2e/`.

**Structure**:
```
testdata/e2e/
├── golden_clean/       # repo that passes all rules
│   ├── input/
│   └── expected.json
├── golden_findings/    # repo with known findings
│   ├── input/
│   └── expected.json
└── e2e_test.go
```

Asserts: full JSON output shape, overall score, exit code.

## Test Helpers

### Fake FactStore

```go
// Used in unit tests for resolvers
type fakeFactStore struct {
    repo    model.RepoFacts
    git     model.GitFacts
    gitOK   bool
    schemas model.SchemaFacts
}
```

Build with helper functions:
```go
facts := newFakeFactStore(model.RepoFacts{
    Root: "/fake",
    ByBase: map[string][]string{
        "claude.md": {"CLAUDE.md"},
    },
})
```

### Fake exec.Runner

```go
runner := exec.NewFake(map[string]exec.FakeResult{
    "git log": {Stdout: "abc123 feat: something\n", ExitCode: 0},
})
```

### Fake llm.Client

```go
client := llm.NewFake(llm.FakeConfig{
    Response: "This is a fake LLM response",
    Model:    "fake-model",
})
```

## What NOT to Test

- Do not test Go standard library behavior
- Do not test exact string formatting of terminal output (test structure instead)
- Do not pin LLM output as golden (it's non-deterministic by design)
- Do not test with real API keys, real git remotes, or real network

## Running Tests

```bash
make test          # unit + pack tests with -race
make test-short    # fast subset
make e2e           # end-to-end (CI default)
make self-scan     # archfit on itself
```

All tests must pass with `-race` flag. Non-determinism is a bug.

## Adding Tests for a New Rule

1. Create `packs/<pack>/fixtures/<rule-id>/input/` with a minimal repo that triggers the rule
2. Run the resolver manually to get the expected output
3. Save as `packs/<pack>/fixtures/<rule-id>/expected.json`
4. Add a row to the table test in `pack_test.go`
5. Verify: `go test -run TestPack/<rule-id> ./packs/<pack>/`

### AST-Dependent Rules

Rules that consume `ASTFacts` require an additional **parse-failure fixture**. This fixture contains a file that is syntactically invalid (e.g., truncated Go source) and asserts that the resolver produces a `severity: warn` / `evidence_strength: strong` finding via `model.ParseFailure` rather than silently returning zero findings. This prevents AST-dependent rules from appearing to pass on repos where the collector could not parse the source.

## Adding Tests for a New Fixer

1. Create a FactStore that produces a finding for the target rule
2. Call `fixer.Plan(ctx, finding, facts)` and assert the proposed changes
3. Apply changes to an in-memory filesystem (map of path → content)
4. Re-run the resolver against the updated facts
5. Assert the finding is gone
