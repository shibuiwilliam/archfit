package packman

import (
	"os"
	"path/filepath"
	"testing"
)

// setupValidPack creates a temporary directory with all required pack files.
func setupValidPack(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Required files.
	writeFile(t, filepath.Join(dir, "AGENTS.md"), "# Agents\n")
	writeFile(t, filepath.Join(dir, "INTENT.md"), "# Intent\n")
	writeFile(t, filepath.Join(dir, "pack.go"), "package mypack\n")

	// Required directories.
	mkdirAll(t, filepath.Join(dir, "resolvers"))
	mkdirAll(t, filepath.Join(dir, "fixtures", "test-rule", "input"))

	// Recommended (optional) items.
	mkdirAll(t, filepath.Join(dir, "rules"))
	writeFile(t, filepath.Join(dir, "context.yaml"), "name: mypack\n")

	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePack_ValidFull(t *testing.T) {
	dir := setupValidPack(t)
	res := ValidatePack(dir)

	if !res.Valid {
		t.Errorf("expected Valid=true, got errors: %v", res.Errors)
	}
	if len(res.Errors) != 0 {
		t.Errorf("expected no errors, got: %v", res.Errors)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("expected no warnings for full pack, got: %v", res.Warnings)
	}
}

func TestValidatePack_MissingAgentsMD(t *testing.T) {
	dir := setupValidPack(t)
	_ = os.Remove(filepath.Join(dir, "AGENTS.md"))

	res := ValidatePack(dir)

	if res.Valid {
		t.Error("expected Valid=false when AGENTS.md is missing")
	}
	assertContains(t, res.Errors, "AGENTS.md")
}

func TestValidatePack_MissingFixtures(t *testing.T) {
	dir := setupValidPack(t)
	_ = os.RemoveAll(filepath.Join(dir, "fixtures"))

	res := ValidatePack(dir)

	if res.Valid {
		t.Error("expected Valid=false when fixtures/ is missing")
	}
	assertContains(t, res.Errors, "fixtures")
}

func TestValidatePack_FixturesWithoutInput(t *testing.T) {
	dir := setupValidPack(t)
	// Remove the input/ subdirectory but keep fixtures/ and its child.
	_ = os.RemoveAll(filepath.Join(dir, "fixtures", "test-rule", "input"))

	res := ValidatePack(dir)

	if res.Valid {
		t.Error("expected Valid=false when fixtures/ has no subdirectory with input/")
	}
	assertContains(t, res.Errors, "input")
}

func TestValidatePack_MinimalPack(t *testing.T) {
	// Only required items, no optional ones.
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "AGENTS.md"), "# Agents\n")
	writeFile(t, filepath.Join(dir, "INTENT.md"), "# Intent\n")
	writeFile(t, filepath.Join(dir, "pack.go"), "package mypack\n")
	mkdirAll(t, filepath.Join(dir, "resolvers"))
	mkdirAll(t, filepath.Join(dir, "fixtures", "rule-001", "input"))

	res := ValidatePack(dir)

	if !res.Valid {
		t.Errorf("expected Valid=true for minimal pack, got errors: %v", res.Errors)
	}
	if len(res.Warnings) == 0 {
		t.Error("expected warnings about missing optional items (rules/, context.yaml)")
	}
	assertContains(t, res.Warnings, "rules/")
	assertContains(t, res.Warnings, "context.yaml")
}

func TestValidatePack_MissingGoFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "AGENTS.md"), "# Agents\n")
	writeFile(t, filepath.Join(dir, "INTENT.md"), "# Intent\n")
	mkdirAll(t, filepath.Join(dir, "resolvers"))
	mkdirAll(t, filepath.Join(dir, "fixtures", "rule-001", "input"))

	res := ValidatePack(dir)

	if res.Valid {
		t.Error("expected Valid=false when no .go file exists")
	}
	assertContains(t, res.Errors, ".go")
}

func TestValidatePack_MissingResolvers(t *testing.T) {
	dir := setupValidPack(t)
	_ = os.RemoveAll(filepath.Join(dir, "resolvers"))

	res := ValidatePack(dir)

	if res.Valid {
		t.Error("expected Valid=false when resolvers/ is missing")
	}
	assertContains(t, res.Errors, "resolvers")
}

// assertContains checks that at least one string in ss contains substr.
func assertContains(t *testing.T, ss []string, substr string) {
	t.Helper()
	for _, s := range ss {
		if contains(s, substr) {
			return
		}
	}
	t.Errorf("expected one of %v to contain %q", ss, substr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
