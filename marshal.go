// SPDX-FileCopyrightText: © 2021 The tadl authors <https://github.com/golangee/tadl/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package tadl

import (
	"fmt"
	"github.com/golangee/tadl/parser"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshal takes Tadl input and parses it into the given struct.
// If "into" is not a struct, this method will fail.
// As this uses go's reflect package, only exported names can be unmarshalled.
// Strict mode requires that all fields of the struct are set and defined exactly once.
// You can set struct tags to influence the unmarshalling process.
// All tags must have the form `tadl:"..."` and are a list of comma separated identifiers.
//
// The first identifier can be used to rename the field, so that an element with the renamed
// name is parsed, and not the name of the struct field.
//
//  // This tadl snippet...
//  #!{item{...}}
//  // could be unmarshalled into this go struct.
//  type Example struct {
//      SomeName Content `tadl:"item"`
//  }
//
// The second identifier is used to specify what kind of thing is being parsed.
// This can be used to parse attributes (attr) or text (text).
//
// Attributes can be parsed into primitive types: string, bool and the integer (signed & unsigned) and float types.
// Should the value not be valid for the target type, e.g. an integer that is too large or a negative value for an uint,
// an error is returned describing the issue.
//
//  // This tadl snippet...
//  #item @key{value} @X{123}
//  // could be unmarshalled into this go struct.
//  type Example struct {
//      SomeName string `tadl:"key,attr"` // Notice how you can rename an attribute
//      X        int    `tadl:",attr"` // You can choose to not rename it, by omitting the rename parameter.
//  }
//
// Text can be parsed into fields marked with 'text'.
// In normal mode the field will have all text that occurred in the element concatenated or an empty string if no
// text was inside the element.
// In strict mode exactly one text item with length > 0 is expected.
// Renaming a text parameter is not an error but pointless, so best leave the rename parameter empty.
//
func Unmarshal(r io.Reader, into interface{}, strict bool) error {
	parse := parser.NewParser("", r)

	if into == nil {
		return fmt.Errorf("cannot unmarshal into nil")
	}

	tree, err := parse.Parse()
	if err != nil {
		return NewUnmarshalError(tree, "parser error", err)
	}

	value := reflect.ValueOf(into)
	unmarshal := unmarshaler{strict: strict}

	if err := unmarshal.node(tree, value); err != nil {
		return err
	}

	return nil
}

// unmarshaler is a helper struct for easier managing the unmarshalling process.
type unmarshaler struct {
	strict bool
}

// While unmarshalling we might need to process a node as an attribute.
// We use this enum to make the decision.
type unmarshalType int

const (
	unmarshalNormal unmarshalType = iota
	unmarshalAttribute
	unmarshalText
)

// UnmarshalError is an error that occurred during unmarshalling.
// It contains the offending node, a string with details and an underlying error (if any).
type UnmarshalError struct {
	Node     *parser.TreeNode
	Detail   string
	wrapping error
}

func NewUnmarshalError(node *parser.TreeNode, detail string, wrapping error) UnmarshalError {
	return UnmarshalError{
		node,
		detail,
		wrapping,
	}
}

func (u UnmarshalError) Error() string {
	if u.wrapping != nil {
		return fmt.Sprintf("cannot unmarshal into '%s', %s: %s", u.Node.Name, u.Detail, u.wrapping.Error())
	}

	return fmt.Sprintf("cannot unmarshal into '%s', %s", u.Node.Name, u.Detail)
}

func (u *UnmarshalError) Unwrap() error {
	return u.wrapping
}

// node will place contents of the tadl node inside the given value.
func (u *unmarshaler) node(node *parser.TreeNode, value reflect.Value) error {
	valueType := value.Type()

	switch value.Kind() {
	case reflect.String:
		text, err := getTextChild(node)
		if err != nil {
			return NewUnmarshalError(node, "expected string", err)
		}

		value.SetString(text)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		text, err := getTextChild(node)
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("integer required for '%s'", valueType.Name()), err)
		}

		i, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("'%s' is not a valid integer", text), err)
		}

		if value.OverflowInt(i) {
			return NewUnmarshalError(node, fmt.Sprintf("value for '%s' out of bounds", valueType.Name()), err)
		}

		value.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		text, err := getTextChild(node)
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("unsigned integer required for '%s'", valueType.Name()), err)
		}

		i, err := strconv.ParseUint(strings.TrimSpace(text), 10, 64)
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("'%s' is not a valid unsigned integer", text), err)
		}

		if value.OverflowUint(i) {
			return NewUnmarshalError(node, fmt.Sprintf("value for '%s' out of bounds", valueType.Name()), err)
		}

		value.SetUint(i)
	case reflect.Bool:
		text, err := getTextChild(node)
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("boolean required for '%s'", valueType.Name()), err)
		}

		b, err := strconv.ParseBool(strings.TrimSpace(text))
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("'%s' is not a valid boolean", text), err)
		}

		value.SetBool(b)
	case reflect.Float64, reflect.Float32:
		text, err := getTextChild(node)
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("float required for '%s'", valueType.Name()), err)
		}

		var bitSize int

		switch value.Kind() {
		case reflect.Float32:
			bitSize = 32
		case reflect.Float64:
			bitSize = 64
		}

		f, err := strconv.ParseFloat(strings.TrimSpace(text), bitSize)
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("'%s' is not a valid float", text), err)
		}

		value.SetFloat(f)
	case reflect.Ptr:
		// Dereference pointer
		return u.node(node, value.Elem())
	case reflect.Slice:
		// Create, process and append children
		elementType := value.Type().Elem()
		for _, child := range node.Children {
			element := reflect.New(elementType).Elem()
			if err := u.node(child, element); err != nil {
				return NewUnmarshalError(node, fmt.Sprintf("cannot read slice children for '%s'", node.Name), err)
			}

			value.Set(reflect.Append(value, element))
		}
	case reflect.Array:
		return NewUnmarshalError(node, "arrays not supported, use a slice instead", nil)
	case reflect.Struct:
		// Iterate over all struct fields.
		for i := 0; i < value.NumField(); i++ {
			fieldType := value.Type().Field(i)
			field := value.Field(i)

			fieldName := fieldType.Name
			unmarshalAs := unmarshalNormal

			// Some tags will change the behavior of how this field will be processed.
			if structTag, ok := fieldType.Tag.Lookup("tadl"); ok {
				tags := strings.Split(structTag, ",")

				// The first tag will rename the field
				if len(tags) > 0 {
					rename := tags[0]
					if len(rename) > 0 {
						fieldName = rename
					}
				}

				// The second tag indicates the type we are parsing
				if len(tags) > 1 {
					as := tags[1]
					switch as {
					case "attr":
						unmarshalAs = unmarshalAttribute
					case "text":
						unmarshalAs = unmarshalText
					case "":
						unmarshalAs = unmarshalNormal
					default:
						return NewUnmarshalError(node, fmt.Sprintf("field type '%s' invalid", as), nil)
					}
				}
			}

			switch unmarshalAs {
			case unmarshalNormal:
				nodeForField, err := u.findSingleChild(node, fieldName)
				if err != nil {
					return err
				}

				if nodeForField == nil {
					continue
				}

				err = u.node(nodeForField, field)
				if err != nil {
					return NewUnmarshalError(node, fmt.Sprintf("while processing field '%s'", fieldType.Name), err)
				}
			case unmarshalAttribute:
				if node.Attributes.Has(fieldName) {
					// We have everything ready to set the attribute.
					// We want to handle integers and strings easily so we recurse here by creating a fake node.
					// As this node is a string, it can *only* be parsed as a primitive type, everything else
					// will return an error, just like we want.
					fakeNode := parser.NewStringNode(node.Attributes[fieldName])

					err := u.node(fakeNode, field)
					if err != nil {
						// We throw away the error, as it was created with a fake node containing useless information.
						return NewUnmarshalError(node, fmt.Sprintf("attribute '%s' requires primitve type", fieldName), nil)
					}
				} else if u.strict {
					return NewUnmarshalError(node, fmt.Sprintf("attribute '%s' required", fieldName), nil)
				}
			case unmarshalText:
				// Text needs a string field to get parsed into. We then collect any text inside this node or
				// expect exactly one text in strict mode.
				if field.Kind() != reflect.String {
					return NewUnmarshalError(node, fmt.Sprintf("'%s' needs to have type string", fieldType.Name), nil)
				}

				foundAny := false

				var text strings.Builder

				for _, c := range node.Children {
					if c.IsText() {
						if foundAny && u.strict {
							return NewUnmarshalError(node, "multiple occurrences of text, where only one is allowed", nil)
						}

						foundAny = true

						text.WriteString(*c.Text)
					}
				}

				if u.strict && !foundAny {
					return NewUnmarshalError(node, "text inside element required", nil)
				}

				field.SetString(text.String())
			default:
				// Should never happen. We provide a helpful message just in case.
				return fmt.Errorf("unmarshal in invalid state: unmarshalType=%v", unmarshalAs)
			}
		}
	default:
		return NewUnmarshalError(node, fmt.Sprintf("with unsupported type '%s' for '%s'", valueType, valueType.Name()), nil)
	}

	return nil
}

