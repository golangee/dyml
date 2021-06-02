package parser2

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// TreeNode is a node in the parse tree.
// For regular nodes Text will always be nil.
// For terminal text nodes Children and Name will be empty and Text will be set.
// TODO The positions from the lexer need to be saved in the nodes.
type TreeNode struct {
	Name       string
	Text       *string
	Attributes map[string]string
	Children   []*TreeNode
}

// NewNode creates a new node for the parse tree.
func NewNode(name string) *TreeNode {
	return &TreeNode{
		Name:       name,
		Attributes: make(map[string]string),
	}
}

// NewTextNode creates a node that will only contain text.
func NewTextNode(text string) *TreeNode {
	return &TreeNode{
		Text: &text,
	}
}

// AddChildren adds children to a node and can be used builder-style.
func (t *TreeNode) AddChildren(children ...*TreeNode) *TreeNode {
	t.Children = append(t.Children, children...)
	return t
}

// AddAttribute adds an attribute to a node and can be used builder-style.
func (t *TreeNode) AddAttribute(key, value string) *TreeNode {
	t.Attributes[key] = value
	return t
}

// tokenWithError is a struct that wraps a Token and an error that may
// have occured while reading that Token.
// This type simplifies storing tokens in the parser.
type tokenWithError struct {
	tok Token
	err error
}

// Parser is used to get a tree representation from Tadl input.
type Parser struct {
	lexer *Lexer
	mode  GrammarMode
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
}

func NewParser(filename string, r io.Reader) *Parser {
	return &Parser{
		lexer: NewLexer(filename, r),
		mode:  G1,
	}
}

// next returns the next token or (nil, io.EOF) if there are no more tokens.
// Repeately calling this can be used to get all tokens by advancing the lexer.
func (p *Parser) next() (Token, error) {

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
			return twe.tok, twe.err
		}
	}

	return tok, err
}

// peek lets you look at the next token without advancing the lexer.
// Under the hood it does advance the lexer, but by using only next() and peek()
// you will get expected behaviour.
func (p *Parser) peek() (Token, error) {

	// Check the buffer for tokens
	if len(p.tokenBuffer) > 0 {
		twe := p.tokenBuffer[0]
		return twe.tok, twe.err
	}

	tok, err := p.lexer.Token()

	if errors.Is(err, io.EOF) {
		// Check tail buffer for tokens that need to be appended
		if len(p.tokenTailBuffer) > 0 {
			twe := p.tokenTailBuffer[0]
			return twe.tok, twe.err
		}
	}

	// Store token+error for use in next()
	p.tokenBuffer = append(p.tokenBuffer, tokenWithError{
		tok: tok,
		err: err,
	})

	return tok, err
}

// Parse returns a parsed tree.
func (p *Parser) Parse() (*TreeNode, error) {

	// TODO G2 mode

	// Prepend and append tokens for the root element.
	// This makes the root just another element, which simplifies parsing a lot.
	p.tokenBuffer = append(p.tokenBuffer,
		tokenWithError{tok: &DefineElement{}},
		tokenWithError{tok: &Identifier{Value: "root"}},
		tokenWithError{tok: &BlockStart{}},
	)
	p.tokenTailBuffer = append(p.tokenTailBuffer,
		tokenWithError{tok: &BlockEnd{}},
	)

	return p.g1Node()
}

// g1Node recusively parses a g1Node and all its children from tokens.
func (p *Parser) g1Node() (*TreeNode, error) {
	// TODO Parse positional information into nodes.

	forwardingNode := false

	// Expect ElementDefinition or CharData
	tok, err := p.next()
	if err != nil {
		return nil, err
	}
	if de, ok := tok.(*DefineElement); ok {
		forwardingNode = de.Forward
	} else if cd, ok := tok.(*CharData); ok {
		return NewTextNode(cd.Value), nil
	} else {
		return nil, NewUnexpectedTokenError(tok, TokenCharData, TokenDefineElement)
	}

	// Expect identifier for new element
	node := NewNode("invalid name")
	tok, err = p.next()
	if err != nil {
		return nil, err
	}
	if id, ok := tok.(*Identifier); ok {
		node.Name = id.Value
	} else {
		return nil, NewUnexpectedTokenError(tok, TokenIdentifier)
	}

	// We now have a valid node.
	// Place our forwardingNodes inside it, if it is not one itself.
	if !forwardingNode {
		node.Children = p.forwardingNodes
		p.forwardingNodes = nil
	}

	// Process all attributes
	// TODO Forwarded attributes
	for {
		tok, _ = p.peek()
		if tok.tokenType() != TokenDefineAttribute {
			break
		}

		p.next() // Pop the token since we know it's a DefineAttribute

		// Read attribute key
		attrKey := ""
		tok, err = p.next()
		if err != nil {
			return nil, err
		}
		if ident, ok := tok.(*Identifier); ok {
			attrKey = ident.Value
		} else {
			return nil, NewUnexpectedTokenError(tok, TokenIdentifier)
		}

		// Read CharData enclosed in brackets as attribute value
		tok, _ = p.next()
		if tok.tokenType() != TokenBlockStart {
			return nil, NewUnexpectedTokenError(tok, TokenBlockStart)
		}

		tok, err = p.next()
		if err != nil {
			return nil, err
		}
		if cd, ok := tok.(*CharData); ok {
			node.AddAttribute(attrKey, cd.Value)
		} else {
			return nil, NewUnexpectedTokenError(tok, TokenCharData)
		}

		tok, _ = p.next()
		if tok.tokenType() != TokenBlockEnd {
			return nil, NewUnexpectedTokenError(tok, TokenBlockEnd)
		}

	}

	// Optional children enclosed in brackets
	tok, _ = p.peek()
	if tok.tokenType() == TokenBlockStart {
		p.next() // Pop the token, we know it's a BlockStart

		// Append children until we encounter a TokenBlockEnd
		for {
			tok, _ = p.peek()
			if tok.tokenType() == TokenBlockEnd {
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
		if tok.tokenType() != TokenBlockEnd {
			return nil, NewUnexpectedTokenError(tok, TokenBlockEnd)
		}
	}

	if forwardingNode {
		// We just parsed a forwarding node. We need to save it, but cannot return it,
		// as it needs to be placed inside the next non-forwarding node.
		// We will parse another node to make it opaque to our caller that this happened.
		p.forwardingNodes = append(p.forwardingNodes, node)
		return p.g1Node()
	}

	return node, nil
}

func NewUnexpectedTokenError(tok Token, wanted ...TokenType) error {
	// TODO Proper error type with positional information
	wantedStrings := []string{}
	for _, tt := range wanted {
		wantedStrings = append(wantedStrings, string(tt))
	}
	wantedString := strings.Join(wantedStrings, ", ")
	return fmt.Errorf("unexpected %s, expected %s", tok.tokenType(), wantedString)
}
