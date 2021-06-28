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

	struc, err := prepareValue(into)
	if err != nil {
		return err
	}

	if err := unmarshalNode(tree, *struc, strict); err != nil {
		return err
	}

	return nil
}

// unmarshalNode will place contents of the tadl node inside the given struct.
// struc needs to be a struct, otherwise this method might panic.
func unmarshalNode(node *parser.TreeNode, struc reflect.Value, strict bool) error {
	for i := 0; i < struc.NumField(); i++ {
		fieldType := struc.Type().Field(i)
		field := struc.Field(i)

		nodeChildren := findChildrenByName(node, fieldType.Name)

		if len(nodeChildren) < 1 {
			if strict {
				return fmt.Errorf("'%s' not defined", fieldType.Name)
			}

			continue
		} else if len(nodeChildren) > 1 && strict {
			return fmt.Errorf("'%s' defined multiple times", fieldType.Name)
		}

		nodeForField := nodeChildren[0]

		switch field.Kind() {
		case reflect.String:
			text, err := getTextChild(nodeForField)
			if err != nil {
				return fmt.Errorf("node for '%s' needs to have a text child", fieldType.Name)
			}

			field.SetString(text)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			text, err := getTextChild(nodeForField)
			if err != nil {
				return fmt.Errorf("node for '%s' needs to have a text child", fieldType.Name)
			}

			i, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
			if err != nil {
				return fmt.Errorf("'%s' is not a valid integer", text)
			}

			if field.OverflowInt(i) {
				return fmt.Errorf("value for '%s' out of bounds", fieldType.Name)
			}

			field.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			text, err := getTextChild(nodeForField)
			if err != nil {
				return fmt.Errorf("node for '%s' needs to have a text child", fieldType.Name)
			}

			i, err := strconv.ParseUint(strings.TrimSpace(text), 10, 64)
			if err != nil {
				return fmt.Errorf("'%s' is not a valid integer", text)
			}

			if field.OverflowUint(i) {
				return fmt.Errorf("value for '%s' out of bounds", fieldType.Name)
			}

			field.SetUint(i)
		case reflect.Ptr:
			return fmt.Errorf("pointer not supported yet")
		case reflect.Array, reflect.Slice:
			return fmt.Errorf("arrays not supported yet")
		case reflect.Struct:
			err := unmarshalNode(nodeForField, field, strict)
			if err != nil {
				return fmt.Errorf("error in '%s': %w", fieldType.Name, err)
			}
		default:
			return fmt.Errorf("cannot unmarshal into '%s' with unsupported type '%v'", fieldType.Name, fieldType.Type)
		}
	}

	return nil
}

// prepareValue accepts a struct or a struct behind any amount of pointers and returns
// a reflect.Value that contains a struct. This returns an error if the given value
// was not a struct.
func prepareValue(v interface{}) (*reflect.Value, error) {
	value := reflect.ValueOf(v)

	for {
		switch value.Kind() {
		case reflect.Struct:
			return &value, nil
		case reflect.Ptr:
			if value.IsNil() {
				return nil, fmt.Errorf("cannot unmarshal into nil")
			}

			value = value.Elem()
		default:
			return nil, fmt.Errorf("can only unmarshal into struct")
		}
	}
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
		return "", fmt.Errorf("only one child supported")
	}

	textChild := node.Children[0]
	if !textChild.IsText() {
		return "", fmt.Errorf("child is not text")
	}

	return *textChild.Text, nil
}
