// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package dyml

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/golangee/dyml/token"

	"github.com/golangee/dyml/parser"
)

// Unmarshaler can be implemented on a struct to define custom unmarshalling behavior.
type Unmarshaler interface {
	UnmarshalDyml(node *parser.TreeNode) error
}

// Unmarshal takes dyml input and parses it into the given struct.
// If "into" is not a struct or a pointer to a struct, this method will panic.
// As this uses go's reflect package, only exported names can be unmarshalled.
// Strict mode requires that all fields of the struct are set and defined exactly once.
// You can set struct tags to influence the unmarshalling process.
// All tags must have the form `dyml:"..."` and are a list of comma separated identifiers.
//
// The first identifier can be used to rename the field, so that an element with the renamed
// name is parsed, and not the name of the struct field.
//
//  // This dyml snippet...
//  #!{item{...}}
//  // could be unmarshalled into this go struct.
//  type Example struct {
//      SomeName Content `dyml:"item"`
//  }
//
// The second identifier is used to specify what kind of thing is being parsed.
// This can be used to parse attributes (attr) or text (text).
//
// Attributes can be parsed into primitive types: string, bool and the integer (signed & unsigned) and float types.
// Should the value not be valid for the target type, e.g. an integer that is too large or a negative value for an uint,
// an error is returned describing the issue.
//
//  // This dyml snippet...
//  #item @key{value} @X{123}
//  // could be unmarshalled into this go struct.
//  type Example struct {
//      SomeName string `dyml:"key,attr"` // Notice how you can rename an attribute
//      X        int    `dyml:",attr"` // You can choose to not rename it, by omitting the rename parameter.
//  }
//
// 'inner' can be used to parse elements that are the contents of the surrounding element.
// Consider this example to parse plain text without surrounding elements:
//
//  // This dyml snippet...
//  #! {
//      "hello"
//      "more text"
//  }
//  // could be unmarshalled into this go struct.
//  type Example struct {
//      Something string `dyml:",inner"`
//  }
//
// When collecting text this way all text inside the node will be concatenated in non-strict mode ("hellomore text" in
// the above example). In strict mode exactly one text child is required.
// In the following example inner is used to parse a map-like Dyml definition into a map without a supporting element.
//
//  // This dyml snippet...
//  #! {
//      A "B"
//      C "D"
//  }
//  // could be unmarshalled into this go struct.
//  type Example struct {
//      Something map[string]string `dyml:",inner"`
//  }
//
//
// dyml can unmarshal into maps. The map key must be a primitive type. The map value must be a primitive
// type, parser.TreeNode or *parser.TreeNode.
// Parsing maps will read first level elements as map keys and the first child of each as the map value.
// In strict mode the map key is required to have exactly one child.
// By specifying parser.TreeNode (or a pointer to it) as the value type you can access the raw tree that would be
// parsed as a value. This is useful if you want to have more control over the value for doing more complex
// manipulations than just parsing a primitive.
//
//  // This dyml snippet...
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
// dyml also supports unmarshalling slices. When no tag is specified in the struct, elements in dyml
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

	if err := unmarshal.doAny(tree, value); err != nil {
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
	mapValueIsCustomType
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

// doAny will parse arbitrary contents of the dyml node into the given value.
// tags are any field tags that may be relevant to process the current node.
func (u *unmarshaler) doAny(node *parser.TreeNode, value reflect.Value, tags ...string) error {
	// Check for custom unmarshalling method.
	customUnmarshalMethod := value.MethodByName("UnmarshalDyml")

	// Handy zero value for comparison.
	zero := reflect.Value{}

	if customUnmarshalMethod == zero && value.CanAddr() {
		// We got no method because we might have been checking for a receiver method on a by-value-reference.
		// Create a pointer to the value and try to find the method on that.
		valuePtr := value.Addr()
		customUnmarshalMethod = valuePtr.MethodByName("UnmarshalDyml")
	}

	if customUnmarshalMethod != zero {
		params := []reflect.Value{reflect.ValueOf(node)}

		// UnmarshalDyml might return an error.
		errValue := customUnmarshalMethod.Call(params)[0]
		if !errValue.IsNil() {
			return errValue.Interface().(error)
		}

		// We have done custom unmarshalling and don't want the default behavior now
		return nil
	}

	switch value.Kind() {
	case reflect.String:
		err := u.doString(node, value)
		if err != nil {
			return err
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		err := u.doInt(node, value)
		if err != nil {
			return err
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		err := u.doUint(node, value)
		if err != nil {
			return err
		}
	case reflect.Bool:
		err := u.doBool(node, value)
		if err != nil {
			return err
		}
	case reflect.Float64, reflect.Float32:
		err := u.doFloat(node, value)
		if err != nil {
			return err
		}
	case reflect.Ptr:
		return u.doPointer(node, value)
	case reflect.Map:
		err := u.doMap(node, value, tags)
		if err != nil {
			return err
		}
	case reflect.Slice:
		err := u.doSlice(node, value, tags)
		if err != nil {
			return err
		}
	case reflect.Array:
		return NewUnmarshalError(node, "arrays not supported, use a slice instead", nil)
	case reflect.Struct:
		err := u.doStruct(node, value)
		if err != nil {
			return err
		}
	default:
		return NewUnmarshalError(
			node,
			fmt.Sprintf("with unsupported type '%s' for '%s'", value.Type(), value.Type().Name()), nil)
	}

	return nil
}

// doSlice parses the children of the node as a slice into value. tags are needed to infer unmarshalling rules.
func (u *unmarshaler) doSlice(node *parser.TreeNode, value reflect.Value, tags []string) error {
	// Figure out type for elements. Should this be a slice we want to know what type is stored in it.
	elementType := value.Type().Elem()
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
		if err := u.doAny(child, element); err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("cannot read slice children for '%s'", node.Name), err)
		}

		value.Set(reflect.Append(value, element))
	}

	return nil
}

