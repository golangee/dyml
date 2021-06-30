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
// Strict mode requires that all fields of the struct are set and defined only once.
// TODO Nice mechanism for attributes
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

// UnmarshalError is an error that occurred during unmarshaling.
// It contains the offending node, a string with details and an underlying error (if any).
type UnmarshalError struct {
	node     *parser.TreeNode
	detail   string
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
		return fmt.Sprintf("cannot unmarshal into '%s', %s: %s", u.node.Name, u.detail, u.wrapping.Error())
	}

	return fmt.Sprintf("cannot unmarshal into '%s', %s", u.node.Name, u.detail)
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
			return NewUnmarshalError(node, fmt.Sprintf("integer required for type '%s'", valueType.Name()), err)
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
			return NewUnmarshalError(node, fmt.Sprintf("unsigned integer required for type '%s'", valueType.Name()), err)
		}

		i, err := strconv.ParseUint(strings.TrimSpace(text), 10, 64)
		if err != nil {
			return NewUnmarshalError(node, fmt.Sprintf("'%s' is not a valid unsigned integer", text), err)
		}

		if value.OverflowUint(i) {
			return NewUnmarshalError(node, fmt.Sprintf("value for '%s' out of bounds", valueType.Name()), err)
		}

		value.SetUint(i)
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

			// Some tags will change the behavior of how this field will be processed.
			if structTag, ok := fieldType.Tag.Lookup("tadl"); ok {
				tags := strings.Split(structTag, ",")

				// The first tag will rename the field
				if len(tags) > 0 {
					rename := tags[0]
					if len(rename) > 0 {
						fieldName = tags[0]
					}
				}
			}

			// There might be several children with a matching name inside the node.
			// In strict mode exactly one is required, otherwise more than one is okay.
			nodeChildren := findChildrenByName(node, fieldName)

			if len(nodeChildren) < 1 {
				if u.strict {
					return NewUnmarshalError(node, fmt.Sprintf("child '%s' required", fieldName), nil)
				}

				continue
			} else if len(nodeChildren) > 1 && u.strict {
				return NewUnmarshalError(node, fmt.Sprintf("'%s' defined multiple times", fieldName), nil)
			}

			nodeForField := nodeChildren[0]

			err := u.node(nodeForField, field)
			if err != nil {
				return NewUnmarshalError(node, fmt.Sprintf("while processing field '%s'", fieldType.Name), err)
			}
		}
	default:
		return NewUnmarshalError(node, fmt.Sprintf("with unsupported type '%s' for '%s'", valueType, valueType.Name()), nil)
	}

	return nil
}

// findChildrenByName returns all direct children of the given node that have
// the given name.
func findChildrenByName(node *parser.TreeNode, name string) []*parser.TreeNode {
	var result []*parser.TreeNode

	for _, child := range node.Children {
		if child.Name == name {
			result = append(result, child)
		}
	}

	return result
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
