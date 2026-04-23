package git_test

import (
	"context"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/adapter/exec"
	"github.com/shibuiwilliam/archfit/internal/collector/git"
)

func TestCollect_HappyPath(t *testing.T) {
	f := exec.NewFake()
	f.Responses["git rev-parse --is-inside-work-tree"] = exec.Result{Stdout: []byte("true\n")}
	f.Responses["git rev-parse HEAD"] = exec.Result{Stdout: []byte("abc123\n")}
	f.Responses["git rev-parse --abbrev-ref HEAD"] = exec.Result{Stdout: []byte("main\n")}
	f.Responses["git rev-list --count HEAD"] = exec.Result{Stdout: []byte("42\n")}
	f.Responses["git log --max-count=50 --pretty=format:%H\t%s"] = exec.Result{
		Stdout: []byte("abc123\tinitial commit\ndef456\tfeat: add thing\n"),
	}

	facts, err := git.Collect(context.Background(), f, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if facts.CurrentCommit != "abc123" {
		t.Errorf("commit: %s", facts.CurrentCommit)
	}
	if facts.CurrentBranch != "main" {
		t.Errorf("branch: %s", facts.CurrentBranch)
	}
	if facts.CommitCount != 42 {
		t.Errorf("count: %d", facts.CommitCount)
	}
	if len(facts.RecentCommits) != 2 || facts.RecentCommits[1].Subject != "feat: add thing" {
		t.Errorf("recent: %+v", facts.RecentCommits)
	}
}

func TestCollect_NotAGitRepo(t *testing.T) {
	f := exec.NewFake() // nothing registered
	_, err := git.Collect(context.Background(), f, "/nogit")
	if err != git.ErrNoGit {
		t.Fatalf("want ErrNoGit, got %v", err)
	}
}
