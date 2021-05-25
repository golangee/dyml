package parser2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestTokenizer(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    *TestSet
		wantErr bool
	}{
		{
			name: "empty",
			text: "",
			want: NewTestSet(),
		},

		{
			name: "space",
			text: " ",
			want: NewTestSet().CharData(" "),
		},

		{
			name: "simple text",
			text: "hello world",
			want: NewTestSet().
				CharData("hello world"),
		},

		{
			name: "escaped simple text",
			text: `hello \wo\#rl\}d`,
			want: NewTestSet().
				CharData(`hello \wo#rl}d`),
		},

		{
			name: "simple element",
			text: `#hello`,
			want: NewTestSet().
				DefineElement().
				Identifier("hello"),
		},

		{
			name: "simple element with attribute and no spaces",
			text: `#hello@id{world}`,
			want: NewTestSet().
				DefineElement().
				Identifier("hello").
				DefineAttribute(false).
				Identifier("id").
				BlockStart().
				CharData("world").
				BlockEnd(),
		},

		{
			name: "simple element with attribute",
			text: `#hello 	@id 	{world}`,
			want: NewTestSet().
				DefineElement().
				Identifier("hello").
				DefineAttribute(false).
				Identifier("id").
				BlockStart().
				CharData("world").
				BlockEnd(),
		},

		{
			name: "more attribs",
			text: `#img @id{5} 	@alt{an image}    @href{https://worldiety.de/yada?a=b&c=d\#anchor-in-string-special-case&%20%C3%A4%23%265%3C%7B%7D}   	`,
			want: NewTestSet().
				DefineElement().
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
			text: `#img@id{5}@alt{an image}@href{https://worldiety.de/yada?a=b&c=d\#anchor-in-string-special-case&%20%C3%A4%23%265%3C%7B%7D}`,
			want: NewTestSet().
				DefineElement().
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
				DefineElement().
				Identifier("hello").
				DefineAttribute(false).
				Identifier("id").
				BlockStart().
				CharData("split\nworld").
				BlockEnd(),
		},

		{
			name: "empty g2",
			text: "#!{}",
			want: NewTestSet().
				G2Preambel().
				BlockStart().
				BlockEnd(),
		},

		{
			name: "basic g2 with a single element",
			text: "#!{hello }",
			want: NewTestSet().
				G2Preambel().
				BlockStart().
				Identifier("hello").
				BlockEnd(),
		},

		{
			name: "g2 with a multiple elements",
			text: "#!{ list another}",
			want: NewTestSet().
				G2Preambel().
				BlockStart().
				Identifier("list").
				Identifier("another").
				BlockEnd(),
		},

		{
			name: "g2 with a string",
			text: `#!{"hello\"\n"}`,
			want: NewTestSet().
				G2Preambel().
				BlockStart().
				CharData(`hello\"\n`).
				BlockEnd(),
		},

		{
			name: "g2 with attributes",
			text: `#!{x @key="value" @@num="5" y}`,
			want: NewTestSet().
				G2Preambel().
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parseTokens(tt.text)
			if !tt.wantErr && err != nil {
				t.Error(err)
				return
			}

			if tt.wantErr && err == nil {
				t.Errorf("expected error")
				return
			}

			tt.want.Assert(tokens, t)
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

func (ts *TestSet) G2Preambel() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*G2Preambel); ok {
			return nil
		}

		return fmt.Errorf("G2Preambel: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) DefineElement() *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if _, ok := t.(*DefineElement); ok {
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

func newTestDec(text string) *Decoder {
	return NewDecoder("parser_test.go", bytes.NewBuffer([]byte(text)))
}

func parseTokens(text string) ([]Token, error) {
	dec := newTestDec(text)
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
