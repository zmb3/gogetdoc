package main

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackages(t *testing.T) {
	cleanup := setGopath(filepath.Join(".", "testdata", "package-doc"), t)
	defer cleanup()

	filename := filepath.Join(".", "testdata", "package-doc", "src", "prog", "main.go")
	tests := []struct {
		Offset int
		Doc    string
	}{
		{107, "\tPackage fmt implements formatted I/O"},                           // import spec
		{113, "Package math provides basic constants and mathematical functions"}, // aliased import
		{118, "Package math provides basic constants and mathematical functions"}, // aliased import
	}
	for _, test := range tests {
		d, err := Run(filename, test.Offset, nil)
		if err != nil {
			t.Error(err)
			continue
		}
		if !strings.HasPrefix(d.Doc, test.Doc) {
			t.Errorf("offset %v: Want '%s', got '%s'", test.Offset, test.Doc, d.Doc)
		}
		if !strings.HasPrefix(d.Decl, "package") {
			t.Errorf("package decl must always start with \"package\", got %q", d.Decl)
		}
	}
}

func TestImportPath(t *testing.T) {
	fset := token.NewFileSet()
	filename := filepath.Join(".", "testdata", "package-doc", "src", "prog", "main.go")
	f, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		t.Error(err)
	}
	if len(f.Imports) != 2 {
		t.Errorf("Want 2 imports, got %d\n", len(f.Imports))
	}
	ip := ImportPath(f.Imports[0])
	if ip != "fmt" {
		t.Errorf("Want 'fmt', got '%s'\n", ip)
	}
}

func TestVendoredPackageImport(t *testing.T) {
	cleanup := setGopath(filepath.Join(".", "testdata", "withvendor"), t)
	defer cleanup()

	filename := filepath.Join(".", "testdata", "withvendor", "src", "main", "main.go")

	doc, err := Run(filename, 39, nil)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Decl != "package vp" {
		t.Errorf("want 'package vp', got '%s'", doc.Decl)
	}
	if doc.Import != "github.com/zmb3/vp" {
		t.Errorf("want 'github.com/zmb3/vp', got %q", doc.Import)
	}
}
