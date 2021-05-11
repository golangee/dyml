package parser2

import "github.com/golangee/tadl/token"

// A CharData token represents a run of raw text.
type CharData struct {
	token.Position
	Value string
}

func (t *CharData) assertToken() {}

// StartElement represents the beginning of an element to parse.
type StartElement struct {
	token.Position
	Value string
}

func (t *StartElement) assertToken() {}
