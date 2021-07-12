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
				NewStringNode("hello world"),
			),
		},
		{
			name: "BlockNoBrackets",
			text: "#title Chapter Two",
			want: NewNode("root").AddChildren(NewNode("title").AddChildren(NewStringNode("ChapterTwo"))),
		},
		{
			name: "different children types",
			text: "hello #item1 world #item2 #item3 more text",
			want: NewNode("root").AddChildren(
				NewStringNode("hello "),
				NewNode("item1"),
				NewStringNode("world "),
				NewNode("item2"),
				NewNode("item3"),
				NewStringNode("more text"),
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
		{
			name: "forwarded elements",
			text: `#A
					##B
					##C
					#D {
						##E
						#F
					}
					#G
				`,
			want: NewNode("root").AddChildren(
				NewNode("A"),
				NewNode("D").AddChildren(
					NewNode("B"),
					NewNode("C"),
					NewNode("F").AddChildren(
						NewNode("E"),
					),
				),
				NewNode("G"),
			),
		},
		{
			name:    "invalid dangling forward element",
			text:    `##item`,
			wantErr: true,
		},
		{
			name: "forwarded attributes",
			text: `#A
					@simple{attribute}
					@@forwarded{attribute}
					@@another{forwarded}
					#B
					@simple{attribute}
					#C`,
			want: NewNode("root").AddChildren(
				NewNode("A").
					AddAttribute("simple", "attribute"),
				NewNode("B").
					AddAttribute("forwarded", "attribute").
					AddAttribute("another", "forwarded").
					AddAttribute("simple", "attribute"),
				NewNode("C"),
			),
		},
		{
			name: "mixed forwarded attributes and elements",
			text: `##subA @@key{value} ##subB @@another_key{more_value} #item`,
			want: NewNode("root").AddChildren(
				NewNode("item").
					AddAttribute("another_key", "more_value").
					AddChildren(
						NewNode("subA"),
						NewNode("subB").
							AddAttribute("key", "value"),
					),
			),
		},
		{
			name:    "invalid simple attribute",
			text:    `@key{value} #item`,
			wantErr: true,
		},
		{
			name:    "invalid attribute defined twice",
			text:    `#item @key{value} @key{value}`,
			wantErr: true,
		},
		{
			name:    "invalid dangling forward attribute",
			text:    `@@key{value}`,
			wantErr: true,
		},
		{
			name: "comment",
			text: "#? this is a comment\nThis is not.",
			want: NewNode("root").AddChildren(
				NewStringCommentNode("this is a comment"),
				NewStringNode("This is not."),
			),
		},
		{
			name: "empty G2",
			text: `#!{}`,
			want: NewNode("root"),
		},
		{
			name: "simple G2",
			text: `#!{item}`,
			want: NewNode("root").AddChildren(
				NewNode("item"),
			),
		},
		{
			name: "siblings G2",
			text: `#!{item, item}`,
			want: NewNode("root").AddChildren(
				NewNode("item"),
				NewNode("item"),
			),
		},
		{
			name: "nested G2",
			text: `#!{item subitem subsubitem "text"}`,
			want: NewNode("root").AddChildren(
				NewNode("item").AddChildren(
					NewNode("subitem").AddChildren(
						NewNode("subsubitem").AddChildren(
							NewStringNode("text"),
						),
					),
				),
			),
		},
		{
			name: "complex siblings and nested G2",
			text: `#!{
						A B {
							C,
							D,
						}
						E {F, G}
						H
					}`,
			want: NewNode("root").AddChildren(
				NewNode("A").AddChildren(
					NewNode("B").AddChildren(
						NewNode("C"),
						NewNode("D"),
					),
				),
				NewNode("E").AddChildren(
					NewNode("F"),
					NewNode("G"),
				),
				NewNode("H"),
			),
		},
		{
			name: "G2 string will stop parsing nested children",
			text: `#!{
						A "hello" B
					}`,
			want: NewNode("root").AddChildren(
				NewNode("A").AddChildren(
					NewStringNode("hello"),
				),
				NewNode("B"),
			),
		},
		{
			name: "simple attribute G2",
			text: `#!{
						item @key="value" @another="key with 'special #chars\""
					}`,
			want: NewNode("root").AddChildren(
				NewNode("item").
					AddAttribute("key", "value").
					AddAttribute("another", `key with 'special #chars"`),
			),
		},
		{
			name: "attribute with siblings G2",
			text: `#!{
						A,
						B C @key="value" D,
						E,
					}`,
			want: NewNode("root").AddChildren(
				NewNode("A"),
				NewNode("B").AddChildren(
					NewNode("C").
						AddAttribute("key", "value").
						AddChildren(
							NewNode("D"),
						),
				),
				NewNode("E"),
			),
		},
		{
			name:    "invalid lonely attribute G2",
			text:    `#!{@key="value"}`,
			wantErr: true,
		},
		{
			name: "invalid attribute defined twice G2",
			text: `#!{
						item @key="value" @key="value"
					}`,
			wantErr: true,
		},
		{
			name: "simple forwarded attribute G2",
			text: `#!{
						@@key="value"
						item
					}`,
			want: NewNode("root").AddChildren(
				NewNode("item").
					AddAttribute("key", "value"),
			),
		},
		{
			name: "forwarded attributes G2",
			text: `#!{
						item,
						@@key="value"
						@@another="one"
						item @not="forwarded",
						parent @@for="child" child,
					}`,
			want: NewNode("root").AddChildren(
				NewNode("item"),
				NewNode("item").
					AddAttribute("not", "forwarded").
					AddAttribute("key", "value").
					AddAttribute("another", "one"),
				NewNode("parent").
					AddChildren(
						NewNode("child").
							AddAttribute("for", "child"),
					),
			),
		},
		{
			name: "invalid dangling forward attribute G2",
			text: `#!{
						item @@key="value"
					}`,
			wantErr: true,
		},
		{
			name: "invalid forward attribute for text G2",
			text: `#!{
						@@key="value" "text"
					}`,
			wantErr: true,
		},
		{
			name: "G1 line in G2",
			text: `#!{
						# This is a G1 text line. #item @key{with value}
					}`,
			want: NewNode("root").AddChildren(
				NewStringNode("This is a G1 text line. "),
				NewNode("item").
					AddAttribute("key", "with value"),
			),
		},
		{
			name: "nested G1 line",
			text: `#!{
						item # text child #child
					}`,
			want: NewNode("root").AddChildren(
				NewNode("item").AddChildren(
					NewStringNode("text child "),
					NewNode("child"),
				),
			),
		},
		{
			name: "forward G1 line",
			text: `#!{
						## forwarded #item @with{attribute}
						parent with children
					}`,
			want: NewNode("root").AddChildren(
				NewNode("parent").AddChildren(
					NewStringNode("forwarded "),
					NewNode("item").AddAttribute("with", "attribute"),
					NewNode("with").AddChildren(
						NewNode("children"),
					),
				),
			),
		},
		{
			name: "empty G1 line",
			text: `#!{
						#
					}`,
			want: NewNode("root"),
		},
		{
			name: "invalid forward G1 line",
			text: `#!{
						## where would this text be forwarded to?
					}`,
			wantErr: true,
		},
		{
			name: "many G1 lines",
			text: `#!{
						# Hello!
						# Hello!
						# Hello!
					}`,
			want: NewNode("root").AddChildren(
				NewStringNode("Hello!"),
				NewStringNode("Hello!"),
				NewStringNode("Hello!"),
			),
		},
		{
			name: "forward G1 line with string",
			text: `#!{
						## hello
						"this is a string"
						item
					}`,
			want: NewNode("root").AddChildren(
				NewStringNode("this is a string"),
				NewNode("item").AddChildren(
					NewStringNode("hello"),
				),
			),
		},
		{
			name: "other group types",
			text: `#!{
						item { X , Y}
						item < X ,Y  >
						item (X, Y )
					}`,
			want: NewNode("root").AddChildren(
				NewNode("item").AddChildren(
					NewNode("X"),
					NewNode("Y"),
				),
				NewNode("item").Block(BlockGeneric).AddChildren(
					NewNode("X"),
					NewNode("Y"),
				),
				NewNode("item").Block(BlockGroup).AddChildren(
					NewNode("X"),
					NewNode("Y"),
				),
			),
		},
		{
			name: "incorrect closing type",
			text: `#!{
						item {>
					}`,
			wantErr: true,
		},
		{
			name: "nested groups",
			text: `#!{
						item< item( item ) >
					}`,
			want: NewNode("root").AddChildren(
				NewNode("item").Block(BlockGeneric).AddChildren(
					NewNode("item").Block(BlockGroup).AddChildren(
						NewNode("item"),
					),
				),
			),
		},
		{
			name:    "invalid root brackets",
			text:    `#!(item)`,
			wantErr: true,
		},
		{
			name: "function definition example",
			text: `#!{
						## Greet someone.
						@@name="The name to greet."
						func Greet(name string)

						## Run complex calculations.
						func Run(x int, y int, z string)
					}`,
			want: NewNode("root").AddChildren(
				NewNode("func").
					AddAttribute("name", "The name to greet.").
					AddChildren(
						NewStringNode("Greet someone."),
						NewNode("Greet").Block(BlockGroup).AddChildren(
							NewNode("name").AddChildren(
								NewNode("string"),
							),
						),
					),
				NewNode("func").
					AddChildren(
						NewStringNode("Run complex calculations."),
						NewNode("Run").Block(BlockGroup).AddChildren(
							NewNode("x").AddChildren(NewNode("int")),
							NewNode("y").AddChildren(NewNode("int")),
							NewNode("z").AddChildren(NewNode("string")),
						),
					),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser("parser_test.go", strings.NewReader(tt.text))
			fmt.Println(parser, parser.visitor, parser.visitor.visitMe)
			tree, err := parser.Parse()
			if tt.name == "BlockNoBrackets" {
				for _, child := range tree.Children {
					fmt.Println(child.Name, child.Text, child.Comment)
				}
			}

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
					nicePath := strings.Join(d.Path, ".")

					// Skip differences on node ranges, as those are too noisy to test.
					// This is a bit hacky, but is fine for testing. It would be nicer to
					// have a custom recursive function to compare nodes.
					if strings.Contains(nicePath, "Range.") {
						continue
					}

					t.Errorf("property '%s' %s, expected %s but got %s",
						nicePath,
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
