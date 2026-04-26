// Binary archfit is the CLI entrypoint. main.go is intentionally the only file
// that wires collectors, the registry, and packs together — this is where the
// explicit (non-init) registration lives, per CLAUDE.md §3.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	osexec "os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/shibuiwilliam/archfit/internal/adapter/exec"
	"github.com/shibuiwilliam/archfit/internal/adapter/llm"
	collectfs "github.com/shibuiwilliam/archfit/internal/collector/fs"
	"github.com/shibuiwilliam/archfit/internal/config"
	"github.com/shibuiwilliam/archfit/internal/contract"
	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/fix"
	"github.com/shibuiwilliam/archfit/internal/fix/static"
	"github.com/shibuiwilliam/archfit/internal/model"
	"github.com/shibuiwilliam/archfit/internal/packman"
	"github.com/shibuiwilliam/archfit/internal/policy"
	"github.com/shibuiwilliam/archfit/internal/report"
	"github.com/shibuiwilliam/archfit/internal/rule"
	"github.com/shibuiwilliam/archfit/internal/version"
	agenttool "github.com/shibuiwilliam/archfit/packs/agent-tool"
	corepack "github.com/shibuiwilliam/archfit/packs/core"
)

// Exit codes are part of the stability contract. See docs/exit-codes.md.
// Do not renumber without an ADR and a major-version bump.
const (
	exitOK              = 0
	exitFindingsAtLevel = 1
	exitUsage           = 2
	exitRuntimeError    = 3
	exitConfigError     = 4
	// exitContractSoftMiss indicates that all hard constraints passed but at
	// least one soft target was missed. Advisory, not blocking.
	// See docs/adr/0008-contract-exit-codes.md.
	exitContractSoftMiss = 5
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return exitUsage
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "scan":
		return cmdScan(rest, stdout, stderr)
	case "score":
		return cmdScore(rest, stdout, stderr)
	case "explain":
		return cmdExplain(rest, stdout, stderr)
	case "list-rules":
		return cmdListRules(rest, stdout, stderr)
	case "list-packs":
		return cmdListPacks(rest, stdout, stderr)
	case "validate-config":
		return cmdValidateConfig(rest, stdout, stderr)
	case "init":
		return cmdInit(rest, stdout, stderr)
	case "check":
		return cmdCheck(rest, stdout, stderr)
	case "report":
		return cmdReport(rest, stdout, stderr)
	case "diff":
		return cmdDiff(rest, stdout, stderr)
	case "fix":
		return cmdFix(rest, stdout, stderr)
	case "trend":
		return cmdTrend(rest, stdout, stderr)
	case "compare":
		return cmdCompare(rest, stdout, stderr)
	case "validate-pack":
		return cmdValidatePack(rest, stdout, stderr)
	case "new-pack":
		return cmdNewPack(rest, stdout, stderr)
	case "test-pack":
		return cmdTestPack(rest, stdout, stderr)
	case "contract":
		return cmdContract(rest, stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "archfit %s\n", version.Version)
		return exitOK
	case "help", "--help", "-h":
		printUsage(stdout)
		return exitOK
	}

	fmt.Fprintf(stderr, "archfit: unknown command %q\n", cmd)
	printUsage(stderr)
	return exitUsage
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `archfit — architecture fitness evaluator for the coding-agent era

usage:
  archfit scan [path]                  run all enabled rules (default: .)
  archfit check <rule-id> [path]       run a single rule against the target
  archfit score [path]                 summary only (same scan, no finding list)
  archfit report [path]                Markdown report (shorthand for scan --format=md)
  archfit diff <baseline.json> [current.json]
                                       compare findings between two scans
  archfit fix [rule-id] [path]         auto-fix findings (strong-evidence rules)
  archfit trend                        show score trends from archived scans
  archfit compare <f1.json> <f2.json> [...]
                                       compare scans across repos
  archfit explain <rule-id>            show a rule's rationale and remediation
  archfit init [path]                  scaffold .archfit.yaml with defaults
  archfit list-rules                   list all registered rules
  archfit list-packs                   list all registered rule packs
  archfit validate-config [path]       check .archfit.yaml without scanning
  archfit contract check [path]         check scan results against .archfit-contract.yaml
  archfit contract init [path]         scaffold a contract from current scan results
  archfit validate-pack <path>         check pack structure
  archfit new-pack <name> [path]       scaffold a new rule pack
  archfit test-pack <path>             run pack tests
  archfit version                      print the version

global flags (where applicable):
  --format {terminal|json|md|sarif}    output format (default: terminal)
  --json                               shorthand for --format=json
  --fail-on {info|warn|error|critical} exit 1 when any finding meets this level (default: error)
  -C <dir>                             change to dir before running (like git -C)
  --config <file>                      path to config file (default: .archfit.yaml in target dir)
  --depth {shallow|standard|deep}      scan depth (default: standard; deep runs verification commands)
  --with-llm                           enrich findings with LLM-authored explanations
                                       (opt-in; requires ANTHROPIC_API_KEY, OPENAI_API_KEY, or GOOGLE_API_KEY)
  --llm-backend {gemini|openai|claude} LLM provider (auto-detected from env if omitted)
  --llm-budget N                       cap the number of LLM calls per run (default: 5)
  --record <dir>                       save scan results (JSON + Markdown) to a timestamped
                                       subdirectory under <dir> (e.g., --record .archfit-records)
  --explain-coverage                   append a summary showing which rules fired vs. passed

Exit codes:
  0   success (or: findings below --fail-on threshold)
  1   findings present at or above --fail-on threshold (or: contract hard violation)
  2   usage error
  3   runtime error
  4   configuration error
  5   contract soft target missed (no hard violations)

See docs/exit-codes.md and PROJECT.md for the full contract.
`)
}

type scanFlags struct {
	format          string
	json            bool
	failOn          string
	workDir         string
	configPath      string
	path            string
	depth           string
	policy          string
	withLLM         bool
	llmBackend      string
	llmBudget       int
	recordDir       string
	explainCoverage bool
}

