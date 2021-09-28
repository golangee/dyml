package streamxmlencoder

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/golangee/dyml/parser"
	"github.com/golangee/dyml/token"
)

func escapeDoubleQuotes(in string) string {
	var out strings.Builder
	var last rune
	for _, c := range in {
		if c == '"' {
			if last == '\\' {
				out.WriteString("\\")
			}
			out.WriteString("\"")
		} else {
			out.WriteRune(c)
		}
		last = c
	}
	return out.String()
}

// Node defines a Node representing a to-be-encoded Element of the dyml-text.
type Node struct {
	name       string
	attributes parser.AttributeList

	opened    bool
	blockType parser.BlockType
}

// NewNode creates a new named Node
func NewNode(name string) *Node {
	return &Node{
		name: name,
	}
}

// Stack holds all already encountered Nodes.
// When a node is closed, it's name is reused and the Node is removed from the stack.
type Stack []*Node

// IsEmpty checks if stack is empty
func (s *Stack) IsEmpty() bool {
	return len(*s) == 0
}

// Push a new node onto the stack
func (s *Stack) Push(n *Node) {
	*s = append(*s, n)
}

// Pop a node from the stack
func (s *Stack) Pop() (*Node, error) {
	if s.IsEmpty() {
		return nil, errors.New("stack is empty, cannot pop")
	}
	index := len(*s) - 1
	element := (*s)[index]
	*s = (*s)[:index]
	return element, nil
}

// IsOpened returns true if the last element on the Stack was already opened before
func (s *Stack) IsOpened() (bool, error) {
	if s == nil || len(*s) == 0 {
		return false, errors.New("stack is empty")
	}
	return (*s)[len(*s)-1].opened, nil
}

// SetOpened sets the opened state of the last element on the Stack.
func (s *Stack) SetOpened() {
	(*s)[len(*s)-1].opened = true
}

// Encoder translates dyml-input to corresponding XML
type Encoder struct {
	visitor    parser.Visitor
	buffWriter *bufio.Writer

	//forwardingAttributes contains all Attributes that have been forwarded to be added to the next viable node.
	forwardedAttributes parser.AttributeList

	// stack and forward are stacks, holding the reduced Node type representing
	// Elements of the syntax tree. Stack is the main stack, receiving all non-forwarded Elements.
	// forward receives all forwarded Nodes, merges them on MergeNodesForwarded() calls
	stack, forward Stack

	// forwardBuilder is a StringBuffer, holding encoded Text that needs to be appended
	// to the output text later. Writes all its content to the io.Writer on writeForwardToWriter()
	forwardBuilder strings.Builder

	// isForwarding indicates the current forward mode
	// if false, all incoming calls mutate the main stack
	// if true, they mutate the forward-stack
	isForwarding bool
}

// NewEncoder creates a new XMLEncoder
// dyml-input is given as an io.Reader instance
func NewEncoder(filename string, r io.Reader, w io.Writer, buffsize int) Encoder {
	encoder := Encoder{
		visitor:    *parser.NewVisitor(nil, token.NewLexer(filename, r)),
		buffWriter: bufio.NewWriterSize(w, buffsize),
	}
	encoder.visitor.SetVisitable(&encoder)
	return encoder
}

// OpenOptional checks if the current Node on the Stack is opened.
// opens it and returns true if it is not, returns false otherwise.
func (e *Encoder) openOptional() (bool, error) {
	if e.stack == nil {
		return false, nil
	}
	if opened, err := e.stack.IsOpened(); err == nil && !opened {
		err := e.write(">")
		if err != nil {
			return false, err
		}
		e.stack.SetOpened()
		return true, nil
	}
	return false, nil
}

// write the given string to the encoder's io.Writer.
func (e *Encoder) write(s string) error {
	if e.isForwarding {
		if _, err := e.forwardBuilder.Write([]byte(s)); err != nil {
			return err
		}
		return nil
	} else {
		if _, err := e.buffWriter.Write([]byte(s)); err != nil {
			return err
		}
	}

	return nil
}

