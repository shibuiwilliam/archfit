// Package schema collects JSON-Schema files from the repository.
//
// Runs after the filesystem collector — it takes RepoFacts as input, selects
// candidate files (*.schema.json under schemas/), reads them, and extracts
// only the minimum agent-facing fields (the top-level "$id"). Parse errors
// are surfaced as SchemaFile.ParseError so resolvers can emit a
// ParseFailure finding, per CLAUDE.md §13.
package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// MaxBytes is the size cap on a single schema file. Above this the file is
// marked as a parse error rather than loaded into memory.
const MaxBytes = 1 << 20 // 1 MiB

// Collect scans repo for JSON Schema files and returns the parsed facts.
func Collect(repo model.RepoFacts) model.SchemaFacts {
	var facts model.SchemaFacts
	for _, f := range repo.Files {
		if !isSchemaCandidate(f.Path) {
			continue
		}
		entry := model.SchemaFile{Path: f.Path}
		if f.Size > MaxBytes {
			entry.ParseError = "schema file larger than 1 MiB — skipped"
			facts.Files = append(facts.Files, entry)
			continue
		}
		abs := filepath.Join(repo.Root, f.Path)
		data, err := os.ReadFile(abs)
		if err != nil {
			entry.ParseError = err.Error()
			facts.Files = append(facts.Files, entry)
			continue
		}
		var top map[string]any
		if err := json.Unmarshal(data, &top); err != nil {
			entry.ParseError = "invalid JSON: " + err.Error()
			facts.Files = append(facts.Files, entry)
			continue
		}
		if id, ok := top["$id"].(string); ok {
			entry.ID = id
		}
		facts.Files = append(facts.Files, entry)
	}
	return facts
}

func isSchemaCandidate(p string) bool {
	if !strings.HasPrefix(p, "schemas/") {
		return false
	}
	return strings.HasSuffix(p, ".schema.json")
}
