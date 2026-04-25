package fix_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shibuiwilliam/archfit/internal/fix"
)

func TestAppendLog_CreatesFileAndAppendsEntries(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "fix-log.json")

	e1 := fix.LogEntry{
		Timestamp: "2026-04-25T10:00:00Z",
		RuleID:    "P1.LOC.001",
		Action:    "applied",
		Files:     []string{"CLAUDE.md"},
		Verified:  true,
	}
	e2 := fix.LogEntry{
		Timestamp: "2026-04-25T10:01:00Z",
		RuleID:    "P7.MRD.002",
		Action:    "rolled_back",
		Files:     []string{"CHANGELOG.md"},
		Verified:  false,
		Error:     "new finding introduced",
	}

	if err := fix.AppendLog(logPath, e1); err != nil {
		t.Fatal(err)
	}
	if err := fix.AppendLog(logPath, e2); err != nil {
		t.Fatal(err)
	}

	entries, err := fix.LoadLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].RuleID != "P1.LOC.001" || !entries[0].Verified {
		t.Errorf("entry 0 mismatch: %+v", entries[0])
	}
	if entries[1].RuleID != "P7.MRD.002" || entries[1].Verified {
		t.Errorf("entry 1 mismatch: %+v", entries[1])
	}
	if entries[1].Error != "new finding introduced" {
		t.Errorf("entry 1 error mismatch: %q", entries[1].Error)
	}
}

func TestLoadLog_NonexistentFileReturnsNil(t *testing.T) {
	entries, err := fix.LoadLog("/nonexistent/path/fix-log.json")
	if err != nil {
		t.Fatal(err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for nonexistent file, got %v", entries)
	}
}

func TestAppendLog_DefaultTimestamp(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "fix-log.json")

	// Empty timestamp should be auto-filled.
	if err := fix.AppendLog(logPath, fix.LogEntry{RuleID: "P1.LOC.001", Action: "applied"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	// Should contain a valid RFC3339 timestamp (starts with 20).
	if len(data) < 20 {
		t.Fatalf("log entry too short: %s", data)
	}

	entries, err := fix.LoadLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Timestamp == "" {
		t.Errorf("expected auto-filled timestamp, got %+v", entries)
	}
}
