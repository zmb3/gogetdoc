package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/loader"
)

// IdentDoc attempts to get the documentation for a *ast.Ident.
func IdentDoc(id *ast.Ident, info *loader.PackageInfo, prog *loader.Program) (*Doc, error) {
	// get definition of identifier
	obj := info.ObjectOf(id)

	// handle packages imported under a different name
	if p, ok := obj.(*types.PkgName); ok {
		return PackageDoc(prog.Fset, p.Imported().Path())
	}

	_, nodes, _ := prog.PathEnclosingInterval(obj.Pos(), obj.Pos())
	for _, node := range nodes {
		//fmt.Printf("for %s: found %T\n%#v\n", id.Name, node, node)
		switch n := node.(type) {
		case *ast.FuncDecl:
			return &Doc{
				Name:  obj.Name(),
				Title: obj.String(), // TODO "relative-to" output format...
				Doc:   n.Doc.Text(),
			}, nil
		case *ast.GenDecl:
			var constValue string
			if n.Tok == token.CONST {
			SpecLoop:
				for _, s := range n.Specs {
					vs := s.(*ast.ValueSpec)
					for _, val := range vs.Values {
						if bl, ok := val.(*ast.BasicLit); ok {
							if bl.Value != "" {
								constValue = bl.Value
								break SpecLoop
							}
						}
					}
				}
			}
			if n.Doc != nil {
				d := &Doc{
					Name:  obj.Name(),
					Title: obj.String(),
					Doc:   n.Doc.Text(),
				}
				if constValue != "" {
					d.Doc += fmt.Sprintf("\nConstant Value: %s", constValue)
				}
				return d, nil
			}
		case *ast.Field:
			// check the doc first, if not present, then look for a comment
			if n.Doc != nil {
				return &Doc{
					Name:  obj.Name(),
					Title: obj.String(),
					Doc:   n.Doc.Text(),
				}, nil
			} else if n.Comment != nil {
				return &Doc{
					Name:  obj.Name(),
					Title: obj.String(),
					Doc:   n.Comment.Text(),
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("No documentation found for %s", obj.Name())
}
