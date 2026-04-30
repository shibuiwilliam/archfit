//go:build tools

// genrules reads packs/<pack>/rules/*.yaml, validates each against
// schemas/rule.schema.json, and emits packs/<pack>/generated_rules.go
// with a GeneratedRules() function. Resolvers are left nil — pack.go
// wires them by rule ID.
//
// Usage: go run -tags tools ./cmd/internal-tools/genrules [packs-dir]
//
// CLAUDE.md §4 requires that generation is an explicit `make generate`
// step with committed output — never implicit during `go build`.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"sigs.k8s.io/yaml"
)

// ruleYAML mirrors the YAML rule structure — just enough for code generation.
type ruleYAML struct {
	ID               string       `json:"id"`
	Principle        string       `json:"principle"`
	Dimension        string       `json:"dimension"`
	Title            string       `json:"title"`
	Severity         string       `json:"severity"`
	EvidenceStrength string       `json:"evidence_strength"`
	Stability        string       `json:"stability"`
	Weight           float64      `json:"weight"`
	Rationale        string       `json:"rationale"`
	Remediation      remYAML      `json:"remediation"`
	AppliesTo        *appliesToYAML `json:"applies_to,omitempty"`
}

type remYAML struct {
	Summary     string `json:"summary"`
	GuideRef    string `json:"guide_ref,omitempty"`
	AutoFixable bool   `json:"auto_fixable,omitempty"`
}

type appliesToYAML struct {
	ProjectTypes []string `json:"project_types,omitempty"`
	Languages    []string `json:"languages,omitempty"`
	PathGlobs    []string `json:"path_globs,omitempty"`
}

var principleConst = map[string]string{
	"P1": "model.P1Locality",
	"P2": "model.P2SpecFirst",
	"P3": "model.P3ShallowExplicitness",
	"P4": "model.P4Verifiability",
	"P5": "model.P5Aggregation",
	"P6": "model.P6Reversibility",
	"P7": "model.P7MachineReadability",
}

var severityConst = map[string]string{
	"info":     "model.SeverityInfo",
	"warn":     "model.SeverityWarn",
	"error":    "model.SeverityError",
	"critical": "model.SeverityCritical",
}

var evidenceConst = map[string]string{
	"strong":  "model.EvidenceStrong",
	"medium":  "model.EvidenceMedium",
	"weak":    "model.EvidenceWeak",
	"sampled": "model.EvidenceSampled",
}

var stabilityConst = map[string]string{
	"experimental": "model.StabilityExperimental",
	"stable":       "model.StabilityStable",
	"deprecated":   "model.StabilityDeprecated",
}

