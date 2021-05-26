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

// GroupStart is a '(' that is the start of a group.
type GroupStart struct {
	token.Position
}

func (t *GroupStart) assertToken() {}

// GroupEnd is a ')' that is the end of a group.
type GroupEnd struct {
	token.Position
}

func (t *GroupEnd) assertToken() {}

// GenericStart is a '<' that is the start of a generic group.
type GenericStart struct {
	token.Position
}

func (t *GenericStart) assertToken() {}

// GenericEnd is a '>' that is the end of a generic group.
type GenericEnd struct {
	token.Position
}

func (t *GenericEnd) assertToken() {}

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

// G1LineEnd is a special newline, that is only emitted when a G1Line ends.
type G1LineEnd struct {
	token.Position
}

func (t *G1LineEnd) assertToken() {}

// Comma ',' is used as a separator in G2.
type Comma struct {
	token.Position
}

func (t *Comma) assertToken() {}

// Pipe '|' is used as a separator in G2.
type Pipe struct {
	token.Position
}

func (t *Pipe) assertToken() {}
