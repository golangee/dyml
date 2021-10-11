// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestLexer(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    *TestSet
		wantErr bool
		// positions is optional to test the correct lexing of positions.
		positions []Position
	}{
		{
			name: "empty",
			text: "",
			want: NewTestSet(),
		},

		{
			name:      "space",
			text:      " ",
			want:      NewTestSet().CharData(" "),
			positions: newTestPositions(1, 1, 1, 2),
		},

		{
			name: "simple text",
			text: "hello world",
			want: NewTestSet().
				CharData("hello world"),
		},

		{
			name: "escaped simple text",
			text: `hello \\wo\#rl\}d`,
			want: NewTestSet().
				CharData(`hello \wo#rl}d`),
		},

		{
			name:    "Whitespace after backslash",
			text:    `#book @id{my-book\ } @author{Torben\}`,
			wantErr: true,
		},

		{
			name: "simple element",
			text: `#hello`,
			want: NewTestSet().
				DefineElement(false).
				Identifier("hello"),
		},

		{
			name: "multiline positions",
			text: `#A
#B
#C`,
			want: NewTestSet().
				DefineElement(false).
				Identifier("A").
				DefineElement(false).
				Identifier("B").
				DefineElement(false).
				Identifier("C"),
			positions: newTestPositions(
				1, 1, 1, 2,
				1, 2, 1, 3,
				2, 1, 2, 2,
				2, 2, 2, 3,
				3, 1, 3, 2,
				3, 2, 3, 3,
			),
		},

		{
			name: "simple element and text",
			text: `#hello world`,
			want: NewTestSet().
				DefineElement(false).
				Identifier("hello").
				CharData("world"),
			positions: newTestPositions(
				1, 1, 1, 2,
				1, 2, 1, 7,
				1, 8, 1, 13,
			),
		},

		{
			name: "simple element with attribute and no spaces",
			text: `#hello@id{world}`,
			want: NewTestSet().
				DefineElement(false).
				Identifier("hello").
				DefineAttribute(false).
				Identifier("id").
				BlockStart().
				CharData("world").
				BlockEnd(),
			positions: newTestPositions(
				1, 1, 1, 2,
				1, 2, 1, 7,
				1, 7, 1, 8,
				1, 8, 1, 10,
				1, 10, 1, 11,
				1, 11, 1, 16,
				1, 16, 1, 17,
			),
		},

		{
			name: "simple element with attribute",
			text: `#hello 	@id 	{world}`,
			want: NewTestSet().
				DefineElement(false).
				Identifier("hello").
				DefineAttribute(false).
				Identifier("id").
				BlockStart().
				CharData("world").
				BlockEnd(),
		},

		{
			name: "more attribs",
			text: `#img @id{5} 	@alt{an image}    @href{https://worldiety.de/yada?a=b&c=d#anchor-in-string-special-case&%20%C3%A4%23%265%3C%7B%7D}   	`,
			want: NewTestSet().
				DefineElement(false).
				Identifier("img").
				DefineAttribute(false).
				Identifier("id").
				BlockStart().
				CharData("5").
				BlockEnd().
				DefineAttribute(false).
				Identifier("alt").
				BlockStart().
				CharData("an image").
				BlockEnd().
				DefineAttribute(false).
				Identifier("href").
				BlockStart().
				CharData(`https://worldiety.de/yada?a=b&c=d#anchor-in-string-special-case&%20%C3%A4%23%265%3C%7B%7D`).
				BlockEnd(),
		},

		{
			name: "more attribs without spaces",
			text: `#img@id{5}@alt{an image}@href{https://worldiety.de/yada?a=b&c=d#anchor-in-string-special-case&%20%C3%A4%23%265%3C%7B%7D}`,
			want: NewTestSet().
				DefineElement(false).
				Identifier("img").
				DefineAttribute(false).
				Identifier("id").
				BlockStart().
				CharData("5").
				BlockEnd().
				DefineAttribute(false).
				Identifier("alt").
				BlockStart().
				CharData("an image").
				BlockEnd().
				DefineAttribute(false).
				Identifier("href").
				BlockStart().
				CharData(`https://worldiety.de/yada?a=b&c=d#anchor-in-string-special-case&%20%C3%A4%23%265%3C%7B%7D`).
				BlockEnd(),
		},

		{
			name: "simple element with attribute and line break",
			text: "#hello @id{split\nworld}",
			want: NewTestSet().
				DefineElement(false).
				Identifier("hello").
				DefineAttribute(false).
				Identifier("id").
				BlockStart().
				CharData("split\nworld").
				BlockEnd(),
		},

		{
			name: "g1 line comment",
			text: "#? This is a comment.\nThis is not.",
			want: NewTestSet().
				G1Comment().
				CharData("This is a comment.").
				CharData("This is not."),
		},

		{
			name:    "invalid blank identifier",
			text:    "# ",
			wantErr: true,
		},

		{
			name: "simple g2",
			text: "#!{hello}",
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				Identifier("hello").
				BlockEnd(),
		},

		{
			name: "named g2",
			text: "#! item {}",
			want: NewTestSet().
				G2Preamble().
				Identifier("item").
				BlockStart().
				BlockEnd(),
		},

		{
			name: "g2 with multiple elements",
			text: "#!{ list another}",
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				Identifier("list").
				Identifier("another").
				BlockEnd(),
		},

		{
			name: "g2 with a string",
			text: `#!{"hello\"\\n"}`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				CharData(`hello"\n`).
				BlockEnd(),
		},

		{
			name: "g2 with attributes",
			text: `#!{x @key="value" @@num="5" y}`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				Identifier("x").
				DefineAttribute(false).
				Identifier("key").
				Assign().
				CharData("value").
				DefineAttribute(true).
				Identifier("num").
				Assign().
				CharData("5").
				Identifier("y").
				BlockEnd(),
		},

		{
			name: "g2 with g1 line",
			text: `#!{# here is another #item @color{blue}}`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				DefineElement(false).
				CharData("here is another ").
				DefineElement(false).
				Identifier("item").
				DefineAttribute(false).
				Identifier("color").
				BlockStart().
				CharData("blue").
				BlockEnd().
				BlockEnd(),
		},

		{
			name: "g1 lines with different endings",
			text: `#!{
						# #item
						# #item{#child}
						# @key{value}
						# text
						#
					}`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				DefineElement(false).
				DefineElement(false).
				Identifier("item").
				G1LineEnd().
				DefineElement(false).
				DefineElement(false).
				Identifier("item").
				BlockStart().
				DefineElement(false).
				Identifier("child").
				BlockEnd().
				G1LineEnd().
				DefineElement(false).
				DefineAttribute(false).
				Identifier("key").
				BlockStart().
				CharData("value").
				BlockEnd().
				G1LineEnd().
				DefineElement(false).
				CharData("text").
				G1LineEnd().
				DefineElement(false).
				G1LineEnd().
				BlockEnd(),
		},

		{
			name: "g2 mixed with multiple g1 lines",
			text: `#!{
				## This is a list with some items
				list {
					item
					## This item likes the #color @format{hex} {\#00FF00}, which is nice.
					item
				}
			}
			`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				DefineElement(true).
				CharData("This is a list with some items").
				G1LineEnd().
				Identifier("list").
				BlockStart().
				Identifier("item").
				DefineElement(true).
				CharData("This item likes the ").
				DefineElement(false).
				Identifier("color").
				DefineAttribute(false).
				Identifier("format").
				BlockStart().
				CharData("hex").
				BlockEnd().
				BlockStart().
				CharData("#00FF00").
				BlockEnd().
				CharData(", which is nice.").
				G1LineEnd().
				Identifier("item").
				BlockEnd().
				BlockEnd(),
		},

		{
			name: "g2 with separated elements",
			text: `#!{item, item ,item , item}`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				Identifier("item").
				Comma().
				Identifier("item").
				Comma().
				Identifier("item").
				Comma().
				Identifier("item").
				BlockEnd(),
		},

		{
			name: "g2 with simple groups",
			text: `#!{ ( ) < >()<> }`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				GroupStart().
				GroupEnd().
				GenericStart().
				GenericEnd().
				GroupStart().
				GroupEnd().
				GenericStart().
				GenericEnd().
				BlockEnd(),
		},

		{
			name: "incomplete g2",
			text: `#!{#`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				DefineElement(false),
		},

		{
			name: "g2 comment",
			text: `#!{
				// This is a comment
				item // Another } comment # with { special characters
			}`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				G2Comment().
				CharData("This is a comment").
				Identifier("item").
				G2Comment().
				CharData("Another } comment # with { special characters").
				BlockEnd(),
		},

		{
			name: "g2 arrow",
			text: `#!{ -> }`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				G2Arrow().
				BlockEnd(),
		},

		{
			name: "semicolon",
			text: `#!{ a; }`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				Identifier("a").
				Semicolon().
				BlockEnd(),
		},

		{
			name: "multiple g2s",
			text: `#!{} hello #!{}`,
			want: NewTestSet().
				G2Preamble().
				BlockStart().
				BlockEnd().
				CharData("hello ").
				G2Preamble().
				BlockStart().
				BlockEnd(),
		},

		{
			name: "g2 in g1",
			text: `#a #!b{} #c`,
			want: NewTestSet().
				DefineElement(false).
				Identifier("a").
				G2Preamble().
				Identifier("b").
				BlockStart().
				BlockEnd().
				DefineElement(false).
				Identifier("c"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parseTokens(tt.text)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error")
				} else {
					// we wanted and got an error, that's okay
				}
			} else {
				if err != nil {
					t.Error(err)
				} else {
					tt.want.Assert(tokens, t)
				}
			}

			// Compare token positions, if any are given
			if len(tt.positions) > 0 {
				if len(tt.positions) != len(tokens) {
					t.Fatalf("expected %d token positions, but got %d", len(tt.positions), len(tokens))
				}

				for i := 0; i < len(tt.positions); i++ {
					expected := tt.positions[i]
					actual := *tokens[i].Pos()

					if !comparePos(expected, actual) {
						t.Errorf("token positions for %s differ, expected: %v, actual: %v", tokens[i].TokenType(), expected, actual)
					}
				}
			}
		})
	}
}

