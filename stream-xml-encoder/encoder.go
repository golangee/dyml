package streamxmlencoder

import (
	"bufio"
	"errors"
	"io"

	"github.com/golangee/tadl/parser"
	"github.com/golangee/tadl/token"
)

//TODO: propagate all errors upwards

// TODO: refactor escaping: incoming strings must be checked for escapable characters
// mainly escape " to \"
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
	var out string
	for _, c := range in {
		if c == '"' {
			out = out + "\""
		} else {
			out = out + string(c)
		}
	}
	return out
}

func escapeDoubleQuotesChar(c *token.CharData) *token.CharData {
	var out string
	for _, c := range c.Value {
		if c == '"' {
			out = out + "\""
		} else {
			out = out + string(c)
		}
	}
	c.Value = out
	return c
}

type Node struct {
	name       string
	attributes parser.AttributeList

	opened bool
}

func NewNode(name string) *Node {
	return &Node{
		name: name,
	}
}

type Stack []Node

// IsEmpty checks if stack is empty
func (s *Stack) IsEmpty() bool {
	return len(*s) == 0
}

// Push a new node onto the stack
func (s *Stack) Push(n Node) {
	*s = append(*s, n)
}

// Pop a node from the stack
func (s *Stack) Pop() (*Node, bool) {
	if s.IsEmpty() {
		return nil, false
	} else {
		index := len(*s) - 1
		element := (*s)[index]
		*s = (*s)[:index]
		return &element, true
	}
}

// IsEmpty returns true if the last element on the Stack was already opened before
func (s *Stack) IsOpened() bool {
	return (*s)[len(*s)-1].opened
}

// SetOpened sets the opened state of the last element on the Stack.
func (s *Stack) SetOpened() {
	(*s)[len(*s)-1].opened = true
}

//TODO: refactor to stack, remove unused fields
/*
// TreeNodeEnc defines the TreeNode structure for encoding tadl input to XML
// inherits from parser.TreeNode the main functionalities,
// adds functionality for writing encoded XML data
type TreeNodeEnc struct {
	Node     *parser.TreeNode
	Parent   *TreeNodeEnc
	Children []*TreeNodeEnc

	written           bool
	attributesWritten bool
	opened            bool
}

// NewNode creates a new named TreeNodeEnc
func NewNode(text string) *Node {
	return &Node{
		name: text,
		opened: false,
	}
}*/

// Encoder translates tadl-input to corresponding XML
type Encoder struct {
	visitor    parser.Visitor
	buffWriter *bufio.Writer

	//forwardingAttributes contains all Attributes that have been forwarded to be added to the next viable node.
	forwardedAttributes parser.AttributeList

	// root and parent are pointers to work with the successively built Tree.
	// root holds the root Node, parent holds the currently to modify Node

	stack   Stack
	forward Stack

	// g2Comments contains all comments in G2 that were eaten from the input,
	// but are not yet placed in a sensible position.
	g2Comments Stack

	buffsize int
}

// NewEncoder creades a new XMLEncoder
// tadl-input is given as an io.Reader instance
func NewEncoder(filename string, r io.Reader, w io.Writer, buffsize int) Encoder {
	encoder := Encoder{
		visitor:    *parser.NewVisitor(nil, token.NewLexer(filename, r)),
		buffWriter: bufio.NewWriterSize(w, buffsize),
		buffsize:   buffsize,
	}
	encoder.visitor.SetVisitable(&encoder)
	return encoder
}

// OpenOptional checks if the current Node on the Stack is opened.
// opens it and returns true if it is not, returns false otherwise.
func (e *Encoder) OpenOptional() bool {
	if !e.stack.IsOpened() {
		e.writeString(gt)
		e.stack.SetOpened()
		return true
	}
	return false
}

// writeString writes the given string to the encoders io.Writer.
func (e *Encoder) writeString(in ...string) error {
	for _, text := range in {
		if _, err := e.buffWriter.Write([]byte(text)); err != nil {
			return err
		}
	}
	return nil
}

