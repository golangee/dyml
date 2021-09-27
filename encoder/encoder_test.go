package streamxmlencoder

import (
	"bytes"
	"strings"
	"testing"
)

//test stream encoding
func TestEncoderStream(t *testing.T) {
	var encoder Encoder
	tests := []struct {
		name     string
		text     string
		want     string
		wantErr  bool
		buffsize int
	}{
		{
			name: "hello world",
			text: `#? saying hello world
							#hello{world}`,
			want:     `<root><!-- saying hello world --><hello _groupType="{}">world</hello></root>`,
			wantErr:  false,
			buffsize: 5,
		},
		{
			name:     "Identifier + Attributes",
			text:     `#book @id{my-book} @author{Torben}`,
			want:     `<root><book id="my-book" author="Torben"></book></root>`,
			wantErr:  false,
			buffsize: 5,
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
					<book _groupType="{}">
						<toc _groupType="{}"></toc>
						<section id="1" _groupType="{}">
							<title _groupType="{}">
								The sections title
							</title>

							The sections text.
						</section>
					</book>
				</root>`,
			wantErr:  false,
			buffsize: 5,
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
						<book id="my-book" author="Torben" _groupType="{}">
							<title _groupType="{}">A very simple book</title>
							<chapter id="ch1" _groupType="{}">
								<title _groupType="{}">Chapter One</title>
								<p _groupType="{}">Hello paragraph.
								Still going on.</p>
							</chapter>

							<chapter id="ch2" _groupType="{}">
								<title _groupType="{}">Chapter Two</title>
								Some <red _groupType="{}"><bold _groupType="{}">Text</bold></red> text.
								The <span style="color:red" _groupType="{}"><span style="font-weight:bold" _groupType="{}">Text </span></span> text.
								<image width="100%" _groupType="{}">https://worldiety.de/favicon.png</image>
							</chapter>
						</book>
					</root>`,
			buffsize: 10,
		},
		{
			name: "equivalent example grammar1.1",
			text: `#list{
							#item1{#key {value}}
							#item2 @id{1}
							#item3 @key{value}
						}`,
			want: `<root>
							<list _groupType="{}">
								<item1 _groupType="{}"><key _groupType="{}">value</key></item1>
								<item2 id="1"></item2>
								<item3 key="value"></item3>
							</list>
						</root>`,
			wantErr:  false,
			buffsize: 5,
		},
		{
			name: "equivalent example grammar1.2",
			text: `#!{
							list{
								item1 key "value",
								@@id="1"
								item2,
								item3 @key="value",
							}
						}`,
			want: `<root>
							<list _groupType="{}">
								<item1><key>value</key></item1>
								<item2 id="1"></item2>
								<item3 key="value"></item3>
							</list>
						</root>`,
			wantErr:  false,
			buffsize: 5,
		},
		{
			name: "simple forwarded attribute G2",
			text: `#!{
							@@key="value"
							item
						}`,
			want: `<root>
							<item key="value"></item>
						</root>`,
			wantErr:  false,
			buffsize: 5,
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
			want: `<root>
							<item></item>
							<item not="forwarded" key="value" another="one"></item>
							<parent>
								<child for="child"></child>
							</parent>
						</root>`,
			wantErr:  false,
			buffsize: 10,
		},
		{
			name: "invalid consecutive commas",
			text: `#!{
							item,
							@@key="value"
							@@another="one"
							item @not="forwarded",
							parent @@for="child" child,,
						}`,
			want: `<root>
							<item></item>
							<item not="forwarded" key="value" another="one"></item>
							<parent>
								<child for="child"></child>
							</parent>
						</root>`,
			wantErr:  true,
			buffsize: 10,
		},

		// TODO: lack of clarity: "->" encoded to "<ret>" or `<ret _token="->">`?
		{
			name: "G2 return arrow, simple",
			text: `#!{
					hello(string) -> (int)
				}`,
			want: `<root>
						<hello _groupType="()">
							<string></string>
							<ret _groupType="()"><int>
								</int>
							</ret>
						</hello>
					</root>`,
			wantErr:  false,
			buffsize: 5,
		},
		{
			name: "g2 invalid return arrow after nothing",
			text: `#!{
							-> (int)
						}`,
			wantErr: true,
		},

		{
			name: "g2 return arrow with generic blocks",
			text: `#!{
							fn x<y> -> <z>
						}`,
			want: `<root>
						<fn>
							<x _groupType="<>">
								<y></y>
								<ret _groupType="<>">
									<z></z>
								</ret>
							</x>
						</fn>
					</root>`,
		},
		{
			name: "g2 invalid return arrow after nothing",
			text: `#!{
							-> (int)
						}`,
			wantErr: true,
		},
		{
			name: "escape quotationmarks",
			text: `#? saying "hello world"
				#hello{world}`,
			want: ` <root>
							<!-- saying "hello world" -->
							<hello _groupType="{}">world
							</hello>
						</root>`,
		},
		//TODO: add tests for g2Arrow, string escaping

		{
			name: "function definition example",
			text: `#!{
							## Greet someone.
							@@name="The name to greet."
							func Greet(name string)

							## Run complex calculations.
							func Run(x int, y int, z string) -> (int, error)
						}`,
			want: `<root>
						<func name="The name to greet.">
							<Greet _groupType="()">
								<name>
									<string></string>
								</name>
							</Greet>
						</func>
						<func>
							<Run _groupType="()">
								<x>
									<int></int>
								</x>
								<y>
									<int></int>
								</y>
								<z>
									<string></string>
								</z>
								<ret _groupType="()">
									<int></int>
									<error></error>
								</ret>
							</Run>
						</func>
					</root>`,
		},
		{
			name: "Escaped backslash",
			text: `#book @id{my-book\\} @author{Torben\\}`,
			// TODO This is invalid XML, backslashes need to be escaped
			want:    `<root><book id="my-book\" author="Torben\"></book></root>`,
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run("stream - "+test.name, func(t *testing.T) {
			writer := new(bytes.Buffer)
			reader := bytes.NewBuffer([]byte(test.text))
			encoder = NewEncoder(test.name, reader, writer, test.buffsize)

			/* first try on testing streaming capability
			go func() {
				err := encoder.Encode()
				if err != nil {
					t.Errorf("Test failed, unexpected error: %v", err)
				}
			}()
			for c := range writer.Bytes() {
				fmt.Printf("%c\n", c)
			}*/

			err := encoder.Encode()

			if !test.wantErr && err != nil {
				t.Error(err)
				return
			}

			if test.wantErr && err == nil {
				t.Errorf("expected error, but did not get one")
				return
			}

			if test.wantErr {
				// We wanted an error and got it, comparing trees would
				// make no sense, so we end this test here.
				return
			}

			if err != nil {
				if test.wantErr == false {
					t.Errorf("Test failed, unexpected error: %v", err)
				}
				// Wanted an error and got it, no need to continue
				return
			}

			val := writer.String()

			if !StringsEqual(test.want, val) {
				t.Errorf("Test '%s' failed. Wanted '%s', got '%s'", test.name, test.want, val)
			}

		})
	}
}

// StringsEqual compares two given strings
// ignores differences in Whitespaces, Tabs and newlines
func StringsEqual(in1, in2 string) bool {
	r := strings.NewReplacer("\n", "", "\t", "", " ", "")
	return r.Replace(in1) == r.Replace(in2)
}
