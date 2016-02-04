gogetdoc
========

Retrieve documentation for items in Go source code.

Go has a variety of tools that make it easy to lookup documentation.
There's the `godoc` HTTP server, the `go doc` command line tool, and https://godoc.org.

These tools are great, but in many cases one may find it valuable to lookup
documentation right from their editor.  The problem with all of these tools
is that they are all meant to be used by a person who knows what they are
looking for.  This makes editor integration difficult, as there isn't an easy way
to say "get me the documentation for this item here."

The `gogetdoc` tool aims to make it easier for editors to provide access to
Go documentation.  Simply give it a filename and offset within the file and
it will figure out what what you're referring to and find the documentation
for it.

## Prerequisites

This tool **requires Go 1.6**, which is currently available by building from tip
or by installing the Go 1.6 release candidate from https://golang.org/dl/.

## Contributions

Are more than welcome!  For small changes feel free to open a pull request.
For larger changes or major features please open an issue to discuss.

## Credits

The following resources served as both inspiration for starting this tool
and help coming up with the implementation.

- Alan Donovan's GothamGo talk "Using `go/types` for Code Comprehension
  and Refactoring Tools" https://youtu.be/p_cz7AxVdfg
- Fatih Arslan's talk at dotGo 2015 "Tools for working with Go Code"
- The `go/types` example repository: https://github.com/golang/example/tree/master/gotypes

## License

3-Clause BSD license - see the LICENSE file for details.
