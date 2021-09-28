// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"errors"
	"fmt"
	"io"

	"github.com/golangee/dyml/token"
)

// TreeNode is a node in the parse tree.
// For regular nodes Text and Comment will always be nil.
// For terminal text nodes Children and Name will be empty and Text will be set.
// For comment nodes Children and Name will be empty and only Comment will be set.
type TreeNode struct {
	Name       string
	Text       *string
	Comment    *string
	Attributes AttributeList
	Children   []*TreeNode
	// BlockType describes the type of brackets the children were surrounded with.
	// This may be BlockNone in which case this node either has no or one children.
	BlockType BlockType
	// Range will span all tokens that were processed to build this node.
	Range token.Position
	// forwarded is set to true when this node was/should be forwarded.
	forwarded bool
}

// NewNode creates a new node for the parse tree.
func NewNode(name string) *TreeNode {
	return &TreeNode{
		Name:       name,
		Attributes: NewAttributeList(),
		BlockType:  BlockNone,
	}
}

// NewTextNode creates a node that will only contain text.
func NewTextNode(cd *token.CharData) *TreeNode {
	return &TreeNode{
		Text: &cd.Value,
		Range: token.Position{
			BeginPos: cd.Begin(),
			EndPos:   cd.End(),
		},
	}
}

// NewCommentNode creates a node that will only contain a comment.
func NewCommentNode(cd *token.CharData) *TreeNode {
	return &TreeNode{
		Comment: &cd.Value,
		Range: token.Position{
			BeginPos: cd.Begin(),
			EndPos:   cd.End(),
		},
	}
}

// NewStringNode will create a text node, like NewTextNode,
// but without positional information. This is only used for testing.
// Use NewTextNode with a CharData token if you can.
func NewStringNode(text string) *TreeNode {
	return &TreeNode{
		Text: &text,
	}
}

// NewStringCommentNode will create a comment node, like NewCommentNode,
// but without positional information. This is only used for testing.
// Use NewCommentNode with a CharData token if you can.
func NewStringCommentNode(text string) *TreeNode {
	return &TreeNode{
		Comment: &text,
	}
}

// AddChildren adds children to a node and can be used builder-style.
func (t *TreeNode) AddChildren(children ...*TreeNode) *TreeNode {
	if t.Children != nil {
		t.Children = append(t.Children, children...)
	} else {
		t.Children = children
	}

	return t
}

// AddAttribute adds an attribute to a node and can be used builder-style.
func (t *TreeNode) AddAttribute(key, value string) *TreeNode {
	t.Attributes.Set(key, value)

	return t
}

// Block is used to set the BlockType of this node.
func (t *TreeNode) Block(blockType BlockType) *TreeNode {
	t.BlockType = blockType

	return t
}

// IsClosedBy returns true if tok is a BlockEnd/GroupEnd/GenericEnd that is the correct
// match for closing this TreeNode.
func (t *TreeNode) IsClosedBy(tok token.Token) bool {
	switch tok.(type) {
	case *token.BlockEnd:
		return t.BlockType == BlockNormal
	case *token.GroupEnd:
		return t.BlockType == BlockGroup
	case *token.GenericEnd:
		return t.BlockType == BlockGeneric
	default:
		return false
	}
}

// IsText returns true if this node is a text only node.
// Only one of IsText, IsComment, IsNode should be true.
func (t *TreeNode) IsText() bool {
	return t.Text != nil
}

// IsComment returns true if this node is a comment node.
// Only one of IsText, IsComment, IsNode should be true.
func (t *TreeNode) IsComment() bool {
	return t.Comment != nil
}

// IsNode returns true if this is a regular node.
// Only one of IsText, IsComment, IsNode should be true.
func (t *TreeNode) IsNode() bool {
	return !t.IsText() && !t.IsComment()
}

// tokenWithError is a struct that wraps a Token and an error that may
// have occurred while reading that Token.
// This type simplifies storing tokens in the parser.
type tokenWithError struct {
	tok token.Token
	err error
}

// BlockType is an addition for nodes that describes with what brackets their children were surrounded.
type BlockType string

const (
	// BlockNone represents no BlockType
	BlockNone BlockType = ""
	// BlockNormal represents curly brackets
	BlockNormal BlockType = "{}"
	// BlockGroup represents round brackets
	BlockGroup BlockType = "()"
	// BlockGeneric represents pointed brackets
	BlockGeneric BlockType = "<>"
)

// Parser is used to get a tree representation from dyml input.
type Parser struct {
	// finalTree is created when Close is called on the last TreeNode in the workingStack.
	finalTree *TreeNode
	// workingStack is the current stack the parser is working on. This is handy for working with
	// open and close calls.
	workingStack []*TreeNode
	// visitor is the visitor that will call this parser's callback methods for constructing the tree.
	visitor *Visitor
	// forwardedAttributes are all attributes that were forwarded and need to be placed in the next node.
	forwardedAttributes AttributeList
	// forwardedNodes is a list of all nodes that should be forwarded into the next normal node.
	// They will be constructed on the workingStack and moved into this list once
	// they have been closed.
	forwardedNodes []*TreeNode
}

// NewParser creates and returns a new Parser with corresponding Visitor
func NewParser(filename string, r io.Reader) *Parser {
	return &Parser{
		visitor: NewVisitor(filename, r),
	}
}

