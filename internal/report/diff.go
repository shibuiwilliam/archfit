package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// BaselineDoc is a subset of the JSON output we need to do a diff. It must
// remain compatible with older JSON versions — only require fields strictly
// needed to identify a finding.
type BaselineDoc struct {
	SchemaVersion string          `json:"schema_version"`
	Findings      []model.Finding `json:"findings"`
}

// DiffResult is the structured outcome of comparing a baseline to a current scan.
type DiffResult struct {
	New       []model.Finding `json:"new"`
	Fixed     []model.Finding `json:"fixed"`
	Unchanged []model.Finding `json:"unchanged"`
}

// LoadBaseline parses a JSON document produced by `archfit scan --json`.
func LoadBaseline(data []byte) (BaselineDoc, error) {
	var doc BaselineDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return doc, fmt.Errorf("baseline: %w", err)
	}
	return doc, nil
}

// Diff compares baseline to current by (rule_id, path, message). Two findings
// are equivalent when all three match — a stable identity without needing a
// synthetic UUID and without being sensitive to non-identifying fields like
// confidence or evidence detail.
func Diff(baseline, current []model.Finding) DiffResult {
	key := func(f model.Finding) string {
		return f.RuleID + "\x00" + f.Path + "\x00" + f.Message
	}
	prev := map[string]model.Finding{}
	for _, f := range baseline {
		prev[key(f)] = f
	}
	var res DiffResult
	curKeys := map[string]bool{}
	for _, f := range current {
		k := key(f)
		curKeys[k] = true
		if _, ok := prev[k]; ok {
			res.Unchanged = append(res.Unchanged, f)
		} else {
			res.New = append(res.New, f)
		}
	}
	for k, f := range prev {
		if !curKeys[k] {
			res.Fixed = append(res.Fixed, f)
		}
	}
	model.SortFindings(res.New)
	model.SortFindings(res.Fixed)
	model.SortFindings(res.Unchanged)
	return res
}

// RenderDiffJSON writes DiffResult as deterministic JSON.
func RenderDiffJSON(w io.Writer, d DiffResult) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	// Ensure non-nil slices for consistent JSON shape.
	if d.New == nil {
		d.New = []model.Finding{}
	}
	if d.Fixed == nil {
		d.Fixed = []model.Finding{}
	}
	if d.Unchanged == nil {
		d.Unchanged = []model.Finding{}
	}
	if err := enc.Encode(map[string]any{
		"new":       d.New,
		"fixed":     d.Fixed,
		"unchanged": d.Unchanged,
		"summary": map[string]int{
			"new":       len(d.New),
			"fixed":     len(d.Fixed),
			"unchanged": len(d.Unchanged),
		},
	}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// RenderDiffTerminal is the human-readable form: grouped, sorted, with counts.
func RenderDiffTerminal(w io.Writer, d DiffResult) error {
	fmt.Fprintf(w, "new: %d   fixed: %d   unchanged: %d\n\n", len(d.New), len(d.Fixed), len(d.Unchanged))
	if len(d.New) > 0 {
		fmt.Fprintln(w, "NEW:")
		printDiffFindings(w, d.New)
		fmt.Fprintln(w)
	}
	if len(d.Fixed) > 0 {
		fmt.Fprintln(w, "FIXED:")
		printDiffFindings(w, d.Fixed)
		fmt.Fprintln(w)
	}
	if len(d.Unchanged) > 0 {
		fmt.Fprintln(w, "UNCHANGED:")
		printDiffFindings(w, d.Unchanged)
	}
	return nil
}

func printDiffFindings(w io.Writer, fs []model.Finding) {
	sorted := append([]model.Finding(nil), fs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if si, sj := sorted[i].Severity.Rank(), sorted[j].Severity.Rank(); si != sj {
			return si > sj
		}
		if sorted[i].RuleID != sorted[j].RuleID {
			return sorted[i].RuleID < sorted[j].RuleID
		}
		return sorted[i].Path < sorted[j].Path
	})
	for _, f := range sorted {
		path := f.Path
		if path == "" {
			path = "(repo)"
		}
		fmt.Fprintf(w, "  [%s] %s %s — %s\n", f.Severity, f.RuleID, path, f.Message)
	}
}