func parseScanFlags(args []string, cmd string) (scanFlags, error) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var f scanFlags
	fs.StringVar(&f.format, "format", "terminal", "output format")
	fs.BoolVar(&f.json, "json", false, "shorthand for --format=json")
	fs.StringVar(&f.failOn, "fail-on", "error", "severity threshold that causes a non-zero exit")
	fs.StringVar(&f.workDir, "C", "", "change to directory before running")
	fs.StringVar(&f.configPath, "config", "", "path to config file (default: .archfit.yaml in target dir)")
	fs.StringVar(&f.depth, "depth", "standard", "scan depth: shallow, standard, or deep")
	fs.StringVar(&f.policy, "policy", "", "path to organization policy file (JSON)")
	fs.BoolVar(&f.withLLM, "with-llm", false, "enrich findings with LLM-authored explanations (opt-in; requires ANTHROPIC_API_KEY, OPENAI_API_KEY, or GOOGLE_API_KEY)")
	fs.StringVar(&f.llmBackend, "llm-backend", "", "LLM provider: gemini, openai, or claude (auto-detected from env if omitted)")
	fs.IntVar(&f.llmBudget, "llm-budget", 5, "maximum LLM calls per run (only when --with-llm)")
	fs.StringVar(&f.recordDir, "record", "", "save scan results (JSON + Markdown) to a timestamped subdirectory under this path")
	fs.BoolVar(&f.explainCoverage, "explain-coverage", false, "append a coverage summary showing which rules fired vs. passed silently")
	if err := fs.Parse(args); err != nil {
		return f, err
	}
	if f.json {
		f.format = "json"
	}
	rest := fs.Args()
	if len(rest) > 1 {
		return f, errors.New("too many positional arguments")
	}
	if len(rest) == 1 {
		f.path = rest[0]
	} else {
		f.path = "."
	}
	if f.workDir != "" {
		f.path = joinIfRelative(f.workDir, f.path)
	}
	return f, nil
}

func joinIfRelative(dir, p string) string {
	if p == "" || p == "." {
		return dir
	}
	if p != "" && (p[0] == '/' || (len(p) >= 2 && p[1] == ':')) {
		return p // absolute
	}
	return filepath.Join(dir, p)
}

// buildRegistry is the ONE place where packs are wired in. Add new packs here.
func buildRegistry() (*rule.Registry, error) {
	reg := rule.NewRegistry()
	if err := corepack.Register(reg); err != nil {
		return nil, err
	}
	if err := agenttool.Register(reg); err != nil {
		return nil, err
	}
	return reg, nil
}

func cmdScan(args []string, stdout, stderr io.Writer) int {
	flags, err := parseScanFlags(args, "scan")
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return exitUsage
	}
	format, err := report.ParseFormat(flags.format)
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return exitUsage
	}
	threshold, err := parseSeverity(flags.failOn)
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return exitUsage
	}

	reg, err := buildRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return exitRuntimeError
	}

	cfg, err := loadConfig(flags.configPath, flags.path)
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return exitConfigError
	}

	rules := filterRules(reg, cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	res, err := core.Scan(ctx, core.ScanInput{
		Root:   flags.path,
		Rules:  rules,
		Runner: exec.NewReal(),
		Depth:  flags.depth,
	})
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return exitRuntimeError
	}

	// LLM enrichment runs only when --with-llm is set. Never on the hot path
	// (CLAUDE.md §13). Failures degrade; they never fail the scan.
	llmClient, err := buildLLMClient(ctx, flags.withLLM, flags.llmBudget, flags.llmBackend)
	if err != nil {
		fmt.Fprintf(stderr, "scan: --with-llm: %v\n", err)
		return exitConfigError
	}
	if llmClient != nil {
		defer func() { _ = llmClient.Close() }()
		if len(res.Findings) == 0 {
			fmt.Fprintln(stderr, "llm: no findings to enrich (score 100)")
		} else {
			enrichFindings(ctx, llmClient, rules, res.Findings, cfg.ProjectType, stderr)
		}
	}

	// Policy enforcement (advisory — does not change exit code).
	if flags.policy != "" {
		pol, perr := policy.Load(flags.policy)
		if perr != nil {
			fmt.Fprintf(stderr, "scan: --policy: %v\n", perr)
			return exitConfigError
		}
		principleScores := make(map[string]float64, len(res.Scores.ByPrinciple))
		for p, v := range res.Scores.ByPrinciple {
			principleScores[string(p)] = v
		}
		var ruleIDs []string
		for _, r := range rules {
			ruleIDs = append(ruleIDs, r.ID)
		}
		violations := policy.Enforce(pol, principleScores, res.Scores.Overall,
			cfg.Packs.Enabled, ruleIDs, "")
		for _, v := range violations {
			fmt.Fprintf(stderr, "policy violation [%s]: %s\n", v.Type, v.Detail)
		}
	}

	if format == report.FormatSARIF {
		if err := report.RenderSARIF(stdout, res, rules, version.Version); err != nil {
			fmt.Fprintf(stderr, "scan: %v\n", err)
			return exitRuntimeError
		}
	} else {
		if err := report.Render(stdout, res, version.Version, cfg.Profile, format); err != nil {
			fmt.Fprintf(stderr, "scan: %v\n", err)
			return exitRuntimeError
		}
	}

	// Record results to a timestamped subdirectory when --record is set.
	if flags.recordDir != "" {
		if err := recordScanResults(flags.recordDir, res, rules, version.Version, cfg.Profile, stderr); err != nil {
			fmt.Fprintf(stderr, "scan: --record: %v\n", err)
			return exitRuntimeError
		}
	}

	// Coverage explanation (--explain-coverage).
	if flags.explainCoverage {
		printCoverageExplanation(stderr, rules, res)
	}

	for _, f := range res.Findings {
		if f.Severity.Rank() >= threshold.Rank() {
			return exitFindingsAtLevel
		}
	}
	return exitOK
}

func printCoverageExplanation(w io.Writer, rules []model.Rule, res core.ScanResult) {
	firedRules := map[string]bool{}
	for _, f := range res.Findings {
		firedRules[f.RuleID] = true
	}
	var clean []string
	for _, r := range rules {
		if !firedRules[r.ID] {
			clean = append(clean, r.ID)
		}
	}
	sort.Strings(clean)
	fmt.Fprintf(w, "coverage: %d rules evaluated, %d with findings, %d clean\n",
		res.RulesEvaluated, res.RulesWithFindings, len(clean))
	if len(clean) > 0 {
		fmt.Fprintf(w, "  clean rules: %s\n", strings.Join(clean, ", "))
	}
	fmt.Fprintln(w, "  hint: rules may pass because they are satisfied OR because they found no applicable signal")
}

