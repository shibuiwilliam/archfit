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
	f.Responses["git log --max-count=2 --numstat --pretty=format:commit %H"] = exec.Result{
		Stdout: []byte("commit abc123\n1\t0\tREADME.md\n\ncommit def456\n2\t1\tapp.go\n1\t0\tgo.mod\n"),
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

func TestCollect_PopulatesFilesChanged(t *testing.T) {
	f := exec.NewFake()
	f.Responses["git rev-parse --is-inside-work-tree"] = exec.Result{Stdout: []byte("true\n")}
	f.Responses["git rev-parse HEAD"] = exec.Result{Stdout: []byte("aaa\n")}
	f.Responses["git rev-parse --abbrev-ref HEAD"] = exec.Result{Stdout: []byte("main\n")}
	f.Responses["git rev-list --count HEAD"] = exec.Result{Stdout: []byte("3\n")}
	f.Responses["git log --max-count=50 --pretty=format:%H\t%s"] = exec.Result{
		Stdout: []byte("aaa\tfeat: add three files\nbbb\tMerge branch 'feature'\nccc\tfeat: add binary\n"),
	}

	// Numstat output with three scenarios:
	// - commit aaa: 3 files changed (normal)
	// - commit bbb: merge commit, 0 numstat lines
	// - commit ccc: binary-only commit (- - notation)
	f.Responses["git log --max-count=3 --numstat --pretty=format:commit %H"] = exec.Result{
		Stdout: []byte(
			"commit aaa\n" +
				"10\t5\tsrc/main.go\n" +
				"3\t1\tsrc/util.go\n" +
				"1\t0\tREADME.md\n" +
				"\n" +
				"commit bbb\n" +
				"\n" +
				"commit ccc\n" +
				"-\t-\tassets/logo.png\n"),
	}

	facts, err := git.Collect(context.Background(), f, "/repo")
	if err != nil {
		t.Fatal(err)
	}

	if len(facts.RecentCommits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(facts.RecentCommits))
	}

	tests := []struct {
		hash string
		want int
		desc string
	}{
		{"aaa", 3, "normal commit with 3 files"},
		{"bbb", 0, "merge commit with no numstat lines"},
		{"ccc", 1, "binary-only commit (- - notation)"},
	}
	for _, tt := range tests {
		for _, c := range facts.RecentCommits {
			if c.Hash == tt.hash {
				if c.FilesChanged != tt.want {
					t.Errorf("%s (%s): FilesChanged = %d, want %d", tt.hash, tt.desc, c.FilesChanged, tt.want)
				}
			}
		}
	}
}

func TestCollect_NumstatFailure_DoesNotBreak(t *testing.T) {
	f := exec.NewFake()
	f.Responses["git rev-parse --is-inside-work-tree"] = exec.Result{Stdout: []byte("true\n")}
	f.Responses["git rev-parse HEAD"] = exec.Result{Stdout: []byte("aaa\n")}
	f.Responses["git rev-parse --abbrev-ref HEAD"] = exec.Result{Stdout: []byte("main\n")}
	f.Responses["git rev-list --count HEAD"] = exec.Result{Stdout: []byte("1\n")}
	f.Responses["git log --max-count=50 --pretty=format:%H\t%s"] = exec.Result{
		Stdout: []byte("aaa\tinit\n"),
	}
	// No numstat response registered — the fake will return an error.
	// Collector should not fail; FilesChanged stays 0.

	facts, err := git.Collect(context.Background(), f, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if facts.RecentCommits[0].FilesChanged != 0 {
		t.Errorf("expected FilesChanged=0 when numstat fails, got %d", facts.RecentCommits[0].FilesChanged)
	}
}

func TestCollect_NotAGitRepo(t *testing.T) {
	f := exec.NewFake() // nothing registered
	_, err := git.Collect(context.Background(), f, "/nogit")
	if err != git.ErrNoGit {
		t.Fatalf("want ErrNoGit, got %v", err)
	}
}