// Parse returns a parsed tree.
func (p *Parser) Parse() (*TreeNode, error) {
	p.visitor.SetVisitable(p)
	err := p.visitor.Run()
	if err != nil {
		return nil, err
	}

	// TODO Make a last check here or in Finalize
	return p.finalTree, nil
}

// getStackTop returns the topmost element in the working stack.
func (p *Parser) getStackTop() (*TreeNode, error) {
	if len(p.workingStack) > 0 {
		return p.workingStack[len(p.workingStack)-1], nil
	} else {
		return nil, errors.New("you found a bug: could not get top of stack in parser")
	}
}

// popStack removes the topmost element from the working stack.
func (p *Parser) popStack() (*TreeNode, error) {
	if len(p.workingStack) > 0 {
		node := p.workingStack[len(p.workingStack)-1]
		p.workingStack = p.workingStack[:len(p.workingStack)-1]
		return node, nil
	} else {
		return nil, errors.New("you found a bug: could not pop stack in parser")
	}
}

// pushStack adds an element to the top of the stack.
func (p *Parser) pushStack(node *TreeNode) {
	p.workingStack = append(p.workingStack, node)
}

// applyForwardedAttributes applies all forwarded attributes to the node.
func (p *Parser) applyForwardedAttributes(node *TreeNode) error {
	// TODO Check for duplicates
	for {
		attr := p.forwardedAttributes.Pop()
		if attr == nil {
			break
		} else {
			node.Attributes.Set(attr.Key, attr.Value)
		}
	}

	return nil
}

// TODO Remove all print methods in the callbacks

func (p *Parser) Open(name token.Identifier) error {
	fmt.Printf("[Parser] Open(%s)\n", name.Value)

	node := NewNode(name.Value)

	if err := p.applyForwardedAttributes(node); err != nil {
		return err
	}

	// Place all forwarded nodes in this node.
	node.AddChildren(p.forwardedNodes...)
	p.forwardedNodes = nil

	p.pushStack(node)

	return nil
}

func (p *Parser) Comment(comment token.CharData) error {
	fmt.Printf("[Parser] Comment(%s)\n", comment.Value)

	top, err := p.getStackTop()
	if err != nil {
		return err
	}
	top.AddChildren(NewCommentNode(&comment))

	return nil
}

func (p *Parser) Text(text token.CharData) error {
	fmt.Printf("[Parser] Text(%s)\n", text.Value)

	top, err := p.getStackTop()
	if err != nil {
		return err
	}
	top.AddChildren(NewTextNode(&text))

	return nil
}

func (p *Parser) OpenReturnArrow(arrow token.G2Arrow) error {
	fmt.Printf("[Parser] OpenReturnArrow\n")
	return nil
}

func (p *Parser) CloseReturnArrow() error {
	fmt.Printf("[Parser] CloseReturnArrow\n")
	return nil
}

func (p *Parser) OpenForward(name token.Identifier) error {
	fmt.Printf("[Parser] OpenForward(%s)\n", name.Value)

	node := NewNode(name.Value)
	node.forwarded = true
	p.pushStack(node)

	if err := p.applyForwardedAttributes(node); err != nil {
		return err
	}

	return nil
}

func (p *Parser) TextForward(text token.CharData) error {
	fmt.Printf("[Parser] TextForward(%s)\n", text.Value)

	node := NewTextNode(&text)
	node.forwarded = true
	p.forwardedNodes = append(p.forwardedNodes, node)

	return nil
}

func (p *Parser) SetBlockType(blockType BlockType) error {
	fmt.Printf("[Parser] SetBlockType(%s)\n", blockType)

	top, err := p.getStackTop()
	if err != nil {
		return err
	}

	top.Block(blockType)

	return nil
}

func (p *Parser) Close() error {
	fmt.Printf("[Parser] Close()\n")

	// TODO This method needs more more length checks.

	// Make the topmost node of the stack a child to the one before it,
	// or set it as the finalTree if there is no parent.
	child, err := p.popStack()
	if err != nil {
		return err
	}

	if child.forwarded {
		p.forwardedNodes = append(p.forwardedNodes, child)
		return nil
	}

	if len(p.workingStack) > 0 {
		p.workingStack[len(p.workingStack)-1].AddChildren(child)
	} else {
		if p.finalTree == nil {
			p.finalTree = child
			fmt.Println("[Parser] Tree was marked as final!")
		} else {
			return errors.New("you found a bug: finalTree already exists")
		}
	}

	return nil
}

func (p *Parser) Attribute(key token.Identifier, value token.CharData) error {
	fmt.Printf("[Parser] Attribute(%s, %s)\n", key.Value, value.Value)

	top, err := p.getStackTop()
	if err != nil {
		return err
	}

	if top.Attributes.Set(key.Value, value.Value) {
		// TODO For the whole file: more positional errors
		return errors.New("attribute already defined")
	}

	return nil
}

func (p *Parser) AttributeForward(key token.Identifier, value token.CharData) error {
	fmt.Printf("[Parser] AttributeForward(%s, %s)\n", key.Value, value.Value)

	p.forwardedAttributes.Add(key.Value, value.Value)

	return nil
}

func (p *Parser) Finalize() error {
	// TODO Make sure that all forwarding attributes and nodes have been processed.
	fmt.Printf("[Parser] Finalize()\n")

	if len(p.forwardedNodes) > 0 {
		return errors.New("there are forwarding nodes left")
	}

	if p.forwardedAttributes.Len() > 0 {
		return errors.New("there are forwarded attributes left")
	}

	return nil
}
