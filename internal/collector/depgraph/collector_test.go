package depgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/model"
)

func TestCollect_GoProject(t *testing.T) {
	dir := t.TempDir()
	modPath := "github.com/example/testmod"
	root := filepath.Join(dir, "testmod")

	// Create go.mod.
	writeFile(t, root, "go.mod", "module "+modPath+"\n\ngo 1.24\n")

	// Create two packages with an import relationship.
	writeFile(t, root, "cmd/main.go", `package main

import "github.com/example/testmod/internal/svc"

func main() { svc.Run() }
`)
	writeFile(t, root, "internal/svc/svc.go", `package svc

func Run() {}
`)

	repo := model.RepoFacts{
		Root: root,
		Files: []model.FileFact{
			{Path: "go.mod", Ext: ".mod"},
			{Path: "cmd/main.go", Ext: ".go"},
			{Path: "internal/svc/svc.go", Ext: ".go"},
		},
		ByPath: map[string]model.FileFact{
			"go.mod":              {Path: "go.mod", Ext: ".mod"},
			"cmd/main.go":         {Path: "cmd/main.go", Ext: ".go"},
			"internal/svc/svc.go": {Path: "internal/svc/svc.go", Ext: ".go"},
		},
	}

	g, err := Collect(repo)
	if err != nil {
		t.Fatal(err)
	}

	if g.PackageCount() != 2 {
		t.Errorf("PackageCount = %d, want 2", g.PackageCount())
	}

	found := false
	for _, e := range g.Edges {
		if e.From == "cmd" && e.To == "internal/svc" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected edge cmd -> internal/svc, got edges: %v", g.Edges)
	}
}

func TestCollect_NoGoMod(t *testing.T) {
	dir := t.TempDir()

	repo := model.RepoFacts{
		Root:   dir,
		Files:  []model.FileFact{{Path: "main.py", Ext: ".py"}},
		ByPath: map[string]model.FileFact{"main.py": {Path: "main.py", Ext: ".py"}},
	}

	g, err := Collect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if g.PackageCount() != 0 {
		t.Errorf("PackageCount = %d, want 0 for non-Go project", g.PackageCount())
	}
}

func TestCollect_NoGoFiles(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "emptymod")

	writeFile(t, root, "go.mod", "module github.com/example/emptymod\n\ngo 1.24\n")
	writeFile(t, root, "README.md", "# Empty\n")

	repo := model.RepoFacts{
		Root: root,
		Files: []model.FileFact{
			{Path: "go.mod", Ext: ".mod"},
			{Path: "README.md", Ext: ".md"},
		},
		ByPath: map[string]model.FileFact{
			"go.mod":    {Path: "go.mod", Ext: ".mod"},
			"README.md": {Path: "README.md", Ext: ".md"},
		},
	}

	g, err := Collect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if g.PackageCount() != 0 {
		t.Errorf("PackageCount = %d, want 0 for project with no .go files", g.PackageCount())
	}
}

// writeFile creates a file at dir/relPath with the given content.
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	abs := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
