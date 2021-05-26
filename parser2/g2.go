package parser2

import (
	"bytes"
	"errors"
	"io"

	"github.com/golangee/tadl/token"
)

// g2Preambel reads the '#!' preambel of G2 grammars.
func (l *Lexer) g2Preambel() (*G2Preambel, error) {
	startPos := l.Pos()

	// Eat '#!' from input
	r, _ := l.nextR()
	if r != '#' {
		return nil, token.NewPosError(l.node(), "expected '#' in g2 mode")
	}
	r, _ = l.nextR()
	if r != '!' {
		return nil, token.NewPosError(l.node(), "expected '!' in g2 mode")
	}

	preambel := &G2Preambel{}
	preambel.Position.BeginPos = startPos
	preambel.Position.EndPos = l.pos

	return preambel, nil
}

// g2CharData reads a "quoted string".
func (l *Lexer) g2CharData() (*CharData, error) {
	startPos := l.Pos()

	// Eat starting '"'
	r, _ := l.nextR()
	if r != '"' {
		return nil, token.NewPosError(l.node(), "expected '\"'")
	}

	var tmp bytes.Buffer
	for {
		r, err := l.nextR()
		if errors.Is(err, io.EOF) {
			break
		}

		if r == '"' && !l.gIsEscaped() {
			break
		}

		tmp.WriteRune(r)
	}

	chardata := &CharData{}
	chardata.Position.BeginPos = startPos
	chardata.Position.EndPos = l.pos
	chardata.Value = tmp.String()

	return chardata, nil
}

// g2Assign reads the '=' in an attribute definition.
func (l *Lexer) g2Assign() (*Assign, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil {
		return nil, err
	}

	if r != '=' {
		return nil, token.NewPosError(l.node(), "expected '=' (attribute definition)")
	}

	assign := &Assign{}
	assign.Position.BeginPos = startPos
	assign.Position.EndPos = l.pos

	return assign, nil
}

// g2Comma reads ',' which separates elements.
func (l *Lexer) g2Comma() (*Comma, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil {
		return nil, err
	}

	if r != ',' {
		return nil, token.NewPosError(l.node(), "expected ','")
	}

	comma := &Comma{}
	comma.Position.BeginPos = startPos
	comma.Position.EndPos = l.pos

	return comma, nil
}

// g2Pipe reads '|' which separates elements.
func (l *Lexer) g2Pipe() (*Pipe, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil {
		return nil, err
	}

	if r != '|' {
		return nil, token.NewPosError(l.node(), "expected '|'")
	}

	pipe := &Pipe{}
	pipe.Position.BeginPos = startPos
	pipe.Position.EndPos = l.pos

	return pipe, nil
}

// g2GroupStart reads the '(' that marks the start of a group.
func (l *Lexer) g2GroupStart() (*GroupStart, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '(' {
		return nil, token.NewPosError(l.node(), "expected '('")
	}

	groupStart := &GroupStart{}
	groupStart.Position.BeginPos = startPos
	groupStart.Position.EndPos = l.pos

	return groupStart, nil

}

// g2GroupEnd reads the ')' that marks the end of a group.
func (l *Lexer) g2GroupEnd() (*GroupEnd, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != ')' {
		return nil, token.NewPosError(l.node(), "expected ')'")
	}

	groupEnd := &GroupEnd{}
	groupEnd.Position.BeginPos = startPos
	groupEnd.Position.EndPos = l.pos

	return groupEnd, nil

}

// g2GenericStart reads the '<' that marks the start of a generic group.
func (l *Lexer) g2GenericStart() (*GenericStart, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '<' {
		return nil, token.NewPosError(l.node(), "expected '<'")
	}

	genericStart := &GenericStart{}
	genericStart.Position.BeginPos = startPos
	genericStart.Position.EndPos = l.pos

	return genericStart, nil

}

// g2GenericEnd reads the '>' that marks the end of a generic group.
func (l *Lexer) g2GenericEnd() (*GenericEnd, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '>' {
		return nil, token.NewPosError(l.node(), "expected '>'")
	}

	genericEnd := &GenericEnd{}
	genericEnd.Position.BeginPos = startPos
	genericEnd.Position.EndPos = l.pos

	return genericEnd, nil

}

// g2CommentStart reads a '//' that marks the start of a line comment in G2.
func (l *Lexer) g2CommentStart() (*G2Comment, error) {
	startPos := l.Pos()

	// Eat '//' from input
	for i := 0; i < 2; i++ {
		r, _ := l.nextR()
		if r != '/' {
			return nil, token.NewPosError(l.node(), "expected '//' for line comment")
		}
	}

	comment := &G2Comment{}
	comment.Position.BeginPos = startPos
	comment.Position.EndPos = l.pos

	return comment, nil

}
