---
id: 0015
title: AST collector — structured code analysis for Phase 1 rules
status: accepted
date: 2026-05-02
supersedes: null
---

## Context

Phase 1 introduces rules that require understanding code structure beyond
file presence and path patterns (PROJECT.md §3.2, §6.1.1):

- **P3.EXP.002** — no `init()` cross-package registration (Go)
- **P3.EXP.003** — reflection / metaprogramming density bounded
- **P3.EXP.004** — single-implementation interfaces flagged (Go)
- **P3.EXP.005** — global mutable state minimized

None of these can be detected by file-presence or regex. They require
parsed syntax trees.

The current FactStore has no AST accessor. Resolvers are pure functions of
FactStore (CLAUDE.md §6 invariant 1), so parsing must happen in a collector,
not in resolvers.

## Decision

### 1. New package: `internal/collector/ast/`

```
internal/collector/ast/
├── ast.go            # Collect() entry point, ASTFacts builder
├── goast/            # Go-only analysis using go/parser + go/ast
│   ├── goast.go      # ParseGoFile(), returns GoFileFacts
│   └── goast_test.go
├── fake.go           # Fake for resolver tests
└── README.md
```

Phase 1 supports **Go only** via `go/parser`. Tree-sitter for cross-language
support (TypeScript, Python, Rust, Java, Ruby) is deferred to Phase 1.5.
The `Collect()` signature is language-agnostic so tree-sitter slots in
without changing the FactStore interface.

### 2. FactStore extension

Add one method using the established optional-return pattern (ADR 0005):

```go
// AST returns AST-derived facts, or (ASTFacts{}, false) when the collector
// was skipped (e.g., no parseable source files, or depth=shallow).
AST() (ASTFacts, bool)
```

`AST()` returns `(zero, false)` when:

- `--depth=shallow` (AST collector is skipped entirely)
- No parseable source files detected

### 3. ASTFacts model types

```go
// ASTFacts is the top-level container for AST-derived facts.
type ASTFacts struct {
    // GoFiles contains per-file analysis for Go source files.
    GoFiles []GoFileFacts
    // ParseFailures records files that could not be parsed.
    // Each entry becomes a ParseFailure finding in the resolver.
    ParseFailures []ParseFailureEntry
}

// GoFileFacts contains structural facts extracted from a single Go file.
type GoFileFacts struct {
    Path           string
    Package        string
    InitFunctions  []InitFact       // init() declarations in this file
    PkgLevelVars   []PkgVarFact     // package-level var declarations
    Interfaces     []InterfaceFact  // interface type declarations
    ReflectImports bool             // imports "reflect" or "reflect" subpkg
    ReflectCalls   int              // count of reflect.* call expressions (depth=deep)
}

// InitFact describes a single init() function.
type InitFact struct {
    Line          int
    // CrossPkgCalls lists function calls to other packages inside init().
    // e.g., ["http.HandleFunc", "sql.Register", "prometheus.MustRegister"]
    CrossPkgCalls []string
}

// PkgVarFact describes a package-level var declaration.
type PkgVarFact struct {
    Name    string
    Line    int
    Mutable bool   // true for var, false for const
    Type    string // best-effort type string; "" if not determinable
}

// InterfaceFact describes an interface type declaration.
type InterfaceFact struct {
    Name       string
    Line       int
    MethodCount int
    // Implementors is populated only at depth=deep. Lists type names in the
    // same module that satisfy the interface (best-effort, no cross-module).
    Implementors []string
}

// ParseFailureEntry records a file that failed to parse.
type ParseFailureEntry struct {
    Path  string
    Error string
}
```

### 4. Depth modes

| Depth | What runs | Cost |
|-------|-----------|------|
| `shallow` | AST collector **skipped** | 0 |
| `standard` | Declaration-level: init functions, pkg-level vars, interface declarations, imports | O(files) with `go/parser.ParseFile` in declaration-only mode |
| `deep` | Full body analysis: reflect call counting, cross-pkg calls in init(), interface implementor search | O(files × complexity) |

### 5. Resource limits

- **File size cap:** 1 MiB. Files larger than this are skipped with a
  `ParseFailureEntry` (reason: "file exceeds size limit").
- **Per-file timeout:** 5 seconds. Parse exceeding this emits a
  `ParseFailureEntry` (reason: "parse timeout exceeded").
- **Total file cap:** 10,000 Go files. Beyond this, the collector stops
  and emits a single `ParseFailureEntry` summarizing the skip.

### 6. Parse failure handling

Parse failures are **never silent**. The collector records them in
`ASTFacts.ParseFailures`. Resolvers that consume AST facts must convert
these into findings using `model.ParseFailure` (severity: warn,
evidence_strength: strong). This is enforced by fixture tests — every
AST-backed rule must have a parse-failure fixture.

### 7. Caching (Phase 1, opt-in)

When `--cache-dir` is set (default: none; convention: `.archfit-cache/ast/`):

- Cache key: SHA-256 of file content + collector version string.
- Cache value: serialized `GoFileFacts` (gob or JSON).
- Stale entries evicted on read if version mismatches.
- Cache is advisory: missing or corrupt entries cause a re-parse, not an error.

Caching is not required for Phase 1 correctness. It is a performance
optimization for large repos and may ship after the initial skeleton.

### 8. Wiring

The collector is wired in `internal/core/scheduler.go`, following the same
pattern as other collectors:

```go
var astFacts model.ASTFacts
astOK := false
if in.Depth != "shallow" {
    a, err := collectast.Collect(ctx, in.Root, in.Depth)
    if err == nil {
        astFacts = a
        astOK = true
    }
}
```

The `newFactStore` constructor gains `ast ASTFacts, astOK bool` parameters.

## Consequences

- `model.FactStore` interface gains one method. All implementations
  (scheduler, test fakes) must be updated. Fakes return `(zero, false)`.
- New model types (`ASTFacts`, `GoFileFacts`, etc.) are added to
  `internal/model/`. This is a public-surface change justified by this ADR.
- Phase 1 rules (P3.EXP.002–005) consume `facts.AST()` and are skipped
  (with `applies_to.languages: [go]`) when AST is unavailable.
- `go/parser` is a stdlib dependency — no new external dependency required.
- Tree-sitter integration (Phase 1.5) will add a `treesitter/` sub-package
  and populate language-specific fact types alongside `GoFileFacts`.
  The `ASTFacts` struct is designed to accommodate this without breaking
  the FactStore interface.
- Existing rules and collectors are unaffected.
