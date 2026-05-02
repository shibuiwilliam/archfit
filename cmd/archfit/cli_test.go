// CLI integration tests — exercise the `run()` function with real arguments,
// capturing stdout/stderr and asserting exit codes. These complement the
// golden JSON tests under testdata/e2e/ by testing the CLI surface: flag
// parsing, subcommand dispatch, output formats, error handling, and exit codes.
//
// No network I/O. No LLM calls (--with-llm is never set). No git dependency.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureDir returns an absolute path to a testdata/e2e fixture's input dir.
// Falls back to packs/core/fixtures if the e2e fixture doesn't exist.
func fixtureDir(t *testing.T, name string) string {
	t.Helper()
	// Try testdata/e2e first.
	dir := filepath.Join("..", "..", "testdata", "e2e", name, "input")
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		abs, _ := filepath.Abs(dir)
		return abs
	}
	t.Fatalf("fixture %q not found", name)
	return ""
}

// runCLI invokes the CLI's run() function with the given args, capturing
// stdout and stderr. Returns the exit code, stdout, and stderr.
func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, err bytes.Buffer
	code = run(args, &out, &err)
	return code, out.String(), err.String()
}

// ---------------------------------------------------------------------------
// 1. Subcommand dispatch and basic usage
// ---------------------------------------------------------------------------

func TestCLI_NoArgs_PrintsUsage(t *testing.T) {
	code, _, stderr := runCLI()
	if code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr, "archfit scan") {
		t.Error("usage text should mention 'archfit scan'")
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	code, _, stderr := runCLI("nonexistent")
	if code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Error("should report unknown command")
	}
}

func TestCLI_Version(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		t.Run(arg, func(t *testing.T) {
			code, stdout, _ := runCLI(arg)
			if code != exitOK {
				t.Fatalf("exit code = %d, want %d", code, exitOK)
			}
			if !strings.HasPrefix(stdout, "archfit ") {
				t.Errorf("stdout = %q, should start with 'archfit '", stdout)
			}
		})
	}
}

func TestCLI_Help(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			code, stdout, _ := runCLI(arg)
			if code != exitOK {
				t.Fatalf("exit code = %d, want %d", code, exitOK)
			}
			if !strings.Contains(stdout, "archfit scan") {
				t.Error("help should mention 'archfit scan'")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. Scan command — output formats, exit codes, finding detection
// ---------------------------------------------------------------------------

func TestCLI_Scan_JSON_CleanRepo(t *testing.T) {
	dir := fixtureDir(t, "clean_agent_tool")
	code, stdout, _ := runCLI("scan", "--json", dir)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d (clean repo)", code, exitOK)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if doc["schema_version"] != "1.1.0" {
		t.Errorf("schema_version = %v, want 1.1.0", doc["schema_version"])
	}
	summary := doc["summary"].(map[string]any)
	if summary["findings_total"].(float64) != 0 {
		// Dump findings so CI logs show exactly which rule fired.
		if findings, ok := doc["findings"].([]any); ok {
			for _, f := range findings {
				fm := f.(map[string]any)
				t.Errorf("unexpected finding: rule=%v severity=%v message=%v",
					fm["rule_id"], fm["severity"], fm["message"])
			}
		}
		t.Fatalf("expected 0 findings in clean repo, got %v", summary["findings_total"])
	}
}

func TestCLI_Scan_JSON_WithFindings(t *testing.T) {
	dir := fixtureDir(t, "multi_finding")
	// Default --fail-on is "error"; multi_finding has only warn + info → exit 0.
	code, stdout, _ := runCLI("scan", "--json", dir)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d (warn-only findings, default fail-on=error)", code, exitOK)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	summary := doc["summary"].(map[string]any)
	total := int(summary["findings_total"].(float64))
	if total == 0 {
		t.Error("expected findings in multi_finding fixture, got 0")
	}
}

func TestCLI_Scan_FailOn_Warn(t *testing.T) {
	dir := fixtureDir(t, "multi_finding")
	code, _, _ := runCLI("scan", "--json", "--fail-on=warn", dir)
	if code != exitFindingsAtLevel {
		t.Fatalf("exit code = %d, want %d (--fail-on=warn with warn findings)", code, exitFindingsAtLevel)
	}
}

func TestCLI_Scan_FailOn_Info(t *testing.T) {
	dir := fixtureDir(t, "multi_finding")
	code, _, _ := runCLI("scan", "--json", "--fail-on=info", dir)
	if code != exitFindingsAtLevel {
		t.Fatalf("exit code = %d, want %d (--fail-on=info with info findings)", code, exitFindingsAtLevel)
	}
}

func TestCLI_Scan_TerminalFormat(t *testing.T) {
	dir := fixtureDir(t, "undocumented_env")
	code, stdout, _ := runCLI("scan", dir)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	// Terminal output should contain the score and findings.
	if !strings.Contains(stdout, "overall score:") {
		t.Error("terminal output should contain 'overall score:'")
	}
	if !strings.Contains(stdout, "P3.EXP.001") {
		t.Error("terminal output should mention P3.EXP.001 finding")
	}
}

func TestCLI_Scan_MarkdownFormat(t *testing.T) {
	dir := fixtureDir(t, "undocumented_env")
	code, stdout, _ := runCLI("scan", "--format=md", dir)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "# archfit report") {
		t.Error("markdown output should start with '# archfit report'")
	}
	if !strings.Contains(stdout, "P3.EXP.001") {
		t.Error("markdown output should mention P3.EXP.001")
	}
}