// doMap will parse the node as a map into value. tags are needed to infer unmarshalling rules.
func (u *unmarshaler) doMap(node *parser.TreeNode, value reflect.Value, tags []string) error {
	mapKeyType := value.Type().Key()
	mapValueType := value.Type().Elem()

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
		valueMode = mapValueIsCustomType
	}

	value.Set(reflect.MakeMap(value.Type()))
	// A map will parse first level children as the key and the first child of those as the value.
	for _, keyNode := range nonCommentChildren(node) {
		if !keyNode.IsNode() {
			if u.strict {
				return NewUnmarshalError(node, "map key must be a node", nil)
			}

			continue
		}

		// Make mapKey be a zero value of the maps key type
		mapKey := reflect.New(mapKeyType).Elem()

		// In order to recursively use u.doAny() to parse values, we will forge a fake text node here
		// and use that to recurse. We use this trick to parse both the key and the value.
		fakeNode := parser.NewStringNode(keyNode.Name)
		if err := u.doAny(fakeNode, mapKey); err != nil {
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
		case mapValueIsCustomType:
			if err := u.doAny(keyNode, mapValue, tags...); err != nil {
				return err
			}
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
			if err := u.doAny(fakeNode, mapValue); err != nil {
				return NewUnmarshalError(node, "value is incompatible with map type", err)
			}
		default:
			return NewUnmarshalError(node,
				fmt.Sprintf("unmarshal has invalid map value mode (%d). this is a bug", valueMode), nil)
		}

		value.SetMapIndex(mapKey, mapValue)
	}

	return nil
}

// doPointer will dereference the pointer in value or create a new zero value for it,
// and then parse the node into that.
func (u *unmarshaler) doPointer(node *parser.TreeNode, value reflect.Value) error {
	// Create value for nil pointer
	if value.IsNil() {
		v := reflect.New(value.Type().Elem())
		value.Set(v)
	}
	// Dereference pointer
	return u.doAny(node, value.Elem())
}

// doFloat parses the node as a float into value.
func (u *unmarshaler) doFloat(node *parser.TreeNode, value reflect.Value) error {
	text, err := getAsText(node)
	if err != nil {
		return NewUnmarshalError(node, fmt.Sprintf("float required for '%s'", value.Type().Name()), err)
	}

	var bitSize int

	switch value.Kind() {
	case reflect.Float32:
		bitSize = 32
	case reflect.Float64:
		bitSize = 64
	default:
		return token.NewPosError(node.Range, "you found a bug: trying to get float bit size for "+value.String())
	}

	f, err := strconv.ParseFloat(strings.TrimSpace(text), bitSize)
	if err != nil {
		return NewUnmarshalError(node, fmt.Sprintf("'%s' is not a valid float", text), err)
	}

	value.SetFloat(f)

	return nil
}

