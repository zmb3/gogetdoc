package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestIdent(t *testing.T) {
	path := filepath.Join("./", "testdata", "idents.go")
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, path)
	if err != nil {
		t.Error(err)
	}
	if len(pkgs) != 1 {
		t.Errorf("Wanted 1 package for %s, got %d packages: %v", path, len(pkgs), pkgs)
	}

	tokFile := FileFromPkg(pkgs[0], path)
	if tokFile == nil {
		t.Fatal("Couldn't get token.File from program")
	}

	tests := []struct {
		Pos  int
		Doc  string
		Decl string
	}{
		{Pos: 190, Doc: "SayHello says hello.\n"},                                                              // method call
		{Pos: 202, Doc: "SayGoodbye says goodbye.\n"},                                                          // function call
		{Pos: 300, Doc: "Message is a message.\n"},                                                             // var (use)
		{Pos: 382, Doc: "Message is a message.\n"},                                                             // var (definition)
		{Pos: 314, Doc: "Sprintf formats according to a format specifier and returns the resulting string.\n"}, // std func
		{Pos: 342, Doc: "Answer is the answer to life the universe and everything.\n\nConstant Value: 42"},     // const (use)
		{Pos: 477, Doc: "Answer is the answer to life the universe and everything.\n\nConstant Value: 42"},     // const (definition)
		{Pos: 143, Doc: "IsNaN reports whether f is an IEEE 754 ``not-a-number'' value.\n"},                    // std func call (alias import)

		// field doc/comment precedence
		{Pos: 623, Doc: "FieldA has doc\n"},
		{Pos: 632, Doc: "FieldB has a comment\n"},

		// GenDecl doc/comment precedence
		{Pos: 984, Doc: "Alpha doc"},
		{Pos: 999, Doc: "Bravo comment"},

		// builtins
		{Pos: 942, Doc: "The error built-in interface type is the conventional"},
		{Pos: 702, Doc: "The append built-in function appends elements to the end"},
		{Pos: 730, Doc: "float32 is the set of all IEEE-754 32-bit floating-point numbers."},
		{Pos: 788, Doc: "iota is a predeclared identifier representing the untyped integer ordinal"},
		{Pos: 831, Doc: "nil is a predeclared identifier representing the zero"},
		{Pos: 881, Doc: "The len built-in function returns the length of v"},
		{Pos: 917, Doc: "The close built-in function closes a channel, which must"},

		// decl
		{Pos: 591, Decl: "type Foo struct {"},
	}

	for _, test := range tests {
		t.Run(test.Doc, func(t *testing.T) {
			pos := tokFile.Pos(test.Pos)
			info, nodes := pathEnclosingInterval(pkgs[0], pos, pos)
			for i := range nodes {
				if ident, ok := nodes[i].(*ast.Ident); ok {
					doc, err := IdentDoc(ident, info, pkgs[0])
					if err != nil {
						t.Fatal(err)
					}
					if !strings.HasPrefix(doc.Doc, test.Doc) {
						t.Errorf("Want %q, got %q\n", test.Doc, doc.Doc)
					}
					if test.Decl != "" && !strings.HasPrefix(doc.Decl, test.Decl) {
						t.Errorf("Decl: want %q, got %q\n", test.Decl, doc.Decl)
					}
					return
				}
			}
			t.Errorf("Couldn't find *ast.Ident at %v\n", test.Pos)
		})
	}
}

func TestModified(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("testdata", "const.go"))
	if err != nil {
		t.Fatal(err)
	}
	contents := `package main

import "fmt"

const (
	Zero = iota
	One
	Two
)

const Three = 3

func main() {
	fmt.Println(Zero, One, Three, Three)
}`
	archive := fmt.Sprintf("%s\n%d\n%s", path, len(contents), contents)

	archiveReader = bytes.NewBufferString(archive)
	defer func() { archiveReader = os.Stdin }()

	d, err := Run(path, int64(118), true)
	if err != nil {
		t.Fatal(err)
	}
	if n := d.Name; n != "Three" {
		t.Errorf("got const %s, want Three", n)
	}
}

func TestConstantValue(t *testing.T) {
	path := filepath.Join("testdata", "const.go")
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, path)
	if err != nil {
		t.Error(err)
	}
	if len(pkgs) != 1 {
		t.Errorf("Wanted 1 package for %s, got %d packages: %v", path, len(pkgs), pkgs)
	}
	prog := pkgs[0]

	for _, offset := range []int64{107, 113, 118, 125} {
		doc, err := DocForPos(prog, path, offset)
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(doc.Doc, "Constant Value:") {
			t.Errorf("Expected doc to contain constant value: %q", doc.Doc)
		}
	}
}

