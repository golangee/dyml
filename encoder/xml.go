package encoder

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/golangee/dyml/parser"
	"github.com/golangee/dyml/token"
	"github.com/golangee/dyml/util"
)

type XMLEncoder struct {
	filename string
	reader   io.Reader
	writer   *bufio.Writer

	// openNodes is a stack of elements that are currently opened,
	// so that the closing tag and other information can be written correctly.
	openNodes []*node
	// forwardedAttributes is a list of attributes that are being forwarded into the next node.
	forwardedAttributes util.AttributeList
	// forwardedNodes are all (text-) nodes that are being forwarded into this node.
	// It is important to note that these nodes are either just text or nodes with no attributes.
	forwardedNodes []*node
	// indent is the current level of indentation for emitting XML.
	indent uint
}

// node is a node that we are currently working on.
type node struct {
	// name is the name in the XML tag which we need to save so that the closing tag can be written.
	name string
	// text is text if this is a text node. For a text node all other attributes are irrelevant.
	text string
	// attributes is a list of attributes this node has.
	attributes util.AttributeList
	// openTagWritten is set to true once we have written the starting XML tag.
	openTagWritten bool
	// isForwarded is true when this node is being forwarded.
	isForwarded bool
	// forwardedNodes contains all nodes that this node is holding until they can be written out.
	forwardedNodes []*node
}

func NewXMLEncoder(filename string, r io.Reader, w io.Writer) *XMLEncoder {
	return &XMLEncoder{
		filename: filename,
		reader:   r,
		writer:   bufio.NewWriter(w),
	}
}

// Encode starts the encoding process, reading input from the reader and writing to the writer.
// There is no up-front validation, which means that in case of an error incomplete output
// already got emitted.
func (e *XMLEncoder) Encode() error {
	v := parser.NewVisitor(e.filename, e.reader)
	v.SetVisitable(e)

	return v.Run()
}

func (e *XMLEncoder) Open(name token.Identifier) error {
	return e.openNode(name.Value)
}

func (e *XMLEncoder) Comment(comment token.CharData) error {
	if err := e.writeTopNodeOpen(); err != nil {
		return err
	}

	return e.writeString(fmt.Sprintf("%s<!-- %s -->\n", e.indentString(), escapeXMLSafe(comment.Value)))
}

func (e *XMLEncoder) Text(text token.CharData) error {
	if err := e.writeTopNodeOpen(); err != nil {
		return err
	}

	return e.writeString(fmt.Sprintf("%s%s\n", e.indentString(), strings.TrimSpace(escapeXMLSafe(text.Value))))
}

func (e *XMLEncoder) OpenReturnArrow(arrow token.G2Arrow, name *token.Identifier) error {
	if name != nil {
		return e.openNode(name.Value)
	}

	return e.openNode("ret")
}

func (e *XMLEncoder) CloseReturnArrow() error {
	return e.Close()
}

func (e *XMLEncoder) SetBlockType(blockType parser.BlockType) error {
	// Not used in XML
	return nil
}

func (e *XMLEncoder) OpenForward(name token.Identifier) error {
	n := &node{
		name:        name.Value,
		isForwarded: true,
	}
	e.push(n)
	e.forwardedNodes = append(e.forwardedNodes, n)

	return nil
}

func (e *XMLEncoder) TextForward(text token.CharData) error {
	n := &node{
		text:        text.Value,
		isForwarded: true,
	}
	e.forwardedNodes = append(e.forwardedNodes, n)

	return nil
}

func (e *XMLEncoder) Close() error {
	// Forwarding nodes should just be popped,
	// they are already inside e.forwardedNodes
	if e.peek().isForwarded {
		e.pop()

		return nil
	}

	if err := e.writeTopNodeOpen(); err != nil {
		return err
	}

	e.indent--

	top := e.pop()

	err := e.writeString(fmt.Sprintf("%s</%s>\n", e.indentString(), top.name))
	if err != nil {
		return err
	}

	return nil
}

