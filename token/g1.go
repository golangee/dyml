// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

// g1Text parses a text sequence until next rune is in stopAt or EOF.
func (l *Lexer) g1Text(stopAt string) (*CharData, error) {
	startPos := l.Pos()

	var tmp bytes.Buffer

	// Keep track of whether the last read char is a '\' to properly escape backslashes
	// and the stopAt characters.
	isEscaping := false

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

		if isEscaping {
			// The last character was a backslash, only backslashes and stopAt characters may follow.
			if strings.ContainsRune(stopAt, r) || r == '\\' {
				// The character was correctly escaped and should be emitted as-is.
				tmp.WriteRune(r)
				isEscaping = false
			} else {
				// Escaping happened, but nothing valid to escape was found!
				return nil, NewPosError(l.node(), fmt.Sprintf("'%c' may not be escaped here", r))
			}
		} else {
			// We are not currently expecting an escaped char, proceed normally.
			if strings.ContainsRune(stopAt, r) {
				// That character is no longer supposed to be in our string, revert the read and stop.
				l.prevR()
				break
			} else if r == '\\' {
				// Enter escape mode and not emit this backslash.
				isEscaping = true
			} else {
				// Any other normal character
				tmp.WriteRune(r)
			}
		}
	}

	text := &CharData{}
	text.Value = tmp.String()
	text.Position.BeginPos = startPos
	text.Position.EndPos = l.pos

	return text, nil
}

func (l *Lexer) g1LineEnd() (*G1LineEnd, error) {
	startPos := l.Pos()

	if r, _ := l.nextR(); r != '\n' {
		return nil, NewPosError(l.node(), "expected newline")
	}

	lineEnd := &G1LineEnd{}
	lineEnd.Position.BeginPos = startPos
	lineEnd.Position.EndPos = l.pos

	return lineEnd, nil
}

// g1CommentStart reads a '#?' that marks the start of a comment in G1.
func (l *Lexer) g1CommentStart() (*G1Comment, error) {
	startPos := l.Pos()

	// Eat '#?' from input
	if r, _ := l.nextR(); r != '#' {
		return nil, NewPosError(l.node(), "expected '#?' for comment")
	}

	if r, _ := l.nextR(); r != '?' {
		return nil, NewPosError(l.node(), "expected '#?' for comment")
	}

	comment := &G1Comment{}
	comment.Position.BeginPos = startPos
	comment.Position.EndPos = l.pos

	return comment, nil
}
