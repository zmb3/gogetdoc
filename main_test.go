package main

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/packages/packagestest"
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

func TestRunInvalidPos(t *testing.T) {
	dir := filepath.Join(".", "testdata", "package")
	mods := []packagestest.Module{
		{Name: "somepkg", Files: packagestest.MustCopyFileTree(dir)},
	}
	packagestest.TestAll(t, func(t *testing.T, exporter packagestest.Exporter) {
		if exporter == packagestest.Modules {
			return // TODO get working with Modules and GOPATH
		}
		exported := packagestest.Export(t, exporter, mods)
		defer exported.Cleanup()

		teardown := setup(exported.Config)
		defer teardown()

		filename := exported.File("somepkg", "idents.go")
		_, err := Run(filename, 5000, nil)
		if err == nil {
			t.Fatal("expected invalid pos error")
		}
	})
}

// github.com/zmb3/gogetdoc/issues/44
func TestInterfaceDecls(t *testing.T) {
	mods := []packagestest.Module{
		{
			Name:  "rabbit",
			Files: packagestest.MustCopyFileTree(filepath.Join(".", "testdata", "interface-decls")),
		},
	}
	// TODO: convert to packagestest.TestAll
	exported := packagestest.Export(t, packagestest.GOPATH, mods)
	defer exported.Cleanup()

	teardown := setup(exported.Config)
	defer teardown()

	filename := exported.File("rabbit", "rabbit.go")

	if expectErr := exported.Expect(map[string]interface{}{
		"decl": func(p token.Position, decl string) {
			doc, err := Run(filename, p.Offset, nil)
			if err != nil {
				t.Error(err)
			}
			if doc.Decl != decl {
				t.Errorf("bad decl, want %q, got %q", decl, doc.Decl)
			}
		},
	}); expectErr != nil {
		t.Fatal(expectErr)
	}
}

func setup(cfg *packages.Config) func() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	err = os.Chdir(cfg.Dir)
	if err != nil {
		panic(err)
	}
	setEnv := func(env []string) {
		for _, assignment := range env {
			if i := strings.Index(assignment, "="); i > 0 {
				os.Setenv(assignment[:i], assignment[i+1:])
			}
		}
	}
	originalEnv := os.Environ()
	setEnv(cfg.Env)
	return func() {
		os.Chdir(dir)
		setEnv(originalEnv)
	}
}
