package ast

import "github.com/golangee/tadl/token"

// A MLNode consists of a name, attributes and other child nodes. Actually it is just xml.
// It seems not to make much sense to reinvent the wheel, like https://docs.rs/crate/doccy/0.3.2 or
// https://www.pml-lang.dev/. For simple examples they are shorter, but that does not mean
// they are more readable. Quite the contrary, in complex cases with more nesting, due to the non-repeating
// node names, there is just a big ball of closing braces. Yaml and Json are no fun either, because embedding
// their notations in character data is impossible (it is just no markup) and nesting becomes a hell.
type MLNode struct {
	Name     MLString
	Attrs    []*MLAttr
	Children []*MLNode
}

type MLAttr struct {
	Key   MLString
	Value MLString
}

type MLString struct {
	token.Position
	Value string
}
