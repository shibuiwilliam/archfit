// Package ecosystem infers typed ecosystem facts from RepoFacts. It runs once
// per scan and provides a unified view that resolvers consume instead of
// maintaining per-resolver keyword tables.
//
// See ADR 0011 for the rationale behind centralizing ecosystem detection.
package ecosystem

import (
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// detectionRule maps a condition to an ecosystem name.
type detectionRule struct {
	name       string
	confidence float64
	// exactFiles are root-level filenames whose presence triggers detection.
	exactFiles []string
	// dirPrefixes are directory prefixes — any file under these triggers detection.
	dirPrefixes []string
}

// ciRules detect CI/CD platforms.
var ciRules = []detectionRule{
	{name: "github-actions", confidence: 1.0, dirPrefixes: []string{".github/workflows/"}},
	{name: "gitlab-ci", confidence: 1.0, exactFiles: []string{".gitlab-ci.yml", ".gitlab-ci.yaml"}},
	{name: "circleci", confidence: 1.0, dirPrefixes: []string{".circleci/"}},
	{name: "buildkite", confidence: 1.0, dirPrefixes: []string{".buildkite/"}},
	{name: "jenkins", confidence: 1.0, exactFiles: []string{"Jenkinsfile"}},
	{name: "travis", confidence: 1.0, exactFiles: []string{".travis.yml"}},
	{name: "azure-pipelines", confidence: 1.0, exactFiles: []string{"azure-pipelines.yml"}},
	{name: "bitbucket-pipelines", confidence: 1.0, exactFiles: []string{"bitbucket-pipelines.yml"}},
	{name: "woodpecker", confidence: 1.0, exactFiles: []string{".woodpecker.yml"}},
}

// deployRules detect deployment platforms and tools.
var deployRules = []detectionRule{
	{name: "docker", confidence: 1.0, exactFiles: []string{
		"Dockerfile", "dockerfile",
		"docker-compose.yml", "docker-compose.yaml",
		"compose.yml", "compose.yaml",
	}},
	{name: "kubernetes", confidence: 1.0, dirPrefixes: []string{"kubernetes/", "k8s/"}},
	{name: "helm", confidence: 1.0, dirPrefixes: []string{"helm/"}},
	{name: "terraform", confidence: 1.0, dirPrefixes: []string{"terraform/", "infra/", "infrastructure/"}},
	{name: "aws-cdk", confidence: 1.0, exactFiles: []string{"cdk.json"}, dirPrefixes: []string{"cdk/"}},
	{name: "serverless", confidence: 1.0, exactFiles: []string{"serverless.yml", "serverless.yaml"}, dirPrefixes: []string{"serverless/"}},
	{name: "cloud-build", confidence: 1.0, exactFiles: []string{"cloudbuild.yaml", "cloudbuild.yml"}},
	{name: "skaffold", confidence: 1.0, exactFiles: []string{"skaffold.yaml"}},
	{name: "heroku", confidence: 1.0, exactFiles: []string{"Procfile"}},
	{name: "render", confidence: 1.0, exactFiles: []string{"render.yaml"}},
	{name: "fly-io", confidence: 1.0, exactFiles: []string{"fly.toml"}},
	{name: "railway", confidence: 1.0, exactFiles: []string{"railway.toml"}},
	{name: "vercel", confidence: 1.0, exactFiles: []string{"vercel.json"}},
	{name: "netlify", confidence: 1.0, exactFiles: []string{"netlify.toml"}},
}

// frameworkRules detect application frameworks.
var frameworkRules = []detectionRule{
	{name: "spring", confidence: 0.9, dirPrefixes: []string{"src/main/resources/"}},
	{name: "rails", confidence: 0.9, dirPrefixes: []string{"config/environments/"}},
}

// fixturePathPrefixes exclude test fixtures from ecosystem detection.
var fixturePathPrefixes = []string{
	"testdata/",
	"fixtures/",
	"test/fixtures/",
	"tests/fixtures/",
	"packs/core/fixtures/",
	"packs/agent-tool/fixtures/",
	".claude/worktrees/",
}

// Collect runs all detection rules against the repo in a single pass and
// returns model.EcosystemFacts. Resolvers consume the result via
// facts.Ecosystems().Has("spring"), facts.Ecosystems().HasCI(), etc.
func Collect(repo model.RepoFacts) model.EcosystemFacts {
	matched := map[string]map[string]bool{}
	confidence := map[string]float64{}

	allRules := make([]detectionRule, 0, len(ciRules)+len(deployRules)+len(frameworkRules))
	allRules = append(allRules, ciRules...)
	allRules = append(allRules, deployRules...)
	allRules = append(allRules, frameworkRules...)

	// Fast path: exact file matches via ByPath map.
	for _, r := range allRules {
		for _, name := range r.exactFiles {
			if _, ok := repo.ByPath[name]; ok {
				if matched[r.name] == nil {
					matched[r.name] = map[string]bool{}
					confidence[r.name] = r.confidence
				}
				matched[r.name][name] = true
			}
		}
	}

	// Single pass over all files for prefix matching.
	for _, f := range repo.Files {
		if isFixture(f.Path) {
			continue
		}
		for _, r := range allRules {
			for _, prefix := range r.dirPrefixes {
				if strings.HasPrefix(f.Path, prefix) {
					if matched[r.name] == nil {
						matched[r.name] = map[string]bool{}
						confidence[r.name] = r.confidence
					}
					matched[r.name][f.Path] = true
					break
				}
			}
		}
	}

	// Build sorted result.
	var detected []model.EcosystemEntry
	for name, markers := range matched {
		paths := make([]string, 0, len(markers))
		for p := range markers {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		if len(paths) > 5 {
			paths = paths[:5]
		}
		detected = append(detected, model.EcosystemEntry{
			Name:       name,
			Confidence: confidence[name],
			Markers:    paths,
		})
	}
	sort.Slice(detected, func(i, j int) bool { return detected[i].Name < detected[j].Name })

	return model.EcosystemFacts{Detected: detected}
}

func isFixture(path string) bool {
	for _, prefix := range fixturePathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
