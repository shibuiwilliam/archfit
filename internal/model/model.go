// Package model defines the core types shared across archfit.
//
// The types here track schemas/rule.schema.json and schemas/output.schema.json.
// If you change a type in a way that affects JSON output, update the schema and
// the golden tests in the same PR, and follow the stability rules in CLAUDE.md §9.
package model

import (
	"context"
	"fmt"
	"regexp"
	"sort"
)

// Principle identifies one of the seven architecture principles archfit evaluates.
type Principle string

// Architecture principle constants (P1 through P7).
const (
	P1Locality            Principle = "P1"
	P2SpecFirst           Principle = "P2"
	P3ShallowExplicitness Principle = "P3"
	P4Verifiability       Principle = "P4"
	P5Aggregation         Principle = "P5"
	P6Reversibility       Principle = "P6"
	P7MachineReadability  Principle = "P7"
)

var allPrinciples = []Principle{
	P1Locality, P2SpecFirst, P3ShallowExplicitness, P4Verifiability,
	P5Aggregation, P6Reversibility, P7MachineReadability,
}

// AllPrinciples returns a copy of all defined principles.
func AllPrinciples() []Principle {
	out := make([]Principle, len(allPrinciples))
	copy(out, allPrinciples)
	return out
}

// Valid reports whether p is a recognised principle.
func (p Principle) Valid() bool {
	switch p {
	case P1Locality, P2SpecFirst, P3ShallowExplicitness, P4Verifiability,
		P5Aggregation, P6Reversibility, P7MachineReadability:
		return true
	}
	return false
}

// Severity classifies a finding's impact level.
type Severity string

