package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/loader"
)

const funcDecl = `package main

import (
	"fmt"
	mth "math"
)

type X struct{}

// SayHello says hello.
func (X) SayHello() {
  fmt.Println("Hello, World", mth.IsNaN(1.23))
}

func main() {
  var x X
  x.SayHello()
  SayGoodbye()
}

// SayGoodbye says goodbye.
func SayGoodbye() {
  fmt.Println("Goodbye")
  fmt.Println(Message, fmt.Sprintf("The answer is %d", Answer))
}

// Message is a message.
var Message = "This is a test."

// Answer is the answer to life the universe and everything.
const Answer = 42

type Foo struct {
	// FieldA has doc
	FieldA string
	FieldB string // FieldB has a comment
}

func (f Foo) Print() {
	fmt.Println(f.FieldA, f.FieldB)
}

var slice = []int{0, 1, 2}

func addInt(i int) {
	slice = append(slice, i)
	if f := float32(i); f > 42 {
		fmt.Println("foo")
	}
}

const (
	A = iota
	B
	C
)

var slice2 = []*Foo{nil, nil, nil}

func test() {
	c := make(chan int)
	if l := len(slice2); l > 0 {
		c <- l
	}
	close(c)
}

func test2() error {
	return nil
}

var (
	// Alpha doc
	Alpha = 0
	Bravo = 1 // Bravo comment
	Charlie = 2
)

type HasUnexported struct {
	Visible string			// Visible is an exported field
	notVisible string		// notVisible is an unexported field
}
`

func TestIdent(t *testing.T) {
	t.Parallel()
	conf := &loader.Config{
		ParserMode: parser.ParseComments,
	}
	astFile, err := conf.ParseFile("test.go", funcDecl)
	if err != nil {
		t.Error(err)
	}

	conf.CreateFromFiles("main", astFile)
	prog, err := conf.Load()
	if err != nil {
		t.Error(err)
	}

	tokFile := FileFromProgram(prog, "test.go")
	if tokFile == nil {
		t.Error("Couldn't get token.File from program")
	}

	tests := []struct {
		Pos  token.Pos
		Doc  string
		Decl string
	}{
		{Pos: tokFile.Pos(191), Doc: "SayHello says hello.\n"},                                                              // method call
		{Pos: tokFile.Pos(205), Doc: "SayGoodbye says goodbye.\n"},                                                          // function call
		{Pos: tokFile.Pos(305), Doc: "Message is a message.\n"},                                                             // var (use)
		{Pos: tokFile.Pos(388), Doc: "Message is a message.\n"},                                                             // var (definition)
		{Pos: tokFile.Pos(318), Doc: "Sprintf formats according to a format specifier and returns the resulting string.\n"}, // std func
		{Pos: tokFile.Pos(346), Doc: "Answer is the answer to life the universe and everything.\n\nConstant Value: 42"},     // const (use)
		{Pos: tokFile.Pos(484), Doc: "Answer is the answer to life the universe and everything.\n\nConstant Value: 42"},     // const (definition)
		{Pos: tokFile.Pos(144), Doc: "IsNaN reports whether f is an IEEE 754 ``not-a-number'' value.\n"},                    // std func call (alias import)

		// field doc/comment precedence
		{Pos: tokFile.Pos(628), Doc: "FieldA has doc\n"},
		{Pos: tokFile.Pos(637), Doc: "FieldB has a comment\n"},

		// GenDecl doc/comment precedence
		{Pos: tokFile.Pos(991), Doc: "Alpha doc"},
		{Pos: tokFile.Pos(1002), Doc: "Bravo comment"},
		{Pos: tokFile.Pos(1029)},

		// builtins
		{Pos: tokFile.Pos(947), Doc: "The error built-in interface type is the conventional"},
		{Pos: tokFile.Pos(707), Doc: "The append built-in function appends elements to the end"},
		{Pos: tokFile.Pos(734), Doc: "float32 is the set of all IEEE-754 32-bit floating-point numbers."},
		{Pos: tokFile.Pos(793), Doc: "iota is a predeclared identifier representing the untyped integer ordinal"},
		{Pos: tokFile.Pos(832), Doc: "nil is a predeclared identifier representing the zero"},
		{Pos: tokFile.Pos(886), Doc: "The len built-in function returns the length of v"},
		{Pos: tokFile.Pos(923), Doc: "The close built-in function closes a channel, which must"},

		// decl
		{Pos: tokFile.Pos(596), Decl: "type Foo struct {"},
	}
TestLoop:
	for _, test := range tests {
		info, nodes, _ := prog.PathEnclosingInterval(test.Pos, test.Pos)
		for i := range nodes {
			if ident, ok := nodes[i].(*ast.Ident); ok {
				doc, err := IdentDoc(&build.Default, ident, info, prog)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(doc.Doc, test.Doc) {
					t.Errorf("Want '%s', got '%s'\n", test.Doc, doc.Doc)
				}
				if test.Decl != "" && !strings.HasPrefix(doc.Decl, test.Decl) {
					t.Errorf("Decl: want '%s', got '%s'\n", test.Decl, doc.Decl)
				}
				continue TestLoop
			}
		}
		t.Errorf("Couldn't find *ast.Ident at %s\n", prog.Fset.Position(test.Pos))
	}
}

