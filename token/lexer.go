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

// maxBufferSize is the maximum number of runes in our buffer. This limits how often prevR can be called.
// Since prevR does not get called that often, a small number is enough here.
const maxBufferSize = 8

// GrammarMode is used to identify if the lexer is
// in grammar 1, grammar 2, or lexing a single line in grammar 1.
type GrammarMode int

const (
	// G1 is the default text-first mode.
	G1 GrammarMode = iota
	// G2 ist the node-first mode.
	G2
	// G1Line is a single G1 line in G2.
	G1Line
	// G1LineForward is the same as G1Line, but the elements should be forwarded.
	G1LineForward
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
	// WantG2AttributeValue is used when we parsed a '=' in G2 and now expect chardata.
	WantG2AttributeValue WantMode = "WantG2AttributeValue"
)

// A Token is an interface for all possible token types.
type Token interface {
	Type() Type
	Pos() *Position
}

type Type string

type runeWithPos struct {
	r    rune
	line int32
	col  int32
	off  int32
}

// Lexer can be used to get individual tokens.
type Lexer struct {
	r      *bufio.Reader
	buf    []runeWithPos
	bufPos int
	// pos is the current lexer position.
	// It is the position of the rune that would be read next by nextR.
	pos  Pos
	mode GrammarMode
	want WantMode
	// To know when we need to switch back from G2, we need to count how many open/closed
	// brackets have occurred. For an open bracket we add one, for a closed bracket we
	// remove one. When the counter then reaches 0 we switch back to G1.
	g2BracketCounter uint
}

