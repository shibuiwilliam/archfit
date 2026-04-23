// Binary archfit is the CLI entrypoint. main.go is intentionally the only file
// that wires collectors, the registry, and packs together — this is where the
// explicit (non-init) registration lives, per CLAUDE.md §3.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/shibuiwilliam/archfit/internal/adapter/exec"
	"github.com/shibuiwilliam/archfit/internal/adapter/llm"
	"github.com/shibuiwilliam/archfit/internal/config"
	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/model"
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
  archfit explain <rule-id>            show a rule's rationale and remediation
  archfit init [path]                  scaffold .archfit.yaml with defaults
  archfit list-rules                   list all registered rules
  archfit list-packs                   list all registered rule packs
  archfit validate-config [path]       check .archfit.yaml without scanning
  archfit version                      print the version

global flags (where applicable):
  --format {terminal|json|md|sarif}    output format (default: terminal)
  --json                               shorthand for --format=json
  --fail-on {info|warn|error|critical} exit 1 when any finding meets this level (default: error)
  -C <dir>                             change to dir before running (like git -C)
  --with-llm                           enrich findings with LLM-authored explanations
                                       (opt-in; requires GOOGLE_API_KEY)
  --llm-budget N                       cap the number of LLM calls per run (default: 5)

Exit codes:
  0   success (or: findings below --fail-on threshold)
  1   findings present at or above --fail-on threshold
  2   usage error
  3   runtime error
  4   configuration error

