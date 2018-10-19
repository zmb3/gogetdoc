package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseValidPos(t *testing.T) {
	fname, offset, err := parsePos("foo.go:#123")
	if fname != "foo.go" {
		t.Errorf("want foo.go, got %v", fname)
	}
	if offset != 123 {
		t.Errorf("want 123, got %v", 123)
	}
	if err != nil {
		t.Error(err)
	}
}

func TestParseEmptyPos(t *testing.T) {
	_, _, err := parsePos("")
	if err == nil {
		t.Error("expected error")
	}
}

func TestParseInvalidPos(t *testing.T) {
	for _, input := range []string{
		"foo.go:123",
		"foo.go#123",
		"foo.go#:123",
		"123",
		"foo.go::123",
		"foo.go##123",
		"#:123",
	} {
		if _, _, err := parsePos(input); err == nil {
			t.Errorf("expected %v to be invalid", input)
		}
	}
}

func TestRunInvalidPosGopath(t *testing.T) {
	cleanup := setGopath(filepath.Join(".", "testdata", "package"), t)
	defer cleanup()
	filename := filepath.Join(".", "testdata", "package", "src", "somepkg", "idents.go")

	_, err := Run(filename, 5000, nil)
	if err == nil {
		t.Fatal("expected invalid pos error")
	}
}

func TestRunOutsideGopath(t *testing.T) {
	tests := []struct {
		Pos int
		Doc string
	}{
		{Pos: 23, Doc: "\tPackage fmt implements formatted I/O"},    // import "fmt"
		{Pos: 48, Doc: "Println formats using the default formats"}, // call fmt.Println()
	}
	filename := filepath.Join(".", "testdata", "amodule", "hello.go")
	for _, test := range tests {
		doc, err := Run(filename, test.Pos, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(doc.Doc, test.Doc) {
			t.Errorf("want '%s', got '%s'", test.Doc, doc.Doc)
		}
	}
}

func TestIssue44(t *testing.T) {
	filename := filepath.Join(".", "testdata", "interface-methods", "rabbit.go")
	for _, test := range []struct {
		Pos  int
		Decl string
	}{
		{87, `func (Thing).Do(s Stuff) Stuff`},
		{151, `func (Thing).DoWithError(s Stuff) (Stuff, error)`},
		{245, `func (Thing).DoWithNoArgs()`},
		{263, `func (Thing).NamedReturns() (s string, err error)`},
		{296, `func (Thing).SameTypeParams(x string, y string)`},
	} {
		doc, err := Run(filename, test.Pos, nil)
		if err != nil {
			t.Fatalf("issue44, pos %d: %v", test.Pos, err)
		}
		if doc.Decl != test.Decl {
			t.Errorf("pos %d, invalid decl: want %q, got %q", test.Pos, test.Decl, doc.Decl)
		}
	}
}
