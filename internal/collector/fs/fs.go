// Package fs collects filesystem facts from a repository root.
//
// Collectors do not judge. They walk once, in a deterministic order, producing
// a typed RepoFacts value. Resolvers then receive that through FactStore.
package fs

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// Default ignores. Repo-local ignore lists arrive via config in Phase 2.
var defaultIgnoredDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"vendor":       {},
	".venv":        {},
	"dist":         {},
	"build":        {},
	"target":       {},
	"bin":          {},
	".cache":       {},
	".idea":        {},
	".vscode":      {},
	"__pycache__":  {},
}

// Limits guard against pathological repos. They can be raised by config later.
const (
	maxFiles   = 50_000
	maxLineLen = 64 * 1024
)

// Collect walks root and returns deterministic facts. Paths are repo-relative
// with forward slashes so fixtures behave identically across OSes.
func Collect(root string) (model.RepoFacts, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return model.RepoFacts{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return model.RepoFacts{}, err
	}
	if !info.IsDir() {
		return model.RepoFacts{}, fmt.Errorf("fs.Collect: %s is not a directory", abs)
	}

	var files []model.FileFact
	byBase := map[string][]string{}
	langs := map[string]int{}

	err = filepath.WalkDir(abs, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			// Continue past unreadable entries; a warn-severity rule surfaces this in Phase 2.
			return nil
		}
		if d.IsDir() {
			if p == abs {
				return nil
			}
			if _, skip := defaultIgnoredDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if len(files) >= maxFiles {
			return filepath.SkipAll
		}
		rel, err := filepath.Rel(abs, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(rel))
		lines := 0
		if fi.Size() > 0 && fi.Size() < 4*1024*1024 && isTextExt(ext) {
			if n, ok := countLines(p); ok {
				lines = n
			}
		}
		ff := model.FileFact{Path: rel, Size: fi.Size(), Lines: lines, Ext: ext}
		files = append(files, ff)
		base := strings.ToLower(filepath.Base(rel))
		byBase[base] = append(byBase[base], rel)
		if lang := languageForExt(ext); lang != "" {
			langs[lang]++
		}
		return nil
	})
	if err != nil {
		return model.RepoFacts{}, err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	byPath := make(map[string]model.FileFact, len(files))
	for _, f := range files {
		byPath[f.Path] = f
	}

	return model.RepoFacts{
		Root:      abs,
		Files:     files,
		ByPath:    byPath,
		ByBase:    byBase,
		Languages: langs,
	}, nil
}

func isTextExt(ext string) bool {
	switch ext {
	case ".go", ".py", ".ts", ".tsx", ".js", ".jsx", ".rs", ".java", ".kt", ".swift",
		".rb", ".php", ".c", ".cc", ".cpp", ".h", ".hpp", ".cs", ".scala", ".clj",
		".md", ".yaml", ".yml", ".json", ".toml", ".xml", ".sh", ".bash", ".zsh",
		".sql", ".proto", ".graphql", ".gql", ".tf", ".hcl", ".dockerfile", ".makefile",
		".html", ".css", ".scss", ".less", ".ini", ".cfg", ".env", ".txt", "":
		return true
	}
	return false
}

func languageForExt(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".tf", ".hcl":
		return "terraform"
	case ".proto":
		return "protobuf"
	case ".graphql", ".gql":
		return "graphql"
	}
	return ""
}

func countLines(p string) (int, bool) {
	f, err := os.Open(p)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 4096), maxLineLen)
	n := 0
	for sc.Scan() {
		n++
	}
	if sc.Err() != nil {
		return 0, false
	}
	return n, true
}