func main() {
	packsDir := "packs"
	if len(os.Args) > 1 {
		packsDir = os.Args[1]
	}

	schemaPath, err := filepath.Abs(filepath.Join("schemas", "rule.schema.json"))
	if err != nil {
		fatal("resolve schema path: %v", err)
	}
	schema, err := jsonschema.NewCompiler().Compile(schemaPath)
	if err != nil {
		fatal("compile rule schema: %v", err)
	}

	entries, err := os.ReadDir(packsDir)
	if err != nil {
		fatal("read packs dir: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		packName := e.Name()
		rulesDir := filepath.Join(packsDir, packName, "rules")
		if _, serr := os.Stat(rulesDir); serr != nil {
			continue
		}
		rules, rerr := loadRules(rulesDir, schema)
		if rerr != nil {
			fatal("pack %s: %v", packName, rerr)
		}
		if len(rules) == 0 {
			continue
		}

		goSrc, gerr := generateGo(packName, rules)
		if gerr != nil {
			fatal("pack %s: generate: %v", packName, gerr)
		}

		outPath := filepath.Join(packsDir, packName, "generated_rules.go")
		if werr := os.WriteFile(outPath, goSrc, 0o644); werr != nil {
			fatal("write %s: %v", outPath, werr)
		}
		fmt.Printf("wrote %s (%d rules)\n", outPath, len(rules))
	}
}

func loadRules(dir string, schema *jsonschema.Schema) ([]ruleYAML, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	var rules []ruleYAML
	for _, f := range files {
		data, rerr := os.ReadFile(f)
		if rerr != nil {
			return nil, fmt.Errorf("read %s: %w", f, rerr)
		}

		// Validate against the JSON Schema. sigs.k8s.io/yaml converts to
		// JSON under the hood, but we need to unmarshal to any for the
		// schema validator.
		var raw any
		if uerr := yaml.Unmarshal(data, &raw); uerr != nil {
			return nil, fmt.Errorf("%s: %w", f, uerr)
		}
		if verr := schema.Validate(raw); verr != nil {
			return nil, fmt.Errorf("%s: schema validation: %w", f, verr)
		}

		var r ruleYAML
		if uerr := yaml.UnmarshalStrict(data, &r); uerr != nil {
			return nil, fmt.Errorf("%s: %w", f, uerr)
		}
		rules = append(rules, r)
	}

	return rules, nil
}

var goTmpl = template.Must(template.New("rules").Funcs(template.FuncMap{
	"gostr":      gostr,
	"gostrslice": gostrslice,
	"principle":  func(s string) string { return principleConst[s] },
	"severity":   func(s string) string { return severityConst[s] },
	"evidence":   func(s string) string { return evidenceConst[s] },
	"stability":  func(s string) string { return stabilityConst[s] },
	"hasAppliesTo": func(a *appliesToYAML) bool {
		return a != nil && len(a.Languages) > 0
	},
}).Parse(`// Code generated by cmd/internal-tools/genrules; DO NOT EDIT.
// Source: packs/{{ .Pack }}/rules/*.yaml

package {{ .Pkg }}

import "github.com/shibuiwilliam/archfit/internal/model"

// GeneratedRules returns rule metadata loaded from YAML. Resolver fields are
// nil — pack.go wires them by ID.
func GeneratedRules() []model.Rule {
	return []model.Rule{
{{- range .Rules }}
		{
			ID:               {{ gostr .ID }},
			Principle:        {{ principle .Principle }},
			Dimension:        {{ gostr .Dimension }},
			Title:            {{ gostr .Title }},
			Severity:         {{ severity .Severity }},
			EvidenceStrength: {{ evidence .EvidenceStrength }},
			Stability:        {{ stability .Stability }},
			Weight:           {{ .Weight }},
			Rationale:        {{ gostr .Rationale }},
			Remediation: model.Remediation{
				Summary:     {{ gostr .Remediation.Summary }},
				GuideRef:    {{ gostr .Remediation.GuideRef }},
				AutoFixable: {{ .Remediation.AutoFixable }},
			},
{{- if hasAppliesTo .AppliesTo }}
			AppliesTo: model.Applicability{
				Languages: {{ gostrslice .AppliesTo.Languages }},
			},
{{- end }}
		},
{{- end }}
	}
}
`))

func generateGo(packName string, rules []ruleYAML) ([]byte, error) {
	// Go package name: convert "agent-tool" → "agenttool".
	pkg := strings.ReplaceAll(packName, "-", "")

	var buf bytes.Buffer
	if err := goTmpl.Execute(&buf, struct {
		Pack  string
		Pkg   string
		Rules []ruleYAML
	}{Pack: packName, Pkg: pkg, Rules: rules}); err != nil {
		return nil, err
	}

	// gofmt the output for consistent formatting.
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w\n%s", err, buf.String())
	}
	return formatted, nil
}

// gostrslice returns a Go []string literal.
func gostrslice(ss []string) string {
	if len(ss) == 0 {
		return "nil"
	}
	var parts []string
	for _, s := range ss {
		parts = append(parts, fmt.Sprintf("%q", s))
	}
	return "[]string{" + strings.Join(parts, ", ") + "}"
}

// gostr returns a Go string literal, using backtick for multi-line strings.
func gostr(s string) string {
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, "\n\r`") {
		// Fall back to quoted string with newlines escaped.
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("%q", s)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "genrules: "+format+"\n", args...)
	os.Exit(1)
}
