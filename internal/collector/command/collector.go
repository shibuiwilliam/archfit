package command

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/shibuiwilliam/archfit/internal/adapter/exec"
)

// commandSpec maps a marker file to the command that should be run.
type commandSpec struct {
	marker string   // file whose presence triggers this command
	name   string   // executable name
	args   []string // arguments
}

// knownCommands is the ordered list of verification commands archfit knows about.
// Order is deterministic: if a repo has both Makefile and go.mod, both run.
var knownCommands = []commandSpec{
	{marker: "Makefile", name: "make", args: []string{"test"}},
	{marker: "package.json", name: "npm", args: []string{"test"}},
	{marker: "go.mod", name: "go", args: []string{"test", "./..."}},
	{marker: "pyproject.toml", name: "pytest", args: nil},
	{marker: "Cargo.toml", name: "cargo", args: []string{"test"}},
}

// Collect detects verification commands from the repo structure at root and
// runs each one through runner, recording timing results. Each command is
// given at most timeout to complete. Results are returned for every detected
// command, including failures. Collect never returns an error — failures are
// captured in TimedResult.Error.
func Collect(ctx context.Context, runner exec.Runner, root string, timeout time.Duration) []TimedResult {
	var results []TimedResult

	for _, spec := range knownCommands {
		path := filepath.Join(root, spec.marker)
		if _, err := os.Stat(path); err != nil {
			continue
		}

		tr := runCommand(ctx, runner, root, spec, timeout)
		results = append(results, tr)
	}

	return results
}

// runCommand executes a single command spec and returns the timed result.
func runCommand(ctx context.Context, runner exec.Runner, root string, spec commandSpec, timeout time.Duration) TimedResult {
	tr := TimedResult{
		Command: spec.name,
		Args:    spec.args,
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := runner.Run(cmdCtx, root, spec.name, spec.args...)
	if err != nil {
		tr.Error = err.Error()
		return tr
	}

	tr.ExitCode = res.ExitCode
	tr.DurationMS = res.Duration.Milliseconds()
	tr.Stdout = truncate(string(res.Stdout), MaxOutputBytes)
	tr.Stderr = truncate(string(res.Stderr), MaxOutputBytes)

	return tr
}

// truncate returns s trimmed to at most maxBytes bytes. If truncation occurs,
// it is a simple byte cut — no attempt to preserve UTF-8 boundaries, since
// this is diagnostic output.
func truncate(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes]
}
