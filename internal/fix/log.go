package fix

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	adaptfs "github.com/shibuiwilliam/archfit/internal/adapter/fs"
)

// DefaultLogPath is the default location for the fix audit log.
const DefaultLogPath = ".archfit-fix-log.json"

// LogEntry records a single fix attempt for audit.
type LogEntry struct {
	Timestamp string   `json:"timestamp"`
	RuleID    string   `json:"rule_id"`
	Action    string   `json:"action"`   // "applied", "rolled_back", "skipped"
	Files     []string `json:"files"`    // paths touched
	Verified  bool     `json:"verified"` // true only if re-scan confirmed fix
	Error     string   `json:"error,omitempty"`
}

// AppendLog appends a single log entry to the audit file as a JSON line.
// Uses the real filesystem. For engine-internal use, see AppendLogFS.
func AppendLog(path string, entry LogEntry) error {
	return AppendLogFS(adaptfs.NewReal(), path, entry)
}

// AppendLogFS appends a log entry using the given filesystem adapter.
func AppendLogFS(fsys adaptfs.FS, path string, entry LogEntry) error {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("fix log: marshal: %w", err)
	}
	data = append(data, '\n')

	// Try OpenFile for append (real FS). If it fails (memory FS), fall back
	// to read-modify-write via WriteFile.
	f, openErr := fsys.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if openErr == nil {
		defer func() { _ = f.Close() }()
		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("fix log: write: %w", err)
		}
		return nil
	}

	// Fallback for memory FS: read existing + append + write back.
	existing, _ := fsys.ReadFile(path) // ignore not-found
	combined := make([]byte, len(existing)+len(data))
	copy(combined, existing)
	copy(combined[len(existing):], data)
	if err := fsys.WriteFile(path, combined, 0o644); err != nil {
		return fmt.Errorf("fix log: write: %w", err)
	}
	return nil
}

// LoadLog reads all entries from the audit file.
func LoadLog(path string) ([]LogEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("fix log: read: %w", err)
	}

	var entries []LogEntry
	dec := json.NewDecoder(strings.NewReader(string(data)))
	for dec.More() {
		var e LogEntry
		if err := dec.Decode(&e); err != nil {
			return entries, fmt.Errorf("fix log: decode: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}
