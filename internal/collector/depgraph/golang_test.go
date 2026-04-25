package depgraph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectGo_BasicImports(t *testing.T) {
	dir := t.TempDir()
	modPath := "github.com/example/myproject"
	root := filepath.Join(dir, "myproject")

	// Create two packages: cmd/app and internal/core.
	// cmd/app imports internal/core.
	writeGoFile(t, root, "cmd/app/main.go", `package main

import "github.com/example/myproject/internal/core"

func main() { core.Run() }
`)
	writeGoFile(t, root, "internal/core/core.go", `package core

func Run() {}
`)

	files := []string{
		filepath.Join(root, "cmd", "app", "main.go"),
		filepath.Join(root, "internal", "core", "core.go"),
	}

	g, err := CollectGo(files, modPath)
	if err != nil {
		t.Fatal(err)
	}

	if g.PackageCount() != 2 {
		t.Errorf("PackageCount = %d, want 2", g.PackageCount())
	}

	// Check that the edge exists.
	found := false
	for _, e := range g.Edges {
		if e.From == "cmd/app" && e.To == "internal/core" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected edge cmd/app -> internal/core, got edges: %v", g.Edges)
	}
}

func TestCollectGo_ExternalImportsExcluded(t *testing.T) {
	dir := t.TempDir()
	modPath := "github.com/example/myproject"
	root := filepath.Join(dir, "myproject")

	writeGoFile(t, root, "pkg/handler.go", `package pkg

import (
	"fmt"
	"github.com/some/external"
	"github.com/example/myproject/internal/util"
)

func Handle() {
	fmt.Println(external.Name)
	util.Help()
}
`)
	writeGoFile(t, root, "internal/util/util.go", `package util

func Help() {}
`)

	files := []string{
		filepath.Join(root, "pkg", "handler.go"),
		filepath.Join(root, "internal", "util", "util.go"),
	}

	g, err := CollectGo(files, modPath)
	if err != nil {
		t.Fatal(err)
	}

	// Only internal edges should exist: pkg -> internal/util.
	for _, e := range g.Edges {
		if e.To == "fmt" || e.To == "github.com/some/external" {
			t.Errorf("external import should not appear as edge: %v", e)
		}
	}

	found := false
	for _, e := range g.Edges {
		if e.From == "pkg" && e.To == "internal/util" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected edge pkg -> internal/util, got edges: %v", g.Edges)
	}
}

func TestCollectGo_TestFilesIncluded(t *testing.T) {
	dir := t.TempDir()
	modPath := "github.com/example/myproject"
	root := filepath.Join(dir, "myproject")

	writeGoFile(t, root, "pkg/handler.go", `package pkg

func Handle() {}
`)
	writeGoFile(t, root, "pkg/handler_test.go", `package pkg

import (
	"testing"
	"github.com/example/myproject/internal/testutil"
)

func TestHandle(t *testing.T) { testutil.Setup() }
`)
	writeGoFile(t, root, "internal/testutil/testutil.go", `package testutil

func Setup() {}
`)

	files := []string{
		filepath.Join(root, "pkg", "handler.go"),
		filepath.Join(root, "pkg", "handler_test.go"),
		filepath.Join(root, "internal", "testutil", "testutil.go"),
	}

	g, err := CollectGo(files, modPath)
	if err != nil {
		t.Fatal(err)
	}

	// The test file's import should create an edge.
	found := false
	for _, e := range g.Edges {
		if e.From == "pkg" && e.To == "internal/testutil" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected edge from test file: pkg -> internal/testutil, got edges: %v", g.Edges)
	}
}

func TestCollectGo_FileCount(t *testing.T) {
	dir := t.TempDir()
	modPath := "github.com/example/myproject"
	root := filepath.Join(dir, "myproject")

	writeGoFile(t, root, "pkg/a.go", `package pkg
`)
	writeGoFile(t, root, "pkg/b.go", `package pkg
`)

	files := []string{
		filepath.Join(root, "pkg", "a.go"),
		filepath.Join(root, "pkg", "b.go"),
	}

	g, err := CollectGo(files, modPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, n := range g.Nodes {
		if n.Package == "pkg" {
			if n.Files != 2 {
				t.Errorf("pkg files = %d, want 2", n.Files)
			}
			return
		}
	}
	t.Error("pkg node not found")
}

// writeGoFile creates a Go source file at dir/relPath with the given content.
func writeGoFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	abs := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