// writeForwardToWriter writes the contents of the forward encoding-buffer to the overlying io.Writer
func (e *Encoder) writeForwardToWriter() error {
	err := e.write(e.forwardBuilder.String())
	if err != nil {
		return err
	}
	e.forwardBuilder.Reset()
	return nil
}

// Encode starts the encoding of dyml-text to XML
// encoded text will be written to the encoders io.Writer
func (e *Encoder) Encode() error {
	err := e.visitor.Run()
	if err != nil {
		return err
	}

	err = e.write("</root>")
	if err != nil {
		return err
	}

	err = e.buffWriter.Flush()
	if err != nil {
		return err
	}

	return nil
}

// Close moves the parent pointer to its current parent Node
func (e *Encoder) Close() error {
	if e.stack == nil || len(e.stack) <= 1 {
		return nil
	}
	if opened, err := e.stack.IsOpened(); err == nil && !opened {
		err = e.write(">")
		if err != nil {
			return err
		}
		e.stack.SetOpened()
	}
	node, err := e.stack.Pop()
	if err != nil {
		return err
	}
	err = e.write(fmt.Sprintf("</%s>", escapeDoubleQuotes(node.name)))
	if err != nil {
		return err
	}

	return nil
}

// NewNode creates a named Node and adds it as a child to the current parent Node
// Opens the new Node
func (e *Encoder) NewNode(name string) error {
	_, err := e.openOptional()
	if err != nil {
		return err
	}

	err = e.write(fmt.Sprintf("<%s", name))
	if err != nil {
		return err
	}

	e.stack.Push(NewNode(escapeDoubleQuotes(name)))
	return nil
}

// NewTextNode creates a new Node with Text based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (e *Encoder) NewTextNode(cd *token.CharData) error {
	if !isWhitespaceOnly(cd.Value) {
		_, err := e.openOptional()
		if err != nil {
			return err
		}

		err = e.write(escapeDoubleQuotes(cd.Value))
		if err != nil {
			return err
		}
	}
	return nil
}

// NewCommentNode creates a new Node with Text as Comment, based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (e *Encoder) NewCommentNode(cd *token.CharData) error {
	if e.isForwarding {
		e.stack.Push(NewNode(cd.Value))
		return nil
	}
	_, err := e.openOptional()
	if err != nil {
		return err
	}
	err = e.write(fmt.Sprintf("<!-- %s -->", escapeDoubleQuotes(cd.Value)))
	if err != nil {
		return err
	}
	return nil
}

// SetBlockType adds the Nodes BlockType to the Node.
// Uses the AddAttribute method to encode the BlockType, as both share the same structure of encoding.
// Attribute: [key]="[value]"		BlockType: _groupType="[BlockType]"
func (e *Encoder) SetBlockType(b parser.BlockType) error {
	if len(e.stack) > 1 {
		err := e.AddAttribute("_groupType", string(b))
		if err != nil {
			return err
		}
	}
	e.stack[len(e.stack)-1].blockType = b
	return nil
}

// GetRootBlockType returns the root-nodes block type.
func (e *Encoder) GetRootBlockType() (parser.BlockType, error) {
	return e.stack[0].blockType, nil
}

// GetBlockType returns the current Nodes block type.
func (e *Encoder) GetBlockType() (parser.BlockType, error) {
	return e.stack[len(e.stack)-1].blockType, nil
}

// GetForwardingLength returns the length of the List of forwaring Nodes
func (e *Encoder) GetForwardingLength() (int, error) {
	return len(e.forward), nil
}

// GetForwardingAttributesLength returns the length of the forwarding AttributeMap
func (e *Encoder) GetForwardingAttributesLength() (int, error) {
	return e.forwardedAttributes.Len(), nil
}

