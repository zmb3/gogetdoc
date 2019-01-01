package somepkg

import (
	"fmt"
	mth "math"
)

type X struct{}

// SayHello says hello.
func (X) SayHello() {
	fmt.Println("Hello, World", mth.IsNaN(1.23)) //@doc("IsNaN", "IsNaN reports whether f")
}

func Baz() {
	var x X      //@decl("X", "type X struct{}")
	x.SayHello() //@decl("ayHello", "func (X) SayHello()")
	SayGoodbye() //@doc("ayGood", "SayGoodbye says goodbye."), decl("ayGood", "func SayGoodbye() (string, error)")
}

// SayGoodbye says goodbye.
func SayGoodbye() (string, error) {
	fmt.Println("Goodbye")
	fmt.Println(Message, fmt.Sprintf("The answer is %d", Answer)) //@const("Answer", "42"), doc("printf", "Sprintf formats according to a"), doc("Message", "Message is a message."), decl("Message", "var Message string")
	return "", nil
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
	fmt.Println(f.FieldA, f.FieldB) //@doc("FieldA", "FieldA has doc"), doc("FieldB", "FieldB has a comment"), decl("FieldA", "field FieldA string")
}

var slice = []int{0, 1, 2}

func addInt(i int) {
	slice = append(slice, i)     //@doc("append", "The append built-in function appends elements")
	if f := float32(i); f > 42 { //@doc("loat32", "float32 is the set of all IEEE-754 32-bit")
		fmt.Println("foo")
	}
}

const (
	A = iota //@doc("iota", "iota is a predeclared identifier representing the untyped integer ordinal")
	B
	C
)

var slice2 = []*Foo{nil, nil, nil} //@doc("nil", "nil is a predeclared identifier representing the zero")

func test() {
	c := make(chan int)
	if l := len(slice2); l > 0 { //@doc("len", "The len built-in function returns")
		c <- l
	}
	close(c) //@doc("close", "The close built-in function closes")
}

func test2() error { //@doc("rror", "The error built-in interface type is the conventional")
	return nil
}

var (
	// Alpha doc
	Alpha   = 0 //@decl("lpha", "var Alpha int")
	Bravo   = 1 // Bravo comment
	Charlie = 2
	Delta   = Bravo //@doc("ravo", "Bravo comment"), decl("ravo", "var Bravo int")
)

type HasUnexported struct { //@exported("Unexported")
	Visible    string // Visible is an exported field
	notVisible string // notVisible is an unexported field
}

type NewString string

var ns NewString //@decl("String", "type NewString string")
