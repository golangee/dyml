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

// g2Pipe reads '|' which separates elements.
func (d *Decoder) g2Pipe() (*Pipe, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil {
		return nil, err
	}

	if r != '|' {
		return nil, token.NewPosError(d.node(), "expected '|'")
	}

	pipe := &Pipe{}
	pipe.Position.BeginPos = startPos
	pipe.Position.EndPos = d.pos

	return pipe, nil
}

// g2GroupStart reads the '(' that marks the start of a group.
func (d *Decoder) g2GroupStart() (*GroupStart, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '(' {
		return nil, token.NewPosError(d.node(), "expected '('")
	}

	groupStart := &GroupStart{}
	groupStart.Position.BeginPos = startPos
	groupStart.Position.EndPos = d.pos

	return groupStart, nil

}

// g2GroupEnd reads the ')' that marks the end of a group.
func (d *Decoder) g2GroupEnd() (*GroupEnd, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != ')' {
		return nil, token.NewPosError(d.node(), "expected ')'")
	}

	groupEnd := &GroupEnd{}
	groupEnd.Position.BeginPos = startPos
	groupEnd.Position.EndPos = d.pos

	return groupEnd, nil

}

// g2GenericStart reads the '<' that marks the start of a generic group.
func (d *Decoder) g2GenericStart() (*GenericStart, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '<' {
		return nil, token.NewPosError(d.node(), "expected '<'")
	}

	genericStart := &GenericStart{}
	genericStart.Position.BeginPos = startPos
	genericStart.Position.EndPos = d.pos

	return genericStart, nil

}

// g2GenericEnd reads the '>' that marks the end of a generic group.
func (d *Decoder) g2GenericEnd() (*GenericEnd, error) {
	startPos := d.Pos()

	r, err := d.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '>' {
		return nil, token.NewPosError(d.node(), "expected '>'")
	}

	genericEnd := &GenericEnd{}
	genericEnd.Position.BeginPos = startPos
	genericEnd.Position.EndPos = d.pos

	return genericEnd, nil

}

// g2CommentStart reads a '//' that marks the start of a line comment in G2.
func (d *Decoder) g2CommentStart() (*G2Comment, error) {
	startPos := d.Pos()

	// Eat '//' from input
	for i := 0; i < 2; i++ {
		r, _ := d.nextR()
		if r != '/' {
			return nil, token.NewPosError(d.node(), "expected '//' for line comment")
		}
	}

	comment := &G2Comment{}
	comment.Position.BeginPos = startPos
	comment.Position.EndPos = d.pos

	return comment, nil

}