// AddAttribute adds a given Attribute to the current parent Node
func (e *Encoder) AddAttribute(key, value string) error {
	//err := e.write(fmt.Sprintf(" ", key, "=\"", escapeDoubleQuotes(value), "\""))
	err := e.write(fmt.Sprintf(` %s="%s"`, key, escapeDoubleQuotes(value)))
	if err != nil {
		return err
	}
	return nil
}

// AddAttributeForward adds a given AttributeMap to the forwaring Attributes
func (e *Encoder) AddAttributeForward(key, value string) error {
	v := escapeDoubleQuotes(value)
	e.forwardedAttributes.Push(&key, &v)
	return nil
}

// AddNodeForward appends a given Node to the list of forwarding Nodes
func (e *Encoder) AddNodeForward(name string) error {
	e.forward.Push(NewNode(escapeDoubleQuotes(name)))
	return nil
}

// MergeAttributes merges the list of forwarded Attributes to the current parent Nodes Attributes
func (e *Encoder) MergeAttributes() error {
	if e.forwardedAttributes.Len() != 0 {
		for i := 0; i < e.forwardedAttributes.Len(); i++ {
			key, value := e.forwardedAttributes.Get(i)
			err := e.write(fmt.Sprintf(`%s="%s"`, *key, escapeDoubleQuotes(*value)))
			if err != nil {
				return err
			}
		}
		e.forwardedAttributes = parser.NewAttributeList()

	}
	return nil
}

// MergeAttributesForwarded adds the buffered forwarding AttributeMap to the latest forwarded Node
func (e *Encoder) MergeAttributesForwarded() error {
	if e.forwardedAttributes.Len() != 0 {
		for i := 0; i < e.forwardedAttributes.Len(); i++ {
			key, value := e.forwardedAttributes.Get(i)
			e.forward[i].attributes.Push(key, value)
		}
		e.forwardedAttributes = parser.NewAttributeList()
	}
	return nil
}

// MergeNodesForwarded appends the current list of forwarding Nodes
// as Children to the current parent Node
func (e *Encoder) MergeNodesForwarded() error {
	if e.forward.IsEmpty() {
		return nil
	}

	err := e.MergeAttributes()
	if err != nil {
		return nil
	}

	_, err = e.openOptional()
	if err != nil {
		return nil
	}

	err = e.writeForwardToWriter()
	if err != nil {
		return nil
	}

	e.forward = nil
	return nil
}

func (e *Encoder) G2AddComment(cd *token.CharData) error {
	err := e.write(fmt.Sprintf("<!-- %s -->", cd.Value))
	if err != nil {
		return err
	}
	return nil
}

// SwitchActiveTree switches the active Tree between the main syntax tree and the forwarding tree
// To modify the forwarding tree, call SwitchActiveTree, call treeCreation functions, call SwitchActiveTree
func (e *Encoder) SwitchActiveTree() error {
	cache := e.stack
	e.stack = e.forward
	e.forward = cache

	e.isForwarding = !e.isForwarding
	return nil
}

// NewStringNode creates a Node with Text and adds it as a child to the current parent Node
// Opens the new Node, used for testing purposes only
func (e *Encoder) NewStringNode(text string) error {
	err := e.write(text)
	if err != nil {
		return err
	}
	return nil
}

// NewStringCommentNode creates a new Node with Text as Comment, based on string and adds it as a child to the current parent Node
// Opens the new Node, used for testing purposes only
func (e *Encoder) NewStringCommentNode(text string) error {
	err := e.write(fmt.Sprintf("<!-- %s -->", text))
	if err != nil {
		return err
	}
	return nil
}

// GetGlobalForward returns the isForwarding flag
// true: forwarding mode, all new nodes/Attributes are added to the forwarding stack to be added later
func (e *Encoder) GetGlobalForward() (bool, error) {
	return e.isForwarding, nil
}

// isWhitespaceOnly checks if the given string consists of whitespaces only
func isWhitespaceOnly(s string) bool {
	for _, char := range s {
		if char != ' ' && char != '\n' && char != '\t' {
			return false
		}
	}
	return true
}
