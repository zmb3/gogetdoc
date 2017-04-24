package main

import (
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
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

const hello2 = `package main

import "fmt"

func Hello() {
	fmt.Println("hello")
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
		{66, "\tPackage fmt implements formatted I/O"},                           // import spec
		{73, "Package math provides basic constants and mathematical functions"}, // aliased import
		{79, "Package math provides basic constants and mathematical functions"}, // aliased import
	}
	for _, test := range tests {
		d, err := DocForPos(&build.Default, prog, "main.go", test.Offset)
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
	doc, err := PackageDoc(&build.Default, fset, ".", "fmt")
	if err != nil {
		t.Error(err)
	}
	if !strings.HasPrefix(doc.Decl, "package fmt") {
		t.Errorf("Want 'package fmt', got %s\n", doc.Decl)
	}
	if doc.Import != "fmt" {
		t.Errorf("want import \"fmt\", got %q", doc.Import)
	}
}

func TestPackageDocDecl(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "main.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Error(err)
	}
	doc, err := PackageDoc(&build.Default, fset, ".", "fmt")
	if err != nil {
		t.Error(err)
	}
	if !strings.HasPrefix(doc.Decl, "package") {
		t.Errorf("package decl must always start with \"package\", got %q", doc.Decl)
	}
}

func TestVendoredPackageImport(t *testing.T) {
	newGopath, err := ioutil.TempDir(".", "gogetdoc-gopath")
	if err != nil {
		t.Fatal(err)
	}
	newGopath, _ = filepath.Abs(newGopath)
	progDir := filepath.Join(newGopath, "src", "github.com", "zmb3", "prog")
	pkgDir := filepath.Join(progDir, "vendor", "github.com", "zmb3", "vp")

	err = os.MkdirAll(pkgDir, 0755)
	if err != nil {
		t.Fatal(err)
	} else {
		defer func() {
			os.RemoveAll(newGopath)
		}()
	}

	err = copyFile(filepath.Join(progDir, "main.go"), filepath.FromSlash("./testdata/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	err = copyFile(filepath.Join(pkgDir, "vp.go"), filepath.FromSlash("./testdata/vp.go"))
	if err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chdir(progDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.Chdir(cwd)
	}()

	ctx := build.Default
	ctx.GOPATH = newGopath
	doc, err := Run(&ctx, "main.go", 39)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Decl != "package vp" {
		t.Errorf("want 'package vp', got '%s'", doc.Decl)
	}
}
