package parser2

import (
	"bytes"
	"errors"
	"io"

	"github.com/golangee/tadl/token"
)

// g2Preambel reads the '#!' preambel of G2 grammars.
func (d *Decoder) g2Preambel() (*G2Preambel, error) {
	startPos := d.Pos()

	// Eat '#!' from input
	r, _ := d.nextR()
	if r != '#' {
		return nil, token.NewPosError(d.node(), "expected '#' in g2 mode")
	}
	r, _ = d.nextR()
	if r != '!' {
		return nil, token.NewPosError(d.node(), "expected '!' in g2 mode")
	}

	preambel := &G2Preambel{}
	preambel.Position.BeginPos = startPos
	preambel.Position.EndPos = d.pos

	return preambel, nil
}

// g2CharData reads a "quoted string".
func (d *Decoder) g2CharData() (*CharData, error) {
	startPos := d.Pos()

	// Eat starting '"'
	r, _ := d.nextR()
	if r != '"' {
		return nil, token.NewPosError(d.node(), "expected '\"'")
	}

	var tmp bytes.Buffer
	for {
		r, err := d.nextR()
		if errors.Is(err, io.EOF) {
			break
		}

		if r == '"' && !d.gIsEscaped() {
			break
		}

		tmp.WriteRune(r)
	}

	chardata := &CharData{}
	chardata.Position.BeginPos = startPos
	chardata.Position.EndPos = d.pos
	chardata.Value = tmp.String()

	return chardata, nil
}

// g2Assign reads the '=' in an attribute definition.
func (d *Decoder) g2Assign() (*Assign, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil {
		return nil, err
	}

	if r != '=' {
		return nil, token.NewPosError(d.node(), "expected '=' (attribute definition)")
	}

	assign := &Assign{}
	assign.Position.BeginPos = startPos
	assign.Position.EndPos = d.pos

	return assign, nil
}

// g2Comma reads ',' which separates elements.
func (d *Decoder) g2Comma() (*Comma, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil {
		return nil, err
	}

	if r != ',' {
		return nil, token.NewPosError(d.node(), "expected ','")
	}

	comma := &Comma{}
	comma.Position.BeginPos = startPos
	comma.Position.EndPos = d.pos

	return comma, nil
}
