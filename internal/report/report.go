// Package report renders ScanResult in the chosen output format.
//
// All renderers must produce deterministic bytes for the same input. JSON
// ordering follows schemas/output.schema.json; findings are sorted upstream
// by model.SortFindings before rendering, so renderers never re-sort.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// OutputSchemaVersion is the version of the JSON output schema we emit.
// Bump per the rules in CLAUDE.md §9.
const OutputSchemaVersion = "0.1.0"

// Format enumerates the supported renderers. Add entries here, not elsewhere.
type Format string

const (
	FormatTerminal Format = "terminal"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "md"
	FormatSARIF    Format = "sarif"
)

func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatTerminal, FormatJSON, FormatMarkdown, FormatSARIF:
		return Format(s), nil
	case "":
		return FormatTerminal, nil
	}
	return "", fmt.Errorf("unknown format %q (want terminal|json|md|sarif)", s)
}

// Render writes res to w in the chosen format.
//
// SARIF is not dispatched from here because it requires the rules that ran
// (not just the findings). Callers that need SARIF call RenderSARIF directly.
// This keeps the generic Render signature stable while still allowing the CLI
// to pick SARIF via --format=sarif.
func Render(w io.Writer, res core.ScanResult, toolVersion, profile string, f Format) error {
	switch f {
	case FormatJSON:
		return renderJSON(w, res, toolVersion, profile)
	case FormatMarkdown:
		return renderMarkdown(w, res, toolVersion, profile)
	case FormatSARIF:
		return ErrSARIFNeedsRegistry
	case FormatTerminal, "":
		return renderTerminal(w, res, toolVersion, profile)
	}
	return fmt.Errorf("unsupported format %q", f)
}

// jsonOutput tracks schemas/output.schema.json exactly. If you change field
// names here, update the schema and bump OutputSchemaVersion per §9.
type jsonOutput struct {
	SchemaVersion string          `json:"schema_version"`
	Tool          jsonTool        `json:"tool"`
	Target        jsonTarget      `json:"target"`
	Summary       jsonSummary     `json:"summary"`
	Scores        jsonScores      `json:"scores"`
	Findings      []model.Finding `json:"findings"`
	Metrics       []model.Metric  `json:"metrics"`
}

type jsonTool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type jsonTarget struct {
	Path    string   `json:"path"`
	Git     *jsonGit `json:"git,omitempty"`
	Profile string   `json:"profile,omitempty"`
}

type jsonGit struct {
	Commit string `json:"commit,omitempty"`
	Branch string `json:"branch,omitempty"`
}

type jsonSummary struct {
	RulesEvaluated int          `json:"rules_evaluated"`
	FindingsTotal  int          `json:"findings_total"`
	BySeverity     jsonSeverity `json:"by_severity"`
}

type jsonSeverity struct {
	Info     int `json:"info"`
	Warn     int `json:"warn"`
	Error    int `json:"error"`
	Critical int `json:"critical"`
}

type jsonScores struct {
	Overall     float64            `json:"overall"`
	ByPrinciple map[string]float64 `json:"by_principle"`
}

func toJSON(res core.ScanResult, toolVersion, profile string) jsonOutput {
	out := jsonOutput{
		SchemaVersion: OutputSchemaVersion,
		Tool:          jsonTool{Name: "archfit", Version: toolVersion},
		Target:        jsonTarget{Path: res.Root, Profile: profile},
		Summary: jsonSummary{
			RulesEvaluated: res.RulesEvaluated,
			FindingsTotal:  len(res.Findings),
		},
		Findings: res.Findings,
		Metrics:  res.Metrics,
	}
	for _, f := range res.Findings {
		switch f.Severity {
		case model.SeverityInfo:
			out.Summary.BySeverity.Info++
		case model.SeverityWarn:
			out.Summary.BySeverity.Warn++
		case model.SeverityError:
			out.Summary.BySeverity.Error++
		case model.SeverityCritical:
			out.Summary.BySeverity.Critical++
		}
	}
	if res.GitAvailable {
		out.Target.Git = &jsonGit{Commit: res.Git.CurrentCommit, Branch: res.Git.CurrentBranch}
	}
	out.Scores.Overall = res.Scores.Overall
	out.Scores.ByPrinciple = map[string]float64{}
	for p, v := range res.Scores.ByPrinciple {
		out.Scores.ByPrinciple[string(p)] = v
	}
	// If the resolver produced nil slices, ensure we emit [] rather than null.
	if out.Findings == nil {
		out.Findings = []model.Finding{}
	}
	if out.Metrics == nil {
		out.Metrics = []model.Metric{}
	}
	return out
}

