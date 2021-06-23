// SPDX-FileCopyrightText: © 2021 The tadl authors <https://github.com/golangee/tadl/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package token

//go:generate go run gen/gen.go

// A CharData token represents a run of text.
type CharData struct {
	Position
	Value string
}

func (t *CharData) String() string {
	return t.Value
}

// Identifier is an identifier as you would expect from a programming language: [0-9a-zA-Z_]+
type Identifier struct {
	Position
	Value string
}

// BlockStart is a '{' that is the start of a block.
type BlockStart struct {
	Position
}

// BlockEnd is a '}' that is the end of a block.
type BlockEnd struct {
	Position
}

// GroupStart is a '(' that is the start of a group.
type GroupStart struct {
	Position
}

// GroupEnd is a ')' that is the end of a group.
type GroupEnd struct {
	Position
}

// GenericStart is a '<' that is the start of a generic group.
type GenericStart struct {
	Position
}

// GenericEnd is a '>' that is the end of a generic group.
type GenericEnd struct {
	Position
}

// G2Preamble is the '#!' preamble for a G2 grammar.
type G2Preamble struct {
	Position
}

// DefineElement is the '#' before the name of an element.
type DefineElement struct {
	Position
	Forward bool
}

// DefineAttribute is the '@' before the name of an attribute.
type DefineAttribute struct {
	Position
	Forward bool
}

// Assign is the '=' in G2 attribute definitions.
type Assign struct {
	Position
}

// G1LineEnd is a special newline, that is only emitted when a G1Line ends.
type G1LineEnd struct {
	Position
}

// Comma ',' is used as a separator in G2.
type Comma struct {
	Position
}

// G1Comment is a '#?' that indicates a comment in G1.
type G1Comment struct {
	Position
}

// G2Comment is a '//' that indicates a comment in G2.
type G2Comment struct {
	Position
}

// G2Arrow is a '->' that indicates a return value in G2.
type G2Arrow struct {
	Position
}
