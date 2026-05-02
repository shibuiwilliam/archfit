package goast_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/collector/ast/goast"
)

// writeTemp writes content to a temp .go file and returns its absolute path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseGoFile_InitFunction(t *testing.T) {
	src := `package main

import "net/http"

func init() {
	http.HandleFunc("/", handler)
}

func handler() {}
`
	abs := writeTemp(t, "main.go", src)
	facts, err := goast.ParseGoFile(abs, "main.go", "deep")
	if err != nil {
		t.Fatal(err)
	}
	if facts.Package != "main" {
		t.Errorf("package = %q, want main", facts.Package)
	}
	if len(facts.InitFunctions) != 1 {
		t.Fatalf("init functions = %d, want 1", len(facts.InitFunctions))
	}
	init0 := facts.InitFunctions[0]
	if init0.Line == 0 {
		t.Error("init line should be > 0")
	}
	// Deep mode extracts cross-pkg calls.
	if len(init0.CrossPkgCalls) != 1 || init0.CrossPkgCalls[0] != "http.HandleFunc" {
		t.Errorf("cross-pkg calls = %v, want [http.HandleFunc]", init0.CrossPkgCalls)
	}
}

func TestParseGoFile_InitFunction_StandardMode(t *testing.T) {
	src := `package main

import "net/http"

func init() {
	http.HandleFunc("/", handler)
}

func handler() {}
`
	abs := writeTemp(t, "main.go", src)
	facts, err := goast.ParseGoFile(abs, "main.go", "standard")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.InitFunctions) != 1 {
		t.Fatalf("init functions = %d, want 1", len(facts.InitFunctions))
	}
	// Standard mode does NOT extract cross-pkg calls.
	if len(facts.InitFunctions[0].CrossPkgCalls) != 0 {
		t.Errorf("standard mode should not extract cross-pkg calls, got %v", facts.InitFunctions[0].CrossPkgCalls)
	}
}

func TestParseGoFile_PkgLevelVars(t *testing.T) {
	src := `package config

var (
	DefaultTimeout = 30
	maxRetries     = 3
)

var GlobalDB *DB
`
	abs := writeTemp(t, "config.go", src)
	facts, err := goast.ParseGoFile(abs, "config.go", "standard")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.PkgLevelVars) != 3 {
		t.Fatalf("pkg vars = %d, want 3", len(facts.PkgLevelVars))
	}
	for _, v := range facts.PkgLevelVars {
		if !v.Mutable {
			t.Errorf("var %s should be mutable", v.Name)
		}
	}
}

func TestParseGoFile_Interfaces(t *testing.T) {
	src := `package svc

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Empty interface{}
`
	abs := writeTemp(t, "svc.go", src)
	facts, err := goast.ParseGoFile(abs, "svc.go", "standard")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Interfaces) != 2 {
		t.Fatalf("interfaces = %d, want 2", len(facts.Interfaces))
	}
	if facts.Interfaces[0].Name != "Reader" || facts.Interfaces[0].MethodCount != 1 {
		t.Errorf("Reader interface: name=%s methods=%d", facts.Interfaces[0].Name, facts.Interfaces[0].MethodCount)
	}
	if facts.Interfaces[1].Name != "Empty" || facts.Interfaces[1].MethodCount != 0 {
		t.Errorf("Empty interface: name=%s methods=%d", facts.Interfaces[1].Name, facts.Interfaces[1].MethodCount)
	}
}

func TestParseGoFile_ReflectImport(t *testing.T) {
	src := `package main

import "reflect"

func foo() {
	reflect.TypeOf(42)
	reflect.ValueOf("hello")
}
`
	abs := writeTemp(t, "main.go", src)

	// Standard mode detects import but does NOT count calls.
	facts, err := goast.ParseGoFile(abs, "main.go", "standard")
	if err != nil {
		t.Fatal(err)
	}
	if !facts.ReflectImports {
		t.Error("expected ReflectImports=true")
	}
	if facts.ReflectCalls != 0 {
		t.Errorf("standard mode: ReflectCalls = %d, want 0", facts.ReflectCalls)
	}

	// Deep mode counts reflect calls.
	facts, err = goast.ParseGoFile(abs, "main.go", "deep")
	if err != nil {
		t.Fatal(err)
	}
	if facts.ReflectCalls != 2 {
		t.Errorf("deep mode: ReflectCalls = %d, want 2", facts.ReflectCalls)
	}
}

func TestParseGoFile_ParseFailure(t *testing.T) {
	src := `package broken

func foo( {  // syntax error
`
	abs := writeTemp(t, "bad.go", src)
	_, err := goast.ParseGoFile(abs, "bad.go", "standard")
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse error") {
		t.Errorf("error should mention parse error, got: %v", err)
	}
}

func TestParseGoFile_FileSizeLimit(t *testing.T) {
	// Create a file just over the 1 MiB limit.
	dir := t.TempDir()
	p := filepath.Join(dir, "big.go")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	// Write a valid Go header then pad to exceed limit.
	header := "package big\n\n"
	if _, err := f.WriteString(header); err != nil {
		t.Fatal(err)
	}
	padding := make([]byte, goast.MaxFileSize+1-len(header))
	for i := range padding {
		padding[i] = ' '
	}
	if _, err := f.Write(padding); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = goast.ParseGoFile(p, "big.go", "standard")
	if err == nil {
		t.Fatal("expected size limit error")
	}
	if !strings.Contains(err.Error(), "size limit") {
		t.Errorf("error should mention size limit, got: %v", err)
	}
}

func TestParseGoFile_NoInit(t *testing.T) {
	src := `package lib

func Add(a, b int) int { return a + b }
`
	abs := writeTemp(t, "lib.go", src)
	facts, err := goast.ParseGoFile(abs, "lib.go", "standard")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.InitFunctions) != 0 {
		t.Errorf("init functions = %d, want 0", len(facts.InitFunctions))
	}
	if facts.ReflectImports {
		t.Error("should not detect reflect import")
	}
}

func TestParseGoFile_MethodInit(t *testing.T) {
	// A method named init on a receiver is NOT a package init function.
	src := `package svc

type Svc struct{}

func (s *Svc) init() {}
`
	abs := writeTemp(t, "svc.go", src)
	facts, err := goast.ParseGoFile(abs, "svc.go", "standard")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.InitFunctions) != 0 {
		t.Errorf("method init() should not count as init function, got %d", len(facts.InitFunctions))
	}
}
