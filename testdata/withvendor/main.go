package main

import (
	"fmt"

	"github.com/zmb3/vp" //@import(".com", "github.com/zmb3/vp"), decl(".com", "package vp")
)

func main() {
	vp.Hello()          //@import("ello", "github.com/zmb3/vp"), doc("ello", "Hello says hello.\n")
	fmt.Println(vp.Foo) //@decl("Foo", "const Foo untyped string")
}
