// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package token

import "strings"

//go:generate go run gen/gen.go

// A CharData token represents a run of text.
type CharData struct {
	Position
	Value string
}

func (t *CharData) String() string {
	return t.Value
}

// SplitLines splits this token into one or more, one for each line.
// This will return empty tokens for empty lines, as the newline-characters
// are not included in the new tokens.
func (t CharData) SplitLines() []*CharData {
	var result []*CharData

	// Keep track of the offset and advance it properly
	offset := t.Begin().Offset

	for i, line := range strings.Split(t.Value, "\n") {
		// The column will be 1, except for the first line.
		col := 1
		if i == 0 {
			col = t.Begin().Col
		}

		result = append(result, &CharData{
			Position: Position{
				BeginPos: Pos{
					File:   t.Begin().File,
					Line:   t.Begin().Line + i,
					Col:    col,
					Offset: offset,
				},
				EndPos: Pos{
					File:   t.EndPos.File,
					Line:   t.Begin().Line + i,
					Col:    col + len(line),
					Offset: offset + len(line),
				},
			},
			Value: line,
		})

		// Add one to the line length for the newline char.
		offset += len(line) + 1
	}

	return result
}

// Identifier is an identifier as you would expect from a programming language: [0-9a-zA-Z_]+.
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

// Semicolon ';' is used as a separator in G2 and is interchangeable with Comma.
type Semicolon struct {
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
