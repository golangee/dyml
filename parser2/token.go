package parser2

//go:generate go run gen/gen.go

import "github.com/golangee/tadl/token"

// A CharData token represents a run of text.
type CharData struct {
	token.Position
	Value string
}

func (t *CharData) String() string {
	return t.Value
}

// Identifier is an identifier as you would expect from a programming language: [0-9a-zA-Z_]+
type Identifier struct {
	token.Position
	Value string
}

// BlockStart is a '{' that is the start of a block.
type BlockStart struct {
	token.Position
}

// BlockEnd is a '}' that is the end of a block.
type BlockEnd struct {
	token.Position
}

// GroupStart is a '(' that is the start of a group.
type GroupStart struct {
	token.Position
}

// GroupEnd is a ')' that is the end of a group.
type GroupEnd struct {
	token.Position
}

// GenericStart is a '<' that is the start of a generic group.
type GenericStart struct {
	token.Position
}

// GenericEnd is a '>' that is the end of a generic group.
type GenericEnd struct {
	token.Position
}

// G2Preamble is the '#!' preamble for a G2 grammar.
type G2Preamble struct {
	token.Position
}

// DefineElement is the '#' before the name of an element.
type DefineElement struct {
	token.Position
	Forward bool
}

// DefineAttribute is the '@' before the name of an attribute.
type DefineAttribute struct {
	token.Position
	Forward bool
}

// Assign is the '=' in G2 attribute definitions.
type Assign struct {
	token.Position
}

// G1LineEnd is a special newline, that is only emitted when a G1Line ends.
type G1LineEnd struct {
	token.Position
}

// Comma ',' is used as a separator in G2.
type Comma struct {
	token.Position
}

// G1Comment is a '#?' that indicates a comment in G1.
type G1Comment struct {
	token.Position
}

// G2Comment is a '//' that indicates a comment in G2.
type G2Comment struct {
	token.Position
}
