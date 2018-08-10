package main

import (
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
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
	newdir, err := ioutil.TempDir(".", "gogetdoc-gopath")
	if err != nil {
		t.Fatal(err)
	}
	newdir, _ = filepath.Abs(newdir)
	defer os.RemoveAll(newdir)

	filename := filepath.Join(newdir, "main.go")
	ioutil.WriteFile(filename, []byte(hello), 0666)

	tests := []struct {
		Offset int64
		Doc    string
	}{
		{66, "\tPackage fmt implements formatted I/O"},                           // import spec
		{73, "Package math provides basic constants and mathematical functions"}, // aliased import
		{79, "Package math provides basic constants and mathematical functions"}, // aliased import
	}
	for _, test := range tests {
		// Reload the packages for each test, since DocForPos changes the ast files.
		pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, filename)
		if err != nil {
			log.Fatal(err)
		}
		if len(pkgs) != 1 {
			t.Errorf("Wanted 1 package for main.go, got %d packages: %v", len(pkgs), pkgs)
		}

		d, err := DocForPos(pkgs[0], filename, test.Offset)
		if err != nil {
			t.Error(err)
			continue
		}
		if !strings.HasPrefix(d.Doc, test.Doc) {
			t.Errorf("offset %v: Want '%s', got '%s'", test.Offset, test.Doc, d.Doc)
		}
	}
}

func TestImportPath(t *testing.T) {
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
	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Error: func(error) {}, // suppress parsing errors
	}
	pkgs, err := packages.Load(cfg, "main.go")
	if err != nil {
		t.Error(err)
	}
	if len(pkgs) != 1 {
		t.Errorf("Wanted 1 package for main.go, got %d packages: %v", len(pkgs), pkgs)
	}

	doc, err := PackageDoc(pkgs[0], "fmt")
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
	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Error: func(error) {}, // suppress parsing errors
	}
	pkgs, err := packages.Load(cfg, "main.go")
	if err != nil {
		t.Error(err)
	}
	if len(pkgs) != 1 {
		t.Errorf("Wanted 1 package for main.go, got %d packages: %v", len(pkgs), pkgs)
	}

	doc, err := PackageDoc(pkgs[0], "fmt")
	if err != nil {
		t.Error(err)
	}
	if !strings.HasPrefix(doc.Decl, "package") {
		t.Errorf("package decl must always start with \"package\", got %q", doc.Decl)
	}
}

func TestVendoredPackageImport(t *testing.T) {
	gopath, cleanup, err := tempGopathDir()
	if err != nil {
		t.Fatal(err)
	}

	defer cleanup()

	progDir := filepath.Join(gopath, "src", "github.com", "zmb3", "prog")
	pkgDir := filepath.Join(progDir, "vendor", "github.com", "zmb3", "vp")

	err = os.MkdirAll(pkgDir, 0755)
	if err != nil {
		t.Fatal(err)
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

	doc, err := Run("main.go", 39)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Decl != "package vp" {
		t.Errorf("want 'package vp', got '%s'", doc.Decl)
	}
}
