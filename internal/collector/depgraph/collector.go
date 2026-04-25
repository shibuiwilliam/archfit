package depgraph

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// Collect builds a dependency graph from the repository's Go source files.
// It reads the go.mod to find the module path, then parses .go files for imports.
// If the repository does not contain a go.mod, an empty graph is returned.
func Collect(repo model.RepoFacts) (Graph, error) {
	// Check if go.mod exists.
	if _, ok := repo.ByPath["go.mod"]; !ok {
		return Graph{}, nil
	}

	modulePath, err := readModulePath(filepath.Join(repo.Root, "go.mod"))
	if err != nil {
		return Graph{}, err
	}
	if modulePath == "" {
		return Graph{}, nil
	}

	// Filter to .go files and build absolute paths.
	var goFiles []string
	for _, f := range repo.Files {
		if f.Ext == ".go" {
			goFiles = append(goFiles, filepath.Join(repo.Root, f.Path))
		}
	}
	if len(goFiles) == 0 {
		return Graph{}, nil
	}

	return CollectGo(goFiles, modulePath)
}

// readModulePath extracts the module path from a go.mod file by finding the
// line that starts with "module ". This avoids importing golang.org/x/mod.
func readModulePath(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", sc.Err()
}
