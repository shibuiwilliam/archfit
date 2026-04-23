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

func TestParseJSON_UnknownFieldRejected(t *testing.T) {
	_, err := config.ParseJSON([]byte(`{"version":1,"unknown":"x"}`))
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestParseJSON_IgnoreValidation(t *testing.T) {
	_, err := config.ParseJSON([]byte(`{
        "version":1,
        "ignore":[{"rule":"P1.LOC.001","reason":"","expires":"2030-01-01"}]
    }`))
	if err == nil {
		t.Fatal("expected error for empty reason")
	}
}

func TestLoad_HappyPath(t *testing.T) {
	dir := t.TempDir()
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
