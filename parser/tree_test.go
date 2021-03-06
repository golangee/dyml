// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package parser_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	. "github.com/golangee/dyml/parser"
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
			want: NewNode("root").Block(BlockNormal),
		},
		{
			name: "just text",
			text: "hello world",
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewStringNode("hello world"),
			),
		},
		{
			name: "BlockNoBrackets",
			text: "#title Chapter Two",
			want: NewNode("root").Block(BlockNormal).AddChildren(NewNode("title").AddChildren(NewStringNode("Chapter Two"))),
		},
		{
			name: "different children types",
			text: "hello #item1 world #item2 #item3 more text",
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewStringNode("hello "),
				NewNode("item1").AddChildren(
					NewStringNode("world "),
				),
				NewNode("item2"),
				NewNode("item3").AddChildren(
					NewStringNode("more text"),
				),
			),
		},
		{
			name: "recursion and whitespace",
			text: "#A   { #B{#C  #D{#E }} } #F",
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("A").Block(BlockNormal).AddChildren(
					NewNode("B").Block(BlockNormal).AddChildren(
						NewNode("C"),
						NewNode("D").Block(BlockNormal).AddChildren(
							NewNode("E"),
						),
					),
				),
				NewNode("F"),
			),
		},
		{
			name: "elements with text",
			text: `#title Hello #subtitle World`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("title").AddChildren(
					NewStringNode("Hello "),
				),
				NewNode("subtitle").AddChildren(
					NewStringNode("World"),
				),
			),
		},
		{
			name: "attributes",
			text: `#item @id{5} @hello{world} @complex{attribute "with" #special\} 'chars}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("item").
					AddAttribute("id", "5").
					AddAttribute("hello", "world").
					AddAttribute("complex", `attribute "with" #special} 'chars`),
			),
		},
		{
			name: "attribute in nested element",
			text: "#item { #subitem @hello{world} }",
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("item").Block(BlockNormal).AddChildren(
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
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("A"),
				NewNode("D").Block(BlockNormal).AddChildren(
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
			want: NewNode("root").Block(BlockNormal).AddChildren(
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
			want: NewNode("root").Block(BlockNormal).AddChildren(
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
			text: "#? This is a comment.\nThis is more comment.",
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewStringCommentNode("This is a comment.\nThis is more comment."),
			),
		},
		{
			name:    "empty G2",
			text:    `#!`,
			wantErr: true,
		},
		{
			name: "simple G2",
			text: `#! item {}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("item").Block(BlockNormal),
			),
		},
		{
			name: "simple G2 alternative brackets",
			text: `#! item ()`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("item").Block(BlockGroup),
			),
		},
		{
			name: "G2 ends with ;",
			text: `some text #! item; more text`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewStringNode("some text "),
				NewNode("item"),
				NewStringNode("more text"),
			),
		},
		{
			name: "G2 ends with ,",
			text: `#! item,`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("item"),
			),
		},
		{
			name: "G2 ends with string",
			text: `some text #! item "hello" more text`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewStringNode("some text "),
				NewNode("item").AddChildren(
					NewStringNode("hello")),
				NewStringNode("more text"),
			),
		},
		{
			name: "G2 with nested items",
			text: `some text #! a b<c> more text`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewStringNode("some text "),
				NewNode("a").AddChildren(
					NewNode("b").Block(BlockGeneric).AddChildren(
						NewNode("c"))),
				NewStringNode("more text"),
			),
		},
		{
			name: "siblings G2",
			text: `#!a{b, c}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("a").Block(BlockNormal).AddChildren(
					NewNode("b"),
					NewNode("c"),
				),
			),
		},
		{
			name: "nested G2",
			text: `#! item {subitem subsubitem "text"}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("item").Block(BlockNormal).AddChildren(
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
			text: `#! g2 {
						A B {
							C,
							D,
						}
						E {F, G}
						H
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("A").AddChildren(
						NewNode("B").Block(BlockNormal).AddChildren(
							NewNode("C"),
							NewNode("D"),
						),
					),
					NewNode("E").Block(BlockNormal).AddChildren(
						NewNode("F"),
						NewNode("G"),
					),
					NewNode("H"),
				),
			),
		},
		{
			name: "G2 string will stop parsing nested children",
			text: `#! g2 {
						A "hello" B
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("A").AddChildren(
						NewStringNode("hello"),
					),
					NewNode("B"),
				),
			),
		},
		{
			name: "simple attribute G2",
			text: `#! g2 {
						item @key="value" @another="key with 'special #chars\""
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("item").
						AddAttribute("key", "value").
						AddAttribute("another", `key with 'special #chars"`),
				),
			),
		},
		{
			name: "attribute with siblings G2",
			text: `#! g2 {
						A,
						B C @key="value" D,
						E,
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
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
			),
		},
		{
			name:    "invalid lonely attribute G2",
			text:    `#! g2 {@key="value"}`,
			wantErr: true,
		},
		{
			name: "invalid attribute defined twice G2",
			text: `#! g2 {
						item @key="value" @key="value"
					}`,
			wantErr: true,
		},
		{
			name: "simple forwarded attribute G2",
			text: `#! g2 {
						@@key="value"
						item
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("item").
						AddAttribute("key", "value"),
				),
			),
		},
		{
			name: "forwarded attributes G2",
			text: `#! g2 {
						item,
						@@key="value"
						@@another="one"
						item @not="forwarded",
						parent @@for="child" child,
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
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
			),
		},
		{
			name: "invalid dangling forward attribute G2",
			text: `#! g2 {
						item @@key="value"
					}`,
			wantErr: true,
		},
		{
			name: "invalid forward attribute for text G2",
			text: `#! g2 {
						@@key="value" "text"
					}`,
			wantErr: true,
		},
		{
			name: "G1 line in G2",
			text: `#! g2 {
						# This is a G1 text line. #item @key{with value}
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewStringNode("This is a G1 text line. "),
					NewNode("item").
						AddAttribute("key", "with value"),
				),
			),
		},
		{
			name: "nested G1 line",
			text: `#! g2 {
						item # text child #child
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("item").AddChildren(
						NewStringNode("text child "),
						NewNode("child"),
					),
				),
			),
		},
		{
			name: "forward G1 line",
			text: `#! g2 {
						## forwarded #item @with{attribute}
						parent with children
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("parent").AddChildren(
						NewStringNode("forwarded "),
						NewNode("item").AddAttribute("with", "attribute"),
						NewNode("with").AddChildren(
							NewNode("children"),
						),
					),
				),
			),
		},
		{
			name: "empty G1 line",
			text: `#! g2 {
						#
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(NewNode("g2").Block(BlockNormal)),
		},
		{
			name: "forwarding node in forwarding line is forbidden",
			text: `#! g2 {
						## ##A #B
						C
					}`,
			wantErr: true,
		},
		{
			name: "forward attributes in forward line",
			text: `#! g2 {
						## @@key{value} #item
						parent
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("parent").AddChildren(
						NewNode("item").AddAttribute("key", "value"),
					),
				),
			),
		},
		{
			name: "invalid forward G1 line",
			text: `#! g2 {
						## where would this text be forwarded to?
					}`,
			wantErr: true,
		},
		{
			name: "many G1 lines",
			text: `#! g2 {
						# Hello!
						# Hello!
						# Hello!
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewStringNode("Hello!"),
					NewStringNode("Hello!"),
					NewStringNode("Hello!"),
				),
			),
		},
		{
			name: "forward G1 line with string",
			text: `#! g2 {
						## hello
						"this is a string"
						item
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewStringNode("this is a string"),
					NewNode("item").AddChildren(
						NewStringNode("hello"),
					),
				),
			),
		},
		{
			name: "other group types",
			text: `#! g2 {
						item { X , Y}
						item < X ,Y  >
						item (X, Y )
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("item").Block(BlockNormal).AddChildren(
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
			),
		},
		{
			name: "incorrect closing type",
			text: `#! g2 {
						item {>
					}`,
			wantErr: true,
		},
		{
			name: "nested groups",
			text: `#! g2 {
						item< item( item ) >
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("item").Block(BlockGeneric).AddChildren(
						NewNode("item").Block(BlockGroup).AddChildren(
							NewNode("item"),
						),
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
			name: "g2 comment",
			text: `#! g2 {
						// First comment
						item // A comment
						,
						item
						// Another comment
						item
						// Last comment
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewStringCommentNode("First comment"),
					NewNode("item").AddChildren(
						NewStringCommentNode("A comment"),
					),
					NewNode("item").AddChildren(
						NewStringCommentNode("Another comment"),
						NewNode("item").AddChildren(
							NewStringCommentNode("Last comment"),
						),
					),
				),
			),
		},

		{
			name: "g2 return arrow",
			text: `#! g2 {
						hello(string) -> (int)
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("hello").Block(BlockGroup).AddChildren(
						NewNode("string"),
						NewNode("ret").Block(BlockGroup).AddChildren(
							NewNode("int"),
						),
					),
				),
			),
		},
		{
			name: "g2 invalid return arrow after nothing",
			text: `#! g2 {
						-> (int)
					}`,
			wantErr: true,
		},
		{
			name: "g2 return arrow after element without block",
			text: `#! g2 {
						x -> (y)
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("x").AddChildren(
						NewNode("ret").Block(BlockGroup).AddChildren(
							NewNode("y"),
						),
					),
				),
			),
		},
		{
			name: "arrow without blocks",
			text: `#! g2 {
						x -> y
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("x").
						AddChildren(NewNode("ret").
							AddChildren(NewNode("y"))),
				),
			),
		},
		{
			name: "multiple arrows",
			text: `#! g2 {
						x -> y -> z
					}`,
			wantErr: true,
		},
		{
			name: "g2 invalid return arrow after comma",
			text: `#! g2 {
						x, -> (y)
					}`,
			wantErr: true,
		},
		{
			name: "g2 arrow directly after preamble",
			text: `#! x -> y`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("x").AddChildren(
					NewNode("ret").AddChildren(
						NewNode("y"),
					),
				),
			),
		},
		{
			name:    "g2 arrow with nothing",
			text:    "#! x -> ;",
			wantErr: true,
		},
		{
			name: "g2 return arrow with generic blocks",
			text: `#! g2 {
						fn x<y> -> <z>
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("fn").AddChildren(
						NewNode("x").Block(BlockGeneric).AddChildren(
							NewNode("y"),
							NewNode("ret").Block(BlockGeneric).AddChildren(
								NewNode("z"),
							),
						),
					),
				),
			),
		},
		{
			name: "function definition example",
			text: `#! g2 {
						## Greet someone.
						@@name="The name to greet."
						func Greet(name string)

						## Run complex calculations.
						func Run(x int, y int, z string) -> (int, error)
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
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
								NewNode("ret").Block(BlockGroup).AddChildren(
									NewNode("int"),
									NewNode("error"),
								),
							),
						),
				),
			),
		},
		{
			name: "trailing commas",
			text: `#! g2 {
						list{
							item1 key "value",
							@@id="1"
							item2,
							item3 @key="value",
						}
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("list").Block(BlockNormal).AddChildren(
						NewNode("item1").
							AddChildren(
								NewNode("key").
									AddChildren(
										NewStringNode("value"))),
						NewNode("item2").AddAttribute("id", "1"),
						NewNode("item3").AddAttribute("key", "value"),
					),
				),
			),
		},
		{
			name: "invalid consecutive commas",
			text: `#! g2 {
						item,
						@@key="value"
						@@another="one"
						item @not="forwarded",
						parent @@for="child" child,,
					}`,
			wantErr: true,
		},
		{
			name: "semicolon as separator",
			text: `#! g2 {
						a; b; c;
					}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("g2").Block(BlockNormal).AddChildren(
					NewNode("a"),
					NewNode("b"),
					NewNode("c"),
				),
			),
		},
		{
			name: "multiple g2s",
			text: `#!a{} #!b{} #!c{d e} #!text{"some text"} #!attr{@@with="some" f @key="attributes"}`,
			want: NewNode("root").Block(BlockNormal).AddChildren(
				NewNode("a").Block(BlockNormal),
				NewNode("b").Block(BlockNormal),
				NewNode("c").Block(BlockNormal).
					Block(BlockNormal).
					AddChildren(NewNode("d").
						AddChildren(NewNode("e"))),
				NewNode("text").
					Block(BlockNormal).
					AddChildren(NewStringNode("some text")),
				NewNode("attr").Block(BlockNormal).AddChildren(
					NewNode("f").
						AddAttribute("with", "some").
						AddAttribute("key", "attributes"),
				),
			),
		},
	}

	t.Parallel()

	for _, ttt := range tests {
		tt := ttt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := NewParser("parser_test.go", strings.NewReader(tt.text))
			tree, err := parser.Parse()

			//nolint
			if tt.wantErr {
				// We wanted an error...
				if err != nil {
					// ...and got one.
					// Comparing trees is nonsense in this case, so we skip that.
					return
				} else {
					// ...but did not get one!
					t.Errorf("expected error, but did not get one")

					return
				}
			} else {
				// We did not want an error...
				if err != nil {
					// ...but got one!
					t.Error(err)

					return
				} else {
					// ...and we did not get one.
					// Everything went fine and we will start comparing trees.
				}
			}

			differences, err := diff.Diff(tt.want, tree,
				diff.Filter(func(path []string, parent reflect.Type, field reflect.StructField) bool {
					// Skip any unexported fields when comparing
					return field.IsExported()
				}))
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
