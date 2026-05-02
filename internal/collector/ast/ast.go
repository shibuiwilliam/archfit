// Package ast collects AST-derived facts from source files. See ADR 0015.
//
// Phase 1 supports Go only (via goast sub-package). Tree-sitter for
// cross-language support is deferred to Phase 1.5.
//
// The collector walks Go files, delegates parsing to goast.ParseGoFile,
// and aggregates results into model.ASTFacts. Parse failures are recorded
// (never silently skipped) so resolvers can emit ParseFailure findings.
package ast

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shibuiwilliam/archfit/internal/collector/ast/goast"
	"github.com/shibuiwilliam/archfit/internal/model"
)

// MaxGoFiles is the cap on Go files to parse. Beyond this the collector stops
// and records a single parse failure summarizing the skip.
const MaxGoFiles = 10000

// Collect walks root for Go source files and returns AST-derived facts.
// depth is "standard" or "deep" (caller should not pass "shallow" — the
// scheduler skips the AST collector entirely for shallow scans).
func Collect(root, depth string) (model.ASTFacts, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return model.ASTFacts{}, err
	}

	var goPaths []string
	err = filepath.WalkDir(abs, func(p string, d os.DirEntry, werr error) error {
		if werr != nil {
			return nil // skip unreadable entries
		}
		// Skip hidden directories and common non-source directories.
		if d.IsDir() {
			base := filepath.Base(p)
			if strings.HasPrefix(base, ".") || base == "vendor" || base == "node_modules" || base == "testdata" || base == "fixtures" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) == ".go" && !strings.HasSuffix(p, "_test.go") {
			goPaths = append(goPaths, p)
		}
		return nil
	})
	if err != nil {
		return model.ASTFacts{}, err
	}

	// Deterministic order.
	sort.Strings(goPaths)

	var facts model.ASTFacts

	// Cap check.
	if len(goPaths) > MaxGoFiles {
		facts.ParseFailures = append(facts.ParseFailures, model.ASTParseFailure{
			Path:  "",
			Error: "Go file count exceeds limit; parsed first 10000, skipped remainder",
		})
		goPaths = goPaths[:MaxGoFiles]
	}

	for _, absPath := range goPaths {
		relPath, _ := filepath.Rel(abs, absPath)
		relPath = filepath.ToSlash(relPath)

		gf, perr := goast.ParseGoFile(absPath, relPath, depth)
		if perr != nil {
			facts.ParseFailures = append(facts.ParseFailures, model.ASTParseFailure{
				Path:  relPath,
				Error: perr.Error(),
			})
			continue
		}
		facts.GoFiles = append(facts.GoFiles, gf)
	}

	return facts, nil
}
