// gogetdoc gets documentation for Go objects given their locations in the source code

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"strings"

	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/packages"
)

var (
	cpuprofile           = flag.String("cpuprofile", "", "write cpu profile to file")
	pos                  = flag.String("pos", "", "Filename and byte offset of item to document, e.g. foo.go:#123")
	modified             = flag.Bool("modified", false, "read an archive of modified files from standard input")
	linelength           = flag.Int("linelength", 80, "maximum length of a line in the output (in Unicode code points)")
	jsonOutput           = flag.Bool("json", false, "enable extended JSON output")
	showUnexportedFields = flag.Bool("u", false, "show unexported fields")
)

const modifiedUsage = `
The archive format for the -modified flag consists of the file name, followed
by a newline, the decimal file size, another newline, and the contents of the file.

This allows editors to supply gogetdoc with the contents of their unsaved buffers.
`

func main() {
	// disable GC as gogetdoc is a short-lived program
	debug.SetGCPercent(-1)

	flag.Var((*buildutil.TagsFlag)(&build.Default.BuildTags), "tags", buildutil.TagsFlagDoc)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, modifiedUsage)
	}
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}
	filename, offset, err := parsePos(*pos)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	d, err := Run(filename, offset)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *jsonOutput {
		json.NewEncoder(os.Stdout).Encode(d)
	} else {
		fmt.Println(d.String())
	}
}

// Run is a wrapper for the gogetdoc command.  It is broken out of main for easier testing.
func Run(filename string, offset int64) (*Doc, error) {
	var parseFile func(fset *token.FileSet, filename string) (*ast.File, error)
	if *modified {
		var overlay map[string][]byte
		overlay, err := buildutil.ParseOverlayArchive(os.Stdin)
		if err != nil {
			log.Fatalln("invalid archive:", err)
		}
		parseFile = func(fset *token.FileSet, filename string) (*ast.File, error) {
			const mode = parser.AllErrors | parser.ParseComments
			return parser.ParseFile(fset, filename, overlay[filename], mode)
		}
	}

	cfg := &packages.Config{
		Mode:      packages.LoadAllSyntax, // want syntax trees of dependencies
		ParseFile: parseFile,              // Use the archive to parse files.
		Tests:     strings.HasSuffix(filename, "_test.go"),
	}
	pkgs, err := packages.Load(cfg, fmt.Sprintf("contains:%s", filename))
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("No package to containing file")
	}

	// Arbitrarily return the first package if there are multiple.
	// TODO: should the user be able to specify which one?
	if len(pkgs) > 1 {
		fmt.Printf("packages not processed: %v\n", pkgs[1:])
	}

	doc, err := DocForPos(pkgs[0], filename, offset)
	if err != nil && len(pkgs[0].Errors) > 0 {
		fmt.Fprintln(os.Stderr, pkgs[0].Errors[0])
	}
	return doc, err
}

// DocForPos attempts to get the documentation for an item given a filename and byte offset.
func DocForPos(lprog *packages.Package, filename string, offset int64) (*Doc, error) {
	tokFile := FileFromProgram(lprog, filename)
	if tokFile == nil {
		return nil, fmt.Errorf("gogetdoc: couldn't find %s in program", filename)
	}
	offPos := tokFile.Pos(int(offset))

	pkgInfo, nodes := pathEnclosingInterval(lprog, offPos, offPos)
	for _, node := range nodes {
		switch i := node.(type) {
		case *ast.ImportSpec:
			return PackageDoc(lprog, ImportPath(i))
		case *ast.Ident:
			// if we can't find the object denoted by the identifier, keep searching)
			if obj := pkgInfo.ObjectOf(i); obj == nil {
				continue
			}
			return IdentDoc(i, pkgInfo, lprog)
		default:
			break
		}
	}
	return nil, errors.New("gogetdoc: no documentation found")
}

// FileFromProgram attempts to locate a token.File from a loaded program.
func FileFromProgram(prog *packages.Package, name string) *token.File {
	for _, astFile := range prog.Syntax {
		tokFile := prog.Fset.File(astFile.Pos())
		if tokFile == nil {
			continue
		}
		tokName := tokFile.Name()
		if runtime.GOOS == "windows" {
			tokName = filepath.ToSlash(tokName)
			name = filepath.ToSlash(name)
		}
		if tokName == name {
			return tokFile
		}
		if sameFile(tokName, name) {
			return tokFile
		}
	}
	return nil
}

func parsePos(p string) (filename string, offset int64, err error) {
	// foo.go:#123
	if p == "" {
		err = errors.New("missing required -pos flag")
		return
	}
	sep := strings.LastIndex(p, ":")
	// need at least 2 characters after the ':'
	// (the # sign and the offset)
	if sep == -1 || sep > len(p)-2 || p[sep+1] != '#' {
		err = fmt.Errorf("invalid option: -pos=%s", p)
		return
	}
	filename = p[:sep]
	offset, err = strconv.ParseInt(p[sep+2:], 10, 32)
	return
}

func sameFile(a, b string) bool {
	if filepath.Base(a) != filepath.Base(b) {
		// We only care about symlinks for the GOPATH itself. File
		// names need to match.
		return false
	}
	if ai, err := os.Stat(a); err == nil {
		if bi, err := os.Stat(b); err == nil {
			return os.SameFile(ai, bi)
		}
	}
	return false
}
