package resolvers

import (
	"context"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// deploymentArtifacts are files/directories whose presence indicates the repo
// deploys something. If any of these exist, rollback documentation is expected.
var deploymentArtifacts = []string{
	"Dockerfile",
	"dockerfile",
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
	"Procfile",
	"app.yaml",
	"app.yml",
	"render.yaml",
	"fly.toml",
	"railway.toml",
	"vercel.json",
	"netlify.toml",
	// AWS CDK
	"cdk.json",
	// Serverless Framework
	"serverless.yml",
	"serverless.yaml",
	// Google Cloud Build
	"cloudbuild.yaml",
	"cloudbuild.yml",
	// Skaffold (Kubernetes dev)
	"skaffold.yaml",
}

// deploymentDirPrefixes are directory prefixes that indicate deployment config.
var deploymentDirPrefixes = []string{
	"kubernetes/",
	"k8s/",
	"deploy/",
	"deployment/",
	"terraform/",
	"infra/",
	"infrastructure/",
	"helm/",
	".github/workflows/",
	".circleci/",
	".gitlab-ci",
	"cdk/",
	"serverless/",
	".buildkite/",
}

// rollbackDocCandidates are the paths archfit accepts as rollback documentation.
var rollbackDocCandidates = []string{
	"RUNBOOK.md",
	"runbook.md",
	"docs/runbook.md",
	"docs/RUNBOOK.md",
	"docs/rollback.md",
	"docs/ROLLBACK.md",
	"docs/deployment.md",
	"docs/DEPLOYMENT.md",
	"docs/deploy.md",
	"docs/operations.md",
	"DEPLOYMENT.md",
	"ROLLBACK.md",
}

// RevP6REV001 fires when the repo has deployment artifacts but no rollback or
// deployment documentation. Reversibility requires knowing how to undo a deploy.
func RevP6REV001(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()

	if !hasDeploymentArtifacts(repo) {
		return nil, nil, nil
	}

	for _, c := range rollbackDocCandidates {
		if _, ok := repo.ByPath[c]; ok {
			return nil, nil, nil
		}
	}

	return []model.Finding{{
		Path:       "docs/",
		Message:    "deployment artifacts detected but no rollback/deployment documentation (RUNBOOK.md, docs/deployment.md, etc.)",
		Confidence: 0.92,
		Evidence: map[string]any{
			"looked_for": rollbackDocCandidates,
		},
	}}, nil, nil
}

func hasDeploymentArtifacts(repo model.RepoFacts) bool {
	for _, name := range deploymentArtifacts {
		if _, ok := repo.ByPath[name]; ok {
			return true
		}
	}
	for _, f := range repo.Files {
		for _, prefix := range deploymentDirPrefixes {
			if strings.HasPrefix(f.Path, prefix) {
				return true
			}
		}
	}
	return false
}
