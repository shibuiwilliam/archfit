// Package config loads .archfit.yaml (or .archfit.json) from the target repo.
//
// Phase 1 design decision: we parse the config as JSON. YAML 1.2 is a strict
// superset of JSON, so a JSON document written into .archfit.yaml is a valid
// YAML document — any YAML-aware tooling round-trips it. Full YAML support
// (anchors, block scalars, unquoted strings) arrives in Phase 2 with yaml.v3.
// Keeping Phase 1 dependency-free is worth the temporary restriction.
//
// DEVELOPMENT_PLAN.md Phase 2 removes this restriction.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Config is the deserialized .archfit.yaml configuration.
type Config struct {
	Version     int                       `json:"version"`
	ProjectType []string                  `json:"project_type,omitempty"`
	Profile     string                    `json:"profile,omitempty"`
	RiskTiers   map[string][]string       `json:"risk_tiers,omitempty"`
	Packs       Packs                     `json:"packs"`
	Overrides   map[string]map[string]any `json:"overrides,omitempty"`
	Ignore      []Ignore                  `json:"ignore,omitempty"`
}

// Packs controls which rule packs are enabled or disabled.
type Packs struct {
	Enabled  []string `json:"enabled,omitempty"`
	Disabled []string `json:"disabled,omitempty"`
}

// Ignore suppresses a rule with a mandatory reason and expiry date.
type Ignore struct {
	Rule    string   `json:"rule"`
	Paths   []string `json:"paths,omitempty"`
	Reason  string   `json:"reason"`
	Expires string   `json:"expires"`
}

// Default returns the default configuration applied when no .archfit.yaml is present.
func Default() Config {
	return Config{
		Version: 1,
		Profile: "standard",
		Packs:   Packs{Enabled: []string{"core"}},
	}
}

// Candidate filenames, in priority order.
var candidates = []string{".archfit.yaml", ".archfit.yml", ".archfit.json"}

// LoadFile reads a configuration from an explicit file path. Use this when the
// caller specifies --config; use Load for the default discovery behaviour.
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	cfg, err := ParseJSON(data)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Load reads a configuration from root, returning Default() with a descriptive
// warning path when none is present. The boolean reports whether a file was found.
func Load(root string) (cfg Config, path string, found bool, err error) {
	for _, name := range candidates {
		p := filepath.Join(root, name)
		data, err := os.ReadFile(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return Config{}, p, true, err
		}
		cfg, err := ParseJSON(data)
		if err != nil {
			return Config{}, p, true, fmt.Errorf("%s: %w", p, err)
		}
		return cfg, p, true, nil
	}
	return Default(), "", false, nil
}

// ParseJSON validates and decodes a JSON config document.
func ParseJSON(data []byte) (Config, error) {
	var raw Config
	dec := json.NewDecoder(newTrimReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return Config{}, err
	}
	if err := raw.Validate(); err != nil {
		return Config{}, err
	}
	return raw, nil
}

// Validate checks that all config fields are well-formed.
func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version %d (want 1)", c.Version)
	}
	if c.Profile != "" {
		switch c.Profile {
		case "strict", "standard", "permissive":
		default:
			return fmt.Errorf("invalid profile %q", c.Profile)
		}
	}
	for i, ig := range c.Ignore {
		if ig.Rule == "" {
			return fmt.Errorf("ignore[%d]: rule is required", i)
		}
		if ig.Reason == "" {
			return fmt.Errorf("ignore[%d]: reason is required (see CLAUDE.md §on suppressions)", i)
		}
		if ig.Expires == "" {
			return fmt.Errorf("ignore[%d]: expires is required", i)
		}
		if _, err := time.Parse("2006-01-02", ig.Expires); err != nil {
			return fmt.Errorf("ignore[%d]: expires must be YYYY-MM-DD: %w", i, err)
		}
	}
	return nil
}

// ExpiredIgnores returns entries whose expires date has passed relative to now.
// Callers surface these as warnings — suppressions must not silently rot.
func (c Config) ExpiredIgnores(now time.Time) []Ignore {
	var out []Ignore
	for _, ig := range c.Ignore {
		t, err := time.Parse("2006-01-02", ig.Expires)
		if err != nil {
			continue
		}
		if now.After(t) {
			out = append(out, ig)
		}
	}
	return out
}

// newTrimReader strips leading whitespace so an accidental BOM-less empty line
// does not confuse DisallowUnknownFields. Tiny, inlined.
type trimReader struct {
	buf []byte
	pos int
}

func newTrimReader(b []byte) *trimReader {
	i := 0
	for i < len(b) && (b[i] == ' ' || b[i] == '\t' || b[i] == '\r' || b[i] == '\n') {
		i++
	}
	return &trimReader{buf: b[i:]}
}

func (r *trimReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.buf) {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.pos:])
	r.pos += n
	return n, nil
}