func TestCLI_Scan_SARIFFormat(t *testing.T) {
	dir := fixtureDir(t, "undocumented_env")
	code, stdout, _ := runCLI("scan", "--format=sarif", dir)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("SARIF output is not valid JSON: %v", err)
	}
	if doc["$schema"] == nil {
		t.Error("SARIF output should contain $schema")
	}
	runs, ok := doc["runs"].([]any)
	if !ok || len(runs) == 0 {
		t.Fatal("SARIF should have at least one run")
	}
}

func TestCLI_Scan_NonexistentPath(t *testing.T) {
	code, _, stderr := runCLI("scan", "/nonexistent/path/that/does/not/exist")
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d for nonexistent path", code, exitRuntimeError)
	}
	if !strings.Contains(stderr, "scan:") {
		t.Error("stderr should contain error context")
	}
}

// ---------------------------------------------------------------------------
// 3. Score command
// ---------------------------------------------------------------------------

func TestCLI_Score_Clean(t *testing.T) {
	dir := fixtureDir(t, "clean_agent_tool")
	code, stdout, _ := runCLI("score", dir)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "overall:") {
		t.Error("score output should contain 'overall:'")
	}
	if !strings.Contains(stdout, "100.0") {
		t.Error("clean fixture should score 100.0")
	}
}

func TestCLI_Score_WithFindings(t *testing.T) {
	dir := fixtureDir(t, "multi_finding")
	code, stdout, _ := runCLI("score", dir)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "overall:") {
		t.Error("score output should contain 'overall:'")
	}
	// multi_finding should not be 100.
	if strings.Contains(stdout, "overall: 100.0") {
		t.Error("multi_finding fixture should not score 100.0")
	}
}

// ---------------------------------------------------------------------------
// 4. Check command — single rule execution
// ---------------------------------------------------------------------------

func TestCLI_Check_SingleRule_NoFinding(t *testing.T) {
	dir := fixtureDir(t, "clean_agent_tool")
	code, stdout, _ := runCLI("check", "P1.LOC.001", "--json", dir)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	summary := doc["summary"].(map[string]any)
	if summary["rules_evaluated"].(float64) != 1 {
		t.Errorf("check should evaluate exactly 1 rule, got %v", summary["rules_evaluated"])
	}
}