// recordScanResults writes JSON and Markdown files to a timestamped subdirectory
// under baseDir. The directory name is the current UTC timestamp in
// YYYYMMDD-HHMMSS format so results sort chronologically.
func recordScanResults(baseDir string, res core.ScanResult, rules []model.Rule, toolVersion, profile string, stderr io.Writer) error {
	now := timeNow().UTC()
	subDir := filepath.Join(baseDir, now.Format("20060102-150405"))
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		return fmt.Errorf("create record directory: %w", err)
	}

	// Write JSON record.
	jsonPath := filepath.Join(subDir, "scan.json")
	jsonFile, err := os.Create(jsonPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", jsonPath, err)
	}
	defer func() { _ = jsonFile.Close() }()
	if err := report.Render(jsonFile, res, toolVersion, profile, report.FormatJSON); err != nil {
		return fmt.Errorf("write JSON record: %w", err)
	}

	// Write Markdown report.
	mdPath := filepath.Join(subDir, "report.md")
	mdFile, err := os.Create(mdPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", mdPath, err)
	}
	defer func() { _ = mdFile.Close() }()
	if err := report.Render(mdFile, res, toolVersion, profile, report.FormatMarkdown); err != nil {
		return fmt.Errorf("write Markdown report: %w", err)
	}

	fmt.Fprintf(stderr, "recorded: %s\n", subDir)
	return nil
}

// timeNow is a package-level variable so tests can override it.
var timeNow = time.Now

func cmdScore(args []string, stdout, stderr io.Writer) int {
	// Score is a thin wrapper: run a scan, emit only the score summary.
	flags, err := parseScanFlags(args, "score")
	if err != nil {
		fmt.Fprintf(stderr, "score: %v\n", err)
		return exitUsage
	}
	reg, err := buildRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "score: %v\n", err)
		return exitRuntimeError
	}
	cfg, err := loadConfig(flags.configPath, flags.path)
	if err != nil {
		fmt.Fprintf(stderr, "score: %v\n", err)
		return exitConfigError
	}
	rules := filterRules(reg, cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	res, err := core.Scan(ctx, core.ScanInput{
		Root:   flags.path,
		Rules:  rules,
		Runner: exec.NewReal(),
		Depth:  flags.depth,
	})
	if err != nil {
		fmt.Fprintf(stderr, "score: %v\n", err)
		return exitRuntimeError
	}

	fmt.Fprintf(stdout, "overall: %.1f\n", res.Scores.Overall)
	for _, p := range model.AllPrinciples() {
		if v, ok := res.Scores.ByPrinciple[p]; ok {
			fmt.Fprintf(stdout, "  %s: %.1f\n", p, v)
		}
	}
	return exitOK
}

func cmdExplain(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	withLLM := fs.Bool("with-llm", false, "append an LLM-authored explanation (requires ANTHROPIC_API_KEY, OPENAI_API_KEY, or GOOGLE_API_KEY)")
	llmBackendStr := fs.String("llm-backend", "", "LLM provider: gemini or openai (auto-detected from env if omitted)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "explain: %v\n", err)
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "usage: archfit explain [--with-llm] <rule-id>")
		return exitUsage
	}
	reg, err := buildRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "explain: %v\n", err)
		return exitRuntimeError
	}
	r, ok := reg.Rule(rest[0])
	if !ok {
		fmt.Fprintf(stderr, "explain: unknown rule %q\n", rest[0])
		return exitUsage
	}
	fmt.Fprintf(stdout, "%s — %s\n", r.ID, r.Title)
	fmt.Fprintf(stdout, "principle: %s  severity: %s  evidence: %s  stability: %s\n",
		r.Principle, r.Severity, r.EvidenceStrength, r.Stability)
	fmt.Fprintf(stdout, "\nRationale:\n  %s\n", r.Rationale)
	fmt.Fprintf(stdout, "\nRemediation:\n  %s\n", r.Remediation.Summary)
	if r.Remediation.GuideRef != "" {
		fmt.Fprintf(stdout, "  guide: %s\n", r.Remediation.GuideRef)
	}
	if *withLLM {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		client, err := buildLLMClient(ctx, true, 1, *llmBackendStr)
		if err != nil {
			fmt.Fprintf(stderr, "explain: --with-llm: %v\n", err)
			return exitConfigError
		}
		defer func() { _ = client.Close() }()

		cfg, _, _, _ := config.Load(".")
		prompt := llm.BuildRulePrompt(r, cfg.ProjectType)
		// Synthetic empty finding — explain is rule-level, not finding-level.
		sug, err := client.Explain(ctx, r, model.Finding{RuleID: r.ID}, prompt)
		if err != nil {
			fmt.Fprintf(stderr, "explain: llm: %v (static explanation above still applies)\n", err)
			return exitOK
		}
		tokenInfo := ""
		if sug.InputTokens > 0 || sug.OutputTokens > 0 {
			tokenInfo = fmt.Sprintf(", %d+%d tokens", sug.InputTokens, sug.OutputTokens)
		}
		fmt.Fprintf(stdout, "\nLLM explanation (%s%s):\n", sug.Model, tokenInfo)
		writeIndentedStdout(stdout, sug.Text, "  ")
	}
	return exitOK
}

// writeIndentedStdout mirrors the one in report/report.go but is local here
// to keep main.go's dependencies tight.
func writeIndentedStdout(w io.Writer, s, prefix string) {
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			fmt.Fprintf(w, "%s%s\n", prefix, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		fmt.Fprintf(w, "%s%s\n", prefix, s[start:])
	}
}

func cmdListRules(_ []string, stdout, stderr io.Writer) int {
	reg, err := buildRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "list-rules: %v\n", err)
		return exitRuntimeError
	}
	for _, r := range reg.Rules() {
		fmt.Fprintf(stdout, "%s  %-8s %s\n", r.ID, r.Severity, r.Title)
	}
	return exitOK
}

func cmdListPacks(_ []string, stdout, stderr io.Writer) int {
	reg, err := buildRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "list-packs: %v\n", err)
		return exitRuntimeError
	}
	packRules := reg.Packs()
	for _, p := range reg.AllPacks() {
		fmt.Fprintf(stdout, "%s (v%s) — %s — %d rules\n", p.Name, p.Version, p.Description, p.RuleCount)
		if ids, ok := packRules[p.Name]; ok {
			for _, id := range ids {
				fmt.Fprintf(stdout, "  %s\n", id)
			}
		}
	}
	return exitOK
}

func cmdValidateConfig(args []string, stdout, stderr io.Writer) int {
	path := "."
	if len(args) == 1 {
		path = args[0]
	} else if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: archfit validate-config [path]")
		return exitUsage
	}
	cfg, cfgPath, present, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(stderr, "validate-config: %v\n", err)
		return exitConfigError
	}
	if !present {
		fmt.Fprintln(stdout, "no .archfit.yaml found; using defaults")
		return exitOK
	}
	fmt.Fprintf(stdout, "%s: ok (profile=%s, packs=%v)\n", cfgPath, cfg.Profile, cfg.Packs.Enabled)
	return exitOK
}

