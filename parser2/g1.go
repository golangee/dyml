package parser2

import (
	"bytes"
	"errors"
	"github.com/golangee/tadl/token"
	"io"
	"strconv"
)

// g1IsEscaped checks if the last read rune is a '/'.
func (d *Decoder) g1IsEscaped() bool {
	if r, ok := d.lastRune(-2); ok {
		return r == '\\'
	}

	return false
}

// g1TerminateStatementSymbol checks for normal space, tab-space or open/close curly brace.
func (d *Decoder) g1TerminateStatementSymbol(r rune) bool {
	return r == ' ' || r == '\t' || r == '{' || r == '}'
}

// g1CharDataBlock expects the following grammar lexem:
//   '{' .* '}'
func (d *Decoder) g1CharDataBlock() (*CharData, error) {
	startOfBlock, err := d.nextR()
	if err != nil {
		return nil, err
	}

	if startOfBlock != '{' {
		return nil, token.NewPosError(d.node(), "expected '{:}' (start of char data block)")
	}

	text, err := d.g1Text(false)
	if err != nil {
		return nil, err
	}

	r, err := d.nextR()
	if err != nil {
		return nil, err
	}

	if r != '}' {
		return nil, token.NewPosError(d.node(), "expected '}' (end of char data block) but got "+debugStrOfRune(r))
	}

	return text, nil
}

func debugStrOfRune(r rune) string {
	return "'" + string(r) + "' (0x" + strconv.FormatInt(int64(r), 16) + ")"
}

// g1Attribute expects the following grammar lexem:
//   ':' Identifier '{' CharData '} (\s|'{'|'}'|EOF)
func (d *Decoder) g1Attribute() (*Attr, error) {
	startPos := d.Pos()
	startOfAttr, err := d.nextR()

	if startOfAttr != ':' || err != nil {
		return nil, token.NewPosError(d.node(), "expected ':' (start of attribute) but got "+debugStrOfRune(startOfAttr)).SetCause(err)
	}

	key, err := d.g1Ident()
	if err != nil {
		return nil, token.NewPosError(d.node(), "expected an identifier for the attribute key").SetCause(err)
	}

	_, err = d.g1ConsumeWhitespace()
	if err != nil {
		return nil, err
	}

	value, err := d.g1CharDataBlock()
	if err != nil {
		return nil, err
	}

	if _, err = d.g1ConsumeWhitespace(); err != nil {
		return nil, err
	}

	attr := &Attr{}
	attr.BeginPos = startPos
	attr.EndPos = d.Pos()
	attr.Key = *key
	attr.Value = *value

	return attr, nil
}

// g1ConsumeWhitespace consumes all simple spaces and tabs. EOF is ignored. Returns the amount of runes ignored.
func (d *Decoder) g1ConsumeWhitespace() (int, error) {
	count := 0
	for {
		r, err := d.nextR()
		if errors.Is(err, io.EOF) {
			return count, nil
		}

		if err != nil {
			return count, err
		}

		if r != ' ' && r != '\t' {
			d.prevR() // revert last read char
			return count, nil
		}

		count++
	}
}

// g1Element expects the following grammar lexem:
//   '#' identifier (\s|EOF|'{'|'}')
func (d *Decoder) g1Element() (*Element, error) {
	startPos := d.Pos()

	startOfElem, err := d.nextR()
	if err != nil {
		return nil, err
	}

	if startOfElem != '#' {
		return nil, token.NewPosError(d.node(), "expected '#' (start of element)")
	}

	var tmp bytes.Buffer
	for {
		r, err := d.nextR()
		if errors.Is(err, io.EOF) {
			break
		}

		if !d.g1IdentChar(r) {
			d.prevR() // revert last read char
			if _, err := d.g1ConsumeWhitespace(); err != nil {
				return nil, err
			}
			break
		}

		tmp.WriteRune(r)
	}

	if tmp.Len() == 0 {
		return nil, token.NewPosError(token.NewNode(startPos, d.Pos()), "empty element name not allowed")
	}

	elem := &Element{}
	elem.BeginPos = startPos
	elem.EndPos = d.Pos()
	elem.Value = CharData{
		Position: elem.Position,
		Value:    tmp.String(),
	}

	elem.Value.BeginPos.Col++ //do not include the #

	return elem, nil
}

// g1IdentChar is [a-z,A-Z]
func (d *Decoder) g1IdentChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || (r == '_')
}

// g1Ident parses a text sequence until next control char # or } or EOF or whitespace.
func (d *Decoder) g1Ident() (*CharData, error) {
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

		if !d.g1IdentChar(r) {
			d.prevR() // reset last read char
			break
		}

		tmp.WriteRune(r)

	}

	text := &CharData{}
	text.Value = tmp.String()
	text.Position.BeginPos = startPos
	text.Position.EndPos = d.pos

	return text, nil
}

// g1Text parses a text sequence until next control char # or } or EOF.
func (d *Decoder) g1Text(escapeHash bool) (*CharData, error) {
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

		if (escapeHash && r == '#') || r == '}' {
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