func TestUnexportedFields(t *testing.T) {
	t.Parallel()
	conf := &loader.Config{
		ParserMode: parser.ParseComments,
	}
	astFile, err := conf.ParseFile("test.go", funcDecl)
	if err != nil {
		t.Error(err)
	}

	conf.CreateFromFiles("main", astFile)
	prog, err := conf.Load()
	if err != nil {
		t.Error(err)
	}

	tokFile := FileFromProgram(prog, "test.go")
	if tokFile == nil {
		t.Error("Couldn't get token.File from program")
	}

	for _, showUnexported := range []bool{true, false} {
		*showUnexportedFields = showUnexported
		doc, err := DocForPos(&build.Default, prog, "test.go", 1052)
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
	ctx, cleanup, err := tempGopath("embed.go", "embed")
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
			doc, err := Run(ctx, "embed.go", test.offset)
			if err != nil {
				t.Fatal(err)
			}
			if doc.Doc != test.want {
				t.Errorf("want %q, got %q", test.want, doc.Doc)
			}
		})
	}
}

func TestIssue20(t *testing.T) {
	ctx, cleanup, err := tempGopath("issue20.go", "foo")
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
			doc, err := Run(ctx, "issue20.go", test.offset)
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
	}
	defer func() {
		os.RemoveAll(newGopath)
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

	ctx := build.Default
	ctx.GOPATH = newGopath
	doc, err := Run(&ctx, "main.go", 63)
	if err != nil {
		t.Fatal(err)
	}

	want := "github.com/zmb3/vp"
	if doc.Import != want {
		t.Errorf("want %s, got %s", want, doc.Import)
	}
}

func tempGopath(filename, pkg string) (ctx *build.Context, cleanup func(), err error) {
	newGopath, err := ioutil.TempDir(".", "gogetdoc-gopath")
	if err != nil {
		return nil, nil, err
	}
	newGopath, _ = filepath.Abs(newGopath)

	pkgDir := filepath.Join(newGopath, "src", "github.com", "zmb3", pkg)
	err = os.MkdirAll(pkgDir, 0755)
	if err != nil {
		os.RemoveAll(newGopath)
		return nil, nil, err
	}

	err = copyFile(filepath.Join(pkgDir, filename), filepath.FromSlash("./testdata/"+filename))
	if err != nil {
		os.RemoveAll(newGopath)
		return nil, nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		os.RemoveAll(newGopath)
		return nil, nil, err
	}
	err = os.Chdir(pkgDir)
	if err != nil {
		os.RemoveAll(newGopath)
		return nil, nil, err
	}

	cleanup = func() {
		os.RemoveAll(newGopath)
		os.Chdir(cwd)
	}
	ctx2 := build.Default
	ctx2.GOPATH = newGopath
	return &ctx2, cleanup, nil
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
