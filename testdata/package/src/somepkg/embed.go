package somepkg

// foo doc
type foo struct {
	i int
}

type bar1 struct {
	foo
	f1 foo
}

type bar2 struct {
	*foo
	f1 *foo
}