// test utils

type TestSet struct {
	checker []func(t Token) error
}

func NewTestSet() *TestSet {
	return &TestSet{}
}

func (ts *TestSet) CharData(value string) *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if cd, ok := t.(*CharData); ok {
			if cd.Value != value {
				return fmt.Errorf("CharData: expected '%s' but got '%s'", value, cd.Value)
			}

			return nil
		}

		return fmt.Errorf("CharData: unexpected type '%v'", reflect.TypeOf(t))
	})

	return ts
}

func (ts *TestSet) Identifier(value string) *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if cd, ok := t.(*Identifier); ok {
			if cd.Value != value {
				return fmt.Errorf("Identifier: expected '%s' but got '%s'", value, cd.Value)
			}

			return nil
		}

		return fmt.Errorf("Identifier: unexpected type '%v'", reflect.TypeOf(t))
	})

	return ts
}

func (ts *TestSet) BlockStart() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*BlockStart); ok {
			return nil
		}

		return fmt.Errorf("BlockStart: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) BlockEnd() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*BlockEnd); ok {
			return nil
		}

		return fmt.Errorf("BlockEnd: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) GroupStart() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*GroupStart); ok {
			return nil
		}

		return fmt.Errorf("GroupStart: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) GroupEnd() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*GroupEnd); ok {
			return nil
		}

		return fmt.Errorf("GroupEnd: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) GenericStart() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*GenericStart); ok {
			return nil
		}

		return fmt.Errorf("GenericStart: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) GenericEnd() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*GenericEnd); ok {
			return nil
		}

		return fmt.Errorf("GenericEnd: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) G2Preamble() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*G2Preamble); ok {
			return nil
		}

		return fmt.Errorf("G2Preamble: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) G2Comment() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*G2Comment); ok {
			return nil
		}

		return fmt.Errorf("G2Comment: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) G2Arrow() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*G2Arrow); ok {
			return nil
		}

		return fmt.Errorf("G2Arrow: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) G1Comment() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*G1Comment); ok {
			return nil
		}

		return fmt.Errorf("G1Comment: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) DefineElement(forward bool) *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if def, ok := t.(*DefineElement); ok {
			if def.Forward != forward {
				return fmt.Errorf("DefineElement: expected forward=%v but got %v", forward, def.Forward)
			}
			return nil
		}

		return fmt.Errorf("DefineElement: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) DefineAttribute(forward bool) *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if attr, ok := t.(*DefineAttribute); ok {
			if attr.Forward != forward {
				return fmt.Errorf("DefineAttribute: expected forward=%v but got %v", forward, attr.Forward)
			}
			return nil
		}

		return fmt.Errorf("DefineAttribute: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) Assign() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*Assign); ok {
			return nil
		}

		return fmt.Errorf("Assign: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) G1LineEnd() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*G1LineEnd); ok {
			return nil
		}

		return fmt.Errorf("G1LineEnd: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) Comma() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*Comma); ok {
			return nil
		}

		return fmt.Errorf("Comma: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) Semicolon() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*Semicolon); ok {
			return nil
		}

		return fmt.Errorf("Semicolon: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) Assert(tokens []Token, t *testing.T) {
	t.Helper()

	if len(ts.checker) != len(tokens) {
		tokenTypesOverview := "["
		for _, token := range tokens {
			tokenTypesOverview += reflect.TypeOf(token).String() + ", "
		}

		tokenTypesOverview += "]"

		t.Fatalf("expected %d parsed tokens but got %d: %s\n%s", len(ts.checker), len(tokens), tokenTypesOverview, toString(tokens))
	}

	for i, token := range tokens {
		if err := ts.checker[i](token); err != nil {
			t.Fatal(err)
		}
	}
}

