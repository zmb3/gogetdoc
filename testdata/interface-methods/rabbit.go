package rabbit

// Thing is an interface
type Thing interface {

	// Do does a thing
	Do(s Stuff) Stuff

	// DoWithError returns multiple values
	DoWithError(s Stuff) (Stuff, error)

	// DoWithNoArgs takes no args and returns no results
	DoWithNoArgs()

	NamedReturns() (s string, err error)

	SameTypeParams(x, y string)
}

// Stuff is a struct
type Stuff struct{}

// ThingImplemented matches Thing interface
type ThingImplemented struct{}

// Do does stuff
func (ti *ThingImplemented) Do(s Stuff) Stuff {
	return s
}
