package parser2

import (
	"bytes"
	"errors"
	"io"
	"strconv"

	"github.com/golangee/tadl/token"
)

func debugStrOfRune(r rune) string {
	return "'" + string(r) + "' (0x" + strconv.FormatInt(int64(r), 16) + ")"
}

// g1Text parses a text sequence until next control char # or } or EOF.
// If stopAtNewline is set, we will stop the parser before the newline rune.
func (d *Decoder) g1Text(stopAtNewline bool) (*CharData, error) {
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

		if stopAtNewline && r == '\n' {
			d.prevR()
			break
		}

		if r == '#' || r == '}' {
			if d.gIsEscaped() {
				// Remove previous '\'
				tmp.Truncate(tmp.Len() - 1)
			} else {
				d.prevR() // reset last read char
				break
			}
		}

		tmp.WriteRune(r)
	}

	text := &CharData{}
	text.Value = tmp.String()
	text.Position.BeginPos = startPos
	text.Position.EndPos = d.pos

	return text, nil
}

func (d *Decoder) g1LineEnd() (*G1LineEnd, error) {
	startPos := d.Pos()

	r, _ := d.nextR()
	if r != '\n' {
		return nil, token.NewPosError(d.node(), "expected newline")
	}

	lineEnd := &G1LineEnd{}
	lineEnd.Position.BeginPos = startPos
	lineEnd.Position.EndPos = d.pos

	return lineEnd, nil
}