// newTestPositions creates new positional information.
// It expects info to have a length divisible by 4, otherwise it will panic.
// The integers are interpreted as repeating instances of Position like this:
// [beginLine, beginCol, endLine, endCol].
func newTestPositions(info ...int) []Position {
	if len(info)%4 != 0 {
		panic("newTestPositions needs length divisible by 4")
	}

	var result []Position

	for i := 0; i < len(info); i += 4 {
		result = append(result, Position{
			BeginPos: Pos{
				Line: info[i],
				Col:  info[i+1],
			},
			EndPos: Pos{
				Line: info[i+2],
				Col:  info[i+3],
			},
		})
	}

	return result
}

// comparePos compares the line and col attributes of the given positions
// and returns true if they are equal.
func comparePos(a, b Position) bool {
	return a.Begin().Col == b.Begin().Col && a.Begin().Line == b.Begin().Line &&
		a.End().Col == b.End().Col && a.End().Line == b.End().Line
}

func newTestLexer(text string) *Lexer {
	return NewLexer("lexer_test.go", bytes.NewBuffer([]byte(text)))
}

func parseTokens(text string) ([]Token, error) {
	dec := newTestLexer(text)

	var res []Token

	for {
		token, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		res = append(res, token)
	}

	return res, nil
}

func toString(i interface{}) string {
	buf, err := json.MarshalIndent(i, " ", " ")
	if err != nil {
		panic(err)
	}

	return string(buf)
}
