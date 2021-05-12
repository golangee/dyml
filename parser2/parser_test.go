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

func TestParser(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    *TestSet
		wantErr bool
	}{
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
				Element("hello"),
		},

		{
			name: "simple element with attribute and no spaces",
			text: `#hello:id{world}`,
			want: NewTestSet().
				Element("hello").
				Attribute("id", "world"),
		},

		{
			name: "simple element with attribute",
			text: `#hello 	:id 	{world}`,
			want: NewTestSet().
				Element("hello").
				Attribute("id", "world"),
		},

		{
			name: "more attribs",
			text: `#img :id{5} 	:alt{an image}    :href{https://worldiety.de/yada?a=b&c=d#anchor-in-string-special-case&%20%C3%A4%23%265%3C%7B%7D}   	`,
			want: NewTestSet().
				Element("img").
				Attribute("id", "5").
				Attribute("alt", "an image").
				Attribute("href", "https://worldiety.de/yada?a=b&c=d#anchor-in-string-special-case&%20%C3%A4%23%265%3C%7B%7D"),
		},

		{
			name: "more attribs without spaces",
			text: `#img:id{5}:alt{an image}:href{https://worldiety.de/yada?a=b&c=d#anchor-in-string-special-case&%20%C3%A4%23%265%3C%7B%7D}`,
			want: NewTestSet().
				Element("img").
				Attribute("id", "5").
				Attribute("alt", "an image").
				Attribute("href", "https://worldiety.de/yada?a=b&c=d#anchor-in-string-special-case&%20%C3%A4%23%265%3C%7B%7D"),
		},

		{
			name: "simple element with attribute and line break",
			text: "#hello :id{split\nworld}",
			want: NewTestSet().
				Element("hello").
				Attribute("id", "split\nworld"),
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

func (ts *TestSet) Element(value string) *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if cd, ok := t.(*Element); ok {
			if cd.Value.String() != value {
				return fmt.Errorf("element: expected '%s' but got '%s': %s", value, cd.Value.String(), toString(cd))
			}

			return nil
		}

		return fmt.Errorf("element: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) Attribute(key, value string) *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if cd, ok := t.(*Attr); ok {
			if cd.Key.String() != key || cd.Value.String() != value {
				return fmt.Errorf("attr: expected '%s' = '%s' but got '%s' = '%s': %s", key, value, cd.Key.String(), cd.Value.String(), toString(cd))
			}

			return nil
		}

		return fmt.Errorf("attr: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
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
