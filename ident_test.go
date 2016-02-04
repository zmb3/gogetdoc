package main

import (
	"go/ast"
	"go/parser"
	"go/token"
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
		{tokFile.Pos(628), "FieldA has doc\n"},
		{tokFile.Pos(637), "FieldB has a comment\n"},
	}
TestLoop:
	for _, test := range tests {
		info, nodes, _ := prog.PathEnclosingInterval(test.Pos, test.Pos)
		for i := range nodes {
			if ident, ok := nodes[i].(*ast.Ident); ok {
				doc, err := IdentDoc(ident, info, prog)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.EqualFold(test.Doc, doc.Doc) {
					t.Errorf("Want '%s', got '%s'\n", test.Doc, doc.Doc)
				}
				continue TestLoop
			}
		}
		t.Errorf("Coudln't find *ast.Ident at %s\n", prog.Fset.Position(test.Pos))
	}
}
