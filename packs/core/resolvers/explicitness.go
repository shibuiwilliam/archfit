package resolvers

import (
	"context"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// envExampleCandidates are the root-level files archfit accepts as "environment
// variables are documented". Keep this list explicit.
var envExampleCandidates = []string{
	".env.example",
	".env.sample",
	".env.template",
	"env.example",
}

// envFilePatterns are file basenames that indicate the repo uses environment
// variables. We exclude the documentation files themselves.
var envFileExclusions = map[string]bool{
	".env.example":  true,
	".env.sample":   true,
	".env.template": true,
	"env.example":   true,
}

// configDocCandidates are paths archfit accepts as "configuration is documented"
// for Spring Boot, Terraform, and Rails config patterns.
var configDocCandidates = []string{
	"config/README.md",
	"config/readme.md",
	"docs/config.md",
	"docs/CONFIG.md",
	"docs/configuration.md",
	"docs/CONFIGURATION.md",
}

// tfvarsDocCandidates are paths archfit accepts as documentation for Terraform
// variable files.
var tfvarsDocCandidates = []string{
	"terraform.tfvars.example",
	"terraform.tfvars.sample",
	"example.tfvars",
	"sample.tfvars",
}

// ExpP3EXP001 fires when the repo has hidden configuration files without
// corresponding documentation. Detects:
//   - .env files without .env.example
//   - Spring Boot application-*.yml profiles without config documentation
//   - Terraform *.tfvars without a tfvars example
//   - Rails config/environments/*.rb without config documentation
func ExpP3EXP001(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()
	var findings []model.Finding

	// Check 1: .env files without documentation.
	if f := checkEnvFiles(repo); f != nil {
		findings = append(findings, *f)
	}

	// Check 2: Spring Boot profile-specific config without documentation.
	if f := checkSpringProfiles(repo); f != nil {
		findings = append(findings, *f)
	}

	// Check 3: Terraform tfvars without example.
	if f := checkTerraformVars(repo); f != nil {
		findings = append(findings, *f)
	}

	// Check 4: Rails config/environments without documentation.
	if f := checkRailsEnvironments(repo); f != nil {
		findings = append(findings, *f)
	}

	return findings, nil, nil
}

// checkEnvFiles detects .env files without .env.example documentation.
func checkEnvFiles(repo model.RepoFacts) *model.Finding {
	for _, name := range envExampleCandidates {
		if _, ok := repo.ByPath[name]; ok {
			return nil
		}
	}
	var envFiles []string
	for _, f := range repo.Files {
		base := fileBase(f.Path)
		if !strings.HasPrefix(base, ".env") {
			continue
		}
		if envFileExclusions[base] {
			continue
		}
		envFiles = append(envFiles, f.Path)
	}
	if len(envFiles) == 0 {
		return nil
	}
	return &model.Finding{
		Message:    "repository uses .env files but has no .env.example to document required environment variables",
		Confidence: 0.95,
		Evidence: map[string]any{
			"env_files":   envFiles,
			"looked_for":  envExampleCandidates,
			"config_type": "dotenv",
		},
	}
}

// checkSpringProfiles detects Spring Boot profile-specific config files
// (application-*.yml, application-*.yaml, application-*.properties) without
// a config/README.md or docs/config.md to document the profile differences.
func checkSpringProfiles(repo model.RepoFacts) *model.Finding {
	if hasConfigDoc(repo) {
		return nil
	}
	var profiles []string
	for _, f := range repo.Files {
		base := fileBase(f.Path)
		if !strings.HasPrefix(base, "application-") {
			continue
		}
		if strings.HasSuffix(base, ".yml") || strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".properties") {
			profiles = append(profiles, f.Path)
		}
	}
	if len(profiles) == 0 {
		return nil
	}
	return &model.Finding{
		Message:    "Spring Boot profile configs detected but no config documentation — differences between profiles are hidden from agents",
		Confidence: 0.90,
		Evidence: map[string]any{
			"profile_files": profiles,
			"looked_for":    configDocCandidates,
			"config_type":   "spring-boot",
		},
	}
}

// checkTerraformVars detects *.tfvars files without a terraform.tfvars.example
// or similar documentation file.
func checkTerraformVars(repo model.RepoFacts) *model.Finding {
	for _, name := range tfvarsDocCandidates {
		if _, ok := repo.ByPath[name]; ok {
			return nil
		}
	}
	var tfvarsFiles []string
	for _, f := range repo.Files {
		base := fileBase(f.Path)
		if strings.HasSuffix(base, ".tfvars") {
			tfvarsFiles = append(tfvarsFiles, f.Path)
		}
	}
	if len(tfvarsFiles) == 0 {
		return nil
	}
	return &model.Finding{
		Message:    "Terraform .tfvars files detected but no terraform.tfvars.example — required variables are hidden from agents",
		Confidence: 0.90,
		Evidence: map[string]any{
			"tfvars_files": tfvarsFiles,
			"looked_for":   tfvarsDocCandidates,
			"config_type":  "terraform",
		},
	}
}

// checkRailsEnvironments detects Rails config/environments/*.rb files without
// a config/README.md or docs/config.md to document environment differences.
func checkRailsEnvironments(repo model.RepoFacts) *model.Finding {
	if hasConfigDoc(repo) {
		return nil
	}
	var envFiles []string
	for _, f := range repo.Files {
		if strings.HasPrefix(f.Path, "config/environments/") && strings.HasSuffix(f.Path, ".rb") {
			envFiles = append(envFiles, f.Path)
		}
	}
	if len(envFiles) == 0 {
		return nil
	}
	return &model.Finding{
		Message:    "Rails environment configs detected (config/environments/) but no config documentation — environment differences are hidden from agents",
		Confidence: 0.90,
		Evidence: map[string]any{
			"env_files":   envFiles,
			"looked_for":  configDocCandidates,
			"config_type": "rails",
		},
	}
}

// hasConfigDoc returns true if any general config documentation file exists.
func hasConfigDoc(repo model.RepoFacts) bool {
	for _, name := range configDocCandidates {
		if _, ok := repo.ByPath[name]; ok {
			return true
		}
	}
	return false
}

// fileBase returns the basename of a path (last segment after /).
func fileBase(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
