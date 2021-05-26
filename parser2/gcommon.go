package parser2

import (
	"bytes"
	"errors"
	"io"

	"github.com/golangee/tadl/token"
)

// gBlockStart reads the '{' that marks the end of a block.
func (d *Decoder) gBlockStart() (*BlockStart, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '{' {
		return nil, token.NewPosError(d.node(), "expected '{'")
	}

	blockStart := &BlockStart{}
	blockStart.Position.BeginPos = startPos
	blockStart.Position.EndPos = d.pos

	return blockStart, nil

}

// gBlockEnd reads the '}' that marks the end of a block.
func (d *Decoder) gBlockEnd() (*BlockEnd, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '}' {
		return nil, token.NewPosError(d.node(), "expected '}'")
	}

	blockEnd := &BlockEnd{}
	blockEnd.Position.BeginPos = startPos
	blockEnd.Position.EndPos = d.pos

	return blockEnd, nil

}

// Skip whitespace characters
func (d *Decoder) gSkipWhitespace() error {

	for {
		r, err := d.nextR()
		if err != nil {
			return err
		}

		switch r {
		case ' ', '\t', '\n':
			continue
		default:
			// We got a non-whitespace, rewind and return
			d.prevR()
			return nil
		}
	}
}

// gIdent parses a text sequence until next control char # or } or EOF or whitespace.
func (d *Decoder) gIdent() (*Identifier, error) {
	startPos := d.Pos()
	var tmp bytes.Buffer
	for {
		r, err := d.nextR()
		if errors.Is(err, io.EOF) {
			if tmp.Len() == 0 {
				return nil, io.EOF
			}

			break
		}

		if err != nil {
			return nil, err
		}

		if !d.gIdentChar(r) {
			d.prevR() // reset last read char
			break
		}

		tmp.WriteRune(r)

	}

	ident := &Identifier{}
	ident.Value = tmp.String()
	ident.Position.BeginPos = startPos
	ident.Position.EndPos = d.pos

	return ident, nil
}

// gIdentChar is [a-zA-Z0-9_]
func (d *Decoder) gIdentChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || (r == '_')
}

// gDefineAttribute reads the '@' that starts an attribute.
func (d *Decoder) gDefineAttribute() (*DefineAttribute, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil {
		return nil, err
	}

	if r != '@' {
		return nil, token.NewPosError(d.node(), "expected '@' (attribute definition)")
	}

	attr := &DefineAttribute{}

	// Check if this is a forwarding attribute
	r, err = d.nextR()
	if r == '@' {
		attr.Forward = true
	} else {
		d.prevR()
	}

	attr.Position.BeginPos = startPos
	attr.Position.EndPos = d.pos

	return attr, nil
}

// gDefineElement reads the '#' that starts an element in G1 or switches to a G1-line in G2.
func (d *Decoder) gDefineElement() (*DefineElement, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil {
		return nil, err
	}

	if r != '#' {
		return nil, token.NewPosError(d.node(), "expected '#' (element definition)")
	}

	define := &DefineElement{}

	// Check if this is a forwarding element
	r, err = d.nextR()
	if r == '#' {
		define.Forward = true
	} else {
		d.prevR()
	}

	define.Position.BeginPos = startPos
	define.Position.EndPos = d.pos

	return define, nil
}

// gIsEscaped checks if the last read rune is a '\'.
func (d *Decoder) gIsEscaped() bool {
	if r, ok := d.lastRune(-2); ok {
		return r == '\\'
	}

	return false
}
