package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"golang.org/x/tools/go/loader"
)

const hello = `// Package main is an example package.
package main

import (
		"fmt"
		mth "math"
)

func main() {
  fmt.Println("Hello, World", mth.IsNaN(1.23))
}
`

func TestPackages(t *testing.T) {
	t.Parallel()
	conf := &loader.Config{
		ParserMode: parser.ParseComments,
	}
	astFile, err := conf.ParseFile("main.go", hello)
	if err != nil {
		t.Error(err)
	}

	conf.CreateFromFiles("main", astFile)
	prog, err := conf.Load()
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		Offset int64
		Doc    string
	}{
		{44, "Package main is an example package"},                               // package doc
		{49, "Package main is an example package"},                               // package doc
		{66, "\tPackage fmt implements formatted I/O"},                           // import spec
		{73, "Package math provides basic constants and mathematical functions"}, // aliased import
		{79, "Package math provides basic constants and mathematical functions"}, // aliased import
	}
	for _, test := range tests {
		d, err := DocForPos(prog, "main.go", test.Offset)
		if err != nil {
			t.Error(err)
			continue
		}
		if !strings.HasPrefix(d.Doc, test.Doc) {
			t.Errorf("Want '%s', got '%s'", test.Doc, d.Doc)
		}
	}
}

func TestImportPath(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "main.go", hello, parser.ImportsOnly)
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

func TestPackageDoc(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "main.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Error(err)
	}
	doc, err := PackageDoc(fset, "fmt")
	if err != nil {
		t.Error(err)
	}
	if !strings.HasPrefix(doc.Decl, "package fmt") {
		t.Errorf("Want 'package fmt', got %s\n", doc.Decl)
	}
}
