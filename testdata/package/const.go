package somepkg

import "fmt"

const (
	Zero = iota
	One
	Two
)

const Three = 3

func main() {
	fmt.Println(Zero, One, Two, Three) //@const("Zero", "0"), const("One", "1"), const("Three", "3")
}
