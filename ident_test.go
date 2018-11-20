package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/buildutil"
)

func TestIdent(t *testing.T) {
	cleanup := setGopath(filepath.Join(".", "testdata", "package"), t)
	defer cleanup()
	filename := filepath.Join(".", "testdata", "package", "src", "somepkg", "idents.go")

	tests := []struct {
		Pos  int
		Doc  string
		Decl string
	}{
		{Pos: 146, Doc: "IsNaN reports whether f is an IEEE 754 ``not-a-number'' value.\n", Decl: "func IsNaN(f float64) (is bool)"},         // std func call (alias import)
		{Pos: 190, Doc: "SayHello says hello.\n", Decl: "func (X) SayHello()"},                                                               // method call
		{Pos: 202, Doc: "SayGoodbye says goodbye.\n", Decl: "func SayGoodbye() (string, error)"},                                             // function call
		{Pos: 319, Doc: "Message is a message.\n", Decl: "var Message string"},                                                               // var (use)
		{Pos: 415, Doc: "Message is a message.\n"},                                                                                           // var (definition)
		{Pos: 329, Doc: "Sprintf formats according to a format specifier and returns the resulting string.\n"},                               // std func
		{Pos: 358, Doc: "Answer is the answer to life the universe and everything.\n\nConstant Value: 42", Decl: "const Answer untyped int"}, // const (use)
		{Pos: 510, Doc: "Answer is the answer to life the universe and everything.\n\nConstant Value: 42"},                                   // const (definition)

		// field doc/comment precedence
		{Pos: 656, Doc: "FieldA has doc\n", Decl: "field FieldA string"},
		{Pos: 665, Doc: "FieldB has a comment\n"},

		// GenDecl doc/comment precedence
		{Pos: 1017, Doc: "Alpha doc", Decl: "var Alpha int"},
		{Pos: 1032, Doc: "Bravo comment", Decl: "var Bravo int"},

		// builtins
		{Pos: 975, Doc: "The error built-in interface type is the conventional"},
		{Pos: 735, Doc: "The append built-in function appends elements to the end", Decl: "func append(slice []Type, elems ...Type) []Type"},
		{Pos: 762, Doc: "float32 is the set of all IEEE-754 32-bit floating-point numbers."},
		{Pos: 821, Doc: "iota is a predeclared identifier representing the untyped integer ordinal"},
		{Pos: 864, Doc: "nil is a predeclared identifier representing the zero"},
		{Pos: 914, Doc: "The len built-in function returns the length of v"},
		{Pos: 950, Doc: "The close built-in function closes a channel, which must"},

		// type spec
		{Pos: 53, Decl: "type X struct{}"},
		{Pos: 1222, Decl: "type NewString string"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("ident pos %d", test.Pos), func(t *testing.T) {
			doc, err := Run(filename, test.Pos, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(doc.Doc, test.Doc) {
				t.Errorf("Want %q, got %q\n", test.Doc, doc.Doc)
			}
			if test.Decl != "" && doc.Decl != test.Decl {
				t.Errorf("Decl: want %q, got %q\n", test.Decl, doc.Decl)
			}
		})
	}
}

func TestModified(t *testing.T) {
	cleanup := setGopath(filepath.Join(".", "testdata", "package"), t)
	defer cleanup()

	filename := filepath.Join(".", "testdata", "package", "src", "somepkg", "const.go")
	path, err := filepath.Abs(filename)
	if err != nil {
		t.Fatal(err)
	}
	contents := `package somepkg

import "fmt"

const (
	Zero = iota
	One
	Two
)

const Three = 3

func main() {
	fmt.Println(Zero, Three, Two, Three)
}
`
	archive := fmt.Sprintf("%s\n%d\n%s", path, len(contents), contents)
	overlay, err := buildutil.ParseOverlayArchive(strings.NewReader(archive))
	if err != nil {
		t.Fatalf("couldn't parse overlay: %v", err)
	}

	d, err := Run(path, 118, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if n := d.Name; n != "Three" {
		t.Errorf("got const %s, want Three", n)
	}
}

func TestConstantValue(t *testing.T) {
	cleanup := setGopath(filepath.Join(".", "testdata", "package"), t)
	defer cleanup()
	filename := filepath.Join(".", "testdata", "package", "src", "somepkg", "const.go")

	for _, offset := range []int{111, 116, 121, 128} {
		doc, err := Run(filename, offset, nil)
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(doc.Doc, "Constant Value:") {
			t.Errorf("Expected doc to contain constant value: %q", doc.Doc)
		}
	}
}

func TestUnexportedFields(t *testing.T) {
	cleanup := setGopath(filepath.Join(".", "testdata", "package"), t)
	defer cleanup()
	filename := filepath.Join(".", "testdata", "package", "src", "somepkg", "idents.go")

	for _, showUnexported := range []bool{true, false} {
		*showUnexportedFields = showUnexported
		doc, err := Run(filename, 1085, nil)
		if err != nil {
			t.Fatalf("showUnexportedFields=%v: %v", showUnexported, err)
		}
		hasUnexportedField := strings.Contains(doc.Decl, "notVisible")
		if hasUnexportedField != *showUnexportedFields {
			t.Errorf("show unexported fields is %v, but got %q", showUnexported, doc.Decl)
		}
	}
}

func TestEmbeddedTypes(t *testing.T) {
	cleanup := setGopath(filepath.Join(".", "testdata", "package"), t)
	defer cleanup()
	filename := filepath.Join(".", "testdata", "package", "src", "somepkg", "embed.go")

	tests := []struct {
		description string
		offset      int
		want        string
	}{
		{"embedded value", 77, "foo doc\n"},
		{"embedded pointer", 113, "foo doc\n"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			doc, err := Run(filename, test.offset, nil)
			if err != nil {
				t.Fatal(err)
			}
			if doc.Doc != test.want {
				t.Errorf("want %q, got %q", test.want, doc.Doc)
			}
			if doc.Pkg != "somepkg" {
				t.Errorf("want package somepkg, got %q", doc.Pkg)
			}
		})
	}
}

func TestIssue20(t *testing.T) {
	cleanup := setGopath(filepath.Join(".", "testdata", "package"), t)
	defer cleanup()
	filename := filepath.Join(".", "testdata", "package", "src", "somepkg", "issue20.go")

	tests := []struct {
		desc   string
		want   string
		offset int
	}{
		{"named type", "var words []string", 116},
		{"unnamed type", "var tests []struct{Name string; args string}", 283},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			doc, err := Run(filename, test.offset, nil)
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
	cleanup := setGopath(filepath.Join(".", "testdata", "withvendor"), t)
	defer cleanup()

	filename := filepath.Join(".", "testdata", "withvendor", "src", "main", "main.go")
	doc, err := Run(filename, 76, nil)
	if err != nil {
		t.Fatal(err)
	}

	want := "github.com/zmb3/vp"
	if doc.Import != want {
		t.Errorf("want %s, got %s", want, doc.Import)
	}
	if doc.Doc != "Hello says hello.\n" {
		t.Errorf("want 'Hello says hello.\n', got %q", doc.Doc)
	}

	doc, err = Run(filename, 99, nil)
	if err != nil {
		t.Fatal(err)
	}

	decl := `const Foo untyped string`
	if decl != doc.Decl {
		t.Errorf("invalid decl: want %q, got %q", decl, doc.Decl)
	}
}

func setGopath(path string, t *testing.T) func() {
	t.Helper()

	orig := os.Getenv("GOPATH")
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	os.Setenv("GOPATH", abs)
	return func() { os.Setenv("GOPATH", orig) }
}
