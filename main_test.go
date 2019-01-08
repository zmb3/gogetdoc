package main

import (
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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
		if exporter == packagestest.Modules && !modulesSupported() {
			t.Skip("Skipping modules test on", runtime.Version())
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

func modulesSupported() bool {
	v := strings.TrimPrefix(runtime.Version(), "go")
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	return minor >= 11
}

func setup(cfg *packages.Config) func() {
	originalDir, err := os.Getwd()
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
	os.Setenv("PWD", cfg.Dir) // https://go-review.googlesource.com/c/tools/+/143517/

	return func() {
		os.Chdir(originalDir)
		setEnv(originalEnv)
	}
}

func TestIssue52(t *testing.T) {
	dir := filepath.Join(".", "testdata", "issue52")
	mods := []packagestest.Module{
		{Name: "issue52", Files: packagestest.MustCopyFileTree(dir)},
	}
	packagestest.TestAll(t, func(t *testing.T, exporter packagestest.Exporter) {
		if exporter == packagestest.Modules && !modulesSupported() {
			t.Skip("Skipping modules test on", runtime.Version())
		}
		exported := packagestest.Export(t, exporter, mods)
		defer exported.Cleanup()

		teardown := setup(exported.Config)
		defer teardown()

		filename := exported.File("issue52", "main.go")

		for _, test := range []struct {
			Pos int
			Doc string
		}{
			{64, "V this works\n"},
			{66, "Foo this doesn't work but should\n"},
		} {
			doc, err := Run(filename, test.Pos, nil)
			if err != nil {
				t.Fatalf("issue52, pos %d: %v", test.Pos, err)
			}
			if doc.Doc != test.Doc {
				t.Errorf("issue52, pos %d, invalid decl: want %q, got %q", test.Pos, test.Doc, doc.Doc)
			}
		}
	})
}
