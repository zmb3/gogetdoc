package main

import (
	"bytes"
	"fmt"
	"go/doc"
)

const (
	indent    = ""
	preIndent = "    "
)

// Doc holds the resulting documentation for a particular item.
type Doc struct {
	Name   string `json:"name"`
	Import string `json:"import"`
	Pkg    string `json:"pkg"`
	Decl   string `json:"decl"`
	Doc    string `json:"doc"`
	Pos    string `json:"pos"`
}

func (d *Doc) String() string {
	buf := &bytes.Buffer{}
	if d.Import != "" {
		fmt.Fprintf(buf, "import \"%s\"\n\n", d.Import)
	}
	fmt.Fprintf(buf, "%s\n\n", d.Decl)
	if d.Doc == "" {
		d.Doc = "Undocumented."
	}
	doc.ToText(buf, d.Doc, indent, preIndent, *linelength)
	return buf.String()
}
