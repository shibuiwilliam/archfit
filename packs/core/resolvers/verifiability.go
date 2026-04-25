package resolvers

import (
	"context"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// verificationEntrypoints are the root-level files archfit recognizes as a
// declared fast-verification entrypoint. Keep this list explicit; adding a new
// one is a deliberate policy choice, not a silent expansion.
var verificationEntrypoints = []string{
	// Task runners
	"Makefile",
	"makefile",
	"justfile",
	"Justfile",
	"Taskfile.yml",
	"Taskfile.yaml",
	// Language-specific build/project files
	"package.json",   // Node/TypeScript (npm/yarn/pnpm)
	"pyproject.toml", // Python (PEP 621)
	"Cargo.toml",     // Rust
	"go.mod",         // Go (`go test ./...` is a universal fallback)
	"pom.xml",        // Java (Maven)
	"build.gradle",   // Java/Kotlin (Gradle)
	"build.gradle.kts",
	"settings.gradle",
	"settings.gradle.kts",
	"Gemfile",        // Ruby (Bundler)
	"Rakefile",       // Ruby (Rake)
	"rakefile",       // Ruby (Rake, lowercase variant)
	"composer.json",  // PHP (Composer)
	"mix.exs",        // Elixir (Mix)
	"build.sbt",      // Scala (sbt)
	"CMakeLists.txt", // C/C++ (CMake)
	"deno.json",      // Deno
	"deno.jsonc",     // Deno (JSONC variant)
	"Earthfile",      // Earthly
	"BUILD.bazel",    // Bazel
	"meson.build",    // Meson (C/C++)
}

// VerP4VER001 fires when the repo has no recognized fast-verification entrypoint at its root.
func VerP4VER001(_ context.Context, facts model.FactStore) ([]model.Finding, []model.Metric, error) {
	repo := facts.Repo()
	for _, name := range verificationEntrypoints {
		if _, ok := repo.ByPath[name]; ok {
			return nil, nil, nil
		}
	}
	return []model.Finding{{
		Message:    "no verification entrypoint at repo root (checked Makefile, justfile, Taskfile, package.json, pyproject.toml, Cargo.toml, go.mod, pom.xml, build.gradle, Gemfile, Rakefile, composer.json, and others)",
		Confidence: 0.98,
		Evidence: map[string]any{
			"looked_for": verificationEntrypoints,
		},
	}}, nil, nil
}
