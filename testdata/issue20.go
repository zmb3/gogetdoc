package foo

import "fmt"

// Foo is a test function.
func Foo() {
	words := []string{}
	for _, word := range words {
		fmt.Println(word)
	}
}

func Bar() {
	tests := []struct {
		Name string
		args string
	}{
		{"Test1", "a b c"},
		{"Test2", "a b c"},
	}
	for _, test := range tests {
		fmt.Println(test.Name)
	}
}
