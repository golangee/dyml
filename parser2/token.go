package parser2

import "github.com/golangee/tadl/token"

// A CharData token represents a run of text.
type CharData struct {
	token.Position
	Value string
}

func (t *CharData) String() string {
	return t.Value
}

func (t *CharData) assertToken() {}

// Identifier is an identifier as you would expect from a programming language: [0-9a-zA-Z_]+
type Identifier struct {
	token.Position
	Value string
}

func (t *Identifier) assertToken() {}

// BlockStart is a '{' that is the start of a block.
type BlockStart struct {
	token.Position
}

func (t *BlockStart) assertToken() {}

// BlockEnd is a '}' that is the end of a block.
type BlockEnd struct {
	token.Position
}

func (t *BlockEnd) assertToken() {}

// G2Preambel is the '#!' preambel for a G2 grammar.
type G2Preambel struct {
	token.Position
}

func (t *G2Preambel) assertToken() {}

// DefineElement is the '#' before the name of an element.
type DefineElement struct {
	token.Position
	Forward bool
}

func (t *DefineElement) assertToken() {}

// DefineAttribute is the '@' before the name of an attribute.
type DefineAttribute struct {
	token.Position
	Forward bool
}

func (t *DefineAttribute) assertToken() {}

// Assign is the '=' in G2 attribute definitions.
type Assign struct {
	token.Position
}

func (t *Assign) assertToken() {}
