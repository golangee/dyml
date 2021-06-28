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
// TODO Better error handling
// TODO Nice mechanism for attributes
func Unmarshal(r io.Reader, into interface{}, strict bool) error {
	parse := parser.NewParser("", r)

	tree, err := parse.Parse()
	if err != nil {
		return fmt.Errorf("cannot unmarshal because of parser error: %w", err)
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

// node will place contents of the tadl node inside the given value.
func (u *unmarshaler) node(node *parser.TreeNode, value reflect.Value) error {
	valueType := value.Type()

	switch value.Kind() {
	case reflect.String:
		text, err := getTextChild(node)
		if err != nil {
			return err
		}
		value.SetString(text)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		text, err := getTextChild(node)
		if err != nil {
			return fmt.Errorf("integer required for type '%s'", valueType.Name())
		}

		i, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			return fmt.Errorf("'%s' is not a valid integer", text)
		}

		if value.OverflowInt(i) {
			return fmt.Errorf("value for '%s' out of bounds", valueType.Name())
		}

		value.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		text, err := getTextChild(node)
		if err != nil {
			return fmt.Errorf("unsigned integer required for type '%s'", valueType.Name())
		}

		i, err := strconv.ParseUint(strings.TrimSpace(text), 10, 64)
		if err != nil {
			return fmt.Errorf("'%s' is not a valid unsigned integer", text)
		}

		if value.OverflowUint(i) {
			return fmt.Errorf("value for '%s' out of bounds", valueType.Name())
		}

		value.SetUint(i)
	case reflect.Ptr:
		// Dereference pointer
		return u.node(node, value.Elem())
	case reflect.Array, reflect.Slice:
		return fmt.Errorf("arrays not supported yet")
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			fieldType := value.Type().Field(i)
			field := value.Field(i)

			nodeChildren := findChildrenByName(node, fieldType.Name)

			if len(nodeChildren) < 1 {
				if u.strict {
					return fmt.Errorf("'%s' not defined", fieldType.Name)
				}

				continue
			} else if len(nodeChildren) > 1 && u.strict {
				return fmt.Errorf("'%s' defined multiple times", fieldType.Name)
			}

			nodeForField := nodeChildren[0]

			err := u.node(nodeForField, field)
			if err != nil {
				return fmt.Errorf("error in '%s': %w", fieldType.Name, err)
			}
		}
	default:
		return fmt.Errorf("cannot unmarshal into '%s' with unsupported type '%v'", valueType.Name(), valueType)
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
func getTextChild(node *parser.TreeNode) (string, error) {
	if len(node.Children) != 1 {
		return "", fmt.Errorf("exactly one text child required")
	}

	textChild := node.Children[0]
	if !textChild.IsText() {
		return "", fmt.Errorf("child is not text")
	}

	return *textChild.Text, nil
}