// findSingleChild returns the child with the given name or an error in strict mode when there is no
// such child or there are multiple children.
// In non-strict mode this method might return (nil, nil) which means that no such child exists, or it will
// return the first item with that name.
func (u *unmarshaler) findSingleChild(node *parser.TreeNode, name string) (*parser.TreeNode, error) {
	var child *parser.TreeNode

	for _, c := range node.Children {
		if c.Name == name {
			if child == nil {
				child = c

				if !u.strict {
					// We found a child and don't care if there are other ones in non-strict mode.
					break
				}
			} else {
				return nil, NewUnmarshalError(node, fmt.Sprintf("'%s' defined multiple times", name), nil)
			}
		}
	}

	if u.strict && child == nil {
		return nil, NewUnmarshalError(node, fmt.Sprintf("child '%s' required", name), nil)
	}

	return child, nil
}

// getTextChild will return a string from the CharData that is the child of the given node.
// If node has more than 1 children this will return an error.
// If the single child is not text, this will return an error.
// If node itself is a text, its text will be returned instead.
func getTextChild(node *parser.TreeNode) (string, error) {
	if node.IsText() {
		return *node.Text, nil
	}

	if len(node.Children) != 1 {
		return "", NewUnmarshalError(node, "exactly one text child required", nil)
	}

	textChild := node.Children[0]
	if !textChild.IsText() {
		return "", NewUnmarshalError(node, "child is not text", nil)
	}

	return *textChild.Text, nil
}