// filterRules applies config ignores and pack enable/disable lists. Phase 1
// supports pack-level enable plus per-rule ignore (no path-scoped ignore yet).
func filterRules(reg *rule.Registry, cfg config.Config) []model.Rule {
	enabled := map[string]bool{}
	for _, p := range cfg.Packs.Enabled {
		enabled[p] = true
	}
	disabled := map[string]bool{}
	for _, p := range cfg.Packs.Disabled {
		disabled[p] = true
	}

	// Default: if no enabled list, enable all known packs.
	packRules := reg.Packs()
	include := map[string]struct{}{}
	for pack, ids := range packRules {
		if len(enabled) > 0 && !enabled[pack] {
			continue
		}
		if disabled[pack] {
			continue
		}
		for _, id := range ids {
			include[id] = struct{}{}
		}
	}

	ignore := map[string]struct{}{}
	for _, ig := range cfg.Ignore {
		if len(ig.Paths) == 0 {
			ignore[ig.Rule] = struct{}{}
		}
	}

	out := make([]model.Rule, 0, len(include))
	for _, r := range reg.Rules() {
		if _, ok := include[r.ID]; !ok {
			continue
		}
		if _, suppressed := ignore[r.ID]; suppressed {
			continue
		}
		out = append(out, r)
	}
	return out
}

// loadConfig loads configuration from --config if set, otherwise discovers from path.
func loadConfig(configPath, path string) (config.Config, error) {
	if configPath != "" {
		return config.LoadFile(configPath)
	}
	cfg, _, _, err := config.Load(path)
	return cfg, err
}

func parseSeverity(s string) (model.Severity, error) {
	sev := model.Severity(s)
	if !sev.Valid() {
		return "", fmt.Errorf("invalid severity %q (want info|warn|error|critical)", s)
	}
	return sev, nil
}

// ----- LLM enrichment (Phase 3a) -----
//
// buildLLMClient returns a ready-to-use llm.Client (Real or OpenAI, wrapped
// in Budget and Cached) when --with-llm is set and an API key is configured.
// It returns (nil, nil) when --with-llm is not set — the caller treats that
// as "do nothing LLM-related". It returns (nil, error) when --with-llm is set
// but the API key is missing; the caller maps that to exit code 4.
func buildLLMClient(ctx context.Context, withLLM bool, budget int, backend string) (llm.Client, error) {
	if !withLLM {
		return nil, nil
	}
	var cfg llm.Config
	var ok bool
	if backend != "" {
		cfg, ok = llm.FromEnvWithBackend(os.Getenv, llm.Backend(backend))
	} else {
		cfg, ok = llm.FromEnv(os.Getenv)
	}
	if !ok {
		return nil, llm.ErrNotConfigured
	}
	var inner llm.Client
	var err error
	switch cfg.Backend {
	case llm.BackendClaude:
		inner, err = llm.NewAnthropic(ctx, cfg)
	case llm.BackendOpenAI:
		inner, err = llm.NewOpenAI(ctx, cfg)
	default:
		inner, err = llm.NewReal(ctx, cfg)
	}
	if err != nil {
		return nil, err
	}
	// Canonical composition: inner → Budget → Cached (outermost).
	// See internal/adapter/llm/budget.go for the rationale.
	return llm.NewCached(llm.NewBudget(inner, budget)), nil
}

// enrichFindings calls the LLM for each finding up to the client's budget.
// Errors are logged to stderr and never fail the scan — base exit code is
// unchanged. Mutates findings in place.
func enrichFindings(ctx context.Context, client llm.Client, rules []model.Rule, findings []model.Finding, projectType []string, stderr io.Writer) {
	if client == nil {
		return
	}
	byID := map[string]model.Rule{}
	for _, r := range rules {
		byID[r.ID] = r
	}
	for i := range findings {
		rl, ok := byID[findings[i].RuleID]
		if !ok {
			continue
		}
		prompt := llm.BuildFindingPrompt(rl, findings[i], projectType)
		sug, err := client.Explain(ctx, rl, findings[i], prompt)
		if err != nil {
			if errors.Is(err, llm.ErrBudgetExhausted) {
				// Quiet exit — user explicitly set the budget.
				return
			}
			fmt.Fprintf(stderr, "llm: finding %s skipped (%v)\n", findings[i].RuleID, err)
			continue
		}
		findings[i].LLMSuggestion = &model.LLMSuggestion{
			Text:         sug.Text,
			Model:        sug.Model,
			CacheHit:     sug.CacheHit,
			Truncated:    sug.Truncated,
			LatencyMS:    sug.LatencyMS,
			InputTokens:  sug.InputTokens,
			OutputTokens: sug.OutputTokens,
		}
	}
}

// ----- Phase 2 subcommands -----

func cmdInit(args []string, stdout, stderr io.Writer) int {
	path := "."
	if len(args) == 1 {
		path = args[0]
	} else if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: archfit init [path]")
		return exitUsage
	}
	target := filepathJoin(path, ".archfit.yaml")
	if _, err := os.Stat(target); err == nil {
		fmt.Fprintf(stderr, "init: %s already exists; refusing to overwrite\n", target)
		return exitConfigError
	}

	// Detect project stack to generate stack-aware defaults.
	projectType, packs := detectProjectStack(path)

	doc := map[string]any{
		"version":      1,
		"project_type": projectType,
		"profile":      "standard",
		"packs":        map[string]any{"enabled": packs},
		"ignore":       []any{},
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return exitRuntimeError
	}
	if err := os.WriteFile(target, append(out, '\n'), 0o644); err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return exitRuntimeError
	}
	if len(projectType) > 0 {
		fmt.Fprintf(stdout, "wrote %s (detected: %s)\n", target, strings.Join(projectType, ", "))
	} else {
		fmt.Fprintf(stdout, "wrote %s\n", target)
	}
	return exitOK
}

// detectProjectStack inspects the filesystem to determine project type and
// appropriate packs. Returns empty slices if detection fails gracefully.
func detectProjectStack(path string) (projectType, packs []string) {
	packs = []string{"core"} // core always enabled

	repo, err := collectfs.Collect(path)
	if err != nil {
		return nil, packs
	}

	// Determine primary language.
	var primaryLang string
	maxCount := 0
	for lang, count := range repo.Languages {
		if count > maxCount {
			primaryLang = lang
			maxCount = count
		}
	}

	// Check for CLI entrypoint (cmd/ or bin/ directory).
	hasCLI := false
	for _, f := range repo.Files {
		if strings.HasPrefix(f.Path, "cmd/") || strings.HasPrefix(f.Path, "bin/") {
			hasCLI = true
			break
		}
	}

	// Check for Terraform.
	hasTerraform := repo.Languages["terraform"] > 0

	// Map language + signals to project type and packs.
	if hasTerraform {
		projectType = append(projectType, "iac")
	}
	if hasCLI {
		projectType = append(projectType, "agent-tool")
		packs = append(packs, "agent-tool")
	}
	switch primaryLang {
	case "go", "python", "typescript", "javascript", "java", "ruby":
		if !hasCLI {
			projectType = append(projectType, "web-saas")
		}
	}

	if len(projectType) == 0 {
		// Fallback: no specific type detected.
		return nil, packs
	}
	return projectType, packs
}