// writeBytes writes the given Byteslice to the encoders io.Writer.
func (e *Encoder) writeBytes(in []byte) error {
	if _, err := e.buffWriter.Write(in); err != nil {
		return err
	}
	return nil
}

// writeAttributes writes all Attributes of the current parent node to the encoders io.Writer.
func (e *Encoder) writeAttributes() error {
	/*if !e.parent.attributesWritten {

		// sorting Attributes alphabetically before writing to the encoders io.Writer
		// TODO: may be inefficient. possible refactor
		keys := make([]string, 0, len(e.parent.Node.Attributes))
		for k, _ := range e.parent.Node.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			err := e.writeString(whitespace, k, equals, dquotes, e.parent.Node.Attributes[k], dquotes)
			if err != nil {
				return err
			}
		}

		/*for key, val := range e.parent.Node.Attributes {
			err := e.writeString(whitespace, key, equals, dquotes, val, dquotes)
			if err != nil {
				return err
			}
		}

		e.parent.attributesWritten = true
	}
	return nil*/
	return nil
}

// Encode starts the encoding of tadl-text to XML
// encoded text will be written to the encoders io.Writer
func (e *Encoder) Encode() error {
	e.writeString(lt, "root")
	err := e.visitor.Run()
	if err != nil {
		return err
	}
	e.writeString(lt, slash, "root", gt)
	e.buffWriter.Flush()
	return nil
}

/*
// open sets the parent pointer to the latest Child of it's current Node
func (e *Encoder) open() {
	e.writeString(gt)


	e.parent = e.parent.Children[len(e.parent.Children)-1]
}*/

// Close moves the parent pointer to its current parent Node
func (e *Encoder) Close() error {
	if !e.stack.IsOpened() {
		e.writeString(gt)
		e.stack.SetOpened()
	}
	node, suc := e.stack.Pop()
	if !suc {
		return errors.New("An error occurred while popping the stack")
	}
	e.writeString(lt, slash, escapeDoubleQuotes(node.name), gt)

	return nil
}

// NewNode creates a named Node and adds it as a child to the current parent Node
// Opens the new Node
func (e *Encoder) NewNode(name string) {
	e.OpenOptional()
	e.writeString(lt, name)
	e.stack.Push(*NewNode(escapeDoubleQuotes(name)))
}

// NewTextNode creates a new Node with Text based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (e *Encoder) NewTextNode(cd *token.CharData) {
	if !isWhitespaceOnly(cd.Value) {
		e.writeString(escapeDoubleQuotes(cd.Value))
	}
}

// NewCommentNode creates a new Node with Text as Comment, based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (e *Encoder) NewCommentNode(cd *token.CharData) {
	e.OpenOptional()
	e.writeString(lt, exclammark, hyphen, hyphen, whitespace, escapeDoubleQuotes(cd.Value), whitespace, hyphen, hyphen, gt)
}

// SetBlockType does nothing, as BlockType is not relevant for encoding.
func (e *Encoder) SetBlockType(b parser.BlockType) {
	return
}

// GetRootBlockType returns BlockNone, as BlockType is not relevant for encoding.
func (e *Encoder) GetRootBlockType() parser.BlockType {
	return parser.BlockNone
}

// GetForwardingLenght returns the lenght of the List of forwaring Nodes
func (e *Encoder) GetForwardingLength() int {
	return len(e.forward)
}

// GetForwardingAttributesLength returns the length of the forwarding AttributeMap
func (e *Encoder) GetForwardingAttributesLength() int {
	return e.forwardedAttributes.Len()
}

// AddAttribute adds a given Attribute to the current parent Node
func (e *Encoder) AddAttribute(key, value string) {
	e.writeString(whitespace, key, equals, dquotes, value, dquotes)
}

