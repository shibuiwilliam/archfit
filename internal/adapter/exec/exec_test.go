package exec_test

import (
	"context"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/adapter/exec"
)

func TestFake_RecordsCallsAndReturnsResponses(t *testing.T) {
	f := exec.NewFake()
	f.Responses["git status"] = exec.Result{Stdout: []byte("clean\n")}
	got, err := f.Run(context.Background(), ".", "git", "status")
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Stdout) != "clean\n" {
		t.Errorf("unexpected stdout %q", got.Stdout)
	}
	if len(f.Calls) != 1 || f.Calls[0] != "git status" {
		t.Errorf("unexpected calls: %v", f.Calls)
	}
}

func TestFake_UnknownCallErrors(t *testing.T) {
	f := exec.NewFake()
	if _, err := f.Run(context.Background(), ".", "nope"); err == nil {
		t.Fatal("expected error for unregistered command")
	}
}