See docs/exit-codes.md and PROJECT.md for the full contract.
`)
}

type scanFlags struct {
	format    string
	json      bool
	failOn    string
	workDir   string
	path      string
	withLLM   bool
	llmBudget int
}

func parseScanFlags(args []string, cmd string) (scanFlags, error) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var f scanFlags
	fs.StringVar(&f.format, "format", "terminal", "output format")
	fs.BoolVar(&f.json, "json", false, "shorthand for --format=json")
	fs.StringVar(&f.failOn, "fail-on", "error", "severity threshold that causes a non-zero exit")
	fs.StringVar(&f.workDir, "C", "", "change to directory before running")
	fs.BoolVar(&f.withLLM, "with-llm", false, "enrich findings with LLM-authored explanations (opt-in; requires GOOGLE_API_KEY)")
	fs.IntVar(&f.llmBudget, "llm-budget", 5, "maximum LLM calls per run (only when --with-llm)")
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
	if len(p) > 0 && (p[0] == '/' || (len(p) >= 2 && p[1] == ':')) {
		return p // absolute
	}
	return dir + string(os.PathSeparator) + p
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

	cfg, cfgPath, present, err := config.Load(flags.path)
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
	})
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return exitRuntimeError
	}

	// LLM enrichment runs only when --with-llm is set. Never on the hot path
	// (CLAUDE.md §13). Failures degrade; they never fail the scan.
	llmClient, err := buildLLMClient(ctx, flags.withLLM, flags.llmBudget)
	if err != nil {
		fmt.Fprintf(stderr, "scan: --with-llm: %v\n", err)
		return exitConfigError
	}
	if llmClient != nil {
		defer llmClient.Close()
		enrichFindings(ctx, llmClient, rules, res.Findings, cfg.ProjectType, stderr)
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

	if present {
		_ = cfgPath // surfaced via the renderer's target.profile field; path reserved for future verbose mode
	}

	for _, f := range res.Findings {
		if f.Severity.Rank() >= threshold.Rank() {
			return exitFindingsAtLevel
		}
	}
	return exitOK
}

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
	cfg, _, _, err := config.Load(flags.path)
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
	withLLM := fs.Bool("with-llm", false, "append an LLM-authored explanation (requires GOOGLE_API_KEY)")
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
		client, err := buildLLMClient(ctx, true, 1)
		if err != nil {
			fmt.Fprintf(stderr, "explain: --with-llm: %v\n", err)
			return exitConfigError
		}
		defer client.Close()

		cfg, _, _, _ := config.Load(".")
		prompt := llm.BuildRulePrompt(r, cfg.ProjectType)
		// Synthetic empty finding — explain is rule-level, not finding-level.
		sug, err := client.Explain(ctx, r, model.Finding{RuleID: r.ID}, prompt)
		if err != nil {
			fmt.Fprintf(stderr, "explain: llm: %v (static explanation above still applies)\n", err)
			return exitOK
		}
		fmt.Fprintf(stdout, "\nLLM explanation (%s):\n", sug.Model)
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
	for name, ids := range reg.Packs() {
		fmt.Fprintf(stdout, "%s (%d rules)\n", name, len(ids))
		for _, id := range ids {
			fmt.Fprintf(stdout, "  %s\n", id)
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

func parseSeverity(s string) (model.Severity, error) {
	sev := model.Severity(s)
	if !sev.Valid() {
		return "", fmt.Errorf("invalid severity %q (want info|warn|error|critical)", s)
	}
	return sev, nil
}

// ----- LLM enrichment (Phase 3a) -----
//
// buildLLMClient returns a ready-to-use llm.Client (Real wrapped in Budget
// and Cached) when --with-llm is set and an API key is configured.
// It returns (nil, nil) when --with-llm is not set — the caller treats that
// as "do nothing LLM-related". It returns (nil, error) when --with-llm is set
// but the API key is missing; the caller maps that to exit code 4.
func buildLLMClient(ctx context.Context, withLLM bool, budget int) (llm.Client, error) {
	if !withLLM {
		return nil, nil
	}
	cfg, ok := llm.FromEnv(os.Getenv)
	if !ok {
		return nil, llm.ErrNotConfigured
	}
	real, err := llm.NewReal(ctx, cfg)
	if err != nil {
		return nil, err
	}
	// Canonical composition: Real → Budget → Cached (outermost).
	// See internal/adapter/llm/budget.go for the rationale.
	return llm.NewCached(llm.NewBudget(real, budget)), nil
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
		rule, ok := byID[findings[i].RuleID]
		if !ok {
			continue
		}
		prompt := llm.BuildFindingPrompt(rule, findings[i], projectType)
		sug, err := client.Explain(ctx, rule, findings[i], prompt)
		if err != nil {
			if errors.Is(err, llm.ErrBudgetExhausted) {
				// Quiet exit — user explicitly set the budget.
				return
			}
			fmt.Fprintf(stderr, "llm: finding %s skipped (%v)\n", findings[i].RuleID, err)
			continue
		}
		findings[i].LLMSuggestion = &model.LLMSuggestion{
			Text:      sug.Text,
			Model:     sug.Model,
			CacheHit:  sug.CacheHit,
			Truncated: sug.Truncated,
			LatencyMS: sug.LatencyMS,
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
	template := []byte(`{
  "version": 1,
  "project_type": [],
  "profile": "standard",
  "packs": {
    "enabled": ["core"]
  },
  "ignore": []
}
`)
	if err := os.WriteFile(target, template, 0o644); err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return exitRuntimeError
	}
	fmt.Fprintf(stdout, "wrote %s\n", target)
	return exitOK
}

// filepathJoin is a thin wrapper that keeps the import list short at the top of main.go.
func filepathJoin(a, b string) string {
	if a == "" || a == "." {
		return b
	}
	if a[len(a)-1] == os.PathSeparator {
		return a + b
	}
	return a + string(os.PathSeparator) + b
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
	})
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return exitRuntimeError
	}
	llmClient, err := buildLLMClient(ctx, flags.withLLM, flags.llmBudget)
	if err != nil {
		fmt.Fprintf(stderr, "check: --with-llm: %v\n", err)
		return exitConfigError
	}
	if llmClient != nil {
		defer llmClient.Close()
		enrichFindings(ctx, llmClient, []model.Rule{r}, res.Findings, nil, stderr)
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
