package main

import (
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages/packagestest"
)

func TestPackageDoc(t *testing.T) {
	dir := filepath.Join(".", "testdata", "package-doc")
	mods := []packagestest.Module{
		{Name: "pkgdoc", Files: packagestest.MustCopyFileTree(dir)},
	}

	packagestest.TestAll(t, func(t *testing.T, exporter packagestest.Exporter) {
		if exporter == packagestest.Modules && !modulesSupported() {
			t.Skip("Skipping modules test on", runtime.Version())
		}
		exported := packagestest.Export(t, exporter, mods)
		defer exported.Cleanup()

		teardown := setup(exported.Config)
		defer teardown()

		filename := exported.File("pkgdoc", "main.go")
		if expectErr := exported.Expect(map[string]interface{}{
			"pkgdoc": func(p token.Position, doc string) {
				d, err := Run(filename, p.Offset, nil)
				if err != nil {
					t.Error(err)
				}
				if !strings.HasPrefix(d.Doc, doc) {
					t.Errorf("expected %q, got %q", doc, d.Doc)
				}
				if !strings.HasPrefix(d.Decl, "package") {
					t.Errorf("expected %q to begin with 'package'", d.Decl)
				}
			},
		}); expectErr != nil {
			t.Fatal(expectErr)
		}
	})
}
