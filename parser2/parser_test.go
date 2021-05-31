package parser2

import (
	"fmt"
	"strings"
	"testing"

	"github.com/r3labs/diff/v2"
)

func TestParser(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    *TreeNode
		wantErr bool
	}{
		{
			name: "empty",
			text: "",
			want: NewNode("root"),
		},
		{
			name: "just text",
			text: "hello world",
			want: NewNode("root").AddChildren(
				NewTextNode("hello world"),
			),
		},
		{
			name: "different children types",
			text: "hello #item1 world #item2 #item3 more text",
			want: NewNode("root").AddChildren(
				NewTextNode("hello "),
				NewNode("item1"),
				NewTextNode("world "),
				NewNode("item2"),
				NewNode("item3"),
				NewTextNode("more text"),
			),
		},
		{
			name: "recursion and whitespace",
			text: "#A   { #B{#C  #D{#E }} } #F",
			want: NewNode("root").AddChildren(
				NewNode("A").AddChildren(
					NewNode("B").AddChildren(
						NewNode("C"),
						NewNode("D").AddChildren(
							NewNode("E"),
						),
					),
				),
				NewNode("F"),
			),
		},
		{
			name: "attributes",
			text: `#item @id{5} @hello{world} @complex{attribute "with" #special 'chars}`,
			want: NewNode("root").AddChildren(
				NewNode("item").
					AddAttribute("id", "5").
					AddAttribute("hello", "world").
					AddAttribute("complex", `attribute "with" #special 'chars`),
			),
		},
		{
			name: "attribute in nested element",
			text: "#item { #subitem @hello{world} }",
			want: NewNode("root").AddChildren(
				NewNode("item").AddChildren(
					NewNode("subitem").AddAttribute("hello", "world"),
				),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			parser := NewParser("parser_test.go", strings.NewReader(tt.text))
			tree, err := parser.Parse()

			if !tt.wantErr && err != nil {
				t.Error(err)
				return
			}

			if tt.wantErr && err == nil {
				t.Errorf("expected error, but did not get one")
				return
			}

			if tt.wantErr {
				// We wanted an error and got it, comparing trees would
				// make no sense, so we end this test here.
				return
			}

			differences, err := diff.Diff(tt.want, tree)
			if err != nil {
				t.Error(err)
				return
			}

			// These descriptions map the type of a change to a more readable format.
			changeTypeDescription := map[string]string{
				"create": "was added",
				"update": "is different",
				"delete": "is missing",
			}

			if len(differences) > 0 {
				for _, d := range differences {
					t.Errorf("property '%s' %s, expected %s but got %s",
						strings.Join(d.Path, "."),
						changeTypeDescription[d.Type],
						PrettyValue(d.From), PrettyValue(d.To))
				}
			}

		})
	}
}

// PrettyValue transforms values into a human readable form.
// Usually "%#v" in fmt.Sprintf can give a nice description of the thing
// you're passing in, but that does not apply to e.g. string pointers.
func PrettyValue(v interface{}) string {

	if s, ok := v.(*string); ok {
		return fmt.Sprintf("%#v", *s)
	}

	return fmt.Sprintf("%#v", v)
}
