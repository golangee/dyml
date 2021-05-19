package parser2

import (
	"bytes"
	"errors"
	"io"

	"github.com/golangee/tadl/token"
)

// Skip whitespace characters and returns the last read rune which is not a whitespace.
func (d *Decoder) g2SkipWhitespace() (rune, error) {

	for {
		r, err := d.nextR()
		if err != nil {
			return r, err
		}

		switch r {
		case ' ', '\t', '\n':
			continue
		default:
			return r, nil
		}
	}
}

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

func (d *Decoder) g2Element() (*Element, error) {
	startPos := d.Pos()

	var tmp bytes.Buffer
	for {
		r, err := d.nextR()
		if errors.Is(err, io.EOF) {
			break
		}

		if !d.g1IdentChar(r) {
			d.prevR() // revert last read char
			break
		}

		tmp.WriteRune(r)
	}

	if tmp.Len() == 0 {
		return nil, token.NewPosError(token.NewNode(startPos, d.Pos()), "empty element name not allowed")
	}

	element := &Element{}
	element.Position.BeginPos = startPos
	element.Position.EndPos = d.pos
	element.Value = CharData{
		Position: element.Position,
		Value:    tmp.String(),
	}

	return element, nil
}

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

		if r == '"' && !d.g1IsEscaped() {
			break
		}

		tmp.WriteRune(r)
	}

	element := &CharData{}
	element.Position.BeginPos = startPos
	element.Position.EndPos = d.pos
	element.Value = tmp.String()

	return element, nil
}

func (d *Decoder) g2BlockStart() (*BlockStart, error) {
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

func (d *Decoder) g2BlockEnd() (*BlockEnd, error) {
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
