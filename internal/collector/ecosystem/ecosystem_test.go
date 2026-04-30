package ecosystem_test

import (
	"testing"

	"github.com/shibuiwilliam/archfit/internal/collector/ecosystem"
	"github.com/shibuiwilliam/archfit/internal/model"
)

func repo(paths ...string) model.RepoFacts {
	files := make([]model.FileFact, len(paths))
	byPath := make(map[string]model.FileFact, len(paths))
	for i, p := range paths {
		f := model.FileFact{Path: p}
		files[i] = f
		byPath[p] = f
	}
	return model.RepoFacts{Files: files, ByPath: byPath}
}

func TestCollect_DetectsGitHubActions(t *testing.T) {
	ef := ecosystem.Collect(repo(".github/workflows/ci.yml", "main.go"))
	if !ef.HasCI() {
		t.Error("expected CI detected")
	}
	if !ef.Has("github-actions") {
		t.Error("expected github-actions ecosystem")
	}
}

func TestCollect_DetectsDocker(t *testing.T) {
	ef := ecosystem.Collect(repo("Dockerfile", "main.go"))
	if !ef.HasDeployment() {
		t.Error("expected deployment detected")
	}
	if !ef.Has("docker") {
		t.Error("expected docker ecosystem")
	}
}

func TestCollect_DetectsSpring(t *testing.T) {
	ef := ecosystem.Collect(repo("src/main/resources/application.yml", "pom.xml"))
	if !ef.Has("spring") {
		t.Error("expected spring ecosystem")
	}
}

func TestCollect_DetectsRails(t *testing.T) {
	ef := ecosystem.Collect(repo("config/environments/production.rb", "Gemfile"))
	if !ef.Has("rails") {
		t.Error("expected rails ecosystem")
	}
}

func TestCollect_DetectsMultiple(t *testing.T) {
	ef := ecosystem.Collect(repo(
		"Dockerfile",
		".github/workflows/ci.yml",
		"src/main/resources/application-dev.yml",
	))
	if !ef.Has("docker") || !ef.Has("github-actions") || !ef.Has("spring") {
		t.Errorf("expected docker + github-actions + spring, got %+v", ef.Detected)
	}
}

func TestCollect_EmptyRepo(t *testing.T) {
	ef := ecosystem.Collect(repo())
	if len(ef.Detected) != 0 {
		t.Errorf("expected empty, got %+v", ef.Detected)
	}
}

func TestCollect_SkipsFixturePaths(t *testing.T) {
	ef := ecosystem.Collect(repo("packs/core/fixtures/P6.REV.001/input/Dockerfile"))
	if ef.Has("docker") {
		t.Error("should skip fixture paths")
	}
}

func TestCIFiles(t *testing.T) {
	ef := ecosystem.Collect(repo(".github/workflows/ci.yml", ".github/workflows/lint.yml", "main.go"))
	files := ef.CIFiles()
	if len(files) != 2 {
		t.Errorf("expected 2 CI files, got %d: %v", len(files), files)
	}
}

func TestHasDeployment_FalseForCIOnly(t *testing.T) {
	ef := ecosystem.Collect(repo(".github/workflows/ci.yml"))
	if ef.HasDeployment() {
		t.Error("CI-only repo should not report deployment")
	}
}
