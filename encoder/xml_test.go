package encoder_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/golangee/dyml/encoder"
)

func TestXMLEncode(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "simple",
			text: "",
			want: "<root></root>",
		},
		{
			name: "hello world",
			text: `#? saying hello world
							#hello{world}`,
			want: `<root><!-- saying hello world --><hello>world</hello></root>`,
		},
		{
			name: "Identifier + Attributes",
			text: `#book @id{my-book} @author{Torben}`,
			want: `<root><book id="my-book" author="Torben"></book></root>`,
		},
		{
			name: "book example",
			text: `#book {
					#toc{}
					#section @id{1} {
					  #title {
						  The sections title
					  }

					  The sections text.
					}
				  }`,
			want: `<root>
					<book>
						<toc></toc>
						<section id="1">
							<title>
								The sections title
							</title>

							The sections text.
						</section>
					</book>
				</root>`,
		},
		{
			name: "complex book example",
			text: `#book @id{my-book} @author{Torben} {
							#title { A very simple book }
							#chapter @id{ch1} {
								#title {
									Chapter One
								}
								#p {
									Hello paragraph.
								Still going on.
								}
							}

							#chapter @id{ch2} {
								#title { Chapter Two }
			 					Some #red{#bold{ Text}} text.
								The #span @style{color:red} { #span @style{font-weight:bold} {Text }} text.
								#image @width{100%} {https://worldiety.de/favicon.png}
							}
						}`,
			want: `<root>
						<book id="my-book" author="Torben">
							<title>A very simple book</title>
							<chapter id="ch1">
								<title>Chapter One</title>
								<p>Hello paragraph.
								Still going on.</p>
							</chapter>

							<chapter id="ch2">
								<title>Chapter Two</title>
								Some <red><bold>Text</bold></red> text.
								The <span style="color:red"><span style="font-weight:bold">Text </span></span> text.
								<image width="100%">https://worldiety.de/favicon.png</image>
							</chapter>
						</book>
					</root>`,
		},
		{
			name: "equivalent example grammar1.1",
			text: `#list{
							#item1{#key {value}}
							#item2 @id{1}
							#item3 @key{value}
						}`,
			want: `<root>
							<list>
								<item1><key>value</key></item1>
								<item2 id="1"></item2>
								<item3 key="value"></item3>
							</list>
						</root>`,
		},
		{
			name: "equivalent example grammar1.2",
			text: `#! list {
						item1 key "value",
						@@id="1"
						item2,
						item3 @key="value",
					}`,
			want: `<root>
							<list>
								<item1><key>value</key></item1>
								<item2 id="1"></item2>
								<item3 key="value"></item3>
							</list>
						</root>`,
		},
		{
			name: "simple forwarded attribute G2",
			text: `#! parent {
							@@key="value"
							item
						}`,
			want: `<root>
						<parent>
							<item key="value"></item>
						</parent>
					</root>`,
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
			want: `<root>
						<g2>
							<item></item>
							<item key="value" another="one" not="forwarded"></item>
							<parent>
								<child for="child"></child>
							</parent>
						</g2>
					</root>`,
		},
		{
			name: "simple G2 return arrow",
			text: `#! g2 {
						hello(string) -> (int)
					}`,
			want: `<root>
						<g2>
							<hello>
								<string></string>
								<ret>
									<int></int>
								</ret>
							</hello>
						</g2>
					</root>`,
		},
		{
			name: "g2 return arrow with generic blocks",
			text: `#! g2 {
						fn x<y> -> <z>
					}`,
			want: `<root>
						<g2>
							<fn>
								<x>
									<y></y>
									<ret>
										<z></z>
									</ret>
								</x>
							</fn>
						</g2>
					</root>`,
		},
		{
			name: "escape quotes",
			text: `#? saying "hello world"
				#hello{world}`,
			want: ` <root>
							<!-- saying &quot;hello world&quot; -->
							<hello>world
							</hello>
						</root>`,
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
			want: `<root>
						<g2>
							<func name="The name to greet.">
								Greet someone.
								<Greet>
									<name>
										<string></string>
									</name>
								</Greet>
							</func>
							<func>
								Run complex calculations.
								<Run>
									<x>
										<int></int>
									</x>
									<y>
										<int></int>
									</y>
									<z>
										<string></string>
									</z>
									<ret>
										<int></int>
										<error></error>
									</ret>
								</Run>
							</func>
						</g2>
					</root>`,
		},
		{
			name: "forward node",
			text: `
					##a
					#b
				`,
			want: `<root><b><a></a></b></root>`,
		},
		{
			name: "backslashes are okay",
			text: `#book @id{my-book\\} @author{Torben\\}`,
			want: `<root><book id="my-book\" author="Torben\"></book></root>`,
		},
		{
			name: "a lot of special chars",
			text: `<tag></tag>&"hello"`,
			want: "<root>&lt;tag&gt;&lt;/tag&gt;&amp;&quot;hello&quot;</root>",
		},
	}

	t.Parallel()

	for _, tt := range tests {
		test := tt

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var writer bytes.Buffer
			reader := bytes.NewBuffer([]byte(test.text))
			enc := encoder.NewXMLEncoder(test.name, reader, &writer)
			err := enc.Encode()
			if err != nil {
				t.Error(err)

				return
			}

			val := writer.String()

			if !StringsEqual(test.want, val) {
				t.Errorf("Test '%s' failed. Wanted '%s', got '%s'", test.name, test.want, val)
			}
		})
	}
}

// StringsEqual compares two given strings but ignores differences in whitespaces, tabs and newlines.
func StringsEqual(in1, in2 string) bool {
	r := strings.NewReplacer("\n", "", "\t", "", " ", "")

	return r.Replace(in1) == r.Replace(in2)
}
