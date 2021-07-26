// SPDX-FileCopyrightText: © 2021 The tadl authors <https://github.com/golangee/tadl/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package tadl

import (
	"fmt"
	"github.com/r3labs/diff/v2"
	"log"
	"strings"
	"testing"
)

func ExampleUnmarshal() {
	type Animal struct {
		Name string `tadl:"name"`
		Age  uint   `tadl:"age"`
	}

	input := strings.NewReader("#name Gopher #age 3")

	var animal Animal

	Unmarshal(input, &animal, false)

	fmt.Printf("Hello %d year old %s!", animal.Age, animal.Name)
	// Output: Hello 3 year old Gopher !
}

func ExampleUnmarshal_Slice() {
	type SimpleSlice struct {
		Nums []int
	}

	input := strings.NewReader(`#!{
		Nums {
			1, 2, 3
		}
	}`)

	var result SimpleSlice

	Unmarshal(input, &result, false)

	fmt.Print(result.Nums)
	// Output: [1 2 3]
}

// ExampleComplexSlice demonstrates more complex slice usage.
// Values will be placed in the correct slices because they
// have a rename tag set.
func ExampleUnmarshal_ComplexSlice() {
	type Animal struct {
		Name string `tadl:"name,attr"`
		Age  uint   `tadl:"age"`
	}

	type ComplexArray struct {
		Animals []Animal `tadl:"animal"`
		Planets []string `tadl:"planet"`
	}

	input := strings.NewReader(`#!{
		animal @name="Dog" {
			age 6
		}
		planet "Earth"
		animal @name="Cat",
		animal @name="Gopher" {
			age 3
		}
		planet "Venus"
		planet "Mars"
	}`)

	var result ComplexArray

	Unmarshal(input, &result, false)

	fmt.Printf("%s, %s", result.Animals[2].Name, result.Planets[0])
	// Output: Gopher, Earth
}

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
					Nums {1, 2, 3, 4}
				}`,
		into: &IntSlice{},
		want: &IntSlice{
			Nums: []int{1, 2, 3, 4},
		},
	})

	testCases = append(testCases, TestCase{
		name: "int slice with comments",
		text: `#!{
					Nums {
						1,
						2, 3, // This comment should be ignored.
						4
					}
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

	type FilteredSlice struct {
		Ints []int `tadl:"i"`
	}

	testCases = append(testCases, TestCase{
		name: "filtered slice",
		text: `#!{
					i 1,
					i 2,
					// please ignore this comment
					hello 123,
					i 3,
					someitem 456,
					"don't mind me"
					some nested things,
					i 4
				}`,
		into: &FilteredSlice{},
		want: &FilteredSlice{
			Ints: []int{1, 2, 3, 4},
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
		Text string `tadl:",inner"`
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
		Value string `tadl:",inner"`
	}

	type TextNested struct {
		Text   string          `tadl:",inner"`
		Inside TextNestedInner `tadl:"inside"`
	}

	testCases = append(testCases, TestCase{
		name: "text in some elements",
		text: `Hello world! #inside Lots of text here :)`,
		into: &TextNested{},
		want: &TextNested{
			Text: "Hello world! ",
			Inside: TextNestedInner{
				Value: "Lots of text here :)",
			},
		},
	})

	type LotsOfText struct {
		Text    string `tadl:",inner"`
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

	type StringStringMap struct {
		Things map[string]string
	}

	testCases = append(testCases, TestCase{
		name: "map[string]string",
		text: `#!{
					Things {
						key1 value,
						key2 "string value"
					}
				}`,
		into: &StringStringMap{},
		want: &StringStringMap{Things: map[string]string{
			"key1": "value",
			"key2": "string value",
		}},
	})

	testCases = append(testCases, TestCase{
		name: "map with comments",
		text: `#!{
					Things {
						// This comment should be ignored
						key1 value,
						// This comment should also be ignored
						key2 "string value"
						// This comment shall too be ignored
					}
				}`,
		into: &StringStringMap{},
		want: &StringStringMap{Things: map[string]string{
			"key1": "value",
			"key2": "string value",
		}},
	})

	type BoolFloatMap struct {
		Things map[bool]float64
	}

	testCases = append(testCases, TestCase{
		name: "map with primitive types",
		text: `#!{
					Things {
						true 123,
						false "123.456"
					}
				}`,
		into: &BoolFloatMap{},
		want: &BoolFloatMap{map[bool]float64{
			true:  123,
			false: 123.456,
		}},
	})

	type InvalidMapKey struct {
		Things map[*InvalidMapKey]int
	}

	testCases = append(testCases, TestCase{
		name:    "invalid map key",
		text:    "#Things",
		into:    &InvalidMapKey{},
		wantErr: true,
	})

	type NillableThing struct {
		Thing *Empty `tadl:"thing"`
	}

	testCases = append(testCases, TestCase{
		name: "nillable field is nil",
		text: "",
		into: &NillableThing{},
		want: &NillableThing{Thing: nil},
	})

	testCases = append(testCases, TestCase{
		name: "nillable field is set",
		text: "#thing",
		into: &NillableThing{},
		want: &NillableThing{Thing: &Empty{}},
	})

	type CustomMapValue struct {
		Name  string
		Value int
	}

	type MapWithCustomValue struct {
		Map map[string]CustomMapValue
	}

	testCases = append(testCases, TestCase{
		name: "map with custom type as value",
		text: `#!{
					Map {
						thingA {
							Name "this is thing A"
							Value 3
						}
						thingB {
							Name "this is thing B"
							Value 5
						}
					}
				}`,
		into: &MapWithCustomValue{},
		want: &MapWithCustomValue{
			map[string]CustomMapValue{
				"thingA": {
					Name:  "this is thing A",
					Value: 3,
				},
				"thingB": {
					Name:  "this is thing B",
					Value: 5,
				},
			},
		},
	})

	type StringA = string
	type StringB string

	type TypeAlias struct {
		StringA StringA
		StringB StringB
	}

	testCases = append(testCases, TestCase{
		name: "map with custom type as value",
		text: `#!{
					StringA "hello"
					StringB "world"
				}`,
		into: &TypeAlias{},
		want: &TypeAlias{
			StringA: "hello",
			StringB: "world",
		},
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

					t.Errorf("property '%s' %s, expected '%v' but got '%v'",
						nicePath,
						changeTypeDescription[d.Type],
						d.From, d.To)
				}
			}
		})
	}
}
