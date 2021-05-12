package parser2

import (
	"bufio"
	"github.com/golangee/tadl/token"
	"io"
	"unicode"
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
	r      *bufio.Reader
	buf    []runeWithPos //TODO truncate to avoid streaming memory leak
	bufPos int
	pos    token.Pos // current position
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

	r, err := d.nextR()
	if err != nil {
		return nil, err
	}

	d.prevR()

	switch r {
	case '#':
		return d.g1Element()
	case ':':
		return d.g1Attribute()
	default:
		return d.g1Text(true)
	}
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
