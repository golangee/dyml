// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"errors"
	"io"
)

// g2Preamble reads the '#!' preamble of G2 grammars.
func (l *Lexer) g2Preamble() (*G2Preamble, error) {
	startPos := l.Pos()

	// Eat '#!' from input
	if r, _ := l.nextR(); r != '#' {
		return nil, NewPosError(l.node(), "expected '#' in g2 mode")
	}

	if r, _ := l.nextR(); r != '!' {
		return nil, NewPosError(l.node(), "expected '!' in g2 mode")
	}

	preamble := &G2Preamble{}
	preamble.Position.BeginPos = startPos
	preamble.Position.EndPos = l.pos

	return preamble, nil
}

// g2Arrow reads the '->' that indicates a return value in G2.
func (l *Lexer) g2Arrow() (*G2Arrow, error) {
	startPos := l.Pos()

	// Eat '->' from input
	if r, _ := l.nextR(); r != '-' {
		return nil, NewPosError(l.node(), "expected '-'")
	}

	if r, _ := l.nextR(); r != '>' {
		return nil, NewPosError(l.node(), "expected '>'")
	}

	arrow := &G2Arrow{}
	arrow.Position.BeginPos = startPos
	arrow.Position.EndPos = l.pos

	return arrow, nil
}

// g2CharData reads a "quoted string".
func (l *Lexer) g2CharData() (*CharData, error) {
	startPos := l.Pos()

	// Eat starting '"'
	r, _ := l.nextR()
	if r != '"' {
		return nil, NewPosError(l.node(), "expected '\"'")
	}

	text, err := l.g1Text("\"")
	if err != nil {
		return nil, err
	}

	// Eat closing '"'
	r, _ = l.nextR()
	if r != '"' {
		return nil, NewPosError(l.node(), "expected '\"'")
	}

	chardata := &CharData{}
	chardata.Position.BeginPos = startPos
	chardata.Position.EndPos = l.pos
	chardata.Value = text.Value

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
		return nil, NewPosError(l.node(), "expected '=' (attribute definition)")
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
		return nil, NewPosError(l.node(), "expected ','")
	}

	comma := &Comma{}
	comma.Position.BeginPos = startPos
	comma.Position.EndPos = l.pos

	return comma, nil
}

// g2GroupStart reads the '(' that marks the start of a group.
func (l *Lexer) g2GroupStart() (*GroupStart, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '(' {
		return nil, NewPosError(l.node(), "expected '('")
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
		return nil, NewPosError(l.node(), "expected ')'")
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
		return nil, NewPosError(l.node(), "expected '<'")
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
		return nil, NewPosError(l.node(), "expected '>'")
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
			return nil, NewPosError(l.node(), "expected '//' for line comment")
		}
	}

	comment := &G2Comment{}
	comment.Position.BeginPos = startPos
	comment.Position.EndPos = l.pos

	return comment, nil
}