// Severity level constants from least to most severe.
const (
	SeverityInfo     Severity = "info"
	SeverityWarn     Severity = "warn"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Rank orders severities from least to most severe. Used for sorting
// findings descending by severity and for --fail-on threshold comparisons.
func (s Severity) Rank() int {
	switch s {
	case SeverityInfo:
		return 1
	case SeverityWarn:
		return 2
	case SeverityError:
		return 3
	case SeverityCritical:
		return 4
	}
	return 0
}

// Valid reports whether s is a recognised severity.
func (s Severity) Valid() bool { return s.Rank() > 0 }

// EvidenceStrength describes how reliably a rule's finding can be verified.
type EvidenceStrength string

// Evidence strength constants from strongest to weakest.
const (
	EvidenceStrong  EvidenceStrength = "strong"
	EvidenceMedium  EvidenceStrength = "medium"
	EvidenceWeak    EvidenceStrength = "weak"
	EvidenceSampled EvidenceStrength = "sampled"
)

// Valid reports whether e is a recognised evidence strength.
func (e EvidenceStrength) Valid() bool {
	switch e {
	case EvidenceStrong, EvidenceMedium, EvidenceWeak, EvidenceSampled:
		return true
	}
	return false
}

// Stability tracks a rule's lifecycle stage.
type Stability string

// Stability lifecycle constants.
const (
	StabilityExperimental Stability = "experimental"
	StabilityStable       Stability = "stable"
	StabilityDeprecated   Stability = "deprecated"
)

// Valid reports whether s is a recognised stability value.
func (s Stability) Valid() bool {
	switch s {
	case StabilityExperimental, StabilityStable, StabilityDeprecated:
		return true
	}
	return false
}

// Applicability declares which project types, languages, and paths a rule targets.
type Applicability struct {
	ProjectTypes []string
	Languages    []string
	PathGlobs    []string
}

// Remediation describes how to fix a finding.
type Remediation struct {
	Summary     string `json:"summary"`
	GuideRef    string `json:"guide_ref,omitempty"`
	AutoFixable bool   `json:"auto_fixable,omitempty"`
}

// Rule is a single archfit evaluation rule with its resolver function.
type Rule struct {
	ID               string
	Principle        Principle
	Dimension        string
	Title            string
	Severity         Severity
	EvidenceStrength EvidenceStrength
	Stability        Stability
	AppliesTo        Applicability
	Rationale        string
	Weight           float64
	Remediation      Remediation
	Resolver         ResolverFunc
}

// ruleIDPattern mirrors schemas/rule.schema.json. Kept here so tests can assert
// the pattern without loading the schema.
var ruleIDPattern = regexp.MustCompile(`^P[1-7]\.[A-Z]{3}\.\d{3}$`)

// Validate checks that all required fields conform to the rule schema.
func (r Rule) Validate() error {
	if !ruleIDPattern.MatchString(r.ID) {
		return fmt.Errorf("rule %q: id must match P<n>.<DIM>.<nnn>", r.ID)
	}
	if !r.Principle.Valid() {
		return fmt.Errorf("rule %s: invalid principle %q", r.ID, r.Principle)
	}
	if !r.Severity.Valid() {
		return fmt.Errorf("rule %s: invalid severity %q", r.ID, r.Severity)
	}
	if !r.EvidenceStrength.Valid() {
		return fmt.Errorf("rule %s: invalid evidence_strength %q", r.ID, r.EvidenceStrength)
	}
	if !r.Stability.Valid() {
		return fmt.Errorf("rule %s: invalid stability %q", r.ID, r.Stability)
	}
	// Severity ↔ evidence matrix (CLAUDE.md §6 invariant 2):
	//   critical → strong only
	//   error    → strong only
	//   warn     → medium, strong, or sampled (weak rejected)
	//   info     → any
	switch {
	case r.Severity == SeverityCritical && r.EvidenceStrength != EvidenceStrong:
		return fmt.Errorf("rule %s: severity critical requires evidence_strength strong, got %s", r.ID, r.EvidenceStrength)
	case r.Severity == SeverityError && r.EvidenceStrength != EvidenceStrong:
		return fmt.Errorf("rule %s: severity error requires evidence_strength strong, got %s", r.ID, r.EvidenceStrength)
	case r.Severity == SeverityWarn && r.EvidenceStrength == EvidenceWeak:
		return fmt.Errorf("rule %s: severity warn requires evidence_strength medium or stronger, got weak", r.ID)
	}
	if r.Resolver == nil {
		return fmt.Errorf("rule %s: resolver must not be nil", r.ID)
	}
	if r.Remediation.Summary == "" {
		return fmt.Errorf("rule %s: remediation.summary is required", r.ID)
	}
	return nil
}

// ResolverFunc is the function signature for rule evaluation logic.
type ResolverFunc func(ctx context.Context, facts FactStore) ([]Finding, []Metric, error)

// Finding is a single rule violation or observation emitted by a resolver.
type Finding struct {
	RuleID           string           `json:"rule_id"`
	Principle        Principle        `json:"principle"`
	Severity         Severity         `json:"severity"`
	EvidenceStrength EvidenceStrength `json:"evidence_strength"`
	Confidence       float64          `json:"confidence"`
	Path             string           `json:"path"`
	Message          string           `json:"message"`
	Evidence         map[string]any   `json:"evidence"`
	Remediation      Remediation      `json:"remediation"`
	// LLMSuggestion is populated only when `--with-llm` is used. It carries
	// the LLM-authored explanation/remediation text. Omitted from JSON output
	// when empty, so the default scan path remains byte-identical (schema
	// version 0.1.0 stays additive per CLAUDE.md §9). See ADR 0003.
	LLMSuggestion *LLMSuggestion `json:"llm_suggestion,omitempty"`
}

// LLMSuggestion is the structured form of an LLM-authored explanation.
// The text is always present; other fields record provenance for auditability.
type LLMSuggestion struct {
	Text         string `json:"text"`
	Model        string `json:"model,omitempty"`
	CacheHit     bool   `json:"cache_hit,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
	LatencyMS    int64  `json:"latency_ms,omitempty"`
	InputTokens  int64  `json:"input_tokens,omitempty"`
	OutputTokens int64  `json:"output_tokens,omitempty"`
}

// Metric is a numeric measurement emitted alongside findings.
type Metric struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Unit      string  `json:"unit,omitempty"`
	Principle string  `json:"principle,omitempty"`
}

// ParseFailure returns a warn-severity, strong-evidence Finding describing a
// collector or resolver's failure to parse input it was asked to interpret.
// This encodes the rule in CLAUDE.md §13: "Parse failures are a finding, not a
// reason to return zero findings." Rules that delegate to a parser should call
// this helper rather than returning an error from the resolver.
func ParseFailure(ruleID, path, detail string) Finding {
	return Finding{
		RuleID:           ruleID,
		Severity:         SeverityWarn,
		EvidenceStrength: EvidenceStrong,
		Confidence:       1.0,
		Path:             path,
		Message:          "parse failure: " + detail,
		Evidence: map[string]any{
			"parse_failure": true,
			"detail":        detail,
		},
	}
}

// SortFindings orders findings deterministically per CLAUDE.md §9:
// severity desc, then rule_id asc, then path asc.
func SortFindings(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		if ri, rj := fs[i].Severity.Rank(), fs[j].Severity.Rank(); ri != rj {
			return ri > rj
		}
		if fs[i].RuleID != fs[j].RuleID {
			return fs[i].RuleID < fs[j].RuleID
		}
		return fs[i].Path < fs[j].Path
	})
}

// FactStore is a read-only view of collected facts. Resolvers receive it; they
// do not build it. See CLAUDE.md §5 — this is how aggregation is enforced.
type FactStore interface {
	// Repo returns the collected repo-wide facts.
	Repo() RepoFacts
	// Git returns git facts, or (GitFacts{}, false) if git was unavailable or disabled.
	Git() (GitFacts, bool)
	// Schemas returns JSON-Schema facts collected from the repo.
	Schemas() SchemaFacts
	// Commands returns command timing facts, or (CommandFacts{}, false) when
	// the command collector was not run (e.g. depth != "deep"). See ADR 0005.
	Commands() (CommandFacts, bool)
	// DepGraph returns dependency graph facts, or (DepGraphFacts{}, false) when
	// the source was not parseable or the collector was skipped. See ADR 0005.
	DepGraph() (DepGraphFacts, bool)
	// Languages returns the language file counts from the filesystem collector.
	// Keys are lowercase language names (e.g. "go", "python", "java").
	// Used by the rule engine to skip rules whose AppliesTo.Languages don't
	// match the repo's detected ecosystem. See ADR 0010.
	Languages() map[string]int
	// Ecosystems returns typed ecosystem facts (CI platforms, deployment tools,
	// frameworks). See ADR 0011. Resolvers should use this instead of maintaining
	// private keyword tables for ecosystem detection.
	Ecosystems() EcosystemFacts
	// AST returns AST-derived facts, or (ASTFacts{}, false) when the collector
	// was skipped (e.g., no parseable source, or depth=shallow). See ADR 0015.
	AST() (ASTFacts, bool)
}

// EcosystemEntry represents a detected framework, platform, or tool. See ADR 0011.
type EcosystemEntry struct {
	Name       string   `json:"name"`
	Confidence float64  `json:"confidence"`
	Markers    []string `json:"markers"`
}

// EcosystemFacts holds typed ecosystem detection results. Resolvers use
// Has/HasCI/HasDeployment/CIFiles instead of private keyword tables.
type EcosystemFacts struct {
	Detected []EcosystemEntry `json:"detected"`
}

// Has returns true if the named ecosystem was detected.
func (ef EcosystemFacts) Has(name string) bool {
	for _, e := range ef.Detected {
		if e.Name == name {
			return true
		}
	}
	return false
}

// HasCI returns true if any CI platform was detected.
func (ef EcosystemFacts) HasCI() bool {
	for _, e := range ef.Detected {
		if isCIPlatform(e.Name) {
			return true
		}
	}
	return false
}

// CIFiles returns all detected CI configuration file paths.
func (ef EcosystemFacts) CIFiles() []string {
	var files []string
	for _, e := range ef.Detected {
		if isCIPlatform(e.Name) {
			files = append(files, e.Markers...)
		}
	}
	return files
}

// HasDeployment returns true if any deployment tool was detected.
func (ef EcosystemFacts) HasDeployment() bool {
	for _, e := range ef.Detected {
		if isDeployPlatform(e.Name) {
			return true
		}
	}
	return false
}

func isCIPlatform(name string) bool {
	switch name {
	case "github-actions", "gitlab-ci", "circleci", "buildkite",
		"jenkins", "travis", "azure-pipelines", "bitbucket-pipelines",
		"woodpecker":
		return true
	}
	return false
}

func isDeployPlatform(name string) bool {
	switch name {
	case "docker", "kubernetes", "helm", "terraform",
		"aws-cdk", "serverless", "cloud-build", "skaffold",
		"heroku", "render", "fly-io", "railway", "vercel", "netlify":
		return true
	}
	return false
}

// SchemaFacts aggregates what the schema collector saw on the repository.
type SchemaFacts struct {
	Files []SchemaFile
}

// SchemaFile is a single JSON-Schema file the collector encountered.
// ParseError is set when the file was found but could not be decoded — the
// consumer decides whether to emit a ParseFailure finding (CLAUDE.md §13).
type SchemaFile struct {
	Path       string
	ID         string // from the top-level "$id" field; empty when absent.
	ParseError string
}

// RepoFacts aggregates filesystem-level facts collected from the repository.
type RepoFacts struct {
	Root      string
	Files     []FileFact
	ByPath    map[string]FileFact
	ByBase    map[string][]string // lowercase basename -> paths
	Languages map[string]int      // language-by-extension -> file count
}

// FileFact holds metadata for a single file in the repository.
type FileFact struct {
	Path  string
	Size  int64
	Lines int
	Ext   string
}

// GitFacts holds facts collected from git history.
type GitFacts struct {
	CommitCount   int
	RecentCommits []Commit
	CurrentBranch string
	CurrentCommit string
}

// Commit represents a single git commit sampled from the repository.
type Commit struct {
	Hash    string
	Subject string
	// FilesChanged is a coarse count from --numstat; 0 if unknown.
	FilesChanged int
}

// CommandFacts holds timing results from running verification commands.
type CommandFacts struct {
	Results []CommandResult
}

// CommandResult records a timed command execution.
type CommandResult struct {
	Command    string `json:"command"`
	DurationMS int64  `json:"duration_ms"`
	ExitCode   int    `json:"exit_code"`
	Layer      string `json:"layer,omitempty"` // verification layer name (e.g. "lint", "unit")
}

// DepGraphFacts holds the dependency graph from source analysis.
type DepGraphFacts struct {
	PackageCount int    `json:"package_count"`
	MaxReach     int    `json:"max_reach"`
	MaxReachPkg  string `json:"max_reach_pkg"`
}

// ASTFacts holds AST-derived facts. See ADR 0015.
// Phase 1 supports Go only (go/parser). Tree-sitter is Phase 1.5.
type ASTFacts struct {
	// GoFiles contains per-file analysis for Go source files.
	GoFiles []GoFileFacts
	// ParseFailures records files that could not be parsed.
	// Resolvers convert these into ParseFailure findings.
	ParseFailures []ASTParseFailure
}

// GoFileFacts contains structural facts extracted from a single Go file.
type GoFileFacts struct {
	Path           string
	Package        string
	InitFunctions  []InitFact
	PkgLevelVars   []PkgVarFact
	Interfaces     []InterfaceFact
	ReflectImports bool // imports "reflect"
	ReflectCalls   int  // count of reflect.* call expressions (depth=deep only)
}

// InitFact describes a single init() function.
type InitFact struct {
	Line int
	// CrossPkgCalls lists qualified function calls to other packages inside
	// init(), e.g. "http.HandleFunc", "sql.Register".
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
	Name        string
	Line        int
	MethodCount int
	// Implementors is populated only at depth=deep. Lists type names in the
	// same module that satisfy the interface (best-effort, no cross-module).
	Implementors []string
}

// ASTParseFailure records a file that failed to parse during AST collection.
type ASTParseFailure struct {
	Path  string
	Error string
}
