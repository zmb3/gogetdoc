package somepkg

// foo doc
type foo struct {
	i int
}

type bar1 struct {
	foo //@doc("foo", "foo doc"), pkg("foo", "somepkg")
	f1  foo
}

type bar2 struct {
	*foo      //@doc("foo", "foo doc"), pkg("foo", "somepkg")
	f1   *foo //@doc("foo", "foo doc"), pkg("foo", "somepkg")
}
