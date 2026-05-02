// Package goast parses Go source files using go/parser and extracts
// structural facts for archfit rules. See ADR 0015.
//
// Two modes:
//   - standard: declaration-level (init functions, pkg-level vars, interfaces, imports).
//   - deep: full body analysis (reflect call counting, cross-pkg calls in init()).
package goast

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/model"
)

// MaxFileSize is the hard limit for a single Go file. Files larger than this
// are skipped with a parse failure entry. Default: 1 MiB.
const MaxFileSize = 1 << 20 // 1 MiB

// ParseGoFile parses a single Go source file and returns structured facts.
// depth is "standard" or "deep". The path should be absolute or repo-relative
// for consistent reporting.
func ParseGoFile(absPath, relPath, depth string) (model.GoFileFacts, error) {
	// Check file size before reading.
	info, err := os.Stat(absPath)
	if err != nil {
		return model.GoFileFacts{}, fmt.Errorf("stat %s: %w", relPath, err)
	}
	if info.Size() > MaxFileSize {
		return model.GoFileFacts{}, fmt.Errorf("file exceeds size limit (%d bytes > %d)", info.Size(), MaxFileSize)
	}

	// Parse mode: declarations only for standard, full for deep.
	mode := parser.ParseComments | parser.SkipObjectResolution
	if depth != "deep" {
		mode |= parser.AllErrors // still want errors even in standard
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, absPath, nil, mode)
	if err != nil {
		return model.GoFileFacts{}, fmt.Errorf("parse error: %w", err)
	}

	facts := model.GoFileFacts{
		Path:    relPath,
		Package: f.Name.Name,
	}

	// Extract declarations.
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == "init" && d.Recv == nil {
				initFact := model.InitFact{
					Line: fset.Position(d.Pos()).Line,
				}
				if depth == "deep" && d.Body != nil {
					initFact.CrossPkgCalls = extractCrossPkgCalls(d.Body)
				}
				facts.InitFunctions = append(facts.InitFunctions, initFact)
			}
		case *ast.GenDecl:
			switch d.Tok {
			case token.VAR:
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range vs.Names {
						facts.PkgLevelVars = append(facts.PkgLevelVars, model.PkgVarFact{
							Name:    name.Name,
							Line:    fset.Position(name.Pos()).Line,
							Mutable: true,
							Type:    typeString(vs.Type),
						})
					}
				}
			case token.TYPE:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					iface, ok := ts.Type.(*ast.InterfaceType)
					if !ok {
						continue
					}
					methodCount := 0
					if iface.Methods != nil {
						methodCount = len(iface.Methods.List)
					}
					facts.Interfaces = append(facts.Interfaces, model.InterfaceFact{
						Name:        ts.Name.Name,
						Line:        fset.Position(ts.Pos()).Line,
						MethodCount: methodCount,
					})
				}
			}
		}
	}

	// Check for reflect import.
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path == "reflect" {
			facts.ReflectImports = true
			break
		}
	}

	// Deep mode: count reflect.* calls in the entire file.
	if depth == "deep" && facts.ReflectImports {
		facts.ReflectCalls = countReflectCalls(f)
	}

	return facts, nil
}

// extractCrossPkgCalls walks an init() body and returns qualified calls
// like "http.HandleFunc", "sql.Register", etc.
func extractCrossPkgCalls(body *ast.BlockStmt) []string {
	var calls []string
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		calls = append(calls, ident.Name+"."+sel.Sel.Name)
		return true
	})
	sort.Strings(calls)
	return calls
}

// countReflectCalls counts call expressions on the "reflect" package identifier.
func countReflectCalls(f *ast.File) int {
	count := 0
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "reflect" {
			count++
		}
		return true
	})
	return count
}

// typeString returns a best-effort string representation of a type expression.
// Returns "" for nil or complex types that aren't worth stringifying.
func typeString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	case *ast.StarExpr:
		if inner := typeString(t.X); inner != "" {
			return "*" + inner
		}
	case *ast.ArrayType:
		if inner := typeString(t.Elt); inner != "" {
			return "[]" + inner
		}
	case *ast.MapType:
		k := typeString(t.Key)
		v := typeString(t.Value)
		if k != "" && v != "" {
			return "map[" + k + "]" + v
		}
	}
	return ""
}
