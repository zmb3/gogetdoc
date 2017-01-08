package main

import (
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"go/types"
	"os"

	"golang.org/x/tools/go/loader"
)

func builtinPackage() *doc.Package {
	buildPkg, err := build.Import("builtin", "", build.ImportComment)
	// should never fail
	if err != nil {
		panic(err)
	}
	include := func(info os.FileInfo) bool {
		return info.Name() == "builtin.go"
	}
	fs := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fs, buildPkg.Dir, include, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	astPkg := astPkgs["builtin"]
	return doc.New(astPkg, buildPkg.ImportPath, doc.AllDecls)
}

// findInBuiltin searches for an identifier in the builtin package.
// It searches in the following order: funcs, constants and variables,
// and finally types.
func findInBuiltin(name string, obj types.Object, prog *loader.Program) (docstring, decl string) {
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
