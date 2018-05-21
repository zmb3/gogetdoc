package main

import (
	"go/build"
	"io/ioutil"
	"os"
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

func TestRunOutsideGopath(t *testing.T) {
	cleanup, err := makeTempWorkspace("hello.go")
	if err != nil {
		t.Fatal(err)
	}

	if cleanup != nil {
		defer cleanup()
	}

	tests := []struct {
		Pos int64
		Doc string
	}{
		{Pos: 23, Doc: "\tPackage fmt implements formatted I/O"},    // import "fmt"
		{Pos: 48, Doc: "Println formats using the default formats"}, // call fmt.Println()
	}

	for _, test := range tests {
		ctx := build.Default
		doc, err := Run(&ctx, "hello.go", test.Pos)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(doc.Doc, test.Doc) {
			t.Errorf("want '%s', got '%s'", test.Doc, doc.Doc)
		}
	}
}

func makeTempWorkspace(fileList ...string) (cleanup func(), err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	tmpDir, err := ioutil.TempDir("", "gogetdoc-tmp")
	if err != nil {
		return nil, err
	}

	for _, file := range fileList {
		err = copyFile(filepath.Join(tmpDir, file), filepath.FromSlash("./testdata/"+file))
		if err != nil {
			os.RemoveAll(tmpDir)
			return nil, err
		}
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	cleanup = func() {
		os.RemoveAll(tmpDir)
		os.Chdir(cwd)
	}
	return cleanup, nil
}
