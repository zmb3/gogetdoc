package main

import (
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages/packagestest"
)

func TestIdent(t *testing.T) {
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

		getDoc := func(p token.Position) *Doc {
			t.Helper()
			doc, docErr := Run(p.Filename, p.Offset, nil)
			if docErr != nil {
				t.Fatal(docErr)
			}
			return doc
		}

		pcmp := func(want, got string) {
			t.Helper()
			if !strings.HasPrefix(got, want) {
				if len(got) > 64 {
					got = got[:64]
				}
				t.Errorf("expected prefix %q in %q", want, got)
			}
		}

		cmp := func(want, got string) {
			t.Helper()
			if got != want {
				t.Errorf("want %q, got %q", want, got)
			}
		}

		if expectErr := exported.Expect(map[string]interface{}{
			"doc":  func(p token.Position, doc string) { pcmp(doc, getDoc(p).Doc) },
			"pkg":  func(p token.Position, pkg string) { cmp(pkg, getDoc(p).Pkg) },
			"decl": func(p token.Position, decl string) { cmp(decl, getDoc(p).Decl) },
			"const": func(p token.Position, val string) {
				d := getDoc(p)
				needle := "Constant Value: " + val
				if !strings.Contains(d.Doc, needle) {
					t.Errorf("Expected %q in %q", needle, d.Doc)
				}
			},
			"exported": func(p token.Position) {
				for _, showUnexported := range []bool{true, false} {
					*showUnexportedFields = showUnexported
					d := getDoc(p)
					hasUnexportedField := strings.Contains(d.Decl, "notVisible")
					if hasUnexportedField != *showUnexportedFields {
						t.Errorf("show unexported fields is %v, but got %q", showUnexported, d.Decl)
					}
				}
			},
		}); expectErr != nil {
			t.Fatal(expectErr)
		}
	})
}

func TestVendoredCode(t *testing.T) {
	dir := filepath.Join(".", "testdata", "withvendor")
	mods := []packagestest.Module{
		{Name: "main", Files: packagestest.MustCopyFileTree(dir)},
	}

	exported := packagestest.Export(t, packagestest.GOPATH, mods)
	defer exported.Cleanup()

	teardown := setup(exported.Config)
	defer teardown()

	filename := exported.File("main", "main.go")
	getDoc := func(p token.Position) *Doc {
		t.Helper()
		doc, docErr := Run(filename, p.Offset, nil)
		if docErr != nil {
			t.Fatal(docErr)
		}
		return doc
	}

	compare := func(want, got string) {
		if want != got {
			t.Errorf("want %q, got %q", want, got)
		}
	}

	if expectErr := exported.Expect(map[string]interface{}{
		"import": func(p token.Position, path string) { compare(path, getDoc(p).Import) },
		"decl":   func(p token.Position, decl string) { compare(decl, getDoc(p).Decl) },
		"doc":    func(p token.Position, doc string) { compare(doc, getDoc(p).Doc) },
	}); expectErr != nil {
		t.Fatal(expectErr)
	}
}