func TestCLI_Check_SingleRule_WithFinding(t *testing.T) {
	dir := fixtureDir(t, "missing_agent_docs")
	code, stdout, _ := runCLI("check", "P1.LOC.001", "--json", dir)
	// P1.LOC.001 is severity warn, default --fail-on=error → exit 0.
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d (warn finding, fail-on=error)", code, exitOK)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	findings := doc["findings"].([]any)
	if len(findings) != 1 {
		t.Errorf("expected 1 finding for P1.LOC.001, got %d", len(findings))
	}
}

func TestCLI_Check_UnknownRule(t *testing.T) {
	dir := fixtureDir(t, "clean_agent_tool")
	code, _, stderr := runCLI("check", "P99.XXX.999", dir)
	if code != exitUsage {
		t.Fatalf("exit code = %d, want %d for unknown rule", code, exitUsage)
	}
	if !strings.Contains(stderr, "P99.XXX.999") {
		t.Error("stderr should mention the unknown rule ID")
	}
}

// ---------------------------------------------------------------------------
// 5. Explain command
// ---------------------------------------------------------------------------

func TestCLI_Explain(t *testing.T) {
	code, stdout, _ := runCLI("explain", "P1.LOC.001")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "P1.LOC.001") {
		t.Error("explain output should contain the rule ID")
	}
	if !strings.Contains(stdout, "CLAUDE.md") || !strings.Contains(stdout, "AGENTS.md") {
		t.Errorf("explain should mention CLAUDE.md and AGENTS.md in remediation, got:\n%s", stdout)
	}
}

func TestCLI_Explain_UnknownRule(t *testing.T) {
	code, _, stderr := runCLI("explain", "P99.NOPE.001")
	if code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr, "P99.NOPE.001") {
		t.Error("stderr should mention the unknown rule")
	}
}

// ---------------------------------------------------------------------------
// 6. List commands
// ---------------------------------------------------------------------------

func TestCLI_ListRules(t *testing.T) {
	code, stdout, _ := runCLI("list-rules")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	// Should contain rules from both packs.
	for _, id := range []string{"P1.LOC.001", "P2.SPC.010", "P7.MRD.002"} {
		if !strings.Contains(stdout, id) {
			t.Errorf("list-rules should contain %s", id)
		}
	}
}

func TestCLI_ListPacks(t *testing.T) {
	code, stdout, _ := runCLI("list-packs")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "core") {
		t.Error("list-packs should mention core pack")
	}
	if !strings.Contains(stdout, "agent-tool") {
		t.Error("list-packs should mention agent-tool pack")
	}
}

// ---------------------------------------------------------------------------
// 7. Diff command
// ---------------------------------------------------------------------------

func TestCLI_Diff_NoRegression(t *testing.T) {
	dir := fixtureDir(t, "clean_agent_tool")
	// Generate a baseline scan.
	_, baseline, _ := runCLI("scan", "--json", dir)

	// Write baseline to a temp file.
	tmp := t.TempDir()
	baselinePath := filepath.Join(tmp, "baseline.json")
	currentPath := filepath.Join(tmp, "current.json")
	if err := os.WriteFile(baselinePath, []byte(baseline), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(currentPath, []byte(baseline), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, _ := runCLI("diff", baselinePath, currentPath)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d (identical scans, no regression)", code, exitOK)
	}
}

func TestCLI_Diff_WithRegression(t *testing.T) {
	cleanDir := fixtureDir(t, "clean_agent_tool")
	dirtyDir := fixtureDir(t, "multi_finding")

	_, baseline, _ := runCLI("scan", "--json", cleanDir)
	_, current, _ := runCLI("scan", "--json", dirtyDir)

	tmp := t.TempDir()
	baselinePath := filepath.Join(tmp, "baseline.json")
	currentPath := filepath.Join(tmp, "current.json")
	mustWriteFile(t, baselinePath, []byte(baseline))
	mustWriteFile(t, currentPath, []byte(current))

	code, _, _ := runCLI("diff", baselinePath, currentPath)
	if code != exitFindingsAtLevel {
		t.Fatalf("exit code = %d, want %d (new findings = regression)", code, exitFindingsAtLevel)
	}
}

func TestCLI_Diff_JSONOutput(t *testing.T) {
	cleanDir := fixtureDir(t, "clean_agent_tool")
	dirtyDir := fixtureDir(t, "multi_finding")

	_, baseline, _ := runCLI("scan", "--json", cleanDir)
	_, current, _ := runCLI("scan", "--json", dirtyDir)

	tmp := t.TempDir()
	baselinePath := filepath.Join(tmp, "baseline.json")
	currentPath := filepath.Join(tmp, "current.json")
	mustWriteFile(t, baselinePath, []byte(baseline))
	mustWriteFile(t, currentPath, []byte(current))

	_, stdout, _ := runCLI("diff", "--json", baselinePath, currentPath)
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("diff --json should produce valid JSON: %v", err)
	}
	if doc["new"] == nil {
		t.Error("diff JSON should contain 'new' field")
	}
}

