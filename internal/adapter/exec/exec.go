// Package exec is archfit's only sanctioned place for running external commands.
//
// CLAUDE.md §2 / §4 make this an aggregation boundary: no other package may
// shell out. Tests use Fake; production uses Real. Resolvers never import this
// package — they consume the facts that collectors extract from it.
package exec

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

// Result captures the complete output of a command invocation.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Duration time.Duration
}

// Runner executes commands. Implementations must never retain the returned
// slices beyond the call.
type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) (Result, error)
}

// Real runs commands on the host. Use only in production paths.
type Real struct {
	// Timeout bounds any single invocation. Zero = no bound.
	Timeout time.Duration
}

// NewReal returns a Real runner with a 30-second default timeout.
func NewReal() *Real { return &Real{Timeout: 30 * time.Second} }

// Run executes the named command in dir and returns its output.
func (r *Real) Run(ctx context.Context, dir, name string, args ...string) (Result, error) {
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}
	start := time.Now()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := Result{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		Duration: time.Since(start),
	}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return res, err
	}
	return res, nil
}

// Fake is the test implementation. Responses are keyed by `name + " " + args...`
// (space-joined). Unknown invocations return an error so tests cannot depend on
// ambient shell state.
type Fake struct {
	Responses map[string]Result
	// Calls records every invocation, in order. Useful for assertions.
	Calls []string
}

// NewFake returns a Fake runner with an empty response map.
func NewFake() *Fake { return &Fake{Responses: map[string]Result{}} }

// Run looks up a pre-registered response keyed by the command string.
func (f *Fake) Run(_ context.Context, _, name string, args ...string) (Result, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	f.Calls = append(f.Calls, key)
	if r, ok := f.Responses[key]; ok {
		return r, nil
	}
	return Result{}, errors.New("exec.Fake: no response registered for: " + key)
}