// AddForwardAttribute adds a given AttributeMap to the forwaring Attributes
func (e *Encoder) AddForwardAttribute(key, value string) {
	e.forwardedAttributes.Push(&AttributeNode{
		Data: Attribute{
			key: key,
			val: value,
		},
	})
}

// AddForwardNode appends a given Node to the list of forwarding Nodes
func (e *Encoder) AddForwardNode(name string) {
	e.forward.Push(*NewNode(escapeDoubleQuotes(name)))
}

// MergeAttributes merges the list of forwarded Attributes to the current parent Nodes Attributes
func (e *Encoder) MergeAttributes() {
	attribute := e.forwardedAttributes.first
	e.writeString(attribute.Data.key, attribute.Data.val)
	for attribute = attribute.Next; attribute != nil; {
		e.writeString(attribute.Data.key, attribute.Data.val)
	}
	e.forwardedAttributes = *NewAttributeList()
}

// MergeAttributesForwarded adds the buffered forwarding AttributeMap to the latest forwarded Node
func (e *Encoder) MergeAttributesForwarded() {
	runner1 := e.forward[len(e.forward)-1].attributes.first
	runner2 := e.forwardedAttributes.first
	for runner2 != nil {
		runner1.Next = runner2
		runner2 = runner2.Next
	}
	e.forwardedAttributes = *NewAttributeList()
}

// AppendForwardingNodes appends the current list of forwarding Nodes
// as Children to the current parent Node
func (e *Encoder) AppendForwardingNodes() {
	/*if e.rootForward != nil && e.rootForward.Children != nil && len(e.rootForward.Children) != 0 {
		e.parent.Children = append(e.parent.Children, e.rootForward.Children...)
		e.rootForward.Children = nil
		e.parentForward = e.rootForward
	}*/
}

// AppendSubTree appends the rootForward Tree to the current parent Nodes Children
func (e *Encoder) AppendSubTree() {
	/*if e.rootForward != nil && len(e.rootForward.Children) != 0 {
		e.parent.Children = append(e.parent.Children, e.rootForward.Children...)
		e.rootForward.Children = nil
	}*/
}

// g2AppendComments will append all comments that were parsed with g2EatComments as children
// into the given node.
func (e *Encoder) G2AppendComments() {
	/*if e.parent != nil {
		e.parent.Children = append(e.parent.Children, e.g2Comments...)
		e.g2Comments = nil
	}*/
}

// G2AddComments adds a new Comment Node based on given CharData to the g2Comments List,
// to be added to the tree later
func (e *Encoder) G2AddComments(cd *token.CharData) {
	//e.g2Comments = append(e.g2Comments, NewCommentNode(escapeDoubleQuotesChar(cd)))
}

// SwitchActiveTree switches the active Tree between the main syntax tree and the forwarding tree
// To modify the forwarding tree, call SwitchActiveTree, call treeCreation functions, call SwitchActiveTree
func (e *Encoder) SwitchActiveTree() {
	/*var cache *TreeNodeEnc = e.parent
	e.parent = e.parentForward
	e.parentForward = cache

	cache = e.root
	e.root = e.rootForward
	e.rootForward = cache
	e.globalForward = !e.globalForward*/
}

// NewStringNode creates a Node with Text and adds it as a child to the current parent Node
// Opens the new Node, used for testing purposes only
func (e *Encoder) NewStringNode(name string) {
	/*e.parent.AddChildren(NewStringNode(escapeDoubleQuotes(name)))
	e.parent.Children[len(e.parent.Children)-1].Parent = e.parent
	e.open()*/
}

// NewStringCommentNode creates a new Node with Text as Comment, based on string and adds it as a child to the current parent Node
// Opens the new Node, used for testing purposes only
func (e *Encoder) NewStringCommentNode(text string) {
	/*e.parent.AddChildren(NewStringCommentNode(escapeDoubleQuotes(text)))
	e.parent.Children[len(e.parent.Children)-1].Parent = e.parent
	e.open()*/
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
