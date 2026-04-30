// End-to-end golden tests.
//
// Each subdirectory under testdata/e2e/ (except this file and "update") is a
// fixture with an `input/` directory and an `expected.json` file. The test
// runs `core.Scan` on input/, canonicalizes the JSON output, and diffs it
// byte-for-byte against expected.json.
//
// To regenerate after an intentional output change:
//
//	go test ./testdata/e2e -update
package e2e_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/report"
	"github.com/shibuiwilliam/archfit/internal/rule"
	agenttool "github.com/shibuiwilliam/archfit/packs/agent-tool"
	corepack "github.com/shibuiwilliam/archfit/packs/core"
)

// update regenerates expected.json files when set.
// Invoke with: go test ./testdata/e2e -update
var update = flag.Bool("update", false, "regenerate expected.json files")

func buildRegistry(t *testing.T) *rule.Registry {
	t.Helper()
	reg := rule.NewRegistry()
	if err := corepack.Register(reg); err != nil {
		t.Fatal(err)
	}
	if err := agenttool.Register(reg); err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestE2E_GoldenJSON(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		fixtureDir := e.Name()
		input := filepath.Join(fixtureDir, "input")
		if _, err := os.Stat(input); err != nil {
			continue
		}
		t.Run(fixtureDir, func(t *testing.T) {
			reg := buildRegistry(t)
			res, err := core.Scan(context.Background(), core.ScanInput{
				Root:  input,
				Rules: reg.Rules(),
				// No runner: git facts are intentionally unavailable so
				// e2e output does not depend on the caller's git state.
			})
			if err != nil {
				t.Fatal(err)
			}

			// Canonicalize: pin the root to a fixed value so golden output
			// is path-independent, and render to JSON deterministically.
			res.Root = "<fixture>"
			var buf bytes.Buffer
			if err := report.Render(&buf, res, "e2e-golden", "standard", report.FormatJSON); err != nil {
				t.Fatal(err)
			}

			expectedPath := filepath.Join(fixtureDir, "expected.json")
			if *update {
				if err := os.WriteFile(expectedPath, buf.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("updated %s", expectedPath)
				return
			}
			want, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("read %s: %v (run with -update to create)", expectedPath, err)
			}
			if !bytes.Equal(bytes.TrimRight(buf.Bytes(), "\n"), bytes.TrimRight(want, "\n")) {
				t.Errorf("e2e output drift in %s.\nrun: go test ./testdata/e2e -update\ndiff (got vs want, first 50 lines):\n%s",
					fixtureDir, firstLines(diffPreview(buf.String(), string(want)), 50))
			}
		})
	}
}

func diffPreview(got, want string) string {
	var b strings.Builder
	b.WriteString("--- got\n+++ want\n")
	gotLines := strings.Split(got, "\n")
	wantLines := strings.Split(want, "\n")
	total := len(gotLines)
	if len(wantLines) > total {
		total = len(wantLines)
	}
	for i := 0; i < total; i++ {
		var g, w string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g != w {
			if g != "" {
				b.WriteString("- " + g + "\n")
			}
			if w != "" {
				b.WriteString("+ " + w + "\n")
			}
		}
	}
	return b.String()
}

func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
