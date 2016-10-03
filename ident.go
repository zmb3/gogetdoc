package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/loader"
)

func findTypeSpec(decl *ast.GenDecl, pos token.Pos) *ast.TypeSpec {
	for _, spec := range decl.Specs {
		typeSpec := spec.(*ast.TypeSpec)
		if typeSpec.Pos() == pos {
			return typeSpec
		}
	}
	return nil
}

func findVarSpec(decl *ast.GenDecl, pos token.Pos) *ast.ValueSpec {
	for _, spec := range decl.Specs {
		varSpec := spec.(*ast.ValueSpec)
		for _, ident := range varSpec.Names {
			if ident.Pos() == pos {
				return varSpec
			}
		}
	}
	return nil
}

func formatNode(n ast.Node, obj types.Object, prog *loader.Program) string {
	var nc ast.Node
	// Render a copy of the node with no documentation.
	// We emit the documentation ourself.
	switch n := n.(type) {
	case *ast.FuncDecl:
		cp := *n
		cp.Doc = nil
		// Don't print the whole function body
		cp.Body = nil
		nc = &cp
	case *ast.GenDecl:
		cp := *n
		cp.Doc = nil
		if len(n.Specs) > 0 {
			// Only print this one type, not all the types in the
			// gendecl
			switch n.Specs[0].(type) {
			case *ast.TypeSpec:
				spec := findTypeSpec(n, obj.Pos())
				if spec != nil {
					specCp := *spec
					if *showUnexportedFields == false {
						trimUnexportedElems(&specCp)
					}
					specCp.Doc = nil
					cp.Specs = []ast.Spec{&specCp}
				}
				cp.Lparen = 0
				cp.Rparen = 0
			case *ast.ValueSpec:
				spec := findVarSpec(n, obj.Pos())
				if spec != nil {
					specCp := *spec
					specCp.Doc = nil
					cp.Specs = []ast.Spec{&specCp}
				}
				cp.Lparen = 0
				cp.Rparen = 0
			}
		}
		nc = &cp
	case *ast.Field:
		// Not supported by go/printer

		// TODO(dominikh): Methods in interfaces are syntactically
		// represented as fields. Using types.Object.String for those
		// causes them to look different from real functions.
		// go/printer doesn't include the import paths in names, while
		// Object.String does. Fix that.

		return obj.String()
	default:
		return obj.String()
	}

	buf := &bytes.Buffer{}
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	err := cfg.Fprint(buf, prog.Fset, nc)
	if err != nil {
		return obj.String()
	}
	return buf.String()
}

// IdentDoc attempts to get the documentation for a *ast.Ident.
func IdentDoc(ctxt *build.Context, id *ast.Ident, info *loader.PackageInfo, prog *loader.Program) (*Doc, error) {
	// get definition of identifier
	obj := info.ObjectOf(id)
	var pos string
	if p := obj.Pos(); p.IsValid() {
		pos = prog.Fset.Position(p).String()
	}

	pkgPath := ""
	if obj.Pkg() != nil {
		pkgPath = obj.Pkg().Path()
	}

	// handle packages imported under a different name
	if p, ok := obj.(*types.PkgName); ok {
		return PackageDoc(ctxt, prog.Fset, "", p.Imported().Path()) // SRCDIR TODO TODO
	}

	_, nodes, _ := prog.PathEnclosingInterval(obj.Pos(), obj.Pos())
	if len(nodes) == 0 {
		// special case - builtins
		doc, decl := findInBuiltin(obj.Name(), obj, prog)
		if doc != "" {
			return &Doc{
				Import: "builtin",
				Name:   obj.Name(),
				Doc:    doc,
				Decl:   decl,
				Pos:    pos,
			}, nil
		}
		return nil, fmt.Errorf("No documentation found for %s", obj.Name())
	}
	var doc *Doc
	for _, node := range nodes {
		switch node.(type) {
		case *ast.FuncDecl, *ast.GenDecl, *ast.Field:
		default:
			continue
		}
		doc = &Doc{
			Import: stripVendorFromImportPath(pkgPath),
			Name:   obj.Name(),
			Decl:   formatNode(node, obj, prog),
			Pos:    pos,
		}
		break
	}
	if doc == nil {
		// This shouldn't happen
		return nil, fmt.Errorf("No documentation found for %s", obj.Name())
	}

	for i, node := range nodes {
		//fmt.Printf("for %s: found %T\n%#v\n", id.Name, node, node)
		switch n := node.(type) {
		case *ast.FuncDecl:
			doc.Doc = n.Doc.Text()
			return doc, nil
		case *ast.GenDecl:
			constValue := ""
			if c, ok := obj.(*types.Const); ok {
				constValue = c.Val().ExactString()
			}

			if len(n.Specs) > 0 {
				switch n.Specs[0].(type) {
				case *ast.TypeSpec:
					spec := findTypeSpec(n, nodes[i-1].Pos())
					if spec.Doc != nil {
						doc.Doc = spec.Doc.Text()
					} else if spec.Comment != nil {
						doc.Doc = spec.Comment.Text()
					}
				case *ast.ValueSpec:
					spec := findVarSpec(n, nodes[i-1].Pos())
					if spec.Doc != nil {
						doc.Doc = spec.Doc.Text()
					} else if spec.Comment != nil {
						doc.Doc = spec.Comment.Text()
					}
				}
			}

			// if we didn't find doc for the spec, check the overall GenDecl
			if doc.Doc == "" && n.Doc != nil {
				doc.Doc = n.Doc.Text()
			}
			if constValue != "" {
				doc.Doc += fmt.Sprintf("\nConstant Value: %s", constValue)
			}

			return doc, nil
		case *ast.Field:
			// check the doc first, if not present, then look for a comment
			if n.Doc != nil {
				doc.Doc = n.Doc.Text()
				return doc, nil
			} else if n.Comment != nil {
				doc.Doc = n.Comment.Text()
				return doc, nil
			}
		}
	}
	return doc, nil
}

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

func stripVendorFromImportPath(ip string) string {
	vendor := "/vendor/"
	l := len(vendor)
	if i := strings.LastIndex(ip, vendor); i != -1 {
		return ip[i+l:]
	}
	return ip
}
