package main

import (
	"errors"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
)

// ImportPath gets the import path from an ImportSpec.
func ImportPath(is *ast.ImportSpec) string {
	s := is.Path.Value
	l := len(s)
	// trim the quotation marks
	return s[1 : l-1]
}

// PackageDoc gets the documentation for the package with the specified import
// path and writes it to out.
func PackageDoc(ctxt *build.Context, fset *token.FileSet, srcDir string, importPath string) (*Doc, error) {
	buildPkg, err := ctxt.Import(importPath, srcDir, build.ImportComment)
	if err != nil {
		return nil, err
	}
	// only parse .go files in the specified package
	filter := func(info os.FileInfo) bool {
		for _, fname := range buildPkg.GoFiles {
			if fname == info.Name() {
				return true
			}
		}
		return false
	}
	// TODO we've already parsed the files via go/loader...can we avoid doing it again?
	pkgs, err := parser.ParseDir(fset, buildPkg.Dir, filter, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if astPkg, ok := pkgs[buildPkg.Name]; ok {
		docPkg := doc.New(astPkg, importPath, 0)
		// TODO: we could also include package-level constants, vars, and functions (like the go doc command)
		return &Doc{
			Name:   buildPkg.Name,
			Decl:   "package " + buildPkg.Name,
			Doc:    docPkg.Doc,
			Import: importPath,
			Pkg:    docPkg.Name,
		}, nil
	}
	return nil, errors.New("No documentation found for " + buildPkg.Name)
}