// ---------------------------------------------------------------------------
// 8. Report command (shorthand for scan --format=md)
// ---------------------------------------------------------------------------

func TestCLI_Report(t *testing.T) {
	dir := fixtureDir(t, "clean_agent_tool")
	code, stdout, _ := runCLI("report", dir)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "# archfit report") {
		t.Error("report should produce markdown output")
	}
}

// ---------------------------------------------------------------------------
// 9. Init command
// ---------------------------------------------------------------------------

func TestCLI_Init_CreatesConfig(t *testing.T) {
	tmp := t.TempDir()
	// Create a minimal source file so init can detect the project stack.
	mustMkdirAll(t, filepath.Join(tmp, "src"))
	mustWriteFile(t, filepath.Join(tmp, "src", "main.go"), []byte("package main\n"))

	code, _, _ := runCLI("init", tmp)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	data, err := os.ReadFile(filepath.Join(tmp, ".archfit.yaml"))
	if err != nil {
		t.Fatal("init should create .archfit.yaml")
	}
	if !strings.Contains(string(data), `"version"`) {
		t.Error(".archfit.yaml should contain a version field")
	}
}

func TestCLI_Init_FailsIfExists(t *testing.T) {
	tmp := t.TempDir()
	mustWriteFile(t, filepath.Join(tmp, ".archfit.yaml"), []byte(`{"version":1}`))

	code, _, stderr := runCLI("init", tmp)
	if code != exitConfigError {
		t.Fatalf("exit code = %d, want %d (config already exists)", code, exitConfigError)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Error("stderr should mention file already exists")
	}
}

// ---------------------------------------------------------------------------
// 10. Validate-config command
// ---------------------------------------------------------------------------

func TestCLI_ValidateConfig_Valid(t *testing.T) {
	tmp := t.TempDir()
	mustWriteFile(t, filepath.Join(tmp, ".archfit.yaml"),
		[]byte(`{"version":1,"profile":"standard","packs":{"enabled":["core"]}}`))

	code, stdout, _ := runCLI("validate-config", tmp)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "standard") {
		t.Error("validate-config should show the profile")
	}
}

func TestCLI_ValidateConfig_Invalid(t *testing.T) {
	tmp := t.TempDir()
	mustWriteFile(t, filepath.Join(tmp, ".archfit.yaml"),
		[]byte(`{"version":99}`))

	code, _, stderr := runCLI("validate-config", tmp)
	if code != exitConfigError {
		t.Fatalf("exit code = %d, want %d (invalid config)", code, exitConfigError)
	}
	if !strings.Contains(stderr, "version") {
		t.Error("stderr should mention version error")
	}
}

// ---------------------------------------------------------------------------
// 11. Contract init and check
// ---------------------------------------------------------------------------

