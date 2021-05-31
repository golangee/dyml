package parser2

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode"

	"github.com/golangee/tadl/token"
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
type WantMode int

const (
	// Nothing special needs to be expected.
	WantNothing WantMode = iota
	WantCommentLine
	WantIdentifier
)

// A Token is an interface for all possible token types.
type Token interface {
	tokenType() TokenType
}

type TokenType string

type runeWithPos struct {
	r    rune
	line int32
	col  int32
}

// Lexer can be used to get individual tokens.
type Lexer struct {
	r         *bufio.Reader
	buf       []runeWithPos //TODO truncate to avoid streaming memory leak
	bufPos    int
	pos       token.Pos // current position
	lastToken Token
	mode      GrammarMode
	want      WantMode
}

// NewLexer creates a new instance, ready to start parsing
func NewLexer(filename string, r io.Reader) *Lexer {
	d := &Lexer{}
	d.r = bufio.NewReader(r)
	d.pos.File = filename
	d.pos.Line = 1
	d.pos.Col = 1

	return d
}

// Token returns the next TADL token in the input stream.
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

	if l.lastToken == nil {
		// No previous token means we just started lexing.
		// Find out if we should switch to g2 by checking if the first two runes are '#!'
		if r1 == '#' && r2 == '!' {
			l.mode = G2
			tok, err = l.g2Preambel()
			l.gSkipWhitespace()
			l.lastToken = tok
			return tok, err
		}
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
			// TODO Comment nodes are not yet supported.
			tok, err = l.g1CommentStart()
			l.want = WantCommentLine
			l.gSkipWhitespace()
		} else if r1 == '#' {
			tok, err = l.gDefineElement()
			l.want = WantIdentifier
		} else if r1 == '@' {
			tok, err = l.gDefineAttribute()
			l.want = WantIdentifier
		} else if r1 == '{' {
			tok, err = l.gBlockStart()
			l.gSkipWhitespace()
		} else if r1 == '}' {
			tok, err = l.gBlockEnd()
			l.gSkipWhitespace()
		} else {
			tok, err = l.g1Text(false)
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
			l.gSkipWhitespace()
		} else if r1 == '#' {
			tok, err = l.gDefineElement()
			l.want = WantIdentifier
		} else if r1 == '@' {
			tok, err = l.gDefineAttribute()
			l.want = WantIdentifier
		} else if r1 == '{' {
			tok, err = l.gBlockStart()
			l.gSkipWhitespace()
		} else if r1 == '}' {
			tok, err = l.gBlockEnd()
			l.gSkipWhitespace()
		} else {
			tok, err = l.g1Text(true)
		}
	case G2:
		if l.want == WantCommentLine {
			tok, err = l.gCommentLine()
			l.want = WantNothing
		} else if r1 == '{' {
			tok, err = l.gBlockStart()
		} else if r1 == '}' {
			tok, err = l.gBlockEnd()
		} else if r1 == '(' {
			tok, err = l.g2GroupStart()
		} else if r1 == ')' {
			tok, err = l.g2GroupEnd()
		} else if r1 == '<' {
			tok, err = l.g2GenericStart()
		} else if r1 == '>' {
			tok, err = l.g2GenericEnd()
		} else if r1 == '"' {
			tok, err = l.g2CharData()
		} else if r1 == '@' {
			tok, err = l.gDefineAttribute()
		} else if r1 == '#' {
			// A '#' marks the start of a G1 line.
			tok, err = l.gDefineElement()
			l.mode = G1Line
		} else if r1 == '=' {
			tok, err = l.g2Assign()
		} else if r1 == ',' {
			tok, err = l.g2Comma()
		} else if r1 == '|' {
			tok, err = l.g2Pipe()
		} else if r1 == '/' {
			tok, err = l.g2CommentStart()
			l.gSkipWhitespace()
			l.want = WantCommentLine
		} else if l.gIdentChar(r1) {
			tok, err = l.gIdent()
		} else {
			return nil, token.NewPosError(l.node(), fmt.Sprintf("unexpected char '%c'", r1))
		}
		l.gSkipWhitespace()
	default:
		return nil, errors.New("lexer in unknown mode")
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

	if tok != nil {
		l.lastToken = tok
	}

	return tok, nil
}

// nextR reads the next rune and updates the position.
func (l *Lexer) nextR() (rune, error) {
	if l.bufPos < len(l.buf) {
		r := l.buf[l.bufPos]
		l.bufPos++
		l.pos.Line = int(r.line)
		l.pos.Col = int(r.col)

		return r.r, nil
	}

	r, size, err := l.r.ReadRune()
	if r == unicode.ReplacementChar {
		return r, token.NewPosError(l.node(), "invalid unicode sequence")
	}

	if err != nil {
		return r, token.NewPosError(l.node(), "unable to read next rune").SetCause(err)
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
func (l *Lexer) node() token.Node {
	return token.NewNode(l.Pos(), l.Pos())
}

// Pos returns the current position of the token parser.
func (l *Lexer) Pos() token.Pos {
	return l.pos
}
