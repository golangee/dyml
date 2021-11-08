// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
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

// gIdent parses an identifier, which is a dot separated sequence of [a-zA-Z0-9_].
func (l *Lexer) gIdent() (*Identifier, error) {
	startPos := l.Pos()

	// When this is true we have to get and identChar, anything is an error.
	// This is true at the start and after a '.'.
	requireChar := true

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

		if requireChar {
			requireChar = false
			// Require a character
			if !l.gIdentChar(r) {
				return nil, NewPosError(l.node(), "expected identifier")
			}
		} else if r == '.' {
			// After a dot we require another identifier.
			requireChar = true
		} else if l.gIdentChar(r) {
			// Okay, will be added to the buffer later
		} else {
			// We reached the end of this identifier, reset the rune and stop
			l.prevR()

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

// gIdentChar is any character of an identifier: [a-zA-Z0-9_].
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
