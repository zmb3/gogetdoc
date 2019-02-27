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
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
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

var archiveReader io.Reader = os.Stdin

const modifiedUsage = `
The archive format for the -modified flag consists of the file name, followed
by a newline, the decimal file size, another newline, and the contents of the file.

This allows editors to supply gogetdoc with the contents of their unsaved buffers.
`

const debugAST = false

func fatal(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func main() {
	// disable GC as gogetdoc is a short-lived program
	debug.SetGCPercent(-1)

	log.SetOutput(ioutil.Discard)

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
			fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fatal(err)
		}
		defer pprof.StopCPUProfile()
	}
	filename, offset, err := parsePos(*pos)
	if err != nil {
		fatal(err)
	}

	var overlay map[string][]byte
	if *modified {
		overlay, err = buildutil.ParseOverlayArchive(archiveReader)
		if err != nil {
			fatal(fmt.Errorf("invalid archive: %v", err))
		}
	}

	d, err := Run(filename, offset, overlay)
	if err != nil {
		fatal(err)
	}

	if *jsonOutput {
		json.NewEncoder(os.Stdout).Encode(d)
	} else {
		fmt.Println(d.String())
	}
}

// Load loads the package containing the specified file and returns the AST file
// containing the search position.  It can optionally load modified files from
// an overlay archive.
func Load(filename string, offset int, overlay map[string][]byte) (*packages.Package, []ast.Node, error) {
	type result struct {
		nodes []ast.Node
		err   error
	}
	ch := make(chan result, 1)

	// Adapted from: https://github.com/ianthehat/godef
	fstat, fstatErr := os.Stat(filename)
	parseFile := func(fset *token.FileSet, fname string, src []byte) (*ast.File, error) {
		var (
			err error
			s   os.FileInfo
		)
		isInputFile := false
		if filename == fname {
			isInputFile = true
		} else if fstatErr != nil {
			isInputFile = false
		} else if s, err = os.Stat(fname); err == nil {
			isInputFile = os.SameFile(fstat, s)
		}

		mode := parser.ParseComments
		if isInputFile && debugAST {
			mode |= parser.Trace
		}
		file, err := parser.ParseFile(fset, fname, src, mode)
		if file == nil {
			if isInputFile {
				ch <- result{nil, err}
			}
			return nil, err
		}
		var keepFunc *ast.FuncDecl
		if isInputFile {
			// find the start of the file (which may be before file.Pos() if there are
			//  comments before the package clause)
			start := file.Pos()
			if len(file.Comments) > 0 && file.Comments[0].Pos() < start {
				start = file.Comments[0].Pos()
			}

			pos := start + token.Pos(offset)
			if pos > file.End() {
				err := fmt.Errorf("cursor %d is beyond end of file %s (%d)", offset, fname, file.End()-file.Pos())
				ch <- result{nil, err}
				return file, err
			}
			path, _ := astutil.PathEnclosingInterval(file, pos, pos)
			if len(path) < 1 {
				err := fmt.Errorf("offset was not a valid token")
				ch <- result{nil, err}
				return nil, err
			}

			// if we are inside a function, we need to retain that function body
			// start from the top not the bottom
			for i := len(path) - 1; i >= 0; i-- {
				if f, ok := path[i].(*ast.FuncDecl); ok {
					keepFunc = f
					break
				}
			}
			ch <- result{path, nil}
		}
		// and drop all function bodies that are not relevant so they don't get
		// type checked
		for _, decl := range file.Decls {
			if f, ok := decl.(*ast.FuncDecl); ok && f != keepFunc {
				f.Body = nil
			}
		}
		return file, err
	}
	cfg := &packages.Config{
		Overlay:   overlay,
		Mode:      packages.LoadAllSyntax,
		ParseFile: parseFile,
		Tests:     strings.HasSuffix(filename, "_test.go"),
	}
	pkgs, err := packages.Load(cfg, fmt.Sprintf("file=%s", filename))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot load package containing %s: %v", filename, err)
	}
	if len(pkgs) == 0 {
		return nil, nil, fmt.Errorf("no package containing file %s", filename)
	}
	// Arbitrarily return the first package if there are multiple.
	// TODO: should the user be able to specify which one?
	if len(pkgs) > 1 {
		log.Printf("packages not processed: %v\n", pkgs[1:])
	}

	r := <-ch
	if r.err != nil {
		return nil, nil, err
	}
	return pkgs[0], r.nodes, nil
}

// Run is a wrapper for the gogetdoc command.  It is broken out of main for easier testing.
func Run(filename string, offset int, overlay map[string][]byte) (*Doc, error) {
	pkg, nodes, err := Load(filename, offset, overlay)
	if err != nil {
		return nil, err
	}
	return DocFromNodes(pkg, nodes)
}

// DocFromNodes gets the documentation from the AST node(s) in the specified package.
func DocFromNodes(pkg *packages.Package, nodes []ast.Node) (*Doc, error) {
	for _, node := range nodes {
		// log.Printf("node is a %T\n", node)
		switch node := node.(type) {
		case *ast.ImportSpec:
			return PackageDoc(pkg, ImportPath(node))
		case *ast.Ident:
			// if we can't find the object denoted by the identifier, keep searching)
			if obj := pkg.TypesInfo.ObjectOf(node); obj == nil {
				continue
			}
			return IdentDoc(node, pkg.TypesInfo, pkg)
		default:
			break
		}
	}
	return nil, errors.New("gogetdoc: no documentation found")
}

// parsePos parses the search position as provided on the command line.
// It should be of the form: foo.go:#123
func parsePos(p string) (filename string, offset int, err error) {
	if p == "" {
		return "", 0, errors.New("missing required -pos flag")
	}
	sep := strings.LastIndex(p, ":")
	// need at least 2 characters after the ':'
	// (the # sign and the offset)
	if sep == -1 || sep > len(p)-2 || p[sep+1] != '#' {
		return "", 0, fmt.Errorf("invalid option: -pos=%s", p)
	}
	filename = p[:sep]
	off, err := strconv.ParseInt(p[sep+2:], 10, 32)
	return filename, int(off), err
}
