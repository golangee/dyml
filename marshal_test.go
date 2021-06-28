package tadl

import (
	"github.com/r3labs/diff/v2"
	"reflect"
	"strings"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	// Base for testing
	type TestCase struct {
		text   string
		strict bool
		// into is an empty instance we will unmarshal into.
		into interface{}
		// want is a filled instance with all values we want.
		want    interface{}
		wantErr bool
	}

	var testCases []TestCase

	type EmptyRoot struct{}

	testCases = append(testCases, TestCase{
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
		text:    "#V 300",
		into:    &OutOfBounds{},
		wantErr: true,
	})

	// Run all test cases
	for _, tc := range testCases {
		testName := reflect.ValueOf(tc.into).Elem().Type().Name()
		t.Run(testName, func(t *testing.T) {
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

					t.Errorf("property '%s' %s, expected %v but got %v",
						nicePath,
						changeTypeDescription[d.Type],
						d.From, d.To)
				}
			}
		})
	}
}
