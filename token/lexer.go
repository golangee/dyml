// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode"
)

// GrammarMode is used to identify if the lexer is
// in grammar 1, grammar 2, or lexing a single line in grammar 1.
type GrammarMode int

const (
	G1 GrammarMode = iota
	G2
	G1Line
)

// WantMode is used to make sure the next token is lexed as a specific thing.
type WantMode string

const (
	// WantNothing indicates that the lexer should operate as usual.
	WantNothing     WantMode = "Nothing"
	WantCommentLine WantMode = "CommentLine"
	WantIdentifier  WantMode = "Identifier"
	// G1 attributes are special, as the whole text inside the brackets has
	// to be lexed as one CharData token. We need several new WantModes to
	// properly expect all tokens in "@key{value}" after a "@" appeared.
	WantG1AttributeIdent    WantMode = "G1AttributeIdent"
	WantG1AttributeStart    WantMode = "G1AttributeStart"
	WantG1AttributeCharData WantMode = "G1AttributeCharData"
	WantG1AttributeEnd      WantMode = "G1AttributeEnd"
)

// A Token is an interface for all possible token types.
type Token interface {
	TokenType() TokenType
	Pos() *Position
}

type TokenType string

type runeWithPos struct {
	r    rune
	line int32
	col  int32
}

// Lexer can be used to get individual tokens.
type Lexer struct {
	r      *bufio.Reader
	buf    []runeWithPos //TODO truncate to avoid streaming memory leak
	bufPos int
	// pos is the current lexer position.
	// It is the position of the rune that would be read next by nextR.
	pos Pos
	// started is only used to detect if the first token is the G2Preamble
	started bool
	mode    GrammarMode
	want    WantMode
}

// NewLexer creates a new instance, ready to start parsing
func NewLexer(filename string, r io.Reader) *Lexer {
	l := &Lexer{}
	l.r = bufio.NewReader(r)
	l.pos.File = filename
	l.pos.Line = 1
	l.pos.Col = 1
	l.want = WantNothing

	return l
}

