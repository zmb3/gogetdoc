package main

import (
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
		Pos token.Pos
		Doc string
	}{
		{tokFile.Pos(191), "SayHello says hello.\n"},                                                              // method call
		{tokFile.Pos(205), "SayGoodbye says goodbye.\n"},                                                          // function call
		{tokFile.Pos(305), "Message is a message.\n"},                                                             // var (use)
		{tokFile.Pos(388), "Message is a message.\n"},                                                             // var (definition)
		{tokFile.Pos(318), "Sprintf formats according to a format specifier and returns the resulting string.\n"}, // std func
		{tokFile.Pos(346), "Answer is the answer to life the universe and everything.\n\nConstant Value: 42"},     // const (use)
		{tokFile.Pos(484), "Answer is the answer to life the universe and everything.\n\nConstant Value: 42"},     // const (definition)
		{tokFile.Pos(144), "IsNaN reports whether f is an IEEE 754 ``not-a-number'' value.\n"},                    // std func call (alias import)

		// field doc/comment precedence
		{tokFile.Pos(628), "FieldA has doc\n"},
		{tokFile.Pos(637), "FieldB has a comment\n"},

		// GenDecl doc/comment precedence
		{tokFile.Pos(991), "Alpha doc"},
		{tokFile.Pos(1002), "Bravo comment"},
		{tokFile.Pos(1029), ""},

		// builtins
		{tokFile.Pos(947), "The error built-in interface type is the conventional"},
		{tokFile.Pos(707), "The append built-in function appends elements to the end"},
		{tokFile.Pos(734), "float32 is the set of all IEEE-754 32-bit floating-point numbers."},
		{tokFile.Pos(793), "iota is a predeclared identifier representing the untyped integer ordinal"},
		{tokFile.Pos(832), "nil is a predeclared identifier representing the zero"},
		{tokFile.Pos(886), "The len built-in function returns the length of v"},
		{tokFile.Pos(923), "The close built-in function closes a channel, which must"},
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
			t.Errorf("show unexported fields is %v, found unexported field is %v", showUnexported, hasUnexportedField)
		}
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
	doc, err := Run(&ctx, "main.go", 63)
	if err != nil {
		t.Fatal(err)
	}

	want := "github.com/zmb3/vp"
	if doc.Import != want {
		t.Errorf("want %s, got %s", want, doc.Import)
	}
}

func copyFile(dst, src string) error {
	orig, err := os.Open(src)
	if err != nil {
		return err
	}
	defer orig.Close()

	copy, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer copy.Close()

	_, err = io.Copy(copy, orig)
	return err
}