func renderJSON(w io.Writer, res core.ScanResult, toolVersion, profile string) error {
	doc := toJSON(res, toolVersion, profile)
	// Use encoding/json with SetIndent for a canonical, diffable form.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func renderTerminal(w io.Writer, res core.ScanResult, toolVersion, profile string) error {
	fmt.Fprintf(w, "archfit %s — target %s (profile=%s)\n", toolVersion, res.Root, profile)
	fmt.Fprintf(w, "rules evaluated: %d, findings: %d\n", res.RulesEvaluated, len(res.Findings))
	fmt.Fprintf(w, "overall score: %.1f\n", res.Scores.Overall)
	if len(res.Scores.ByPrinciple) > 0 {
		keys := make([]string, 0, len(res.Scores.ByPrinciple))
		for p := range res.Scores.ByPrinciple {
			keys = append(keys, string(p))
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "  %s: %.1f\n", k, res.Scores.ByPrinciple[model.Principle(k)])
		}
	}
	if len(res.Findings) == 0 {
		fmt.Fprintln(w, "no findings")
		return nil
	}
	fmt.Fprintln(w, "findings:")
	for _, f := range res.Findings {
		path := f.Path
		if path == "" {
			path = "(repo)"
		}
		fmt.Fprintf(w, "  [%s] %s %s — %s\n", f.Severity, f.RuleID, path, f.Message)
		if f.LLMSuggestion != nil && f.LLMSuggestion.Text != "" {
			cacheTag := ""
			if f.LLMSuggestion.CacheHit {
				cacheTag = " (cached)"
			}
			fmt.Fprintf(w, "    ─ llm%s:\n", cacheTag)
			writeIndented(w, f.LLMSuggestion.Text, "      ")
		}
	}
	if len(res.Errors) > 0 {
		fmt.Fprintln(w, "rule errors:")
		for _, e := range res.Errors {
			fmt.Fprintf(w, "  %s: %s\n", e.RuleID, e.Err)
		}
	}
	return nil
}

// writeIndented prints s with the given prefix in front of every line,
// preserving line breaks. Used for multi-line LLM suggestions.
func writeIndented(w io.Writer, s, prefix string) {
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			fmt.Fprintf(w, "%s%s\n", prefix, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		fmt.Fprintf(w, "%s%s\n", prefix, s[start:])
	}
}

func renderMarkdown(w io.Writer, res core.ScanResult, toolVersion, profile string) error {
	fmt.Fprintf(w, "# archfit report\n\n")
	fmt.Fprintf(w, "- tool: archfit %s\n- target: `%s`\n- profile: `%s`\n- rules evaluated: %d\n- overall score: **%.1f**\n\n",
		toolVersion, res.Root, profile, res.RulesEvaluated, res.Scores.Overall)

	if len(res.Scores.ByPrinciple) > 0 {
		fmt.Fprintln(w, "## Score by principle")
		fmt.Fprintln(w)
		keys := make([]string, 0, len(res.Scores.ByPrinciple))
		for p := range res.Scores.ByPrinciple {
			keys = append(keys, string(p))
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "- %s: %.1f\n", k, res.Scores.ByPrinciple[model.Principle(k)])
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "## Findings")
	fmt.Fprintln(w)
	if len(res.Findings) == 0 {
		fmt.Fprintln(w, "_none_")
		return nil
	}
	fmt.Fprintln(w, "| Severity | Rule | Path | Message |")
	fmt.Fprintln(w, "|---|---|---|---|")
	for _, f := range res.Findings {
		path := f.Path
		if path == "" {
			path = "(repo)"
		}
		fmt.Fprintf(w, "| %s | %s | `%s` | %s |\n", f.Severity, f.RuleID, path, f.Message)
	}
	return nil
}