// filepathJoin is a thin wrapper that keeps the import list short at the top of main.go.
func filepathJoin(a, b string) string {
	if a == "" || a == "." {
		return b
	}
	return filepath.Join(a, b)
}

func cmdCheck(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: archfit check <rule-id> [path] [--json]")
		return exitUsage
	}
	ruleID := args[0]
	rest := args[1:]
	flags, err := parseScanFlags(rest, "check")
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return exitUsage
	}
	format, err := report.ParseFormat(flags.format)
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return exitUsage
	}

	reg, err := buildRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return exitRuntimeError
	}
	r, ok := reg.Rule(ruleID)
	if !ok {
		fmt.Fprintf(stderr, "check: unknown rule %q\n", ruleID)
		return exitUsage
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	res, err := core.Scan(ctx, core.ScanInput{
		Root:   flags.path,
		Rules:  []model.Rule{r},
		Runner: exec.NewReal(),
		Depth:  flags.depth,
	})
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return exitRuntimeError
	}
	llmClient, err := buildLLMClient(ctx, flags.withLLM, flags.llmBudget, flags.llmBackend)
	if err != nil {
		fmt.Fprintf(stderr, "check: --with-llm: %v\n", err)
		return exitConfigError
	}
	if llmClient != nil {
		defer func() { _ = llmClient.Close() }()
		if len(res.Findings) == 0 {
			fmt.Fprintln(stderr, "llm: no findings to enrich")
		} else {
			enrichFindings(ctx, llmClient, []model.Rule{r}, res.Findings, nil, stderr)
		}
	}
	if format == report.FormatSARIF {
		if err := report.RenderSARIF(stdout, res, []model.Rule{r}, version.Version); err != nil {
			fmt.Fprintf(stderr, "check: %v\n", err)
			return exitRuntimeError
		}
	} else {
		if err := report.Render(stdout, res, version.Version, "standard", format); err != nil {
			fmt.Fprintf(stderr, "check: %v\n", err)
			return exitRuntimeError
		}
	}
	for _, f := range res.Findings {
		if f.Severity.Rank() >= model.SeverityError.Rank() {
			return exitFindingsAtLevel
		}
	}
	return exitOK
}

func cmdReport(args []string, stdout, stderr io.Writer) int {
	// `report` is scan --format=md. We rewrite args and dispatch.
	argsCopy := append([]string{"--format=md"}, args...)
	return cmdScan(argsCopy, stdout, stderr)
}

func cmdDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "diff: %v\n", err)
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) < 1 || len(rest) > 2 {
		fmt.Fprintln(stderr, "usage: archfit diff <baseline.json> [current.json]")
		fmt.Fprintln(stderr, "       when current.json is omitted, it is read from stdin")
		return exitUsage
	}
	baselineBytes, err := os.ReadFile(rest[0])
	if err != nil {
		fmt.Fprintf(stderr, "diff: %v\n", err)
		return exitRuntimeError
	}
	var currentBytes []byte
	if len(rest) == 2 {
		currentBytes, err = os.ReadFile(rest[1])
		if err != nil {
			fmt.Fprintf(stderr, "diff: %v\n", err)
			return exitRuntimeError
		}
	} else {
		currentBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(stderr, "diff: %v\n", err)
			return exitRuntimeError
		}
	}
	baseline, err := report.LoadBaseline(baselineBytes)
	if err != nil {
		fmt.Fprintf(stderr, "diff: %v\n", err)
		return exitConfigError
	}
	current, err := report.LoadBaseline(currentBytes)
	if err != nil {
		fmt.Fprintf(stderr, "diff: %v\n", err)
		return exitConfigError
	}
	d := report.Diff(baseline.Findings, current.Findings)
	if *jsonOut {
		if err := report.RenderDiffJSON(stdout, d); err != nil {
			fmt.Fprintf(stderr, "diff: %v\n", err)
			return exitRuntimeError
		}
	} else {
		if err := report.RenderDiffTerminal(stdout, d); err != nil {
			fmt.Fprintf(stderr, "diff: %v\n", err)
			return exitRuntimeError
		}
	}
	// Regressions (new findings) → exit 1 so CI can gate on them.
	if len(d.New) > 0 {
		return exitFindingsAtLevel
	}
	return exitOK
}

// ----- Fix engine (Pillar 1) -----

// buildFixEngine registers all static fixers explicitly. Same pattern as
// buildRegistry(). See ADR 0004.
func buildFixEngine() *fix.Engine {
	e := fix.NewEngine()
	e.Register(static.NewLocP1LOC001())
	e.Register(static.NewLocP1LOC002())
	e.Register(static.NewVerP4VER001())
	e.Register(static.NewMrdP7MRD001())
	e.Register(static.NewMrdP7MRD002())
	e.Register(static.NewMrdP7MRD003())
	e.Register(static.NewSpcP2SPC010())
	return e
}

