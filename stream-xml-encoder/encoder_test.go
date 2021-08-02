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
		name    string
		text    string
		want    string
		wantErr bool
	}{
		/**/ {
			name: "hello world",
			text: `#? saying hello world
						#hello{world}`,
			want:    `<root><!-- saying hello world --><hello>world</hello></root>`,
			wantErr: false,
		},
		{
			name:    "Identifier + Attributes",
			text:    `#book @id{my-book} @author{Torben}`,
			want:    `<root><book id="my-book" author="Torben"></book></root>`,
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
		/*{
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
			writer := new(bytes.Buffer)
			reader := bytes.NewBuffer([]byte(test.text))
			encoder = NewEncoder(test.name, reader, writer, 10)

			err := encoder.Encode()
			if err != nil {
				t.Errorf("Test failed, unexpected error: %v", err)
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
	var cleanIn1 string = strings.Replace(strings.Replace(strings.Replace(in1, "\n", "", -1), " ", "", -1), "\t", "", -1)
	var cleanIn2 string = strings.Replace(strings.Replace(strings.Replace(in2, "\n", "", -1), " ", "", -1), "\t", "", -1)
	return cleanIn1 == cleanIn2
}
