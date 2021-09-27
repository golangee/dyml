package streamxmlencoder

import (
	"bufio"
	"errors"
	"io"
	"strings"

	"github.com/golangee/tadl/parser"
	"github.com/golangee/tadl/token"
)

const (
	whitespace = " "
	exclammark = "!"
	dquotes    = "\""
	slash      = "/"
	hyphen     = "-"
	lt         = "<"
	equals     = "="
	gt         = ">"
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

func escapeDoubleQuotesChar(c *token.CharData) *token.CharData {
	return &token.CharData{
		Position: c.Position,
		Value:    escapeDoubleQuotes(c.Value),
	}
}

// Node defines a Node representing a to-be-encoded Element of the tadl-text.
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

// Encoder translates tadl-input to corresponding XML
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

	// g2Comments contains all comments in G2 that were eaten from the input,
	// but are not yet placed in a sensible position.
	g2Comments Stack

	// globalForward indicates the current forward mode
	// if false, all incoming calls mutate the main stack
	// if true, they mutate the forward-stack
	globalForward bool
}

// NewEncoder creades a new XMLEncoder
// tadl-input is given as an io.Reader instance
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
		err := e.writeString(gt)
		if err != nil {
			return false, err
		}
		e.stack.SetOpened()
		return true, nil
	}
	return false, nil
}

// writeString writes the given string to the encoders io.Writer.
func (e *Encoder) writeString(in ...string) error {
	if e.globalForward {
		for _, text := range in {
			if _, err := e.forwardBuilder.Write([]byte(text)); err != nil {
				return err
			}
		}
		return nil
	}
	for _, text := range in {
		if _, err := e.buffWriter.Write([]byte(text)); err != nil {
			return err
		}
	}
	return nil
}

// writeForwardToWriter writes the contents of the forward encoding-buffer to the overlying io.Writer
func (e *Encoder) writeForwardToWriter() error {
	err := e.writeString(e.forwardBuilder.String())
	if err != nil {
		return err
	}
	e.forwardBuilder.Reset()
	return nil
}

// Encode starts the encoding of tadl-text to XML
// encoded text will be written to the encoders io.Writer
func (e *Encoder) Encode() error {
	err := e.visitor.Run()
	if err != nil {
		return err
	}

	err = e.writeString(lt, slash, "root", gt)
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
		err = e.writeString(gt)
		if err != nil {
			return err
		}
		e.stack.SetOpened()
	}
	node, err := e.stack.Pop()
	if err != nil {
		return err
	}
	err = e.writeString(lt, slash, escapeDoubleQuotes(node.name), gt)
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

	err = e.writeString(lt, name)
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

		err = e.writeString(escapeDoubleQuotes(cd.Value))
		if err != nil {
			return err
		}
	}
	return nil
}

// NewCommentNode creates a new Node with Text as Comment, based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (e *Encoder) NewCommentNode(cd *token.CharData) error {
	if e.globalForward {
		e.stack.Push(NewNode(cd.Value))
		return nil
	}
	_, err := e.openOptional()
	if err != nil {
		return err
	}
	err = e.writeString(lt, exclammark, hyphen, hyphen, whitespace, escapeDoubleQuotes(cd.Value), whitespace, hyphen, hyphen, gt)
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
	err := e.writeString(whitespace, key, equals, dquotes, escapeDoubleQuotes(value), dquotes)
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
		len := e.forwardedAttributes.Len()
		for i := 0; i < len; i++ {
			key, value := e.forwardedAttributes.Get(i)
			err := e.writeString(whitespace, *key, equals, dquotes, *value, dquotes)
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
		len := e.forwardedAttributes.Len()
		for i := 0; i < len; i++ {
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

// G2AppendComments will append all comments that were parsed with g2EatComments as children
// into the given node.
func (e *Encoder) G2AppendComments() error {
	for _, comment := range e.g2Comments {
		err := e.writeString(lt, exclammark, hyphen, hyphen, whitespace, comment.name, whitespace, hyphen, hyphen, gt)
		if err != nil {
			return err
		}
	}
	return nil
}

// G2AddComments adds a new Comment Node based on given CharData to the g2Comments List,
// to be added to the tree later
func (e *Encoder) G2AddComments(cd *token.CharData) error {
	e.g2Comments = append(e.g2Comments, NewNode(string(escapeDoubleQuotesChar(cd).Value)))
	return nil
}

// SwitchActiveTree switches the active Tree between the main syntax tree and the forwarding tree
// To modify the forwarding tree, call SwitchActiveTree, call treeCreation functions, call SwitchActiveTree
func (e *Encoder) SwitchActiveTree() error {
	cache := e.stack
	e.stack = e.forward
	e.forward = cache

	e.globalForward = !e.globalForward
	return nil
}

// NewStringNode creates a Node with Text and adds it as a child to the current parent Node
// Opens the new Node, used for testing purposes only
func (e *Encoder) NewStringNode(text string) error {
	err := e.writeString(text)
	if err != nil {
		return err
	}
	return nil
}

// NewStringCommentNode creates a new Node with Text as Comment, based on string and adds it as a child to the current parent Node
// Opens the new Node, used for testing purposes only
func (e *Encoder) NewStringCommentNode(text string) error {
	err := e.writeString(lt, exclammark, hyphen, hyphen, whitespace, text, whitespace, hyphen, hyphen, gt)
	if err != nil {
		return err
	}
	return nil
}

// GetGlobalForward returns the globalForward flag
// true: forwarding mode, all new nodes/Attributes are added to the forwarding stack to be added later
func (e *Encoder) GetGlobalForward() (bool, error) {
	return e.globalForward, nil
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