func cmdFix(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fix", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var (
		fixAll     = fs.Bool("all", false, "fix all fixable findings")
		dryRun     = fs.Bool("dry-run", false, "show what would change without applying")
		planOnly   = fs.Bool("plan", false, "show fix plan and exit")
		jsonOut    = fs.Bool("json", false, "emit fix result as JSON")
		workDir    = fs.String("C", "", "change to directory before running")
		withLLM    = fs.Bool("with-llm", false, "enrich fix content with LLM (opt-in)")
		llmBackend = fs.String("llm-backend", "", "LLM provider: gemini, openai, or claude")
		llmBudget  = fs.Int("llm-budget", 5, "max LLM calls per run")
	)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "fix: %v\n", err)
		return exitUsage
	}
	rest := fs.Args()

	// Determine rule ID and path from positional args.
	var ruleID, path string
	switch {
	case *fixAll && len(rest) > 1:
		fmt.Fprintln(stderr, "usage: archfit fix --all [path]")
		return exitUsage
	case *fixAll && len(rest) == 1:
		path = rest[0]
	case *fixAll:
		path = "."
	case len(rest) == 0:
		fmt.Fprintln(stderr, "usage: archfit fix [--all] [rule-id] [path]")
		return exitUsage
	case len(rest) == 1:
		ruleID = rest[0]
		path = "."
	case len(rest) == 2:
		ruleID = rest[0]
		path = rest[1]
	default:
		fmt.Fprintln(stderr, "usage: archfit fix [--all] [rule-id] [path]")
		return exitUsage
	}

	if *workDir != "" {
		path = joinIfRelative(*workDir, path)
	}

	// Suppress unused variable warnings for LLM flags (used in Step 6).
	_, _, _ = *withLLM, *llmBackend, *llmBudget

	reg, err := buildRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "fix: %v\n", err)
		return exitRuntimeError
	}

	cfg, _, _, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(stderr, "fix: %v\n", err)
		return exitConfigError
	}

	rules := filterRules(reg, cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initial scan to get current findings and facts.
	scanRes, err := core.Scan(ctx, core.ScanInput{
		Root:   path,
		Rules:  rules,
		Runner: exec.NewReal(),
	})
	if err != nil {
		fmt.Fprintf(stderr, "fix: initial scan: %v\n", err)
		return exitRuntimeError
	}

	engine := buildFixEngine()

	var ruleIDs []string
	if ruleID != "" {
		ruleIDs = []string{ruleID}
	}

	// Build a scanner function for verification.
	scanner := func(ctx context.Context) (core.ScanResult, error) {
		return core.Scan(ctx, core.ScanInput{
			Root:   path,
			Rules:  rules,
			Runner: exec.NewReal(),
		})
	}

	// We need a FactStore for the fixers. Re-collect facts.
	facts := &cliFactStore{scanRes: scanRes, path: path}

	fixResult, err := engine.Fix(ctx, fix.Input{
		Root:     path,
		RuleIDs:  ruleIDs,
		DryRun:   *dryRun || *planOnly,
		Facts:    facts,
		Findings: scanRes.Findings,
		Scanner:  scanner,
	})
	if err != nil {
		fmt.Fprintf(stderr, "fix: %v\n", err)
		return exitRuntimeError
	}

	// Render output.
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(fixResult); err != nil {
			fmt.Fprintf(stderr, "fix: %v\n", err)
			return exitRuntimeError
		}
	} else {
		fmt.Fprint(stdout, fixResult.Plan.Summary())
		if !*dryRun && !*planOnly {
			if fixResult.Verified {
				fmt.Fprintf(stdout, "\nall fixes verified by re-scan\n")
			} else if len(fixResult.Applied) > 0 {
				fmt.Fprintf(stdout, "\nfixes rolled back — verification failed\n")
				if len(fixResult.NewIssues) > 0 {
					fmt.Fprintf(stdout, "new issues introduced: %d\n", len(fixResult.NewIssues))
				}
			}
		}
	}

	if !fixResult.Verified && len(fixResult.Applied) > 0 {
		return exitFindingsAtLevel
	}
	return exitOK
}

// cliFactStore wraps a ScanResult to provide a minimal FactStore for fixers.
// The fixers only need Repo() for path lookups and root directory.
type cliFactStore struct {
	scanRes core.ScanResult
	path    string
}

func (f *cliFactStore) Repo() model.RepoFacts {
	// Re-collect filesystem facts. This is lightweight and avoids storing
	// a full FactStore from the scan (which is internal to core.Scan).
	repo, err := collectForFix(f.path)
	if err != nil {
		return model.RepoFacts{Root: f.path}
	}
	return repo
}

func (f *cliFactStore) Git() (model.GitFacts, bool) {
	return f.scanRes.Git, f.scanRes.GitAvailable
}

func (f *cliFactStore) Schemas() model.SchemaFacts {
	return model.SchemaFacts{}
}

func (f *cliFactStore) Commands() (model.CommandFacts, bool) {
	return model.CommandFacts{}, false
}

func (f *cliFactStore) DepGraph() (model.DepGraphFacts, bool) {
	return model.DepGraphFacts{}, false
}

// collectForFix does a lightweight filesystem collect for the fix engine.
func collectForFix(root string) (model.RepoFacts, error) {
	// Import cycle prevention: we call the collector directly.
	// This is acceptable because main.go is the wiring layer.
	return collectfs.Collect(root)
}

// ----- Validate-pack subcommand (Step 16, ADR 0006) -----

func cmdValidatePack(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: archfit validate-pack <path>")
		return exitUsage
	}
	dir := args[0]
	res := packman.ValidatePack(dir)

	for _, e := range res.Errors {
		fmt.Fprintf(stderr, "error: %s\n", e)
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", w)
	}
	if res.Valid {
		fmt.Fprintln(stdout, "pack structure is valid")
		return exitOK
	}
	return exitFindingsAtLevel
}

// ----- Trend subcommand -----

func cmdTrend(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("trend", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	historyDir := fs.String("history", ".archfit-history", "directory containing archived scan JSON files")
	since := fs.String("since", "", "only show entries on or after this ISO date (YYYY-MM-DD)")
	format := fs.String("format", "terminal", "output format: terminal, json, csv")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "trend: %v\n", err)
		return exitUsage
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(stderr, "usage: archfit trend [--history <dir>] [--since <date>] [--format {terminal|json|csv}]")
		return exitUsage
	}

	dirEntries, err := os.ReadDir(*historyDir)
	if err != nil {
		fmt.Fprintf(stderr, "trend: %v\n", err)
		return exitRuntimeError
	}

	var files []string
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if *since != "" && e.Name() < *since {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)

	if len(files) == 0 {
		fmt.Fprintln(stderr, "trend: no .json files found in "+*historyDir)
		return exitOK
	}

	type trendEntry struct {
		File        string             `json:"file"`
		Overall     float64            `json:"overall"`
		Delta       float64            `json:"delta"`
		ByPrinciple map[string]float64 `json:"by_principle,omitempty"`
	}

	var trendData []trendEntry
	var prevOverall float64
	first := true

	for _, name := range files {
		data, err := os.ReadFile(filepath.Join(*historyDir, name))
		if err != nil {
			fmt.Fprintf(stderr, "trend: skipping %s: %v\n", name, err)
			continue
		}
		doc, err := report.LoadBaseline(data)
		if err != nil {
			fmt.Fprintf(stderr, "trend: skipping %s: %v\n", name, err)
			continue
		}
		delta := 0.0
		if !first {
			delta = doc.Scores.Overall - prevOverall
		}
		prevOverall = doc.Scores.Overall
		first = false

		trendData = append(trendData, trendEntry{
			File:        name,
			Overall:     doc.Scores.Overall,
			Delta:       math.Round(delta*10) / 10,
			ByPrinciple: doc.Scores.ByPrinciple,
		})
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(trendData); err != nil {
			fmt.Fprintf(stderr, "trend: %v\n", err)
			return exitRuntimeError
		}
	case "csv":
		fmt.Fprintln(stdout, "file,overall,delta")
		for _, e := range trendData {
			fmt.Fprintf(stdout, "%s,%.1f,%.1f\n", e.File, e.Overall, e.Delta)
		}
	default:
		fmt.Fprintf(stdout, "%-40s %8s %8s\n", "FILE", "SCORE", "DELTA")
		fmt.Fprintf(stdout, "%-40s %8s %8s\n", strings.Repeat("-", 40), "--------", "--------")
		for _, e := range trendData {
			deltaStr := fmt.Sprintf("%+.1f", e.Delta)
			if e.Delta == 0 {
				deltaStr = "  ---"
			}
			fmt.Fprintf(stdout, "%-40s %8.1f %8s\n", e.File, e.Overall, deltaStr)
		}
	}
	return exitOK
}

