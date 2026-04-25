# API Reference — Core Types and Interfaces

This document lists the key types and interfaces Claude Code needs when working on archfit. All types are in `internal/model/model.go` unless noted.

## Principles

```go
type Principle string

const (
    P1Locality            Principle = "P1"
    P2SpecFirst           Principle = "P2"
    P3ShallowExplicitness Principle = "P3"
    P4Verifiability       Principle = "P4"
    P5Aggregation         Principle = "P5"
    P6Reversibility       Principle = "P6"
    P7MachineReadability  Principle = "P7"
)
```

## Severity

```go
type Severity string

const (
    SeverityInfo     Severity = "info"      // Rank 1
    SeverityWarn     Severity = "warn"      // Rank 2
    SeverityError    Severity = "error"     // Rank 3
    SeverityCritical Severity = "critical"  // Rank 4
)
```

**Constraint**: `error` severity requires `strong` evidence (enforced by `Rule.Validate()`).

## Rule

```go
type Rule struct {
    ID               string           // "P1.LOC.001" — must match ^P[1-7]\.[A-Z]{3}\.\d{3}$
    Principle        Principle
    Dimension        string           // "LOC", "SPC", "MRD", etc.
    Title            string
    Severity         Severity
    EvidenceStrength EvidenceStrength // strong, medium, weak, sampled
    Stability        Stability        // experimental, stable, deprecated
    AppliesTo        Applicability
    Rationale        string
    Weight           float64          // for scoring normalization
    Remediation      Remediation
    Resolver         ResolverFunc     // (ctx, FactStore) → ([]Finding, []Metric, error)
}
```

## Finding

```go
type Finding struct {
    RuleID           string           `json:"rule_id"`
    Principle        Principle        `json:"principle"`
    Severity         Severity         `json:"severity"`
    EvidenceStrength EvidenceStrength `json:"evidence_strength"`
    Confidence       float64          `json:"confidence"`      // 0.0–1.0
    Path             string           `json:"path"`            // repo-relative
    Message          string           `json:"message"`
    Evidence         map[string]any   `json:"evidence"`        // must be JSON-marshalable
    Remediation      Remediation      `json:"remediation"`
    LLMSuggestion    *LLMSuggestion   `json:"llm_suggestion,omitempty"`
}
```

**Sort order** (deterministic): severity desc → rule_id asc → path asc.

## FactStore

```go
type FactStore interface {
    Repo() RepoFacts                    // always available
    Git() (GitFacts, bool)              // false when git unavailable
    Schemas() SchemaFacts               // always available
    Commands() (CommandFacts, bool)     // false when depth != "deep"
    DepGraph() (DepGraphFacts, bool)    // false when source not parseable
}
```

### RepoFacts

```go
type RepoFacts struct {
    Root      string                     // absolute path to repo root
    Files     []FileFact                 // all files found
    ByPath    map[string]FileFact        // path → fact
    ByBase    map[string][]string        // lowercase basename → paths
    Languages map[string]int             // extension → count
}
```

### GitFacts

```go
type GitFacts struct {
    CommitCount   int
    RecentCommits []Commit
    CurrentBranch string
    CurrentCommit string
}
```

### SchemaFacts

```go
type SchemaFacts struct {
    Files []SchemaFile  // path, $id, parse error
}
```

## ResolverFunc

```go
type ResolverFunc func(ctx context.Context, facts FactStore) ([]Finding, []Metric, error)
```

**Rules for resolvers**:
- Must be pure functions of FactStore — no I/O, no side effects
- Return `[]Finding` (may be empty), `[]Metric` (may be empty), error
- Use `model.ParseFailure()` for parse errors instead of returning an error
- Never import from `internal/adapter/` or `internal/collector/`

## Fix Engine (internal/fix/)

### Fixer Interface

```go
type Fixer interface {
    RuleID() string
    Plan(ctx context.Context, finding model.Finding, facts model.FactStore) ([]Change, error)
    NeedsLLM() bool
}
```

### Change

```go
type Change struct {
    Path    string       // repo-relative
    Action  ChangeAction // "create", "modify", "append"
    Content []byte
    Preview string       // human-readable description
}
```

### Engine

```go
type Engine struct { /* ... */ }

func NewEngine() *Engine
func (e *Engine) Register(f Fixer)
func (e *Engine) Fix(ctx context.Context, input Input) (Result, error)

type Input struct {
    Root     string
    RuleIDs  []string                                      // empty = all fixable
    DryRun   bool
    Facts    model.FactStore
    Findings []model.Finding
    Scanner  func(ctx context.Context) (core.ScanResult, error)
}

type Result struct {
    Plan      Plan
    Applied   []AppliedChange
    Verified  bool
    NewIssues []model.Finding
}
```

## Rule Registry (internal/rule/)

```go
type Registry struct { /* ... */ }

func NewRegistry() *Registry
func (r *Registry) Register(packName string, rules ...model.Rule) error
func (r *Registry) Rules() []model.Rule           // all rules, sorted
func (r *Registry) Rule(id string) (model.Rule, bool)
func (r *Registry) Packs() map[string][]string    // pack name → rule IDs
func (r *Registry) AllPacks() []PackInfo           // pack metadata
```

## LLM Client (internal/adapter/llm/)

```go
type Client interface {
    Explain(ctx context.Context, rule model.Rule, finding model.Finding, prompt Prompt) (Suggestion, error)
    Close() error
}

type Suggestion struct {
    Text      string
    Model     string
    CacheHit  bool
    Truncated bool
    LatencyMS int64
}
```

**Composition**: `inner → Budget → Cached` (outermost).

## Score (internal/score/)

```go
type Scores struct {
    Overall     float64
    ByPrinciple map[Principle]float64
}

func Compute(rules []model.Rule, findings []model.Finding) Scores
```
