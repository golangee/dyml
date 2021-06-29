package streamxmlencoder

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// test recursive encoding, using the parser and a full tree buffer
func TestEncoderRek(t *testing.T) {
	var encoder XMLEncoder
	tests := []struct {
		name          string
		text          string
		want, wantAlt string
		wantErr       bool
	}{
		{
			name: "hello world",
			text: `#? saying hello world
		 			#hello{world}`,
			want: `<root>
			<!-- saying
				 hello world

			-->
			<hello>world</hello>
			</root>`,
			wantErr: false,
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
			wantErr: false,
		},
		{
			// might fail, as the sequence in which Attributes are added to an identifier may differ,
			// #book @id{my-book} @author{Torben} can be translated to:
			// <book id="my-book" author="Torben"> or <book author="Torben id="my-book"">
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
			wantAlt: `<root>
			<book author="Torben id="my-book"">
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
			name: "equivalent example grammar1",
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
			wantErr: false,
		},
		/*{
			name: "equivalent example grammar2",
			text: `#!{
						list{
							item1 key "value",
							@@1
							item2,
							item3 @key="value",
						}
					}`,
			want: `<root>
						<list>
							<item1><key>value</key></item1>
							<item2 id="1"/>
							<item3 key="value"/>
						</list>
					</root>`,
		},
		{
			name: "complex node first",
			text: `#!{
						# just a text line

						## This is a forward text node. It contains a non-recursive grammar 1, so e.g. #ref{id} is possible.
						type Person struct {
							## ...is the first name
							Firstname int32

							## ...come get some.
							@@stuff ...is the stuff parameter.
							@@other="...is the other parameter."
							func Get(stuff string, other []int, list Map<X,Y>) ->(int32|error) // note the different closing rules here
						}
					}`,
			want: `<root>
					just a text line
					<type>
						This is a forward text node. It contains a non-recursive grammar 1, so e.g. <ref>id</ref> is possible.
						<Person>
							<struct _groupType="{}">
								<Firstname>
									...is the first name
									<int32/>
								</Firstname>

								<func stuff="...is the stuff parameter." other="...is the other parameter.">
									...come get some.
									<Get _groupType="()">
										<stuff><string/></stuff>
										<other>
											<SLICE><int/></SLICE>
										</other>
										<list><Map _groupType="<>"><X/><Y/></List></list>
									</Get>
									<ret _token="->" _groupType="()">
										<int32/>
										<error/>
									</ret>
								</func> <!-- note the different closing rules here-->
							</struct>
						</Person>
					</type>

				</root>`,
			wantErr: false,
		},*/
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			var output string
			var b bytes.Buffer
			encoder = NewEncoder(test.name, bytes.NewBuffer([]byte(test.text)), bufio.NewWriter(&b))
			output, err := encoder.EncodeToXML()

			if test.wantErr {
				if err == nil {
					t.Errorf("expected Error")
				}
			} else {
				if err != nil {
					t.Error(err)
				} else {
					if !StringsEqual(output, test.want) {
						// if a testcase includes multiple Attributes on a single Node, the sequence of these Attributes is not fixed,
						// as it passes a non ordered slice. In this Case, an alternative testcase is used, switching the corresponding Attributes
						if test.wantAlt == "" || !StringsEqual(output, test.wantAlt) {
							t.Errorf("Test %s failed: \n%s\n\n\n does not equal \n\n\n%s \n(Ignoring Whitespaces, Tabs, newlines)", test.name, output, test.want)
						}
					}
				}
			}

		})
	}
}

//test stream encoding
func TestEncoderStream(t *testing.T) {
	var encoder XMLEncoder
	tests := []struct {
		name          string
		text          string
		want, wantAlt [20]string
		wantErr       bool
	}{
		/*{
			name: "hello world",
			text: `#? saying hello world
		 			#hello{world}`,
			want:    [20]string{`<root>`, `<!-- saying hello world -->`, `<hello>`, `world`, `</hello>`, `</root>`},
			wantErr: false,
		},*/
		{
			name:    "Identifier + Attributes",
			text:    `#book @id{my-book} @author{Torben}`,
			want:    [20]string{`<root>`, `<book id="my-book" author="Torben">`, `</book>`, `</root>`},
			wantErr: false,
		},
		/*{
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
			wantErr: false,
		},
		{
			// might fail, as the sequence in which Attributes are added to an identifier
			// depends on the sequence they lie in the Nodes' Attributes-Slice
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
			wantAlt: `<root>
			<book author="Torben id="my-book"">
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
			name: "equivalent example grammar1",
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
			wantErr: false,
		},*/
	}
	for _, test := range tests {
		t.Run("stream - "+test.name, func(t *testing.T) {
			var output string
			var b bytes.Buffer
			var err error

			writer := bufio.NewWriter(&b)
			reader := bytes.NewBuffer([]byte(test.text))

			/*lexer := parser2.NewLexer("default", reader)
			tok, err := lexer.Token()
			fmt.Println("reader ", reader)
			fmt.Println("token, err ", tok, err)
			fmt.Println("tokentype, err ", tok.TokenType(), err)*/

			encoder = NewEncoder(test.name, reader, writer)
			fmt.Println("encoder, output ", encoder, output)

			i := 0
			for output, err := encoder.Next(); output == "" || err != io.EOF || err != nil; i++ {
				fmt.Println(output)

				if output != test.want[i] {
					t.Errorf("Test %s failed: \n%s\n\n\n does not equal \n\n\n%s \n(Ignoring Whitespaces, Tabs, newlines)", test.name, output, test.want)
				} else {
					fmt.Printf("Successfully translated Element: " + test.want[i])
				}
				output, err = encoder.Next()
			}
			if err != nil {
				t.Error(err)
			}

		})
	}
}

// StringsEqual compares two given strings
// ignores differences in Whitespaces, Tabs and newlines
func StringsEqual(in1, in2 string) bool {
	var cleanIn1 string = strings.Replace(strings.Replace(strings.Replace(in1, "\n", "", -1), " ", "", -1), "\t", "", -1)
	var cleanIn2 string = strings.Replace(strings.Replace(strings.Replace(in2, "\n", "", -1), " ", "", -1), "\t", "", -1)
	if cleanIn1 == cleanIn2 {
		return true
	}
	return false
}
