package schema_test

import (
	"os"
	"path/filepath"
	"testing"

	collectfs "github.com/shibuiwilliam/archfit/internal/collector/fs"
	"github.com/shibuiwilliam/archfit/internal/collector/schema"
)

func TestCollect_IDExtractionAndParseError(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("schemas/a.schema.json", `{"$id":"https://example.com/a","type":"object"}`)
	write("schemas/b.schema.json", `{"type":"object"}`) // no $id
	write("schemas/broken.schema.json", `{ not valid json`)
	write("other/not-a.schema.json", `{}`) // ignored (wrong directory)

	repo, err := collectfs.Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := schema.Collect(repo)
	if len(got.Files) != 3 {
		t.Fatalf("want 3 schema entries, got %d: %+v", len(got.Files), got.Files)
	}
	byPath := map[string]string{}
	for _, f := range got.Files {
		if f.ParseError != "" {
			byPath[f.Path] = "ERR:" + f.ParseError
		} else {
			byPath[f.Path] = "ID:" + f.ID
		}
	}
	if byPath["schemas/a.schema.json"] != "ID:https://example.com/a" {
		t.Errorf("a: %v", byPath["schemas/a.schema.json"])
	}
	if byPath["schemas/b.schema.json"] != "ID:" {
		t.Errorf("b should have empty id, got: %v", byPath["schemas/b.schema.json"])
	}
	if v, ok := byPath["schemas/broken.schema.json"]; !ok || v == "ID:" {
		t.Errorf("broken should carry ParseError: %v", byPath["schemas/broken.schema.json"])
	}
}
