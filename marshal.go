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
// 'inner' can be used to parse elements that are the contents of the surrounding element.
// Consider this example to parse plain text without surrounding elements:
//
//  // This tadl snippet...
//  #! {
//      "hello"
//      "more text"
//  }
//  // could be unmarshalled into this go struct.
//  type Example struct {
//      Something string `tadl:",inner"`
//  }
//
// When collecting text this way all text inside the node will be concatenated in non-strict mode ("hellomore text" in
// the above example). In strict mode exactly one text child is required.
// In the following example inner is used to parse a map-like Tadl definition into a map without a supporting element.
//
//  // This tadl snippet...
//  #! {
//      A "B"
//      C "D"
//  }
//  // could be unmarshalled into this go struct.
//  type Example struct {
//      Something map[string]string `tadl:",inner"`
//  }
//
//
// Tadl can unmarshal into maps. The map key must be a primitive type. The map value must be a primitive
// type, parser.TreeNode or *parser.TreeNode.
// Parsing maps will read first level elements as map keys and the first child of each as the map value.
// In strict mode the map key is required to have exactly one child.
// By specifying parser.TreeNode (or a pointer to it) as the value type you can access the raw tree that would be
// parsed as a value. This is useful if you want to have more control over the value for doing more complex
// manipulations than just parsing a primitive.
//
//  // This tadl snippet...
//  #! {
//      SomeMap {
//          a 123,  // Numbers are valid identifiers, so this works
//          b "1.5" // but all other values should be enclosed in quotes.
//      }
//  }
//  // could be unmarshalled into this go struct.
//  type Example struct {
//      SomeMap map[string]float64
//  }
//
// Tadl also supports unmarshalling slices. When no tag is specified in the struct, elements in Tadl
// are unmarshalled into the slice directly. Should you specify a tag on the field in your struct,
// then only elements with that tag will be parsed. See the examples for more details.
//
func Unmarshal(r io.Reader, into interface{}, strict bool) error {
	parse := parser.NewParser("", r)

	if into == nil {
		return fmt.Errorf("cannot unmarshal into nil")
	}

	tree, err := parse.Parse()
	if err != nil {
		return err
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
	unmarshalInner
)

// unmarshalMapValue is a helper to decide what kind of map value should be unmarshalled.
type unmarshalMapValue int

const (
	mapValueIsPrimitive unmarshalMapValue = iota
	mapValueIsNode
	mapValueIsNodePointer
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
// tags are any field tags that may be relevant to process the current node.
func (u *unmarshaler) node(node *parser.TreeNode, value reflect.Value, tags ...string) error {
	valueType := value.Type()

	switch value.Kind() {
	case reflect.String:
		text, err := u.findText(node)
		if err != nil {
			return NewUnmarshalError(node, "expected string", err)
		}

		value.SetString(text)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		text, err := getAsText(node)
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
		text, err := getAsText(node)
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
		text, err := getAsText(node)
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("boolean required for '%s'", valueType.Name()), err)
		}

		b, err := strconv.ParseBool(strings.TrimSpace(text))
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("'%s' is not a valid boolean", text), err)
		}

		value.SetBool(b)
	case reflect.Float64, reflect.Float32:
		text, err := getAsText(node)
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
		// Create value for nil pointer
		if value.IsNil() {
			v := reflect.New(valueType.Elem())
			value.Set(v)
		}
		// Dereference pointer
		return u.node(node, value.Elem())
	case reflect.Map:
		mapKeyType := valueType.Key()
		mapValueType := valueType.Elem()

		// Maps must have primitive key.
		if !u.isPrimitive(mapKeyType) {
			return NewUnmarshalError(node, fmt.Sprintf("map key type '%s' is not primitive", mapKeyType.String()), nil)
		}

		// Map value must be primitive or a (pointer to) parser.TreeNode
		var valueMode unmarshalMapValue
		if u.isPrimitive(mapValueType) {
			valueMode = mapValueIsPrimitive
		} else if mapValueType == reflect.TypeOf(parser.TreeNode{}) {
			valueMode = mapValueIsNode
		} else if mapValueType == reflect.TypeOf(&parser.TreeNode{}) {
			valueMode = mapValueIsNodePointer
		} else {
			return NewUnmarshalError(node, "map value must be primitive type or (*)parser.TreeNode", nil)
		}

		value.Set(reflect.MakeMap(valueType))
		// A map will parse first level children as the key and the first child of those as the value.
		for _, keyNode := range nonCommentChildren(node) {
			if !keyNode.IsNode() {
				if u.strict {
					return NewUnmarshalError(node, "map key must be a node", nil)
				} else {
					continue
				}
			}

			// Make mapKey be a zero value of the maps key type
			mapKey := reflect.New(mapKeyType).Elem()

			// In order to recursively use u.node() to parse values, we will forge a fake text node here
			// and use that to recurse. We use this trick to parse both the key and the value.
			fakeNode := parser.NewStringNode(keyNode.Name)
			if err := u.node(fakeNode, mapKey); err != nil {
				return NewUnmarshalError(node, "invalid map key", err)
			}

			// Now that we parsed the key we continue with parsing the value
			keyNodeChildren := nonCommentChildren(keyNode)
			if len(keyNodeChildren) == 0 {
				return NewUnmarshalError(node, fmt.Sprintf("no value in map for key '%v'", mapKey), nil)
			} else if u.strict && len(keyNodeChildren) != 1 {
				return NewUnmarshalError(node, fmt.Sprintf("key '%v' needs exactly one value", mapKey), nil)
			}

			valueNode := keyNodeChildren[0]

			// Make mapValue be a zero value of the maps value type
			mapValue := reflect.New(mapValueType).Elem()

			switch valueMode {
			case mapValueIsNodePointer:
				mapValue = reflect.ValueOf(keyNode)
			case mapValueIsNode:
				mapValue = reflect.ValueOf(*keyNode)
			case mapValueIsPrimitive:
				if u.strict && len(nonCommentChildren(valueNode)) > 0 {
					return NewUnmarshalError(node, fmt.Sprintf("value for key '%v' must have no children", mapKey), nil)
				}

				var primitiveValueToParse string

				if valueNode.IsNode() {
					primitiveValueToParse = valueNode.Name
				} else if valueNode.IsText() {
					primitiveValueToParse = *valueNode.Text
				} else {
					return NewUnmarshalError(node, fmt.Sprintf("value for key '%v' must be node or text", mapKey), nil)
				}

				fakeNode := parser.NewStringNode(primitiveValueToParse)
				if err := u.node(fakeNode, mapValue); err != nil {
					return NewUnmarshalError(node, "value is incompatible with map type", err)
				}

			default:
				return NewUnmarshalError(node, fmt.Sprintf("unmarshal has invalid map value mode (%d). this is a bug", valueMode), nil)
			}

			value.SetMapIndex(mapKey, mapValue)
		}
	case reflect.Slice:
		// Figure out type for elements. Should this be a slice we want to know what type is stored in it.
		elementType := valueType.Elem()
		if elementType.Kind() == reflect.Slice {
			elementType = elementType.Elem()
		}

		// Create, process and append children
		for _, child := range nonCommentChildren(node) {
			if len(tags) > 0 {
				// Use rename tag to filter for slice elements with the given name.
				if child.Name != tags[0] {
					continue
				}
			}

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

			var tags []string

			// Some tags will change the behavior of how this field will be processed.
			if structTag, ok := fieldType.Tag.Lookup("tadl"); ok {
				tags = strings.Split(structTag, ",")

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
					case "inner":
						unmarshalAs = unmarshalInner
					case "":
						unmarshalAs = unmarshalNormal
					default:
						return NewUnmarshalError(node, fmt.Sprintf("field type '%s' invalid", as), nil)
					}
				}
			}

			switch unmarshalAs {
			case unmarshalNormal:
				// Should the field be a slice and a rename param is set, then we need to pass the whole node in,
				// not just a subnode, to allow for filtering of elements.
				if field.Kind() == reflect.Slice && len(tags) > 0 && len(tags[0]) > 0 {
					if err := u.node(node, field, tags...); err != nil {
						return err
					}
				} else {
					nodeForField, err := u.findSingleChild(node, fieldName)
					if err != nil {
						return err
					}

					if nodeForField == nil {
						continue
					}

					err = u.node(nodeForField, field, tags...)
					if err != nil {
						return NewUnmarshalError(node, fmt.Sprintf("while processing field '%s'", fieldType.Name), err)
					}
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
			case unmarshalInner:
				if err := u.node(node, field); err != nil {
					return NewUnmarshalError(node, "'inner' struct tag caused an error", err)
				}
			default:
				// Should never happen. We provide a helpful message just in case.
				return fmt.Errorf("unmarshal in invalid state: unmarshalType=%v. this is a bug", unmarshalAs)
			}
		}
	default:
		return NewUnmarshalError(node, fmt.Sprintf("with unsupported type '%s' for '%s'", valueType, valueType.Name()), nil)
	}

	return nil
}