// ----- Compare subcommand (Step 18) -----

// compareEntry holds extracted score data from a single scan JSON file.
type compareEntry struct {
	File        string             `json:"file"`
	Overall     float64            `json:"overall"`
	ByPrinciple map[string]float64 `json:"by_principle"`
	Findings    int                `json:"findings"`
}

func cmdCompare(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "terminal", "output format: terminal, json, csv, md")
	sortBy := fs.String("sort", "overall", "sort field: overall, name")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "compare: %v\n", err)
		return exitUsage
	}
	files := fs.Args()
	if len(files) < 2 {
		fmt.Fprintln(stderr, "usage: archfit compare <file1.json> <file2.json> [...]")
		return exitUsage
	}

	var entries []compareEntry
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(stderr, "compare: %v\n", err)
			return exitRuntimeError
		}
		doc, err := report.LoadBaseline(data)
		if err != nil {
			fmt.Fprintf(stderr, "compare: %s: %v\n", f, err)
			return exitRuntimeError
		}
		entries = append(entries, compareEntry{
			File:        filepath.Base(f),
			Overall:     doc.Scores.Overall,
			ByPrinciple: doc.Scores.ByPrinciple,
			Findings:    len(doc.Findings),
		})
	}

	// Sort entries.
	switch *sortBy {
	case "name":
		sort.Slice(entries, func(i, j int) bool { return entries[i].File < entries[j].File })
	default: // "overall" — descending
		sort.Slice(entries, func(i, j int) bool { return entries[i].Overall > entries[j].Overall })
	}

	// Collect principle keys present across all entries for column headers.
	principles := model.AllPrinciples()

	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(entries); err != nil {
			fmt.Fprintf(stderr, "compare: %v\n", err)
			return exitRuntimeError
		}
	case "csv":
		hdr := []string{"file", "overall"}
		for _, p := range principles {
			hdr = append(hdr, string(p))
		}
		hdr = append(hdr, "findings")
		fmt.Fprintln(stdout, strings.Join(hdr, ","))
		for _, e := range entries {
			row := []string{e.File, fmt.Sprintf("%.1f", e.Overall)}
			for _, p := range principles {
				v := e.ByPrinciple[string(p)]
				row = append(row, fmt.Sprintf("%.1f", v))
			}
			row = append(row, fmt.Sprintf("%d", e.Findings))
			fmt.Fprintln(stdout, strings.Join(row, ","))
		}
	case "md":
		hdr := []string{"File", "Overall"}
		sep := []string{"---", "---"}
		for _, p := range principles {
			hdr = append(hdr, string(p))
			sep = append(sep, "---")
		}
		hdr = append(hdr, "Findings")
		sep = append(sep, "---")
		fmt.Fprintf(stdout, "| %s |\n", strings.Join(hdr, " | "))
		fmt.Fprintf(stdout, "| %s |\n", strings.Join(sep, " | "))
		for _, e := range entries {
			row := []string{e.File, fmt.Sprintf("%.1f", e.Overall)}
			for _, p := range principles {
				v := e.ByPrinciple[string(p)]
				row = append(row, fmt.Sprintf("%.1f", v))
			}
			row = append(row, fmt.Sprintf("%d", e.Findings))
			fmt.Fprintf(stdout, "| %s |\n", strings.Join(row, " | "))
		}
	default: // terminal
		// Build header.
		fmt.Fprintf(stdout, "%-30s %8s", "FILE", "OVERALL")
		for _, p := range principles {
			fmt.Fprintf(stdout, " %6s", string(p))
		}
		fmt.Fprintf(stdout, " %8s\n", "FINDINGS")

		fmt.Fprintf(stdout, "%-30s %8s", strings.Repeat("-", 30), "--------")
		for range principles {
			fmt.Fprintf(stdout, " %6s", "------")
		}
		fmt.Fprintf(stdout, " %8s\n", "--------")

		for _, e := range entries {
			fmt.Fprintf(stdout, "%-30s %8.1f", e.File, e.Overall)
			for _, p := range principles {
				v := e.ByPrinciple[string(p)]
				fmt.Fprintf(stdout, " %6.1f", v)
			}
			fmt.Fprintf(stdout, " %8d\n", e.Findings)
		}
	}
	return exitOK
}

// ----- Pack SDK subcommands -----

func cmdNewPack(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintln(stderr, "usage: archfit new-pack <name> [path]")
		return exitUsage
	}
	name := args[0]
	base := "."
	if len(args) == 2 {
		base = args[1]
	}
	packDir := filepath.Join(base, name)
	if _, err := os.Stat(packDir); err == nil {
		fmt.Fprintf(stderr, "new-pack: directory %s already exists\n", packDir)
		return exitConfigError
	}

	// Scaffold the pack directory structure.
	dirs := []string{
		packDir,
		filepath.Join(packDir, "resolvers"),
		filepath.Join(packDir, "fixtures"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			fmt.Fprintf(stderr, "new-pack: %v\n", err)
			return exitRuntimeError
		}
	}

	// Convert hyphenated name to underscore for Go package name.
	goPkg := strings.ReplaceAll(name, "-", "_")

	files := map[string]string{
		filepath.Join(packDir, "AGENTS.md"):    "# " + name + " pack\n\nAgent-facing documentation for the " + name + " rule pack.\n",
		filepath.Join(packDir, "INTENT.md"):    "# " + name + " — intent\n\nThis pack checks...\n",
		filepath.Join(packDir, "pack.go"):      "package " + goPkg + "\n\nimport (\n\t\"github.com/shibuiwilliam/archfit/internal/model\"\n\t\"github.com/shibuiwilliam/archfit/internal/rule\"\n)\n\n// PackName is the unique identifier for this pack.\nconst PackName = \"" + name + "\"\n\n// Rules returns the rules in this pack.\nfunc Rules() []model.Rule {\n\treturn []model.Rule{\n\t\t// Add rules here.\n\t}\n}\n\n// Register adds this pack's rules to the registry.\nfunc Register(reg *rule.Registry) error {\n\treturn reg.Register(PackName, Rules()...)\n}\n",
		filepath.Join(packDir, "pack_test.go"): "package " + goPkg + "_test\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fmt.Fprintf(stderr, "new-pack: %v\n", err)
			return exitRuntimeError
		}
	}

	fmt.Fprintf(stdout, "created pack scaffold at %s\n", packDir)
	return exitOK
}

