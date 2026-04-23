package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	collectfs "github.com/shibuiwilliam/archfit/internal/collector/fs"
)

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCollect_BasicAndDeterministic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "CLAUDE.md", "hi\nthere\n")
	writeFile(t, dir, "cmd/archfit/main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, dir, ".git/HEAD", "ref: refs/heads/main\n") // must be ignored
	writeFile(t, dir, "node_modules/a/b.js", "x")            // must be ignored

	facts, err := collectfs.Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(facts.Files), facts.Files)
	}
	if facts.Files[0].Path != "CLAUDE.md" || facts.Files[1].Path != "cmd/archfit/main.go" {
		t.Errorf("paths not sorted as expected: %+v", facts.Files)
	}
	if facts.Languages["go"] != 1 {
		t.Errorf("go count expected 1, got %d", facts.Languages["go"])
	}
	if got := facts.ByBase["claude.md"]; len(got) != 1 || got[0] != "CLAUDE.md" {
		t.Errorf("byBase missing CLAUDE.md: %v", got)
	}
}
