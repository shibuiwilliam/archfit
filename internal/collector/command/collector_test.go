package command_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shibuiwilliam/archfit/internal/adapter/exec"
	"github.com/shibuiwilliam/archfit/internal/collector/command"
)

func TestCollect_Makefile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("test:\n\techo ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := exec.NewFake()
	f.Responses["make test"] = exec.Result{
		Stdout:   []byte("PASS"),
		ExitCode: 0,
		Duration: 1500 * time.Millisecond,
	}

	results := command.Collect(context.Background(), f, root, 30*time.Second)
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Command != "make" {
		t.Errorf("command: got %q, want %q", r.Command, "make")
	}
	if len(r.Args) != 1 || r.Args[0] != "test" {
		t.Errorf("args: got %v, want [test]", r.Args)
	}
	if r.DurationMS != 1500 {
		t.Errorf("duration_ms: got %d, want 1500", r.DurationMS)
	}
	if r.Stdout != "PASS" {
		t.Errorf("stdout: got %q, want %q", r.Stdout, "PASS")
	}
	if r.Error != "" {
		t.Errorf("unexpected error: %s", r.Error)
	}
}

func TestCollect_GoMod(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := exec.NewFake()
	f.Responses["go test ./..."] = exec.Result{
		Stdout:   []byte("ok"),
		ExitCode: 0,
		Duration: 5 * time.Second,
	}

	results := command.Collect(context.Background(), f, root, 30*time.Second)
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Command != "go" {
		t.Errorf("command: got %q, want %q", r.Command, "go")
	}
	if len(r.Args) != 2 || r.Args[0] != "test" || r.Args[1] != "./..." {
		t.Errorf("args: got %v, want [test ./...]", r.Args)
	}
	if r.DurationMS != 5000 {
		t.Errorf("duration_ms: got %d, want 5000", r.DurationMS)
	}
}

func TestCollect_NoRecognizedFiles(t *testing.T) {
	root := t.TempDir()
	// No marker files — expect empty results.

	f := exec.NewFake()
	results := command.Collect(context.Background(), f, root, 30*time.Second)
	if len(results) != 0 {
		t.Fatalf("want 0 results, got %d", len(results))
	}
}

func TestCollect_StdoutTruncation(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create output larger than MaxOutputBytes (4096).
	bigOutput := strings.Repeat("x", command.MaxOutputBytes+500)
	f := exec.NewFake()
	f.Responses["go test ./..."] = exec.Result{
		Stdout:   []byte(bigOutput),
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	}

	results := command.Collect(context.Background(), f, root, 30*time.Second)
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if len(results[0].Stdout) != command.MaxOutputBytes {
		t.Errorf("stdout length: got %d, want %d", len(results[0].Stdout), command.MaxOutputBytes)
	}
}

func TestCollect_CommandError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Do not register a response for "cargo test" so the fake returns an error.
	f := exec.NewFake()

	results := command.Collect(context.Background(), f, root, 30*time.Second)
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Error == "" {
		t.Error("expected non-empty Error for unregistered command")
	}
	if r.Command != "cargo" {
		t.Errorf("command: got %q, want %q", r.Command, "cargo")
	}
}

func TestCollect_MultipleMarkers(t *testing.T) {
	root := t.TempDir()
	// Both Makefile and go.mod present — both should run.
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("test:"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := exec.NewFake()
	f.Responses["make test"] = exec.Result{Duration: time.Second}
	f.Responses["go test ./..."] = exec.Result{Duration: 2 * time.Second}

	results := command.Collect(context.Background(), f, root, 30*time.Second)
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	if results[0].Command != "make" {
		t.Errorf("first command: got %q, want %q", results[0].Command, "make")
	}
	if results[1].Command != "go" {
		t.Errorf("second command: got %q, want %q", results[1].Command, "go")
	}
}
