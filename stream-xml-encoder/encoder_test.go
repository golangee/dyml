package streamxmlencoder

import (
	"testing"
)

func TestEncoder(t *testing.T) {
	var encoder XMLEncoder
	tests := []struct {
		name    string
		text    string
		want    string
		wantErr bool
	}{
		{
			name: "hello world",
			text: `#? saying
						hello world
		 
		 			#hello{world}`,
			want: `<root>
			<!-- saying
				 hello world
			
			-->
			<hello>world</hello>
			</root>`,
			wantErr: false,
		},
		/*{
			name: "book example",
			text: `#book {
				#toc{}
				#section #:1 {
				  #title {
					  The sections title
				  }

				  The sections text.
				}
			  }`,
			want: `<root>
				<book>
					<toc/>
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
			name: "complex book example",
			text: `#book #:my-book ##author Torben {
					#title A very simple book
					#chapter :id{ch1} {
						#title Chapter One
						#p Hello paragraph.
						Still going on.
					}

					#chapter :id{ch2} {
						#title Chapter Two
						Some #red{#bold Text} text.
						The #span ##style{color:red} { #span ##style{font-weight:bold} Text } text.
						#image ##width{100%} https://worldiety.de/favicon.png
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
			name: "equivalent example grammar1",
			text: `#list{
					#item1{#key value}
					#item2 :id{1}
					#item3 :key{value}
			  	}`,
			want: `<root>
						<list>
							<item1><key>value</key></item1>
					 		<item2 id="1"/>
					 		<item3 key="value"/>
						</list>
					</root>`,
			wantErr: false,
		},
		{
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
			encoder = NewEncoderFromNameAndString(test.name, test.text)
			output, err := encoder.EncodeToXML()

			if test.wantErr {
				if err == nil {
					t.Errorf("expected Error")
				}
			} else {
				if err != nil {
					t.Error(err)
				} else {
					if output != test.want {
						t.Errorf("Test %s failed: %s does not equal %s", test.name, output, test.want)
					}
				}
			}

		})
	}
}
