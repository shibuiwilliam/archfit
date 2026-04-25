package depgraph

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

// CollectGo builds a dependency graph from Go source files.
// It uses go/parser from the standard library to extract import statements.
// The files parameter is a list of absolute file paths to .go files.
// modulePath is the Go module path (e.g., "github.com/shibuiwilliam/archfit")
// — only imports within this module are tracked (external deps are ignored).
func CollectGo(files []string, modulePath string) (Graph, error) {
	type pkgInfo struct {
		files int
		// imports is a set of internal import paths (relative to module).
		imports map[string]struct{}
	}

	pkgs := map[string]*pkgInfo{}
	fset := token.NewFileSet()

	for _, path := range files {
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			// Skip files that fail to parse; a parse-failure finding
			// is the resolver's responsibility (CLAUDE.md §13).
			continue
		}

		// Derive the package directory relative to the module root.
		// We infer module root by stripping the modulePath-derived suffix,
		// but since we only have file paths, we use the directory of the file
		// relative to wherever the module root is. The caller is expected to
		// pass files rooted under the module. We detect the module root by
		// finding the modulePath's last element in the file path.
		pkgDir := packageDir(path, modulePath)
		if pkgDir == "" {
			continue
		}

		info, ok := pkgs[pkgDir]
		if !ok {
			info = &pkgInfo{imports: map[string]struct{}{}}
			pkgs[pkgDir] = info
		}
		info.files++

		for _, imp := range f.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(impPath, modulePath) {
				continue
			}
			// Convert to module-relative path.
			rel := strings.TrimPrefix(impPath, modulePath)
			rel = strings.TrimPrefix(rel, "/")
			if rel == "" {
				rel = "."
			}
			if rel != pkgDir {
				info.imports[rel] = struct{}{}
				// Ensure the target package exists as a node even if we
				// haven't parsed any of its files yet.
				if _, exists := pkgs[rel]; !exists {
					pkgs[rel] = &pkgInfo{imports: map[string]struct{}{}}
				}
			}
		}
	}

	// Build sorted nodes.
	nodeNames := make([]string, 0, len(pkgs))
	for name := range pkgs {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	nodes := make([]Node, 0, len(nodeNames))
	for _, name := range nodeNames {
		nodes = append(nodes, Node{Package: name, Files: pkgs[name].files})
	}

	// Build sorted, deduplicated edges.
	type edgeKey struct{ from, to string }
	seen := map[edgeKey]struct{}{}
	var edges []Edge
	for _, name := range nodeNames {
		targets := make([]string, 0, len(pkgs[name].imports))
		for t := range pkgs[name].imports {
			targets = append(targets, t)
		}
		sort.Strings(targets)
		for _, t := range targets {
			k := edgeKey{name, t}
			if _, dup := seen[k]; !dup {
				seen[k] = struct{}{}
				edges = append(edges, Edge{From: name, To: t})
			}
		}
	}

	return Graph{Nodes: nodes, Edges: edges}, nil
}

// packageDir derives a module-relative package directory from an absolute file
// path and the module path. For example, given:
//
//	path       = "/home/user/archfit/internal/model/model.go"
//	modulePath = "github.com/shibuiwilliam/archfit"
//
// The last element of modulePath is "archfit". We find that in the file path
// and take everything after it as the package directory: "internal/model".
func packageDir(path, modulePath string) string {
	// Use the last path element of the module as the anchor.
	moduleBase := filepath.Base(modulePath)
	// Normalize to forward slashes for consistent matching.
	normalized := filepath.ToSlash(path)
	dir := filepath.ToSlash(filepath.Dir(normalized))

	// Find the module base in the directory path.
	idx := -1
	parts := strings.Split(dir, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == moduleBase {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ""
	}

	// Everything after the module base directory is the package path.
	rel := strings.Join(parts[idx+1:], "/")
	if rel == "" {
		rel = "."
	}
	return rel
}