// NewLexer creates a new instance, ready to start parsing.
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
// The lexer start of in G1 mode. Should a user of a Lexer detect a token that
// indicates a mode change, it is THEIR responsibility to change the lexer's
// mode accordingly.
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

	// Special handling for G1 attributes
	//nolint:exhaustive
	switch l.want {
	case WantG1AttributeIdent:
		tok, err = l.gIdent()
		if err != nil {
			return nil, err
		}

		if l.mode == G1Line {
			_ = l.gSkipWhitespace('\n')
		} else {
			_ = l.gSkipWhitespace()
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
			tok, err = l.gText("}\n")
		} else {
			tok, err = l.gText("}")
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
			_ = l.gSkipWhitespace('\n')
		} else {
			_ = l.gSkipWhitespace()
		}

		return tok, err
	}

	switch l.mode {
	case G1:
		if l.want == WantIdentifier {
			tok, err = l.gIdent()
			_ = l.gSkipWhitespace()
			l.want = WantNothing
		} else if l.want == WantCommentLine {
			tok, err = l.gText("#")
			l.want = WantNothing
		} else if r1 == '#' && r2 == '!' {
			tok, err = l.g2Preamble()
			l.mode = G2
			_ = l.gSkipWhitespace()
		} else if r1 == '#' && r2 == '?' {
			tok, err = l.g1CommentStart()
			l.want = WantCommentLine
			_ = l.gSkipWhitespace()
		} else if r1 == '#' {
			tok, err = l.gDefineElement()
			l.want = WantIdentifier
		} else if r1 == '@' {
			tok, err = l.gDefineAttribute()
			l.want = WantG1AttributeIdent
		} else if r1 == '{' {
			tok, err = l.gBlockStart()
			_ = l.gSkipWhitespace()
		} else if r1 == '}' {
			tok, err = l.gBlockEnd()
			_ = l.gSkipWhitespace()
		} else {
			tok, err = l.gText("#}")
		}
	case G1Line:
		if r1 == '\n' {
			// Newline marks the end of this G1Line. Switch back to G2.
			tok, err = l.g1LineEnd()
			l.want = WantNothing
			l.mode = G2
			_ = l.gSkipWhitespace()
		} else if l.want == WantIdentifier {
			tok, err = l.gIdent()
			l.want = WantNothing
			_ = l.gSkipWhitespace('\n')
		} else if r1 == '#' {
			tok, err = l.gDefineElement()
			l.want = WantIdentifier
		} else if r1 == '@' {
			tok, err = l.gDefineAttribute()
			l.want = WantG1AttributeIdent
		} else if r1 == '{' {
			tok, err = l.gBlockStart()
			_ = l.gSkipWhitespace('\n')
		} else if r1 == '}' {
			tok, err = l.gBlockEnd()
			_ = l.gSkipWhitespace('\n')
		} else {
			tok, err = l.gText("#}\n")
		}
	case G2:
		if l.want == WantCommentLine {
			tok, err = l.gText("\n")
			l.want = WantNothing
			_ = l.gSkipWhitespace()
		} else if l.want == WantG2AttributeValue {
			tok, err = l.g2CharData()
			l.want = WantNothing
			_ = l.gSkipWhitespace()
		} else if r1 == '{' {
			tok, err = l.gBlockStart()
			l.g2BracketCounter++
			_ = l.gSkipWhitespace()
		} else if r1 == '}' {
			tok, err = l.gBlockEnd()
			l.g2BracketCounter--
			l.checkSwitchToG1()
			_ = l.gSkipWhitespace()
		} else if r1 == '(' {
			tok, err = l.g2GroupStart()
			l.g2BracketCounter++
			_ = l.gSkipWhitespace()
		} else if r1 == ')' {
			tok, err = l.g2GroupEnd()
			l.g2BracketCounter--
			l.checkSwitchToG1()
			_ = l.gSkipWhitespace()
		} else if r1 == '<' {
			tok, err = l.g2GenericStart()
			l.g2BracketCounter++
			_ = l.gSkipWhitespace()
		} else if r1 == '>' {
			tok, err = l.g2GenericEnd()
			l.g2BracketCounter--
			l.checkSwitchToG1()
			_ = l.gSkipWhitespace()
		} else if r1 == '"' {
			tok, err = l.g2CharData()
			l.checkSwitchToG1()
			_ = l.gSkipWhitespace()
		} else if r1 == '@' {
			tok, err = l.gDefineAttribute()
		} else if r1 == '#' {
			// A '#' marks the start of a G1 line.
			tok, err = l.gDefineElement()
			l.mode = G1Line
			_ = l.gSkipWhitespace('\n')
		} else if r1 == '=' {
			tok, err = l.g2Assign()
			l.want = WantG2AttributeValue
			_ = l.gSkipWhitespace()
		} else if r1 == ',' {
			tok, err = l.g2Comma()
			l.checkSwitchToG1()
			_ = l.gSkipWhitespace()
		} else if r1 == ';' {
			tok, err = l.g2Semicolon()
			l.checkSwitchToG1()
			_ = l.gSkipWhitespace()
		} else if r1 == '/' {
			tok, err = l.g2CommentStart()
			l.want = WantCommentLine
			_ = l.gSkipWhitespace('\n')
		} else if r1 == '-' && r2 == '>' {
			tok, err = l.g2Arrow()
			_ = l.gSkipWhitespace()
		} else if l.gIdentChar(r1) {
			tok, err = l.gIdent()
			_ = l.gSkipWhitespace()
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

// checkSwitchToG1 will check the bracketCounter and, if it is 0, set the lexer's mode to G1.
func (l *Lexer) checkSwitchToG1() {
	if l.g2BracketCounter == 0 {
		l.mode = G1
	}
}

// nextR reads the next rune and updates the position.
func (l *Lexer) nextR() (rune, error) {
	if l.bufPos < len(l.buf) {
		r := l.buf[l.bufPos]
		l.bufPos++
		l.pos.Line = int(r.line)
		// col needs to be incremented so that the lexer points to the next rune.
		l.pos.Col = int(r.col) + 1
		l.pos.Offset = int(r.off) + 1

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
		off:  int32(l.pos.Offset),
	})
	l.bufPos++

	// Should the buffer get longer than maxBufferSize we will remove the first element from it.
	if len(l.buf) > maxBufferSize {
		l.buf = l.buf[1:]
		l.bufPos = len(l.buf)
	}

	l.pos.Offset += size
	l.pos.Col++

	if r == '\n' {
		l.pos.Line++
		l.pos.Col = 1
	}

	return r, err
}

// prevR unreads the current rune. panics if out of balance with nextR or if it was called
// more than maxBufferSize times in succession.
func (l *Lexer) prevR() {
	l.bufPos--
	r := l.buf[l.bufPos]
	l.pos.Line = int(r.line)
	l.pos.Col = int(r.col)
	l.pos.Offset = int(r.off)
}

// node returns a fake node for positional errors.
func (l *Lexer) node() Node {
	return NewNode(l.Pos(), l.Pos())
}

// Pos returns the current position of the token parser.
func (l *Lexer) Pos() Pos {
	return l.pos
}
