package parser

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/golangee/tadl/ast"
	"github.com/golangee/tadl/token"
	"io"
	"io/ioutil"
	"reflect"
)

func ParseMarkup(filename string) (*ast.MLNode, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open markup file: %w", err)
	}

	stack := append([]*ast.MLNode{}, &ast.MLNode{})
	dec := xml.NewDecoder(bytes.NewReader(buf))

	var lastPos token.Pos
	for {
		tok, err := dec.Token()
		if tok == nil && err == io.EOF {
			return stack[0], nil
		}

		line, col := resolveOffsetLocation(buf, int(dec.InputOffset()))
		pos := token.Pos{
			File: filename,
			Line: line,
			Col:  col,
		}

		if err != nil {
			return nil, token.NewPosError(token.NewNode(lastPos, pos), err.Error()).SetCause(err)
		}

		switch t := tok.(type) {
		case xml.CharData:
			fmt.Printf("text:'%s'\n", string(t))
		case xml.StartElement:
			fmt.Printf("elem(: '%s'\n", t.Name.Local)
		case xml.EndElement:
			fmt.Printf("'%s')\n", t.Name.Local)
		default:
			panic(reflect.TypeOf(t))
		}

		fmt.Println(reflect.TypeOf(tok).String())

		lastPos = pos
	}
}

func resolveOffsetLocation(buf []byte, offset int) (line, col int) {
	line = 1
	col = 1
	for i, b := range buf {
		if b == '\n' {
			col = 1
			line++
		} else {
			col++
		}

		if i == offset {
			return line, col
		}
	}

	return
}
