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

// Element represents the declaration of a new element to insert at the current position.
// Positions marks the beginning of the element (including #) until the last element.
type Element struct {
	token.Position
	Value CharData
}

func (t *Element) assertToken() {}

// Attr describes an attribute, which must have a unique Key and a an optional string Value.
type Attr struct {
	token.Position
	Key   CharData
	Value CharData
}

func (t *Attr) assertToken() {}


type BlockStart struct{
	token.Position
}

func (t *BlockStart) assertToken() {}


type BlockEnd struct{
	token.Position
}

func (t *BlockEnd) assertToken() {}