// Package packman provides pack structure validation for archfit rule packs.
// It checks that a directory follows the required pack layout without
// importing or compiling Go code. See ADR 0006 for the design rationale.
package packman

import (
	"os"
	"path/filepath"
)

// ValidationResult holds the outcome of validating a pack directory.
type ValidationResult struct {
	// Valid is true when no required structural elements are missing.
	Valid bool `json:"valid"`
	// Errors lists missing required files or directories.
	Errors []string `json:"errors"`
	// Warnings lists missing but recommended files or directories.
	Warnings []string `json:"warnings"`
}

// ValidatePack checks that dir follows the archfit pack structure.
//
// Required files: AGENTS.md, INTENT.md
// Required: at least one .go file in the pack root
// Required dirs: resolvers/, fixtures/ (with at least one subdirectory containing input/)
// Recommended: rules/ directory, context.yaml
func ValidatePack(dir string) ValidationResult {
	var res ValidationResult

	// Required files.
	for _, name := range []string{"AGENTS.md", "INTENT.md"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			res.Errors = append(res.Errors, "missing required file: "+name)
		}
	}

	// At least one .go file in the pack root.
	if !hasGoFile(dir) {
		res.Errors = append(res.Errors, "no .go file found in pack root (expected at least one, e.g. pack.go)")
	}

	// Required directories.
	for _, name := range []string{"resolvers"} {
		path := filepath.Join(dir, name)
		if !isDir(path) {
			res.Errors = append(res.Errors, "missing required directory: "+name+"/")
		}
	}

	// fixtures/ must exist and contain at least one subdirectory with input/.
	fixturesDir := filepath.Join(dir, "fixtures")
	if !isDir(fixturesDir) {
		res.Errors = append(res.Errors, "missing required directory: fixtures/")
	} else if !hasFixtureWithInput(fixturesDir) {
		res.Errors = append(res.Errors, "fixtures/ must contain at least one subdirectory with an input/ directory")
	}

	// Recommended (warnings only).
	if !isDir(filepath.Join(dir, "rules")) {
		res.Warnings = append(res.Warnings, "missing recommended directory: rules/ (YAML rule definitions)")
	}
	if _, err := os.Stat(filepath.Join(dir, "context.yaml")); os.IsNotExist(err) {
		res.Warnings = append(res.Warnings, "missing recommended file: context.yaml")
	}

	res.Valid = len(res.Errors) == 0
	return res
}

// hasGoFile reports whether dir contains at least one .go file.
func hasGoFile(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".go" {
			return true
		}
	}
	return false
}

// isDir reports whether path exists and is a directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// hasFixtureWithInput reports whether fixturesDir contains at least one
// subdirectory that itself contains an input/ directory.
func hasFixtureWithInput(fixturesDir string) bool {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			inputDir := filepath.Join(fixturesDir, e.Name(), "input")
			if isDir(inputDir) {
				return true
			}
		}
	}
	return false
}