// Token returns the next dyml token in the input stream.
// At the end of the input stream, Token returns nil, io.EOF.
func (l *Lexer) Token() (Token, error) {
	// Peek the first two runes.
	// The second one is only used to detect the g2 grammar.
	r1, err := l.nextR()
	if err != nil {
		return nil, err
	}

	r2, err := l.nextR()
	if err == nil {
		l.prevR()
	}

	l.prevR()

	var tok Token

	if !l.started {
		l.started = true
		// Find out if we should switch to g2 by checking if the first two runes are '#!'
		if r1 == '#' && r2 == '!' {
			l.mode = G2
			tok, err = l.g2Preamble()
			l.gSkipWhitespace()

			return tok, err
		}
	}

	// Special handling for G1 attributes
	switch l.want {
	case WantG1AttributeIdent:
		tok, err = l.gIdent()
		if err != nil {
			return nil, err
		}

		if l.mode == G1Line {
			l.gSkipWhitespace('\n')
		} else {
			l.gSkipWhitespace()
		}

		l.want = WantG1AttributeStart

		return tok, err
	case WantG1AttributeStart:
		tok, err = l.gBlockStart()
		if err != nil {
			return nil, err
		}

		l.want = WantG1AttributeCharData

		return tok, err
	case WantG1AttributeCharData:
		if l.mode == G1Line {
			tok, err = l.g1Text("}\n")
		} else {
			tok, err = l.g1Text("}")
		}

		if err != nil {
			return nil, err
		}

		l.want = WantG1AttributeEnd

		return tok, err
	case WantG1AttributeEnd:
		tok, err = l.gBlockEnd()
		if err != nil {
			return nil, err
		}

		l.want = WantNothing

		if l.mode == G1Line {
			l.gSkipWhitespace('\n')
		} else {
			l.gSkipWhitespace()
		}

		return tok, err
	}

	switch l.mode {
	case G1:
		if l.want == WantIdentifier {
			tok, err = l.gIdent()
			l.gSkipWhitespace()
			l.want = WantNothing
		} else if l.want == WantCommentLine {
			tok, err = l.gCommentLine()
			l.want = WantNothing
		} else if r1 == '#' && r2 == '?' {
			tok, err = l.g1CommentStart()
			l.want = WantCommentLine
			l.gSkipWhitespace()
		} else if r1 == '#' {
			tok, err = l.gDefineElement()
			l.want = WantIdentifier
		} else if r1 == '@' {
			tok, err = l.gDefineAttribute()
			l.want = WantG1AttributeIdent
		} else if r1 == '{' {
			tok, err = l.gBlockStart()
			l.gSkipWhitespace()
		} else if r1 == '}' {
			tok, err = l.gBlockEnd()
			l.gSkipWhitespace()
		} else {
			tok, err = l.g1Text("#}")
		}
	case G1Line:
		if r1 == '\n' {
			// Newline marks the end of this G1Line. Switch back to G2.
			tok, err = l.g1LineEnd()
			l.mode = G2
			l.want = WantNothing
			l.gSkipWhitespace()
		} else if l.want == WantIdentifier {
			tok, err = l.gIdent()
			l.want = WantNothing
			l.gSkipWhitespace('\n')
		} else if r1 == '#' {
			tok, err = l.gDefineElement()
			l.want = WantIdentifier
		} else if r1 == '@' {
			tok, err = l.gDefineAttribute()
			l.want = WantG1AttributeIdent
		} else if r1 == '{' {
			tok, err = l.gBlockStart()
			l.gSkipWhitespace('\n')
		} else if r1 == '}' {
			tok, err = l.gBlockEnd()
			l.gSkipWhitespace('\n')
		} else {
			tok, err = l.g1Text("#}\n")
		}
	case G2:
		if l.want == WantCommentLine {
			tok, err = l.gCommentLine()
			l.want = WantNothing
			l.gSkipWhitespace()
		} else if r1 == '{' {
			tok, err = l.gBlockStart()
			l.gSkipWhitespace()
		} else if r1 == '}' {
			tok, err = l.gBlockEnd()
			l.gSkipWhitespace()
		} else if r1 == '(' {
			tok, err = l.g2GroupStart()
			l.gSkipWhitespace()
		} else if r1 == ')' {
			tok, err = l.g2GroupEnd()
			l.gSkipWhitespace()
		} else if r1 == '<' {
			tok, err = l.g2GenericStart()
			l.gSkipWhitespace()
		} else if r1 == '>' {
			tok, err = l.g2GenericEnd()
			l.gSkipWhitespace()
		} else if r1 == '"' {
			tok, err = l.g2CharData()
			l.gSkipWhitespace()
		} else if r1 == '@' {
			tok, err = l.gDefineAttribute()
		} else if r1 == '#' {
			// A '#' marks the start of a G1 line.
			tok, err = l.gDefineElement()
			l.mode = G1Line
			l.gSkipWhitespace('\n')
		} else if r1 == '=' {
			tok, err = l.g2Assign()
			l.gSkipWhitespace()
		} else if r1 == ',' {
			tok, err = l.g2Comma()
			l.gSkipWhitespace()
		} else if r1 == '/' {
			tok, err = l.g2CommentStart()
			l.want = WantCommentLine
			l.gSkipWhitespace('\n')
		} else if r1 == '-' && r2 == '>' {
			tok, err = l.g2Arrow()
			l.gSkipWhitespace()
		} else if l.gIdentChar(r1) {
			tok, err = l.gIdent()
			l.gSkipWhitespace()
		} else {
			return nil, NewPosError(l.node(), fmt.Sprintf("unexpected char '%c'", r1))
		}
	default:
		return nil, fmt.Errorf("lexer is in unknown mode (%d), this is a bug", l.mode)
	}

	// An EOF might occur while reading a token.
	// If we got a token while reading, we do not want to return EOF just yet.
	// That will then happen in the next call to Token().
	if err != nil {
		if errors.Is(err, io.EOF) {
			if tok == nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return tok, nil
}

// nextR reads the next rune and updates the position.
func (l *Lexer) nextR() (rune, error) {
	if l.bufPos < len(l.buf) {
		r := l.buf[l.bufPos]
		l.bufPos++
		l.pos.Line = int(r.line)
		// col needs to be incremented so that the lexer points to the next rune.
		l.pos.Col = int(r.col) + 1

		if r.r == '\n' {
			l.pos.Line++
			l.pos.Col = 1
		}

		return r.r, nil
	}

	r, size, err := l.r.ReadRune()
	if r == unicode.ReplacementChar {
		return r, NewPosError(l.node(), "invalid unicode sequence")
	}

	if err != nil {
		return r, NewPosError(l.node(), "unable to read next rune").SetCause(err)
	}

	l.buf = append(l.buf, runeWithPos{
		r:    r,
		line: int32(l.pos.Line),
		col:  int32(l.pos.Col),
	})
	l.bufPos++

	l.pos.Offset += size
	l.pos.Col++

	if r == '\n' {
		l.pos.Line++
		l.pos.Col = 1
	}

	return r, err
}

// prevR unreads the current rune. panics if out of balance with nextR
func (l *Lexer) prevR() rune {
	l.bufPos--
	r := l.buf[l.bufPos]
	l.pos.Line = int(r.line)
	l.pos.Col = int(r.col)

	return r.r
}

// lastRune returns the last read rune without un-read or false.
func (l *Lexer) lastRune(offset int) (rune, bool) {
	if len(l.buf) < -offset {
		return unicode.ReplacementChar, false
	}

	return l.buf[len(l.buf)+offset].r, true
}

// node returns a fake node for positional errors.
func (l *Lexer) node() Node {
	return NewNode(l.Pos(), l.Pos())
}

// Pos returns the current position of the token parser.
func (l *Lexer) Pos() Pos {
	return l.pos
}
