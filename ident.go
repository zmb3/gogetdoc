package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"

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
	// Render a copy of the node with no documentation. We
	// emit the documentation ourself.
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
				specCp := *spec
				specCp.Doc = nil
				cp.Specs = []ast.Spec{&specCp}
				cp.Lparen = 0
				cp.Rparen = 0
			case *ast.ValueSpec:
				spec := findVarSpec(n, obj.Pos())
				specCp := *spec
				specCp.Doc = nil
				cp.Specs = []ast.Spec{&specCp}
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
func IdentDoc(id *ast.Ident, info *loader.PackageInfo, prog *loader.Program) (*Doc, error) {
	// get definition of identifier
	obj := info.ObjectOf(id)
	pkgPath := ""
	if obj.Pkg() != nil {
		pkgPath = obj.Pkg().Path()
	}

	// handle packages imported under a different name
	if p, ok := obj.(*types.PkgName); ok {
		return PackageDoc(prog.Fset, p.Imported().Path())
	}

	_, nodes, _ := prog.PathEnclosingInterval(obj.Pos(), obj.Pos())
	if len(nodes) == 0 {
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
			Import: pkgPath,
			Name:   obj.Name(),
			Decl:   formatNode(node, obj, prog),
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
			// TODO "relative-to" output format...
			doc.Doc = n.Doc.Text()
			return doc, nil
		case *ast.GenDecl:
			constValue := ""
			if c, ok := obj.(*types.Const); ok {
				constValue = c.Val().ExactString()
			}
			if n.Doc != nil {
				if len(n.Specs) > 0 {
					switch n.Specs[0].(type) {
					case *ast.TypeSpec:
						doc.Doc = findTypeSpec(n, nodes[i-1].Pos()).Doc.Text()
					case *ast.ValueSpec:
						doc.Doc = findVarSpec(n, nodes[i-1].Pos()).Doc.Text()
					}
				}
				if doc.Doc == "" {
					doc.Doc = n.Doc.Text()
				}
				if constValue != "" {
					doc.Doc += fmt.Sprintf("\nConstant Value: %s", constValue)
				}
				return doc, nil
			}
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