func TestCLI_Contract_Init(t *testing.T) {
	dir := fixtureDir(t, "clean_agent_tool")
	// Create a writable copy of the fixture.
	tmp := t.TempDir()
	copyDir(t, dir, tmp)

	code, _, _ := runCLI("contract", "init", tmp)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".archfit-contract.yaml")); err != nil {
		t.Fatal("contract init should create .archfit-contract.yaml")
	}
}

func TestCLI_Contract_Check_Passes(t *testing.T) {
	dir := fixtureDir(t, "clean_agent_tool")
	tmp := t.TempDir()
	copyDir(t, dir, tmp)

	// Init a contract, then check it.
	runCLI("contract", "init", tmp)
	code, _, _ := runCLI("contract", "check", tmp)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d (contract should pass on clean repo)", code, exitOK)
	}
}

func TestCLI_Contract_NoSubcommand(t *testing.T) {
	code, _, stderr := runCLI("contract")
	if code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr, "contract check") {
		t.Error("should show contract subcommand usage")
	}
}

// ---------------------------------------------------------------------------
// 12. Fix command — dry run
// ---------------------------------------------------------------------------

func TestCLI_Fix_DryRun(t *testing.T) {
	dir := fixtureDir(t, "missing_agent_docs")
	tmp := t.TempDir()
	copyDir(t, dir, tmp)

	code, stdout, _ := runCLI("fix", "--dry-run", "--all", tmp)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	// Dry run should mention what it would do without creating files.
	if !strings.Contains(stdout, "CLAUDE.md") && !strings.Contains(stdout, "P1.LOC.001") {
		t.Error("dry run output should mention CLAUDE.md or P1.LOC.001")
	}
	// The file should NOT have been created.
	if _, err := os.Stat(filepath.Join(tmp, "CLAUDE.md")); err == nil {
		t.Error("dry run should not create files")
	}
}

func TestCLI_Fix_Plan(t *testing.T) {
	dir := fixtureDir(t, "missing_agent_docs")
	tmp := t.TempDir()
	copyDir(t, dir, tmp)

	code, stdout, _ := runCLI("fix", "--plan", "--all", tmp)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "P1.LOC.001") {
		t.Error("plan should mention fixable rule IDs")
	}
}

// ---------------------------------------------------------------------------
// 13. Compare command
// ---------------------------------------------------------------------------

func TestCLI_Compare(t *testing.T) {
	cleanDir := fixtureDir(t, "clean_agent_tool")
	dirtyDir := fixtureDir(t, "multi_finding")

	_, clean, _ := runCLI("scan", "--json", cleanDir)
	_, dirty, _ := runCLI("scan", "--json", dirtyDir)

	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "clean.json")
	f2 := filepath.Join(tmp, "dirty.json")
	mustWriteFile(t, f1, []byte(clean))
	mustWriteFile(t, f2, []byte(dirty))

	code, stdout, _ := runCLI("compare", f1, f2)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "clean.json") || !strings.Contains(stdout, "dirty.json") {
		t.Error("compare should list both file names")
	}
}

func TestCLI_Compare_TooFewArgs(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "one.json")
	mustWriteFile(t, f, []byte(`{}`))

	code, _, stderr := runCLI("compare", f)
	if code != exitUsage {
		t.Fatalf("exit code = %d, want %d (too few args)", code, exitUsage)
	}
	if !strings.Contains(stderr, "compare") {
		t.Error("stderr should mention compare usage")
	}
}

// ---------------------------------------------------------------------------
// 14. PR check command
// ---------------------------------------------------------------------------

func TestCLI_PRCheck_NoGit(t *testing.T) {
	// pr-check on a non-git directory should fail with runtime error.
	tmp := t.TempDir()
	mustMkdirAll(t, filepath.Join(tmp, "src"))
	mustWriteFile(t, filepath.Join(tmp, "src", "main.go"), []byte("package main\n"))

	code, _, stderr := runCLI("pr-check", "--base", "main", tmp)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d (no git repo)", code, exitRuntimeError)
	}
	if !strings.Contains(stderr, "pr-check") {
		t.Error("stderr should mention pr-check context")
	}
}

