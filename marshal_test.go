package tadl

import (
	"fmt"
	"github.com/r3labs/diff/v2"
	"log"
	"strings"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	// Base for testing
	type TestCase struct {
		name   string
		text   string
		strict bool
		// into is an empty instance we will unmarshal into.
		into interface{}
		// want is a filled instance with all values we want.
		want    interface{}
		wantErr bool
	}

	var testCases []TestCase

	// Test cases always follow this pattern:
	// 1. Define all required types
	// 2. Define testcase using those types

	type EmptyRoot struct{}

	testCases = append(testCases, TestCase{
		name: "empty",
		text: "",
		into: &EmptyRoot{},
		want: &EmptyRoot{},
	})

	type SimpleRoot struct {
		S string
		I int8
		U uint64
	}

	testCases = append(testCases, TestCase{
		name: "struct with some types",
		text: "#S hello #I -5 #U 3000",
		into: &SimpleRoot{},
		want: &SimpleRoot{
			S: "hello ",
			I: -5,
			U: 3000,
		},
	})

	type OutOfBounds struct {
		V int8
	}

	testCases = append(testCases, TestCase{
		name:    "out of bounds int8",
		text:    "#V 300",
		into:    &OutOfBounds{},
		wantErr: true,
	})

	type Empty struct{}

	type EmptyElement struct {
		EmptyEl Empty
	}

	testCases = append(testCases, TestCase{
		name: "empty element",
		text: "#Empty",
		into: &EmptyElement{},
		want: &EmptyElement{
			EmptyEl: Empty{},
		},
	})

	type SimpleText struct {
		Text string
	}

	testCases = append(testCases, TestCase{
		name: "simple test",
		text: "#Text hello",
		into: &SimpleText{},
		want: &SimpleText{
			Text: "hello",
		},
	})

	testCases = append(testCases, TestCase{
		name: "absent empty element is correctly parsed in non-strict mode",
		text: "",
		into: &EmptyElement{},
		want: &EmptyElement{
			EmptyEl: Empty{},
		},
	})

	testCases = append(testCases, TestCase{
		name:    "absent empty element is denied in strict mode",
		text:    "",
		into:    &EmptyElement{},
		strict:  true,
		wantErr: true,
	})

	type IntSlice struct {
		Nums []int
	}

	testCases = append(testCases, TestCase{
		name: "int slice",
		text: `#!{
					Nums {"1" "2" "3" "4"}
				}`,
		into: &IntSlice{},
		want: &IntSlice{
			Nums: []int{1, 2, 3, 4},
		},
	})

	type EmptyStructSlice struct {
		Things []Empty
	}

	testCases = append(testCases, TestCase{
		name: "slice of empty structs",
		text: `#!{
					Things {Empty, Empty, Empty}
				}`,
		into: &EmptyStructSlice{},
		want: &EmptyStructSlice{
			Things: []Empty{{}, {}, {}},
		},
	})

	testCases = append(testCases, TestCase{
		name:    "do not unmarshal into nil",
		text:    "whatever",
		into:    nil,
		wantErr: true,
	})

	type SimpleRename struct {
		Field string `tadl:"item"`
	}

	testCases = append(testCases, TestCase{
		name: "field rename",
		text: `#item hello`,
		into: &SimpleRename{},
		want: &SimpleRename{Field: "hello"},
	})

	type InvalidFieldType struct {
		V string `tadl:",not-a-type"`
	}

	testCases = append(testCases, TestCase{
		name:    "invalid field type",
		text:    `#V hello`,
		into:    &InvalidFieldType{},
		wantErr: true,
	})

	type SimpleAttributeInner struct {
		Attribute string  `tadl:",attr"`
		Renamed   int     `tadl:"x,attr"`
		Boolean   bool    `tadl:"b,attr"`
		Float     float64 `tadl:"f,attr"`
	}

	type SimpleAttribute struct {
		Inner SimpleAttributeInner `tadl:"item"`
	}

	testCases = append(testCases, TestCase{
		name: "simple attribute",
		text: `#item @Attribute{Hello world!} @x{123} @b{true} @f{123.456}`,
		into: &SimpleAttribute{},
		want: &SimpleAttribute{
			Inner: SimpleAttributeInner{
				Attribute: "Hello world!",
				Renamed:   123,
				Boolean:   true,
				Float:     123.456,
			},
		},
	})

	type RequiredAttributeStrictInner struct {
		Attribute string `tadl:",attr"`
	}

	type RequiredAttributeStrict struct {
		Inner RequiredAttributeStrictInner `tadl:"item"`
	}

	testCases = append(testCases, TestCase{
		name:    "strict mode requires attribute to be set",
		text:    `#item`,
		into:    &RequiredAttributeStrict{},
		strict:  true,
		wantErr: true,
	})

	type TextDirectly struct {
		Text string `tadl:",text"`
	}

	testCases = append(testCases, TestCase{
		name: "plain text in root element",
		text: `Hello world!`,
		into: &TextDirectly{},
		want: &TextDirectly{
			Text: "Hello world!",
		},
	})

	testCases = append(testCases, TestCase{
		name: "empty text zero value",
		text: ``,
		into: &TextDirectly{},
		want: &TextDirectly{
			Text: "",
		},
	})

	testCases = append(testCases, TestCase{
		name:    "text required in strict mode",
		text:    ``,
		strict:  true,
		into:    &TextDirectly{},
		wantErr: true,
	})

	type TextNestedInner struct {
		Value string `tadl:",text"`
	}

	type TextNested struct {
		Text  string          `tadl:",text"`
		Inner TextNestedInner `tadl:"inner"`
	}

	testCases = append(testCases, TestCase{
		name: "text in some elements",
		text: `Hello world! #inner Lots of text here :)`,
		into: &TextNested{},
		want: &TextNested{
			Text: "Hello world! ",
			Inner: TextNestedInner{
				Value: "Lots of text here :)",
			},
		},
	})

	type LotsOfText struct {
		Text    string `tadl:",text"`
		Element Empty  `tadl:"item"`
	}

	testCases = append(testCases, TestCase{
		name: "scattered text will be concatenated",
		text: `hello #item{} this is text`,
		into: &LotsOfText{},
		want: &LotsOfText{
			Text:    "hello this is text",
			Element: Empty{},
		},
	})

	testCases = append(testCases, TestCase{
		name:    "scattered text is forbidden in strict mode",
		text:    `hello #item{} this is text`,
		strict:  true,
		into:    &LotsOfText{},
		wantErr: true,
	})

	// Run all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Unmarshal(strings.NewReader(tc.text), tc.into, tc.strict)

			if err != nil {
				if tc.wantErr {
					// We got an expected error.
					return
				} else {
					t.Fatal(err)
				}
			} else {
				if tc.wantErr {
					t.Fatal("expected an error, but got none")
				}
			}

			differences, err := diff.Diff(tc.want, tc.into)
			if err != nil {
				log.Println(fmt.Errorf("cannot compare test result: %w", err))
				t.SkipNow()
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

					t.Errorf("property '%s' %s, expected %v but got %v",
						nicePath,
						changeTypeDescription[d.Type],
						d.From, d.To)
				}
			}
		})
	}
}
