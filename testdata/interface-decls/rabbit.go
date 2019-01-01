package rabbit

type Thing interface {
	Do(s Stuff) Stuff                    //@decl("Do", "func (Thing).Do(s Stuff) Stuff")
	DoWithError(s Stuff) (Stuff, error)  //@decl("hError", "func (Thing).DoWithError(s Stuff) (Stuff, error)")
	DoWithNoArgs()                       //@decl("WithNoArgs", "func (Thing).DoWithNoArgs()")
	NamedReturns() (s string, err error) //@decl("medReturns", "func (Thing).NamedReturns() (s string, err error)")
	SameTypeParams(x, y string)          //@decl("TypeParams", "func (Thing).SameTypeParams(x string, y string)")
}

type Stuff struct{}
type ThingImplemented struct{}

func (ti *ThingImplemented) Do(s Stuff) Stuff { //@decl("tuff", "type Stuff struct{}")
	return s
}
