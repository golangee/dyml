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
				StartElement("hello"),
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

func (ts *TestSet) StartElement(value string) *TestSet {
	ts.checker = append(ts.checker, func(t Token) error {
		if cd, ok := t.(*StartElement); ok {
			if cd.Value != value {
				return fmt.Errorf("StartElement: expected '%s' but got '%s': %s", value, cd.Value, toString(cd))
			}

			return nil
		}

		return fmt.Errorf("StartElement: unexpected type '%v': %s", reflect.TypeOf(t), toString(t))
	})

	return ts
}

func (ts *TestSet) Assert(tokens []Token, t *testing.T) {
	if len(ts.checker) != len(tokens) {
		t.Fatalf("expected %d parsed tokens but got %d: %s", len(ts.checker), len(tokens), toString(tokens))
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
