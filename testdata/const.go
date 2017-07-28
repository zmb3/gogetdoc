package main

import "fmt"

const (
	Zero = iota
	One
	Two
)

const Three = 3

func main() {
	fmt.Println(Zero, One, Two, Three)
}
