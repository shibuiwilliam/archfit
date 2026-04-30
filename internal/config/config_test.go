package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shibuiwilliam/archfit/internal/config"
)

func TestLoad_MissingReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	cfg, p, present, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatalf("expected not present, got %q", p)
	}
	if cfg.Profile != "standard" || len(cfg.Packs.Enabled) != 1 {
		t.Errorf("unexpected default: %+v", cfg)
	}
}

func TestParse_UnknownFieldRejected(t *testing.T) {
	_, err := config.Parse([]byte(`{"version":1,"unknown":"x"}`))
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestParse_IgnoreValidation(t *testing.T) {
	_, err := config.Parse([]byte(`{
        "version":1,
        "ignore":[{"rule":"P1.LOC.001","reason":"","expires":"2030-01-01"}]
    }`))
	if err == nil {
		t.Fatal("expected error for empty reason")
	}
}

func TestLoad_JSON_HappyPath(t *testing.T) {
	dir := t.TempDir()
	// JSON-formatted config must continue to work (YAML 1.2 is a superset).
	if err := os.WriteFile(filepath.Join(dir, ".archfit.yaml"), []byte(`{
      "version":1,
      "profile":"strict",
      "packs":{"enabled":["core"]},
      "ignore":[{"rule":"P1.LOC.001","reason":"wip","expires":"2030-01-01"}]
    }`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, present, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !present || cfg.Profile != "strict" {
		t.Errorf("unexpected cfg: %+v present=%v", cfg, present)
	}
	if exp := cfg.ExpiredIgnores(time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)); len(exp) != 1 {
		t.Errorf("expired should include expired entry")
	}
}

func TestLoad_YAML_WithComments(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `# archfit configuration
version: 1
profile: standard

# Enable both packs for this project
packs:
  enabled:
    - core
    - agent-tool

ignore:
  - rule: P1.LOC.002
    reason: legacy slices being removed
    expires: "2030-06-30"
`
	if err := os.WriteFile(filepath.Join(dir, ".archfit.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, present, err := config.Load(dir)
	if err != nil {
		t.Fatalf("YAML with comments should parse: %v", err)
	}
	if !present {
		t.Fatal("expected config to be present")
	}
	if cfg.Profile != "standard" {
		t.Errorf("profile = %q, want standard", cfg.Profile)
	}
	if len(cfg.Packs.Enabled) != 2 {
		t.Errorf("packs.enabled = %v, want [core agent-tool]", cfg.Packs.Enabled)
	}
	if len(cfg.Ignore) != 1 || cfg.Ignore[0].Rule != "P1.LOC.002" {
		t.Errorf("ignore = %+v, want 1 entry for P1.LOC.002", cfg.Ignore)
	}
}

func TestParse_VerificationLayers(t *testing.T) {
	yamlContent := `
version: 1
packs:
  enabled: [core]
verification:
  lint:
    command: make lint
    timeout_s: 5
  unit:
    command: make test
    timeout_s: 60
`
	cfg, err := config.Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("YAML with verification should parse: %v", err)
	}
	if len(cfg.Verification) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(cfg.Verification))
	}
	lint := cfg.Verification["lint"]
	if lint.Command != "make lint" || lint.TimeoutS != 5 {
		t.Errorf("lint layer = %+v, want command='make lint', timeout_s=5", lint)
	}
	unit := cfg.Verification["unit"]
	if unit.Command != "make test" || unit.TimeoutS != 60 {
		t.Errorf("unit layer = %+v, want command='make test', timeout_s=60", unit)
	}
}

func TestParse_YAML_UnquotedStrings(t *testing.T) {
	yamlContent := `
version: 1
profile: permissive
packs:
  enabled: [core]
risk_tiers:
  high:
    - src/auth/**
    - infra/**
`
	cfg, err := config.Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("YAML with unquoted strings should parse: %v", err)
	}
	if cfg.Profile != "permissive" {
		t.Errorf("profile = %q, want permissive", cfg.Profile)
	}
	if len(cfg.RiskTiers["high"]) != 2 {
		t.Errorf("risk_tiers.high = %v, want 2 entries", cfg.RiskTiers["high"])
	}
}

func TestParse_YAML_UnknownFieldRejected(t *testing.T) {
	_, err := config.Parse([]byte("version: 1\nbogus_field: true\n"))
	if err == nil {
		t.Fatal("YAML with unknown field should be rejected by strict parsing")
	}
}

// ParseJSON is the backward-compatible alias — verify it still works.
func TestParseJSON_BackwardCompat(t *testing.T) {
	cfg, err := config.ParseJSON([]byte(`{"version":1,"packs":{"enabled":["core"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
}
