// Package config loads .archfit.yaml (or .archfit.yml / .archfit.json) from the
// target repo.
//
// Parsing uses sigs.k8s.io/yaml which accepts both YAML 1.2 and JSON. Existing
// JSON-formatted configs continue to work; idiomatic YAML (comments, unquoted
// strings, block scalars) is now also supported.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// sigs.k8s.io/yaml preserves `json:"..."` tag semantics on Go structs,
	// so no struct tag migration is needed. Justified in docs/dependencies.md.
	"sigs.k8s.io/yaml"
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
	cfg, err := Parse(data)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w\nhint: archfit reads YAML 1.2; check indentation and quoting", path, err)
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
		cfg, err := Parse(data)
		if err != nil {
			return Config{}, p, true, fmt.Errorf("%s: %w\nhint: archfit reads YAML 1.2; check indentation and quoting", p, err)
		}
		return cfg, p, true, nil
	}
	return Default(), "", false, nil
}

// Parse validates and decodes a YAML (or JSON) config document.
// It rejects unknown fields to catch typos early.
func Parse(data []byte) (Config, error) {
	var raw Config
	if err := yaml.UnmarshalStrict(data, &raw); err != nil {
		return Config{}, err
	}
	if err := raw.Validate(); err != nil {
		return Config{}, err
	}
	return raw, nil
}

// ParseJSON is a backward-compatible alias for Parse. JSON is valid YAML 1.2.
func ParseJSON(data []byte) (Config, error) {
	return Parse(data)
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
