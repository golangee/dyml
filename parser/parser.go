// SPDX-FileCopyrightText: © 2021 The tadl authors <https://github.com/golangee/tadl/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"errors"
	"github.com/golangee/tadl/token"
	"io"
)

// TreeNode is a node in the parse tree.
// For regular nodes Text and Comment will always be nil.
// For terminal text nodes Children and Name will be empty and Text will be set.
// For comment nodes Children and Name will be empty and only Comment will be set.
type TreeNode struct {
	Name       string
	Text       *string
	Comment    *string
	Attributes AttributeMap
	Children   []*TreeNode
	// BlockType describes the type of brackets the children were surrounded with.
	BlockType BlockType
	// Range will span all tokens that were processed to build this node.
	Range token.Position
}

// NewNode creates a new node for the parse tree.
func NewNode(name string) *TreeNode {
	return &TreeNode{
		Name:       name,
		Attributes: NewAttributeMap(),
		BlockType:  BlockNormal,
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
	t.Children = append(t.Children, children...)

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

// isClosedBy returns true if tok is a BlockEnd/GroupEnd/GenericEnd that is the correct
// match for closing this TreeNode.
func (t *TreeNode) isClosedBy(tok token.Token) bool {
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

// AttributeMap is a custom map[string]string to make the
// handling of attributes easier.
type AttributeMap map[string]string

func NewAttributeMap() AttributeMap {
	return make(map[string]string)
}

// Set sets a key to a value in this map.
func (a AttributeMap) Set(key, value string) {
	a[key] = value
}

// Has returns true if the given key is in the map and false otherwise.
func (a AttributeMap) Has(key string) bool {
	_, ok := a[key]
	return ok
}

// Merge returns a new AttributeMap with all keys from this and the other AttributeMap.
func (a AttributeMap) Merge(other AttributeMap) AttributeMap {
	result := NewAttributeMap()

	for k, v := range a {
		result[k] = v
	}

	for k, v := range other {
		result[k] = v
	}

	return result
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
	BlockNone    BlockType = ""
	BlockNormal  BlockType = "{}"
	BlockGroup   BlockType = "()"
	BlockGeneric BlockType = "<>"
)

// Parser is used to get a tree representation from Tadl input.
type Parser struct {
	lexer *token.Lexer
	mode  token.GrammarMode
	// tokenBuffer contains all tokens that need to be processed next.
	// These could be peeked tokens or tokens that were added in the parser.
	// When it is empty, we can call lexer.Token() to get the next token.
	tokenBuffer []tokenWithError
	// tokenTailBuffer contains all tokens that need to be processed once
	// lexer.Token() returns no more tokens. tokenTailBuffer will contain
	// tokens that were added from parser code.
	tokenTailBuffer []tokenWithError
	// forwardingNodes is a list of all nodes that were defined as forwarded.
	// They will be inserted into the next node.
	forwardingNodes []*TreeNode
	// g2Comments contains all comments in G2 that were eaten from the input,
	// but are not yet placed in a sensible position.
	g2Comments []*TreeNode
}

func NewParser(filename string, r io.Reader) *Parser {
	return &Parser{
		lexer: token.NewLexer(filename, r),
		mode:  token.G1,
	}
}

// next returns the next token or (nil, io.EOF) if there are no more tokens.
// Repeatedly calling this can be used to get all tokens by advancing the lexer.
func (p *Parser) next() (token.Token, error) {
	// Check the buffer for tokens
	if len(p.tokenBuffer) > 0 {
		twe := p.tokenBuffer[0]
		p.tokenBuffer = p.tokenBuffer[1:] // pop token

		return twe.tok, twe.err
	}

	tok, err := p.lexer.Token()

	if errors.Is(err, io.EOF) {
		// Check tail buffer for tokens that need to be appended
		if len(p.tokenTailBuffer) > 0 {
			twe := p.tokenTailBuffer[0]
			p.tokenTailBuffer = p.tokenTailBuffer[1:] // pop token

			// Tail tokens are generated and have no positional information associated.
			// We fix that here, so that potential errors point to the right place.
			if twe.tok != nil {
				lexPos := p.lexer.Pos()
				twe.tok.Pos().SetBegin(lexPos.File, lexPos.Line, lexPos.Col)
				twe.tok.Pos().SetEnd(lexPos.File, lexPos.Line, lexPos.Col)
			}

			return twe.tok, twe.err
		}
	}

	return tok, err
}

// peek lets you look at the next token without advancing the lexer.
// Under the hood it does advance the lexer, but by using only next() and peek()
// you will get expected behaviour.
func (p *Parser) peek() (token.Token, error) {
	// Check the buffer for tokens
	if len(p.tokenBuffer) > 0 {
		twe := p.tokenBuffer[0]
		return twe.tok, twe.err
	}

	tok, err := p.next()

	// Store token+error for use in next()
	p.tokenBuffer = append(p.tokenBuffer, tokenWithError{
		tok: tok,
		err: err,
	})

	return tok, err
}

// Parse returns a parsed tree.
func (p *Parser) Parse() (*TreeNode, error) {
	// Peek the first token to check if we should set G2 mode.
	tok, err := p.peek()

	// Edge case: When the input is empty we do not want the EOF in our buffer, as we will append tailTokens later.
	if errors.Is(err, io.EOF) {
		p.next()
	}

	var tree *TreeNode

	if tok != nil && tok.TokenType() == token.TokenG2Preamble {
		// Prepare G2 by switching out the preamble for a root identifier.
		p.mode = token.G2
		p.next()
		p.tokenBuffer = append(p.tokenBuffer,
			tokenWithError{tok: &token.Identifier{Value: "root"}},
		)

		tree, err = p.g2Node()
		if err != nil {
			return nil, err
		}
	} else {
		// Prepare G1.
		// Prepend and append tokens for the root element.
		// This makes the root just another element, which simplifies parsing a lot.
		p.tokenBuffer = append([]tokenWithError{
			{tok: &token.DefineElement{}},
			{tok: &token.Identifier{Value: "root"}},
			{tok: &token.BlockStart{}},
		},
			p.tokenBuffer...,
		)
		p.tokenTailBuffer = append(p.tokenTailBuffer,
			tokenWithError{tok: &token.BlockEnd{}},
		)

		tree, err = p.g1Node()
		if err != nil {
			return nil, err
		}
	}

	// All forwarding nodes should have been processed earlier.
	if len(p.forwardingNodes) > 0 {
		return nil, token.NewPosError(p.forwardingNodes[0].Range, "there is no node to forward this node into")
	}

	// The root element should always have curly brackets.
	if tree.BlockType != BlockNormal {
		return nil, token.NewPosError(tree.Range, "root element must have curly brackets")
	}

	return tree, nil
}

// g1Node recursively parses a G1 node and all its children from tokens.
func (p *Parser) g1Node() (*TreeNode, error) {
	forwardingNode := false
	node := NewNode("invalid name") // name will be set later
	node.Range.BeginPos = p.lexer.Pos()

	// Parse forwarding attributes
	forwardedAttributes, err := p.parseAttributes(true)
	if err != nil {
		return nil, err
	}

	// Expect ElementDefinition or CharData
	tok, err := p.next()
	if err != nil {
		return nil, err
	}

	switch t := tok.(type) {
	case *token.DefineElement:
		forwardingNode = t.Forward
	case *token.CharData:
		return NewTextNode(t), nil
	case *token.G1Comment:
		// Expect CharData as comment
		tok, err = p.next()
		if err != nil {
			return nil, err
		}

		if cd, ok := tok.(*token.CharData); ok {
			return NewCommentNode(cd), nil
		} else {
			return nil, token.NewPosError(
				tok.Pos(),
				"expected a comment",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
		}
	default:
		return nil, token.NewPosError(
			tok.Pos(),
			"this token is not valid here",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenDefineElement, token.TokenCharData))
	}

	// Expect identifier for new element
	tok, err = p.next()
	if err != nil {
		return nil, err
	}

	if id, ok := tok.(*token.Identifier); ok {
		node.Name = id.Value
	} else {
		return nil, token.NewPosError(
			tok.Pos(),
			"this token is not valid here",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenIdentifier))
	}

	// We now have a valid node.
	// Place our forwardingNodes inside it, if it is not one itself.
	if !forwardingNode {
		node.Children = p.forwardingNodes
		p.forwardingNodes = nil
	}

	// Process non-forwarding attributes.
	attributes, err := p.parseAttributes(false)
	if err != nil {
		return nil, err
	}

	node.Attributes = forwardedAttributes.Merge(attributes)

	tok, _ = p.peek()
	switch t := tok.(type) {
	case *token.BlockStart:
		// Optional children enclosed in brackets
		p.next() // Pop the token, we know it's a BlockStart

		node.BlockType = BlockNormal

		// Append children until we encounter a TokenBlockEnd
		for {
			tok, _ = p.peek()
			if tok.TokenType() == token.TokenBlockEnd {
				break
			}

			child, err := p.g1Node()
			if err != nil {
				return nil, err
			}

			node.AddChildren(child)
		}

		// Expect a BlockEnd
		tok, err = p.next()
		if err != nil {
			return nil, err
		}

		if tok.TokenType() != token.TokenBlockEnd {
			return nil, token.NewPosError(
				tok.Pos(),
				"use a '}' here to close the element",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenBlockEnd))
		}
	case *token.CharData:
		// An element can contain a single CharData token, which will become a child of the current node.
		p.next()
		node.AddChildren(NewTextNode(t))
	}

	if forwardingNode {
		// We just parsed a forwarding node. We need to save it, but cannot return it,
		// as it needs to be placed inside the next non-forwarding node.
		// We will parse another node to make it opaque to our caller that this happened.
		p.forwardingNodes = append(p.forwardingNodes, node)
		return p.g1Node()
	}

	node.Range.EndPos = p.lexer.Pos()

	return node, nil
}

// g1LineNodes returns all nodes that were encountered in a G1 line.
// This function will eat the beginning DefineElement and the ending G1LineEnd token.
func (p *Parser) g1LineNodes() ([]*TreeNode, error) {
	// Expect beginning '#'
	tok, err := p.next()
	if err != nil {
		return nil, err
	}

	var forward bool

	if de, ok := tok.(*token.DefineElement); ok {
		forward = de.Forward
	} else {
		return nil, token.NewPosError(
			tok.Pos(),
			"start of G1 line expected",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenDefineElement))
	}

	p.mode = token.G1Line

	// Read g1Nodes until we encounter G1LineEnd
	var nodes []*TreeNode

	for {
		tok, _ := p.peek()
		if tok != nil && tok.TokenType() == token.TokenG1LineEnd {
			p.next()
			break
		}

		node, err := p.g1Node()
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	p.mode = token.G2

	// Should this be a forwarding G1 line, we will store the children for later
	// and return an empty array here.
	if forward {
		p.forwardingNodes = append(p.forwardingNodes, nodes...)
		return []*TreeNode{}, nil
	} else {
		return nodes, nil
	}
}

// g2Node recursively parses a G2 node and all its children from tokens.
func (p *Parser) g2Node() (*TreeNode, error) {
	if err := p.g2EatComments(); err != nil {
		return nil, err
	}

	var nodeName string

	nodeStart := p.lexer.Pos()

	// Read forward attributes
	forwardedAttributes, err := p.parseAttributes(true)
	if err != nil {
		return nil, err
	}

	if err := p.g2EatComments(); err != nil {
		return nil, err
	}

	// Expect identifier or text
	tok, err := p.next()
	if err != nil {
		return nil, err
	}

	switch t := tok.(type) {
	case *token.Identifier:
		nodeName = t.Value
	case *token.CharData:
		if len(forwardedAttributes) > 0 {
			// We have forwarded attributes for a text, where an identifier would be appropriate.
			return nil, token.NewPosError(
				tok.Pos(),
				"attributes cannot be forwarded into this text",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
		}

		return NewTextNode(t), nil
	default:
		return nil, token.NewPosError(
			tok.Pos(),
			"this token is not valid here",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData, token.TokenIdentifier))
	}

	// From this point onwards we will be handling a valid node
	node := NewNode(nodeName)
	node.Range.BeginPos = nodeStart

	p.g2AppendComments(node)

	// Insert forwarded nodes
	node.Children = p.forwardingNodes
	p.forwardingNodes = nil

	// Read attributes
	attributes, err := p.parseAttributes(false)
	if err != nil {
		return nil, err
	}

	node.Attributes = forwardedAttributes.Merge(attributes)

	if err := p.g2EatComments(); err != nil {
		return nil, err
	}

	p.g2AppendComments(node)

	// Process children
	tok, err = p.peek()
	if err != nil {
		return nil, err
	}

	switch t := tok.(type) {
	case *token.CharData:
		p.next()

		node.AddChildren(NewTextNode(t))
	case *token.DefineElement:
		children, err := p.g1LineNodes()
		if err != nil {
			return nil, err
		}

		node.AddChildren(children...)
	case *token.BlockStart, *token.GenericStart, *token.GroupStart:
		p.next()

		// Set BlockType
		switch t.(type) {
		case *token.BlockStart:
			node.BlockType = BlockNormal
		case *token.GroupStart:
			node.BlockType = BlockGroup
		case *token.GenericStart:
			node.BlockType = BlockGeneric
		}

		// Parse children
		for {
			if err := p.g2EatComments(); err != nil {
				return nil, err
			}

			p.g2AppendComments(node)

			tok, err = p.peek()
			if err != nil {
				return nil, err
			}

			if node.isClosedBy(tok) {
				p.next() // pop closing token

				break
			} else if tok.TokenType() == token.TokenDefineElement {
				children, err := p.g1LineNodes()
				if err != nil {
					return nil, err
				}
				node.AddChildren(children...)
			} else {
				child, err := p.g2Node()
				if err != nil {
					return nil, err
				}

				node.AddChildren(child)
			}
		}
	case *token.BlockEnd, *token.GroupEnd, *token.GenericEnd:
		// Any closing token ends this node and will be handled by the parent.
	case *token.Comma:
		// Comma ends a node definition
		p.next()
	default:
		child, err := p.g2Node()
		if err != nil {
			return nil, err
		}

		node.AddChildren(child)
	}

	if err := p.g2EatComments(); err != nil {
		return nil, err
	}

	p.g2AppendComments(node)

	node.Range.EndPos = p.lexer.Pos()

	return node, nil
}

// g2EatComments will read all G2 comments from the current lexer position and store them in
// p.g2Comments so that the can be placed in a sensible node with g2AppendComments.
func (p *Parser) g2EatComments() error {
	for {
		tok, err := p.peek()
		if err != nil {
			// Do not report an error at this point, as some other function will handle it.
			break
		}

		if tok.TokenType() != token.TokenG2Comment {
			// The next thing is not a comment, which means that we are done.
			break
		}

		p.next() // Pop G2Comment

		tok, err = p.next()
		if err != nil {
			return err
		}

		// Expect CharData as comment
		if cd, ok := tok.(*token.CharData); ok {
			p.g2Comments = append(p.g2Comments, NewCommentNode(cd))
		} else {
			return token.NewPosError(
				tok.Pos(),
				"empty comment is not valid",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
		}
	}

	return nil
}

// g2AppendComments will append all comments that were parsed with g2EatComments as children
// into the given node.
func (p *Parser) g2AppendComments(node *TreeNode) {
	node.Children = append(node.Children, p.g2Comments...)
	p.g2Comments = nil
}

// parseAttributes eats consecutive attributes from the lexer and returns them in an AttributeMap.
// forwarding specifies if regular or forwarding attributes should be parsed.
// The function returns when a non-attribute is encountered. Should an attribute be parsed
// that is the wrong type of forwarding, it will return an error.
// This function can read attributes in modes G1, G2.
func (p *Parser) parseAttributes(wantForward bool) (AttributeMap, error) {
	result := NewAttributeMap()

	isG1 := p.mode == token.G1 || p.mode == token.G1Line

	for {
		if err := p.g2EatComments(); err != nil {
			return nil, err
		}

		tok, err := p.peek()
		if err != nil {
			break
		}

		if attr, ok := tok.(*token.DefineAttribute); ok {
			if wantForward && !attr.Forward {
				return nil, token.NewPosError(
					tok.Pos(),
					"this should be a forward attribute or removed",
				).SetCause(NewForwardAttrError())
			}

			if !wantForward && attr.Forward {
				// The next forwarding attribute is not for us, but for the next element.
				// Stop parsing attributes here.
				break
			}

			if wantForward != attr.Forward {
				// Should never happen, as the two if-blocks make this impossible.
				panic("Sanity check failed, wantForward != attr.Forward")
			}

			p.next() // pop DefineAttribute
		} else {
			// The next token is not a DefineAttribute
			break
		}

		var attrKey, attrValue string

		// Read attribute key
		tok, err = p.next()
		if err != nil {
			return nil, err
		}

		if ident, ok := tok.(*token.Identifier); ok {
			attrKey = ident.Value
		} else {
			return nil, token.NewPosError(
				tok.Pos(),
				"an identifier is required as an attribute key",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenIdentifier))
		}

		if result.Has(attrKey) {
			return nil, token.NewPosError(
				tok.Pos(),
				"cannot define same attribute twice",
			)
		}

		// Read CharData enclosed in brackets as attribute value in G1.
		// Read CharData after Assign in G2.

		tok, _ = p.next()
		if isG1 {
			if tok.TokenType() != token.TokenBlockStart {
				return nil, token.NewPosError(
					tok.Pos(),
					"attribute value must be enclosed in '{}'",
				).SetCause(NewUnexpectedTokenError(tok, token.TokenBlockStart))
			}
		} else {
			if tok.TokenType() != token.TokenAssign {
				return nil, token.NewPosError(
					tok.Pos(),
					"'=' is expected here",
				).SetCause(NewUnexpectedTokenError(tok, token.TokenAssign))
			}
		}

		tok, err = p.next()
		if err != nil {
			return nil, err
		}

		if cd, ok := tok.(*token.CharData); ok {
			attrValue = cd.Value
		} else {
			return nil, token.NewPosError(
				tok.Pos(),
				"attribute value is required",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
		}

		result.Set(attrKey, attrValue)

		if isG1 {
			tok, _ = p.next()
			if tok.TokenType() != token.TokenBlockEnd {
				return nil, token.NewPosError(
					tok.Pos(),
					"attribute value needs to be closed with '}'",
				).SetCause(NewUnexpectedTokenError(tok, token.TokenBlockEnd))
			}
		}
	}

	return result, nil
}