func (e *XMLEncoder) Attribute(key token.Identifier, value token.CharData) error {
	n := e.peek()
	attr := util.Attribute{
		Key:   key.Value,
		Value: value.Value,
		Range: token.Position{
			BeginPos: key.Begin(),
			EndPos:   value.End(),
		},
	}

	if n.attributes.Set(attr) {
		return token.NewPosError(attr.Range, "key defined twice")
	}

	return nil
}

func (e *XMLEncoder) AttributeForward(key token.Identifier, value token.CharData) error {
	attr := util.Attribute{
		Key:   key.Value,
		Value: value.Value,
		Range: token.Position{
			BeginPos: key.Begin(),
			EndPos:   value.End(),
		},
	}

	if e.forwardedAttributes.Set(attr) {
		return token.NewPosError(attr.Range, "key defined twice")
	}

	return nil
}

func (e *XMLEncoder) Finalize() error {
	if e.writer.Flush() != nil {
		return fmt.Errorf("failed to flush written XML: %w", e.writer.Flush())
	}

	return nil
}

// writeString is a convenience method to write strings to the underlying writer.
func (e *XMLEncoder) writeString(s string) error {
	_, err := e.writer.WriteString(s)

	return err
}

// openNode puts a node on our working stack but does not write it yet.
// However, its parent node might get written out, since we know that it will not get any more attributes.
func (e *XMLEncoder) openNode(name string) error {
	if err := e.writeTopNodeOpen(); err != nil {
		return err
	}

	// Put the node on our stack, so we know how to close it.
	e.push(&node{
		name:           name,
		attributes:     e.forwardedAttributes,
		forwardedNodes: e.forwardedNodes,
	})

	e.forwardedAttributes = util.AttributeList{}
	e.forwardedNodes = nil

	return nil
}

// writeTopNodeOpen writes the topmost stack node to the writer.
func (e *XMLEncoder) writeTopNodeOpen() error {
	top := e.peek()
	if top != nil && !top.openTagWritten {
		top.openTagWritten = true

		// Build the opening tag with all attributes
		var tag strings.Builder

		tag.WriteString(e.indentString())
		tag.WriteString("<")
		tag.WriteString(top.name)

		for {
			attr := top.attributes.Pop()
			if attr == nil {
				break
			}

			tag.WriteString(fmt.Sprintf(` %s="%s"`, attr.Key, escapeXMLSafe(attr.Value)))
		}
		tag.WriteString(">\n")

		e.indent++

		// Place all forwarded nodes here
		for _, forwardedNode := range top.forwardedNodes {
			if len(forwardedNode.name) > 0 {
				tag.WriteString(fmt.Sprintf("%[1]s<%[2]s></%[2]s>\n", e.indentString(), forwardedNode.name))
			} else if len(forwardedNode.text) > 0 {
				tag.WriteString(fmt.Sprintf("%s%s\n", e.indentString(), escapeXMLSafe(forwardedNode.text)))
			}
		}

		top.forwardedNodes = nil

		err := e.writeString(tag.String())
		if err != nil {
			return err
		}
	}

	return nil
}

// push a node onto our working stack.
func (e *XMLEncoder) push(n *node) {
	e.openNodes = append(e.openNodes, n)
}

// peek at the top element in our working stack. Might return nil if the stack is empty.
func (e *XMLEncoder) peek() *node {
	if len(e.openNodes) > 0 {
		n := e.openNodes[len(e.openNodes)-1]

		return n
	}

	return nil
}

// pop the top node from the working stack. Might return nil if the stack is empty.
func (e *XMLEncoder) pop() *node {
	if len(e.openNodes) > 0 {
		n := e.openNodes[len(e.openNodes)-1]
		e.openNodes = e.openNodes[:len(e.openNodes)-1]

		return n
	}

	return nil
}

// indentString returns a string with a number of spaces that matches the
// current indentation level.
func (e *XMLEncoder) indentString() string {
	var tmp strings.Builder
	for i := uint(0); i < e.indent; i++ {
		tmp.WriteString("    ")
	}
	return tmp.String()
}

// escapeXMLSafe replaces all occurrences of reserved characters in XML: <>&".
func escapeXMLSafe(s string) string {
	replacer := strings.NewReplacer("<", "&lt;", ">", "&gt;", "&", "&amp;", `"`, "&quot;")

	return replacer.Replace(s)
}