func TestUnexportedFields(t *testing.T) {
	path := filepath.Join("testdata", "idents.go")
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, path)
	if err != nil {
		t.Error(err)
	}
	if len(pkgs) != 1 {
		t.Errorf("Wanted 1 package for %s, got %d packages: %v", path, len(pkgs), pkgs)
	}
	prog := pkgs[0]

	for _, showUnexported := range []bool{true, false} {
		*showUnexportedFields = showUnexported
		doc, err := DocForPos(prog, path, 1051)
		if err != nil {
			t.Error(err)
		}
		hasUnexportedField := strings.Contains(doc.Decl, "notVisible")
		if hasUnexportedField != *showUnexportedFields {
			t.Errorf("show unexported fields is %v, but got %q", showUnexported, doc.Decl)
		}
	}
}

func TestEmbeddedTypes(t *testing.T) {
	cleanup, err := tempGopath("embed.go", "embed")
	if err != nil {
		t.Fatal(err)
	}

	if cleanup != nil {
		defer cleanup()
	}

	tests := []struct {
		description string
		offset      int64
		want        string
	}{
		{"embedded value", 75, "foo doc\n"},
		{"embedded pointer", 112, "foo doc\n"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			doc, err := Run("embed.go", test.offset, false)
			if err != nil {
				t.Fatal(err)
			}
			if doc.Doc != test.want {
				t.Errorf("want %q, got %q", test.want, doc.Doc)
			}
			if doc.Pkg != "embed" {
				t.Errorf("want package embed, got %q", doc.Pkg)
			}
		})
	}
}

func TestIssue20(t *testing.T) {
	cleanup, err := tempGopath("issue20.go", "foo")
	if err != nil {
		t.Fatal(err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	tests := []struct {
		desc   string
		want   string
		offset int64
	}{
		{"named type", "var words []string", 114},
		{"unnamed type", "var tests []struct{Name string; args string}", 281},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			doc, err := Run("issue20.go", test.offset, false)
			if err != nil {
				t.Fatal(err)
			}

			if doc.Decl != test.want {
				t.Errorf("want %s, got %s", test.want, doc.Decl)
			}

			if doc.Doc != "" {
				t.Errorf("expect doc to be empty, but got %q", doc.Doc)
			}
		})
	}
}

func TestVendoredIdent(t *testing.T) {
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
	defer func() {
		os.RemoveAll(gopath)
	}()

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

	doc, err := Run("main.go", 63, false)
	if err != nil {
		t.Fatal(err)
	}

	want := "github.com/zmb3/vp"
	if doc.Import != want {
		t.Errorf("want %s, got %s", want, doc.Import)
	}
}

func tempGopathDir() (string, func(), error) {
	gopath, err := ioutil.TempDir("", "gogetdoc")
	if err != nil {
		return "", nil, err
	}
	oldgopath := os.Getenv("GOPATH")
	os.Setenv("GOPATH", gopath)
	cleanup := func() {
		os.Setenv("GOPATH", oldgopath)
		os.RemoveAll(gopath)
	}

	return gopath, cleanup, err
}

func tempGopath(filename, pkg string) (cleanup func(), err error) {
	gopath, gopathcleanup, err := tempGopathDir()
	if err != nil {
		return nil, err
	}

	pkgDir := filepath.Join(gopath, "src", "github.com", "zmb3", pkg)
	err = os.MkdirAll(pkgDir, 0755)
	if err != nil {
		os.RemoveAll(gopath)
		return nil, err
	}

	err = copyFile(filepath.Join(pkgDir, filename), filepath.FromSlash("./testdata/"+filename))
	if err != nil {
		os.RemoveAll(gopath)
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		os.RemoveAll(gopath)
		return nil, err
	}
	err = os.Chdir(pkgDir)
	if err != nil {
		os.RemoveAll(gopath)
		return nil, err
	}

	cleanup = func() {
		os.Chdir(cwd)
		gopathcleanup()
	}
	return cleanup, nil
}

func copyFile(dst, src string) error {
	orig, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copying file %s: %v", src, err)
	}
	defer orig.Close()

	copy, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating copy %s: %v", dst, err)
	}
	defer copy.Close()

	_, err = io.Copy(copy, orig)
	return err
}
