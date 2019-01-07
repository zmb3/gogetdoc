package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/packages/packagestest"
)

const contents = `package somepkg

import "fmt"

const (
Zero = iota
One
Two
)

const Three = 3

func main() {
	fmt.Println(Zero, Three, Two, Three)
}
`

func TestModified(t *testing.T) {
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

		path := exported.File("somepkg", "const.go")
		archive := fmt.Sprintf("%s\n%d\n%s", path, len(contents), contents)
		overlay, err := buildutil.ParseOverlayArchive(strings.NewReader(archive))
		if err != nil {
			t.Fatalf("couldn't parse overlay: %v", err)
		}

		d, err := Run(path, 114, overlay)
		if err != nil {
			t.Fatal(err)
		}
		if n := d.Name; n != "Three" {
			t.Errorf("got const %s, want Three", n)
		}
	})
}
