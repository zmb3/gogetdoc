package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/doc"

	"golang.org/x/tools/go/packages"
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
func PackageDoc(from *packages.Package, importPath string) (*Doc, error) {
	pkg := from.Imports[importPath]
	if pkg == nil {
		return nil, fmt.Errorf("package %s not in import map of packages %v", importPath, from)
	}
	if len(pkg.Syntax) == 0 {
		return nil, errors.New("no documentation found for " + pkg.Name)
	}

	fileMap := make(map[string]*ast.File)
	for _, file := range pkg.Syntax {
		fileMap[pkg.Fset.File(file.Pos()).Name()] = file
	}
	astPkg := &ast.Package{
		Name:  pkg.Name,
		Files: fileMap,
	}

	docPkg := doc.New(astPkg, importPath, 0)
	// TODO: we could also include package-level constants, vars, and functions (like the go doc command)
	return &Doc{
		Name:   pkg.Name,
		Decl:   "package " + pkg.Name,
		Doc:    docPkg.Doc,
		Import: importPath,
		Pkg:    docPkg.Name,
	}, nil
}