func cmdTestPack(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: archfit test-pack <path>")
		return exitUsage
	}
	packPath := args[0]

	cmd := osexec.Command("go", "test", "-race", "-count=1", packPath)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*osexec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "test-pack: %v\n", err)
		return exitRuntimeError
	}
	return exitOK
}

// ----- Contract subcommands -----

func cmdContract(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, `usage:
  archfit contract check [path]   check scan results against .archfit-contract.yaml
  archfit contract init [path]    scaffold a contract from current scan results`)
		return exitUsage
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "check":
		return cmdContractCheck(rest, stdout, stderr)
	case "init":
		return cmdContractInit(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "archfit contract: unknown subcommand %q\n", sub)
		return exitUsage
	}
}

func cmdContractCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("contract check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	formatStr := fs.String("format", "terminal", "output format")
	jsonFlag := fs.Bool("json", false, "shorthand for --format=json")
	workDir := fs.String("C", "", "change to directory before running")
	configPath := fs.String("config", "", "path to config file")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "contract check: %v\n", err)
		return exitUsage
	}
	if *jsonFlag {
		*formatStr = "json"
	}
	path := "."
	if rest := fs.Args(); len(rest) == 1 {
		path = rest[0]
	}
	if *workDir != "" {
		path = joinIfRelative(*workDir, path)
	}

	// Load contract.
	c, cpath, found, err := contract.Load(path)
	if err != nil {
		fmt.Fprintf(stderr, "contract check: %v\n", err)
		return exitConfigError
	}
	if !found {
		fmt.Fprintln(stderr, "contract check: no .archfit-contract.yaml found — run 'archfit contract init' to create one")
		return exitOK
	}
	fmt.Fprintf(stderr, "contract: loaded %s\n", cpath)

	// Run scan.
	reg, err := buildRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "contract check: %v\n", err)
		return exitRuntimeError
	}
	cfg, err := loadConfig(*configPath, path)
	if err != nil {
		fmt.Fprintf(stderr, "contract check: %v\n", err)
		return exitConfigError
	}
	rules := filterRules(reg, cfg)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	res, err := core.Scan(ctx, core.ScanInput{Root: path, Rules: rules, Runner: exec.NewReal()})
	if err != nil {
		fmt.Fprintf(stderr, "contract check: %v\n", err)
		return exitRuntimeError
	}

	// Check contract.
	result := contract.Check(c, res.Scores, res.Findings)

	// Render.
	if *formatStr == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	} else {
		renderContractTerminal(stdout, result)
	}

	if !result.Passed {
		return exitFindingsAtLevel
	}
	if len(result.SoftMisses) > 0 {
		return exitContractSoftMiss
	}
	return exitOK
}

func renderContractTerminal(w io.Writer, r contract.CheckResult) {
	if r.Passed {
		fmt.Fprintln(w, "contract: PASSED (all hard constraints satisfied)")
	} else {
		fmt.Fprintln(w, "contract: FAILED")
	}
	if len(r.HardViolations) > 0 {
		fmt.Fprintln(w, "hard violations:")
		for _, v := range r.HardViolations {
			fmt.Fprintf(w, "  ✗ %s\n", v.Detail)
		}
	}
	if len(r.SoftMisses) > 0 {
		fmt.Fprintln(w, "soft target misses:")
		for _, m := range r.SoftMisses {
			fmt.Fprintf(w, "  ○ %s\n", m.Detail)
		}
	}
	if len(r.BudgetStatus) > 0 {
		fmt.Fprintln(w, "area budgets:")
		for _, b := range r.BudgetStatus {
			status := "ok"
			if b.Exhausted {
				status = "EXHAUSTED"
			}
			fmt.Fprintf(w, "  %s: %d/%d findings (%s)\n", b.Budget.Path, b.Current, b.Budget.MaxFindings, status)
		}
	}
}

func cmdContractInit(args []string, stdout, stderr io.Writer) int {
	path := "."
	if len(args) == 1 {
		path = args[0]
	} else if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: archfit contract init [path]")
		return exitUsage
	}

	// Check if contract already exists.
	if _, _, found, _ := contract.Load(path); found {
		fmt.Fprintln(stderr, "contract init: .archfit-contract.yaml already exists")
		return exitConfigError
	}

	// Run scan to get current scores.
	reg, err := buildRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "contract init: %v\n", err)
		return exitRuntimeError
	}
	cfg, cerr := loadConfig("", path)
	if cerr != nil {
		fmt.Fprintf(stderr, "contract init: %v\n", cerr)
		return exitConfigError
	}
	rules := filterRules(reg, cfg)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	res, err := core.Scan(ctx, core.ScanInput{Root: path, Rules: rules, Runner: exec.NewReal()})
	if err != nil {
		fmt.Fprintf(stderr, "contract init: %v\n", err)
		return exitRuntimeError
	}

	// Generate contract: hard constraints set 5 points below current scores.
	c := contract.Contract{Version: 1}
	overallFloor := math.Floor(res.Scores.Overall/5) * 5 // round down to nearest 5
	if overallFloor < 0 {
		overallFloor = 0
	}
	c.HardConstraints = append(c.HardConstraints, contract.Constraint{
		Principle: "overall",
		MinScore:  overallFloor,
		Scope:     "**",
		Rationale: "Do not let overall fitness drop below current level",
	})

	// Add a soft target at current score (aspirational: maintain or improve).
	if res.Scores.Overall < 100 {
		c.SoftTargets = append(c.SoftTargets, contract.Target{
			Principle:   "overall",
			TargetScore: 100,
			Current:     res.Scores.Overall,
		})
	}

	// Add a default agent directive.
	c.AgentDirectives = append(c.AgentDirectives, contract.AgentDirective{
		When:   "finding.severity >= error",
		Action: "stop and ask the user before proceeding",
	})

	// Write .archfit-contract.yaml.
	out, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "contract init: %v\n", err)
		return exitRuntimeError
	}
	contractPath := filepath.Join(path, ".archfit-contract.yaml")
	if err := os.WriteFile(contractPath, append(out, '\n'), 0o644); err != nil {
		fmt.Fprintf(stderr, "contract init: %v\n", err)
		return exitRuntimeError
	}
	fmt.Fprintf(stdout, "created %s (overall floor: %.0f, current: %.1f)\n", contractPath, overallFloor, res.Scores.Overall)
	return exitOK
}