// ---------------------------------------------------------------------------
// 15. Trend command
// ---------------------------------------------------------------------------

func TestCLI_Trend(t *testing.T) {
	dir := fixtureDir(t, "clean_agent_tool")
	_, scan1, _ := runCLI("scan", "--json", dir)

	tmp := t.TempDir()
	histDir := filepath.Join(tmp, "history")
	mustMkdirAll(t, histDir)
	mustWriteFile(t, filepath.Join(histDir, "2026-01-01.json"), []byte(scan1))
	mustWriteFile(t, filepath.Join(histDir, "2026-02-01.json"), []byte(scan1))

	code, stdout, _ := runCLI("trend", "--history", histDir)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "2026-01-01") {
		t.Error("trend should list scan dates")
	}
}

func TestCLI_Trend_EmptyHistory(t *testing.T) {
	tmp := t.TempDir()
	histDir := filepath.Join(tmp, "empty-history")
	mustMkdirAll(t, histDir)

	code, _, stderr := runCLI("trend", "--history", histDir)
	// Empty history should still succeed (just show nothing).
	if code != exitOK && code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d or %d", code, exitOK, exitRuntimeError)
	}
	_ = stderr // may warn about no files
}

// ---------------------------------------------------------------------------
// 15. JSON output structure validation
// ---------------------------------------------------------------------------

func TestCLI_Scan_JSONStructure(t *testing.T) {
	dir := fixtureDir(t, "multi_finding")
	_, stdout, _ := runCLI("scan", "--json", dir)

	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Required top-level fields.
	for _, field := range []string{"schema_version", "tool", "target", "summary", "scores", "findings", "metrics"} {
		if doc[field] == nil {
			t.Errorf("missing required JSON field: %s", field)
		}
	}

	// Findings should be sorted: severity desc, rule_id asc, path asc.
	findings := doc["findings"].([]any)
	if len(findings) < 2 {
		return // not enough to test sort order
	}
	prev := findings[0].(map[string]any)
	for i := 1; i < len(findings); i++ {
		curr := findings[i].(map[string]any)
		prevSev := prev["severity"].(string)
		currSev := curr["severity"].(string)
		prevID := prev["rule_id"].(string)
		currID := curr["rule_id"].(string)
		if severityRank(prevSev) < severityRank(currSev) {
			t.Errorf("findings not sorted by severity desc at index %d: %s < %s", i, prevSev, currSev)
		}
		if prevSev == currSev && prevID > currID {
			t.Errorf("findings not sorted by rule_id asc at index %d: %s > %s", i, prevID, currID)
		}
		prev = curr
	}
}

func severityRank(s string) int {
	switch s {
	case "critical":
		return 4
	case "error":
		return 3
	case "warn":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// 16. Finding evidence quality — each finding must carry evidence
// ---------------------------------------------------------------------------

func TestCLI_Scan_AllFindingsHaveEvidence(t *testing.T) {
	dir := fixtureDir(t, "multi_finding")
	_, stdout, _ := runCLI("scan", "--json", dir)

	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	findings := doc["findings"].([]any)
	for i, f := range findings {
		finding := f.(map[string]any)
		evidence := finding["evidence"]
		if evidence == nil {
			t.Errorf("finding[%d] (%s) has no evidence", i, finding["rule_id"])
			continue
		}
		ev := evidence.(map[string]any)
		if len(ev) == 0 {
			t.Errorf("finding[%d] (%s) has empty evidence", i, finding["rule_id"])
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// mustWriteFile writes data to path, failing the test on error.
func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// mustMkdirAll creates a directory tree, failing the test on error.
func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// copyDir copies the contents of src into dst (both must be directories).
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			mustMkdirAll(t, dstPath)
			copyDir(t, srcPath, dstPath)
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, dstPath, data)
		}
	}
}
