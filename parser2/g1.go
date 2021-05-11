package parser2

import (
	"bytes"
	"errors"
	"github.com/golangee/tadl/token"
	"io"
)

// g1IsEscaped checks if the last read rune is a '/'.
func (d *Decoder) g1IsEscaped() bool {
	if r, ok := d.lastRune(-2); ok {
		return r == '\\'
	}

	return false
}

func (d *Decoder) g1StartElement() (*StartElement, error) {
	startPos := d.Pos()
	startPos.Col--
	var tmp bytes.Buffer
	for {
		r, err := d.nextR()
		if errors.Is(err, io.EOF) {
			if tmp.Len() == 0 {
				return nil, io.EOF
			}

			break
		}

		if r == ' ' || r == '\t' {
			d.prevR() // revert last read char
			break
		}

		tmp.WriteRune(r)
	}

	if tmp.Len() == 0 {
		return nil, token.NewPosError(token.NewNode(startPos, d.Pos()), "empty element name not allowed")
	}

	start := &StartElement{}
	start.BeginPos = startPos
	start.EndPos = d.Pos()
	start.Value = tmp.String()

	return start, nil
}

// g1Text parses a text sequence until next control char # or } or EOF.
func (d *Decoder) g1Text() (*CharData, error) {
	startPos := d.Pos()
	var tmp bytes.Buffer
	for {
		r, err := d.nextR()
		if errors.Is(err, io.EOF) {
			if tmp.Len() == 0 {
				return nil, io.EOF
			}

			break
		}

		if err != nil {
			return nil, err
		}

		tmp.WriteRune(r)

		if r == '#' || r == '}' {
			if d.g1IsEscaped() {
				tmp.Truncate(tmp.Len() - 2)
				tmp.WriteRune(r)
			} else {
				tmp.Truncate(tmp.Len() - 1)
				d.prevR() // reset last read char
				break
			}

		}
	}

	text := &CharData{}
	text.Value = tmp.String()
	text.Position.BeginPos = startPos
	text.Position.EndPos = d.pos

	return text, nil
}
