package main

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
)

func builtinPackage() *doc.Package {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadFiles}, "builtin")
	if err != nil {
		log.Fatalf("error getting metadata of builtin: %v", err)
	}
	pkg := pkgs[0]

	fs := token.NewFileSet()
	fileMap := make(map[string]*ast.File)
	for _, filename := range pkg.GoFiles {
		file, err := parser.ParseFile(fs, filename, nil, parser.ParseComments)
		if err != nil {
			log.Fatal(err)
		}
		fileMap[filename] = file
	}

	astPkg := &ast.Package{
		Name:  pkg.Name,
		Files: fileMap,
	}
	return doc.New(astPkg, "builtin", doc.AllDecls)
}

// findInBuiltin searches for an identifier in the builtin package.
// It searches in the following order: funcs, constants and variables,
// and finally types.
func findInBuiltin(name string, obj types.Object, prog *packages.Package) (docstring, decl string) {
	pkg := builtinPackage()

	consts := make([]*doc.Value, 0, 2*len(pkg.Consts))
	vars := make([]*doc.Value, 0, 2*len(pkg.Vars))
	funcs := make([]*doc.Func, 0, 2*len(pkg.Funcs))

	consts = append(consts, pkg.Consts...)
	vars = append(vars, pkg.Vars...)
	funcs = append(funcs, pkg.Funcs...)

	for _, t := range pkg.Types {
		funcs = append(funcs, t.Funcs...)
		consts = append(consts, t.Consts...)
		vars = append(vars, t.Vars...)
	}

	// funcs
	for _, f := range funcs {
		if f.Name == name {
			return f.Doc, formatNode(f.Decl, obj, prog)
		}
	}

	// consts/vars
	for _, v := range consts {
		for _, n := range v.Names {
			if n == name {
				return v.Doc, ""
			}
		}
	}

	for _, v := range vars {
		for _, n := range v.Names {
			if n == name {
				return v.Doc, ""
			}
		}
	}

	// types
	for _, t := range pkg.Types {
		if t.Name == name {
			return t.Doc, formatNode(t.Decl, obj, prog)
		}
	}

	return "", ""
}
