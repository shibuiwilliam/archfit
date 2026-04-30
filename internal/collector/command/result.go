// Package command runs verification commands (make test, go test, npm test, etc.)
// and records their wall-clock duration for the verification_latency_s metric.
//
// This is a collector: it gathers facts through internal/adapter/exec and never
// makes judgements. Resolvers consume TimedResult through the FactStore.
package command

// MaxOutputBytes caps stdout/stderr retained per command to prevent memory issues.
const MaxOutputBytes = 4 * 1024

// TimedResult records the outcome of a timed command execution.
type TimedResult struct {
	Command    string   `json:"command"`
	Args       []string `json:"args,omitempty"`
	ExitCode   int      `json:"exit_code"`
	DurationMS int64    `json:"duration_ms"`      // wall-clock ms
	Layer      string   `json:"layer,omitempty"`  // verification layer name
	Stdout     string   `json:"stdout,omitempty"` // truncated to MaxOutputBytes
	Stderr     string   `json:"stderr,omitempty"` // truncated to MaxOutputBytes
	Error      string   `json:"error,omitempty"`  // non-empty if command failed to start
}
