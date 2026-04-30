// Schema conformance test — validates every golden expected.json file against
// schemas/output.schema.json. This guarantees that the renderer and the schema
// stay in sync: if anyone adds a field to the renderer without updating the
// schema (or vice versa), this test fails.
//
// Uses github.com/santhosh-tekuri/jsonschema/v6 — a pure-Go JSON Schema
// validator. Justified in docs/dependencies.md; the dependency is test-only
// in practice (no production import), but go.mod lists it unconditionally
// because Go modules do not distinguish test vs non-test deps.
package report_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// schemaPath returns the absolute path to schemas/output.schema.json,
// relative to this test file's location in internal/report/.
func schemaPath(t *testing.T) string {
	t.Helper()
	// internal/report/ → ../../schemas/output.schema.json
	p, err := filepath.Abs(filepath.Join("..", "..", "schemas", "output.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("schema not found: %s", p)
	}
	return p
}

// compileSchema compiles the output schema for reuse across subtests.
func compileSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	c := jsonschema.NewCompiler()
	sch, err := c.Compile(schemaPath(t))
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

// collectExpectedJSON walks root and returns all paths named "expected.json".
func collectExpectedJSON(t *testing.T, root string) []string {
	t.Helper()
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "expected.json" && !info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

// TestOutputSchema_E2EFixtures validates every testdata/e2e/**/expected.json
// against schemas/output.schema.json.
func TestOutputSchema_E2EFixtures(t *testing.T) {
	sch := compileSchema(t)
	root := filepath.Join("..", "..", "testdata", "e2e")
	files := collectExpectedJSON(t, root)
	if len(files) == 0 {
		t.Fatal("no expected.json files found under testdata/e2e/")
	}

	for _, path := range files {
		rel, _ := filepath.Rel(filepath.Join("..", ".."), path)
		t.Run(rel, func(t *testing.T) {
			validateFile(t, sch, path)
		})
	}
}

// TestOutputSchema_PackFixtures validates every packs/**/fixtures/**/expected.json
// against schemas/output.schema.json.
func TestOutputSchema_PackFixtures(t *testing.T) {
	sch := compileSchema(t)
	root := filepath.Join("..", "..", "packs")
	files := collectExpectedJSON(t, root)
	// Pack fixtures may use a subset schema (per-rule findings only, not full
	// scan output). Skip files that are not full scan output documents.
	var fullOutputFiles []string
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		// Full output documents contain "schema_version". Per-rule fixture
		// expected.json files are arrays or objects without that key.
		if strings.Contains(string(data), `"schema_version"`) {
			fullOutputFiles = append(fullOutputFiles, path)
		}
	}
	if len(fullOutputFiles) == 0 {
		t.Skip("no full-output expected.json files found under packs/")
	}

	for _, path := range fullOutputFiles {
		rel, _ := filepath.Rel(filepath.Join("..", ".."), path)
		t.Run(rel, func(t *testing.T) {
			validateFile(t, sch, path)
		})
	}
}

// TestOutputSchema_RequiresRulesWithFindings confirms the schema rejects output
// missing the rules_with_findings field. This prevents silent regression if
// someone removes the field from the renderer.
func TestOutputSchema_RequiresRulesWithFindings(t *testing.T) {
	sch := compileSchema(t)

	// Minimal valid document missing rules_with_findings.
	doc := map[string]any{
		"schema_version": "0.2.0",
		"tool":           map[string]any{"name": "archfit", "version": "test"},
		"target":         map[string]any{"path": "."},
		"summary": map[string]any{
			"rules_evaluated": 1,
			// rules_with_findings intentionally omitted
			"findings_total": 0,
			"by_severity":    map[string]any{"info": 0, "warn": 0, "error": 0, "critical": 0},
		},
		"scores": map[string]any{
			"overall":      100.0,
			"by_principle": map[string]any{"P1": 100.0},
		},
		"findings": []any{},
		"metrics":  []any{},
	}

	if err := sch.Validate(doc); err == nil {
		t.Fatal("schema should reject output missing rules_with_findings")
	}
}

func validateFile(t *testing.T, sch *jsonschema.Schema, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if err := sch.Validate(doc); err != nil {
		t.Errorf("schema validation failed for %s:\n%v", path, err)
	}
}
