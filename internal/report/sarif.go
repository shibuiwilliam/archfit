// SARIF 2.1.0 renderer.
//
// The goal is a document that GitHub Code Scanning accepts — not the full
// SARIF spec surface, which is large. We emit only the fields that are
// required or meaningfully consumed by common SARIF readers:
//
//   - $schema and version
//   - runs[].tool.driver: name, version, informationUri, rules[]
//   - runs[].results[]: ruleId, level, message.text, locations[], properties.evidence
//
// SARIF spec: https://docs.oasis-open.org/sarif/sarif/v2.1.0/
// GitHub's subset: https://docs.github.com/en/code-security/code-scanning/integrating-with-code-scanning/sarif-support-for-code-scanning
//
// Stability: breaking changes to this output are treated as breaking changes to
// archfit's public API. Add fields freely; never silently rename or remove.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/shibuiwilliam/archfit/internal/core"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// SARIFInformationURI is the link GitHub shows for the rule row header.
const SARIFInformationURI = "https://github.com/shibuiwilliam/archfit"

type sarifDoc struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name,omitempty"`
	ShortDescription sarifText              `json:"shortDescription"`
	FullDescription  sarifText              `json:"fullDescription,omitempty"`
	Help             sarifText              `json:"help,omitempty"`
	DefaultConfig    sarifConfig            `json:"defaultConfiguration"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID     string                 `json:"ruleId"`
	Level      string                 `json:"level"`
	Message    sarifText              `json:"message"`
	Locations  []sarifLocation        `json:"locations,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

// severityToSARIFLevel maps archfit severities to SARIF's level enum.
// SARIF has four values (none/note/warning/error); we do not emit none.
func severityToSARIFLevel(s model.Severity) string {
	switch s {
	case model.SeverityInfo:
		return "note"
	case model.SeverityWarn:
		return "warning"
	case model.SeverityError, model.SeverityCritical:
		return "error"
	}
	return "warning"
}

// RenderSARIF writes a SARIF 2.1.0 document to w. The rules slice is the set
// of rules that ran (not the full registry) — callers pass this in so the
// SARIF output reflects what was actually evaluated.
func RenderSARIF(w io.Writer, res core.ScanResult, rules []model.Rule, toolVersion string) error {
	doc := sarifDoc{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:           "archfit",
					Version:        toolVersion,
					InformationURI: SARIFInformationURI,
					Rules:          make([]sarifRule, 0, len(rules)),
				},
			},
			Results: make([]sarifResult, 0, len(res.Findings)),
		}},
	}

	for _, r := range rules {
		doc.Runs[0].Tool.Driver.Rules = append(doc.Runs[0].Tool.Driver.Rules, sarifRule{
			ID:               r.ID,
			Name:             r.Title,
			ShortDescription: sarifText{Text: r.Title},
			FullDescription:  sarifText{Text: r.Rationale},
			Help:             sarifText{Text: r.Remediation.Summary},
			DefaultConfig:    sarifConfig{Level: severityToSARIFLevel(r.Severity)},
			Properties: map[string]interface{}{
				"principle":         string(r.Principle),
				"dimension":         r.Dimension,
				"evidence_strength": string(r.EvidenceStrength),
				"stability":         string(r.Stability),
			},
		})
	}

	for _, f := range res.Findings {
		result := sarifResult{
			RuleID:  f.RuleID,
			Level:   severityToSARIFLevel(f.Severity),
			Message: sarifText{Text: f.Message},
			Properties: map[string]interface{}{
				"principle":         string(f.Principle),
				"evidence_strength": string(f.EvidenceStrength),
				"confidence":        f.Confidence,
				"evidence":          f.Evidence,
				"remediation":       f.Remediation,
			},
		}
		// SARIF requires at least one location per result in strict mode; we
		// emit a synthetic repo-root location when the finding has no path.
		uri := f.Path
		if uri == "" {
			uri = "."
		}
		result.Locations = []sarifLocation{{
			PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: uri},
			},
		}}
		doc.Runs[0].Results = append(doc.Runs[0].Results, result)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

// ErrSARIFNeedsRegistry is returned by Render when asked to emit SARIF without
// the registry of rules in scope. The CLI supplies rules via RenderSARIF directly.
var ErrSARIFNeedsRegistry = fmt.Errorf("SARIF renderer requires the list of evaluated rules; use RenderSARIF")
