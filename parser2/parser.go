package parser2

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode"

	"github.com/golangee/tadl/token"
)

// A Token is an interface holding one of the token types:
// Element, EndElement, Comment
type Token interface {
	assertToken()
}

type runeWithPos struct {
	r    rune
	line int32
	col  int32
}

// Decoder implements a TADL stream parser.
type Decoder struct {
	r         *bufio.Reader
	buf       []runeWithPos //TODO truncate to avoid streaming memory leak
	bufPos    int
	pos       token.Pos // current position
	lastToken Token
	// g2Mode is true if the decoder has been set to node first mode with #!{...}
	g2Mode bool
}

// NewDecoder creates a new instance, ready to start parsing
func NewDecoder(filename string, r io.Reader) *Decoder {
	d := &Decoder{}
	d.r = bufio.NewReader(r)
	d.pos.File = filename
	d.pos.Line = 1
	d.pos.Col = 1

	return d
}

// Token returns the next TADL token in the input stream.
// At the end of the input stream, Token returns nil, io.EOF.
func (d *Decoder) Token() (Token, error) {
	// Peek the first two runes.
	// The second one is only used to detect the g2 grammar.
	r1, err := d.nextR()
	if err != nil {
		return nil, err
	}
	r2, err := d.nextR()
	if err == nil {
		d.prevR()
	}
	d.prevR()

	var tok Token

	if d.lastToken == nil {
		// No previous token means we just started lexing.
		// Find out if we should switch to g2 by checking if the first two runes are '#!'
		if r1 == '#' && r2 == '!' {
			d.g2Mode = true
			tok, err = d.g2Preambel()
			d.gSkipWhitespace()
			d.lastToken = tok
			return tok, err
		}
	}

	if d.g2Mode {
		// G2 grammar rules

		if r1 == '{' {
			tok, err = d.gBlockStart()
		} else if r1 == '}' {
			tok, err = d.gBlockEnd()
		} else if r1 == '"' {
			tok, err = d.g2CharData()
		} else if r1 == '@' {
			tok, err = d.gDefineAttribute()
		} else if r1 == '#' {
			tok, err = d.gDefineElement()
		} else if r1 == '=' {
			tok, err = d.g2Assign()
		} else if d.gIdentChar(r1) {
			tok, err = d.gIdent()
		} else {
			return nil, token.NewPosError(d.node(), fmt.Sprintf("unexpected char '%c'", r1))
		}
		d.gSkipWhitespace()
	} else {
		// G1 grammar rules

		if _, ok := d.lastToken.(*DefineElement); ok {
			tok, err = d.gIdent()
			d.gSkipWhitespace()
		} else if _, ok := d.lastToken.(*DefineAttribute); ok {
			tok, err = d.gIdent()
			d.gSkipWhitespace()
		} else if r1 == '#' {
			tok, err = d.gDefineElement()
		} else if r1 == '@' {
			tok, err = d.gDefineAttribute()
		} else if r1 == '{' {
			tok, err = d.gBlockStart()
		} else if r1 == '}' {
			tok, err = d.gBlockEnd()
			d.gSkipWhitespace()
		} else {
			tok, err = d.g1Text()
		}
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
		d.lastToken = tok
	}

	return tok, nil
}

// nextR reads the next rune and updates the position.
func (d *Decoder) nextR() (rune, error) {
	if d.bufPos < len(d.buf) {
		r := d.buf[d.bufPos]
		d.bufPos++
		d.pos.Line = int(r.line)
		d.pos.Col = int(r.col)

		return r.r, nil
	}

	r, size, err := d.r.ReadRune()
	if r == unicode.ReplacementChar {
		return r, token.NewPosError(d.node(), "invalid unicode sequence")
	}

	if err != nil {
		return r, token.NewPosError(d.node(), "unable to read next rune").SetCause(err)
	}

	d.buf = append(d.buf, runeWithPos{
		r:    r,
		line: int32(d.pos.Line),
		col:  int32(d.pos.Col),
	})
	d.bufPos++

	d.pos.Offset += size
	d.pos.Col++

	if r == '\n' {
		d.pos.Line++
		d.pos.Col = 1
	}

	return r, err
}

// prevR unreads the current rune. panics if out of balance with nextR
func (d *Decoder) prevR() rune {
	d.bufPos--
	r := d.buf[d.bufPos]
	d.pos.Line = int(r.line)
	d.pos.Col = int(r.col)

	return r.r
}

// lastRune returns the last read rune without un-read or false.
func (d *Decoder) lastRune(offset int) (rune, bool) {
	if len(d.buf) < -offset {
		return unicode.ReplacementChar, false
	}

	return d.buf[len(d.buf)+offset].r, true
}

// node returns a fake node for positional errors.
func (d *Decoder) node() token.Node {
	return token.NewNode(d.Pos(), d.Pos())
}

// Pos returns the current position of the token parser.
func (d *Decoder) Pos() token.Pos {
	return d.pos
}
