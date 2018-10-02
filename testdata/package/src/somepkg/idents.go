package somepkg

import (
	"fmt"
	mth "math"
)

type X struct{}

// SayHello says hello.
func (X) SayHello() {
	fmt.Println("Hello, World", mth.IsNaN(1.23))
}

func Baz() {
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
	Alpha   = 0
	Bravo   = 1 // Bravo comment
	Charlie = 2
)

type HasUnexported struct {
	Visible    string // Visible is an exported field
	notVisible string // notVisible is an unexported field
}
