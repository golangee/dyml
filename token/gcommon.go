// SPDX-FileCopyrightText: © 2021 The tadl authors <https://github.com/golangee/tadl/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

// gBlockStart reads the '{' that marks the start of a block.
func (l *Lexer) gBlockStart() (*BlockStart, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '{' {
		return nil, NewPosError(l.node(), "expected '{'")
	}

	blockStart := &BlockStart{}
	blockStart.Position.BeginPos = startPos
	blockStart.Position.EndPos = l.pos

	return blockStart, nil
}

// gBlockEnd reads the '}' that marks the end of a block.
func (l *Lexer) gBlockEnd() (*BlockEnd, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if r != '}' {
		return nil, NewPosError(l.node(), "expected '}'")
	}

	blockEnd := &BlockEnd{}
	blockEnd.Position.BeginPos = startPos
	blockEnd.Position.EndPos = l.pos

	return blockEnd, nil
}

// gSkipWhitespace skips whitespace characters.
// Any whitespace characters in dontSkip will not be skipped.
func (l *Lexer) gSkipWhitespace(dontSkip ...rune) error {
	whitespaces := " \n\t"
	dontSkipStr := string(dontSkip)

	for {
		r, err := l.nextR()
		if err != nil {
			return err
		}

		if strings.ContainsRune(whitespaces, r) && !strings.ContainsRune(dontSkipStr, r) {
			// skip this character
			continue
		} else {
			// We got a non-whitespace, rewind and return
			l.prevR()

			return nil
		}
	}
}

// gIdent parses a text sequence until next control char # or } or EOF or whitespace.
func (l *Lexer) gIdent() (*Identifier, error) {
	startPos := l.Pos()

	var tmp bytes.Buffer

	for {
		r, err := l.nextR()
		if errors.Is(err, io.EOF) {
			if tmp.Len() == 0 {
				return nil, io.EOF
			}

			break
		}

		if err != nil {
			return nil, err
		}

		if !l.gIdentChar(r) {
			l.prevR() // reset last read char

			break
		}

		tmp.WriteRune(r)
	}

	if tmp.Len() == 0 {
		return nil, NewPosError(l.node(), "expected identifier")
	}

	ident := &Identifier{}
	ident.Value = tmp.String()
	ident.Position.BeginPos = startPos
	ident.Position.EndPos = l.pos

	return ident, nil
}

// gIdentChar is [a-zA-Z0-9_]
func (l *Lexer) gIdentChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || (r == '_')
}

// gDefineAttribute reads the '@' that starts an attribute.
func (l *Lexer) gDefineAttribute() (*DefineAttribute, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil {
		return nil, err
	}

	if r != '@' {
		return nil, NewPosError(l.node(), "expected '@' (attribute definition)")
	}

	attr := &DefineAttribute{}

	// Check if this is a forwarding attribute
	r, err = l.nextR()
	if r == '@' {
		attr.Forward = true
	} else if err == nil {
		l.prevR()
	}

	attr.Position.BeginPos = startPos
	attr.Position.EndPos = l.pos

	return attr, nil
}

// gDefineElement reads the '#' that starts an element in G1 or switches to a G1-line in G2.
func (l *Lexer) gDefineElement() (*DefineElement, error) {
	startPos := l.Pos()

	r, err := l.nextR()
	if err != nil {
		return nil, err
	}

	if r != '#' {
		return nil, NewPosError(l.node(), "expected '#' (element definition)")
	}

	define := &DefineElement{}

	// Check if this is a forwarding element
	r, err = l.nextR()
	if r == '#' {
		define.Forward = true
	} else if err == nil {
		l.prevR()
	}

	define.Position.BeginPos = startPos
	define.Position.EndPos = l.pos

	return define, nil
}

// gIsEscaped checks if the last read rune is a '\'.
func (l *Lexer) gIsEscaped() bool {
	if r, ok := l.lastRune(-2); ok {
		return r == '\\'
	}

	return false
}

// gCommentLine reads arbitrary text for the rest of the line.
func (l *Lexer) gCommentLine() (*CharData, error) {
	startPos := l.Pos()

	var tmp bytes.Buffer

	for {
		r, err := l.nextR()
		if errors.Is(err, io.EOF) {
			if tmp.Len() == 0 {
				return nil, io.EOF
			}

			break
		}

		if err != nil {
			return nil, err
		}

		if r == '\n' {
			break
		}

		tmp.WriteRune(r)
	}

	text := &CharData{}
	text.Value = tmp.String()
	text.Position.BeginPos = startPos
	text.Position.EndPos = l.pos

	return text, nil
}
