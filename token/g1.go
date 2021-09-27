// SPDX-FileCopyrightText: © 2021 The tadl authors <https://github.com/golangee/tadl/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

// g1Text parses a text sequence until next rune is in stopAt or EOF.
func (l *Lexer) g1Text(stopAt string) (*CharData, error) {
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

		if strings.ContainsRune(stopAt, r) {
			if l.gIsEscaped() {

				// Remove previous '\'
				tmp.Truncate(tmp.Len() - 1)
				//l.prevR()
				//break
			} else {
				l.prevR() // reset last read char

				break
			}
		} else {
			if l.gIsEscaped() {
				if r != '\\' {
					return nil, NewPosError(l.node(), "unexpected rune")
				}
			}
		}

		tmp.WriteRune(r)
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