// doBool parses the node as a boolean into value.
func (u *unmarshaler) doBool(node *parser.TreeNode, value reflect.Value) error {
	text, err := getAsText(node)
	if err != nil {
		return NewUnmarshalError(node, fmt.Sprintf("boolean required for '%s'", value.Type().Name()), err)
	}

	b, err := strconv.ParseBool(strings.TrimSpace(text))
	if err != nil {
		return NewUnmarshalError(node, fmt.Sprintf("'%s' is not a valid boolean", text), err)
	}

	value.SetBool(b)

	return nil
}

// doUint parses the node as an unsigned integer into value.
func (u *unmarshaler) doUint(node *parser.TreeNode, value reflect.Value) error {
	text, err := getAsText(node)
	if err != nil {
		return NewUnmarshalError(node, fmt.Sprintf("unsigned integer required for '%s'", value.Type().Name()), err)
	}

	i, err := strconv.ParseUint(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return NewUnmarshalError(node, fmt.Sprintf("'%s' is not a valid unsigned integer", text), err)
	}

	if value.OverflowUint(i) {
		return NewUnmarshalError(node, fmt.Sprintf("value for '%s' out of bounds", value.Type().Name()), err)
	}

	value.SetUint(i)

	return nil
}

// doInt parses the node as a signed integer into value.
func (u *unmarshaler) doInt(node *parser.TreeNode, value reflect.Value) error {
	text, err := getAsText(node)
	if err != nil {
		return NewUnmarshalError(node, fmt.Sprintf("integer required for '%s'", value.Type().Name()), err)
	}

	i, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return NewUnmarshalError(node, fmt.Sprintf("'%s' is not a valid integer", text), err)
	}

	if value.OverflowInt(i) {
		return NewUnmarshalError(node, fmt.Sprintf("value for '%s' out of bounds", value.Type().Name()), err)
	}

	value.SetInt(i)

	return nil
}

// doString parses the node as a string into value.
func (u *unmarshaler) doString(node *parser.TreeNode, value reflect.Value) error {
	text, err := u.findText(node)
	if err != nil {
		return NewUnmarshalError(node, "expected string", err)
	}

	value.SetString(text)

	return nil
}

// doStruct parses the node as a struct into value.
func (u *unmarshaler) doStruct(node *parser.TreeNode, value reflect.Value) error {
	// Iterate over all struct fields.
	for i := 0; i < value.NumField(); i++ {
		fieldType := value.Type().Field(i)
		field := value.Field(i)

		fieldName := fieldType.Name
		unmarshalAs := unmarshalNormal

		var tags []string

		// Some tags will change the behavior of how this field will be processed.
		if structTag, ok := fieldType.Tag.Lookup("dyml"); ok {
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
				if err := u.doSlice(node, field, tags); err != nil {
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

				err = u.doAny(nodeForField, field, tags...)
				if err != nil {
					return NewUnmarshalError(node, fmt.Sprintf("while processing field '%s'", fieldType.Name), err)
				}
			}
		case unmarshalAttribute:
			attr := node.Attributes.Get(fieldName)
			if attr != nil {
				// We have everything ready to set the attribute.
				// We want to handle integers and strings easily so we recurse here by creating a fake node.
				// As this node is a string, it can *only* be parsed as a primitive type, everything else
				// will return an error, just like we want.
				fakeNode := parser.NewStringNode(attr.Value)

				err := u.doAny(fakeNode, field)
				if err != nil {
					// We throw away the error, as it was created with a fake node containing useless information.
					return NewUnmarshalError(node, fmt.Sprintf("attribute '%s' requires primitve type", fieldName), nil)
				}
			} else if u.strict {
				return NewUnmarshalError(node, fmt.Sprintf("attribute '%s' required", fieldName), nil)
			}
		case unmarshalInner:
			if err := u.doAny(node, field); err != nil {
				return NewUnmarshalError(node, "'inner' struct tag caused an error", err)
			}
		default:
			// Should never happen. We provide a helpful message just in case.
			return fmt.Errorf("unmarshal in invalid state: unmarshalType=%v. this is a bug", unmarshalAs)
		}
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
	default:
		return false
	}
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
