package resolvers

import (
	"context"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// apiContractFiles are root-relative paths whose presence signals a
// machine-readable API contract. Any one of these satisfies the rule.
var apiContractBasenames = []string{
	"openapi.yaml", "openapi.yml", "openapi.json",
	"swagger.yaml", "swagger.yml", "swagger.json",
	"schema.graphql",
}

// apiContractExtensionDirs are (directory prefix, extension) pairs. A file
// matching both satisfies the rule (e.g., proto/service.proto).
var apiContractExtensionDirs = []struct {
	dirPrefix string
	ext       string
}{
	{"proto/", ".proto"},
	{"api/", ".proto"},
	{"api/", ".yaml"},
	{"api/", ".yml"},
	{"api/", ".json"},
	{"graphql/", ".graphql"},
	{"graphql/", ".gql"},
}

// serviceIndicatorSegments are path segments that suggest the repo is an
// HTTP/gRPC service (as opposed to a library or CLI-only tool). These are
// checked case-insensitively against each segment of each file path.
var serviceIndicatorSegments = []string{
	"handler", "handlers",
	"controller", "controllers",
	"router", "routers",
	"routes",
	"endpoint", "endpoints",
	"server",
	"views",    // Django/Flask
	"resource", // JAX-RS
}

// serviceIndicatorBasenames are exact basenames (case-insensitive) that
// suggest the repo serves an API.
var serviceIndicatorBasenames = []string{
	"app.py",    // FastAPI/Flask/Django
	"server.go", // Go HTTP
	"server.ts", // Node/Express
	"server.js", // Node/Express
	"app.ts",    // Node
	"app.js",    // Node/Express
	"main.py",   // FastAPI
	"routes.rb", // Rails
	"routes.py", // Flask/Django
	"wsgi.py",   // Django
	"asgi.py",   // Django/Starlette
}

// SpcP2SPC001 fires when the repo looks like an HTTP/gRPC/GraphQL service
// (based on directory and filename patterns) but has no machine-readable API
// contract (OpenAPI, Protobuf, or GraphQL schema).
func SpcP2SPC001(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	// 1. Check if any API contract file exists → rule satisfied.
	if hasAPIContract(repo) {
		return nil, nil, nil
	}

	// 2. Check if the repo looks like a service.
	indicators := findServiceIndicators(repo)
	if len(indicators) == 0 {
		return nil, nil, nil // not a service → rule does not apply
	}

	return []model.Finding{{
		Message:    "repository appears to serve an API but has no machine-readable contract (OpenAPI, Protobuf, or GraphQL schema)",
		Confidence: 0.85,
		Evidence: map[string]any{
			"service_indicators": truncateSlice(indicators, 10),
			"looked_for_contracts": append(
				apiContractBasenames,
				"proto/*.proto", "api/*.proto", "api/*.yaml", "graphql/*.graphql",
			),
		},
	}}, nil, nil
}

func hasAPIContract(repo model.RepoFacts) bool {
	// Check basenames at any path.
	for _, name := range apiContractBasenames {
		if _, ok := repo.ByBase[name]; ok {
			return true
		}
	}
	// Check extension+directory patterns.
	for _, f := range repo.Files {
		if isFixtureOrTestdata(f.Path) {
			continue
		}
		for _, ed := range apiContractExtensionDirs {
			if strings.HasPrefix(f.Path, ed.dirPrefix) && strings.HasSuffix(f.Path, ed.ext) {
				return true
			}
		}
	}
	return false
}

func findServiceIndicators(repo model.RepoFacts) []string {
	var indicators []string
	seen := map[string]bool{}

	for _, f := range repo.Files {
		if isFixtureOrTestdata(f.Path) {
			continue
		}
		// Check basename matches.
		base := fileBase(f.Path)
		for _, ib := range serviceIndicatorBasenames {
			if strings.EqualFold(base, ib) && !seen[f.Path] {
				indicators = append(indicators, f.Path)
				seen[f.Path] = true
			}
		}
		// Check path segment matches.
		segments := strings.Split(strings.ToLower(f.Path), "/")
		for _, seg := range segments {
			for _, is := range serviceIndicatorSegments {
				if seg == is && !seen[f.Path] {
					indicators = append(indicators, f.Path)
					seen[f.Path] = true
				}
			}
		}
	}
	return indicators
}