// isPrimitive returns true if the given type is a primitive one.
func (u *unmarshaler) isPrimitive(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Bool, reflect.Float32, reflect.Float64, reflect.String:
		return true
	}

	return false
}

// nonCommentChildren returns all children of the given node that are not comments.
func nonCommentChildren(node *parser.TreeNode) []*parser.TreeNode {
	var result []*parser.TreeNode

	for _, child := range node.Children {
		if !child.IsComment() {
			result = append(result, child)
		}
	}

	return result
}

// findSingleChild returns the child with the given name or an error in strict mode when there is no
// such child or there are multiple children.
// In non-strict mode this method might return (nil, nil) which means that no such child exists, or it will
// return the first item with that name.
func (u *unmarshaler) findSingleChild(node *parser.TreeNode, name string) (*parser.TreeNode, error) {
	var child *parser.TreeNode

	for _, c := range nonCommentChildren(node) {
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

// findText will find text inside the children of the given node or will return the text of a text node directly.
// In strict mode exactly one text child is required.
// In non-strict mode all text children will be concatenated. This might then return an empty string
// if there are no text children.
func (u *unmarshaler) findText(node *parser.TreeNode) (string, error) {
	if node.IsText() {
		return *node.Text, nil
	}

	foundAny := false

	var text strings.Builder

	for _, c := range nonCommentChildren(node) {
		if c.IsText() {
			if foundAny && u.strict {
				return "", NewUnmarshalError(node, "multiple occurrences of text, where only one is allowed", nil)
			}

			foundAny = true

			text.WriteString(*c.Text)
		}
	}

	if u.strict && !foundAny {
		return "", NewUnmarshalError(node, "text inside element required", nil)
	}

	return text.String(), nil
}

// getAsText will return a string from the given node.
// This can either be from the node itself should it either:
//  - Be text, in which case that is returned
//  - Be a node with no children, in which case the node's name is returned.
// If node has exactly one child then the text from that will be returned according
// to the same rules.
func getAsText(node *parser.TreeNode) (string, error) {
	if node.IsText() {
		return *node.Text, nil
	}

	children := nonCommentChildren(node)

	if node.IsNode() {
		if len(children) == 0 {
			return node.Name, nil
		}

		if len(children) == 1 {
			child := children[0]

			if child.IsText() {
				return *child.Text, nil
			}

			if child.IsNode() {
				if len(nonCommentChildren(child)) == 0 {
					return child.Name, nil
				}

				return "", NewUnmarshalError(node, "child must not have children", nil)
			}

			return "", NewUnmarshalError(node, "child is not text", nil)
		}

		return "", NewUnmarshalError(node, "more than one child found", nil)
	}

	return "", NewUnmarshalError(node, "must be node or text-node", nil)
}
