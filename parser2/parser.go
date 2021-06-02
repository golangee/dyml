package parser2

import (
	"errors"
	"fmt"
	"github.com/golangee/tadl/token"
	"io"
	"strings"
)

// TreeNode is a node in the parse tree.
// For regular nodes Text will always be nil.
// For terminal text nodes Children and Name will be empty and Text will be set.
type TreeNode struct {
	Name       string
	Text       *string
	Attributes AttributeMap
	Children   []*TreeNode
	// Range will span all tokens that were processed to build this node.
	Range token.Position
}

// NewNode creates a new node for the parse tree.
func NewNode(name string) *TreeNode {
	return &TreeNode{
		Name:       name,
		Attributes: NewAttributeMap(),
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
	t.Attributes.Set(key, value)
	return t
}

// AttributeMap is a simple wrapper around a map[string]string to make the
// handling of attributes easier.
type AttributeMap map[string]string

func NewAttributeMap() AttributeMap {
	return make(map[string]string)
}

// Set sets a key to a value in this map.
func (a AttributeMap) Set(key, value string) {
	a[key] = value
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
// have occured while reading that Token.
// This type simplifies storing tokens in the parser.
type tokenWithError struct {
	tok Token
	err error
}

// UnexpectedTokenError is returned when a token appeared that the parser did not expect.
// It provides expected alternatives for tokens that were expected instead.
type UnexpectedTokenError struct {
	tok      Token
	expected []TokenType
}

func NewUnexpectedTokenError(tok Token, expected ...TokenType) error {
	return UnexpectedTokenError{
		tok:      tok,
		expected: expected,
	}
}

func (u UnexpectedTokenError) Error() string {
	// Build a pretty string with expected tokens
	expectedStrings := []string{}
	for _, tt := range u.expected {
		expectedStrings = append(expectedStrings, string(tt))
	}
	expected := strings.Join(expectedStrings, ", ")
	return fmt.Sprintf(
		"unexpected %s at %s, expected %s",
		u.tok.tokenType(),
		u.tok.position().Begin(),
		expected)
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
// Repeatedly calling this can be used to get all tokens by advancing the lexer.
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

			// Tail tokens are generated and have no positional information associated.
			// We fix that here, so that potential errors point to the right place.
			if twe.tok != nil {
				lexPos := p.lexer.Pos()
				twe.tok.position().SetBegin(lexPos.File, lexPos.Line, lexPos.Col)
				twe.tok.position().SetEnd(lexPos.File, lexPos.Line, lexPos.Col)
			}

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
	if de, ok := tok.(*DefineElement); ok {
		forwardingNode = de.Forward
	} else if cd, ok := tok.(*CharData); ok {
		return NewTextNode(cd.Value), nil
	} else {
		return nil, NewUnexpectedTokenError(tok, TokenCharData, TokenDefineElement)
	}

	// Expect identifier for new element
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

	// Process non-forwarding attributes.
	attributes, err := p.parseAttributes(false)
	if err != nil {
		return nil, err
	}

	node.Attributes = forwardedAttributes.Merge(attributes)

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

	node.Range.EndPos = p.lexer.Pos()

	return node, nil
}

// parseAttributes eats consecutive attributes from the lexer and returns them in an AttributeMap.
// forwarding specifies if regular or forwarding attributes should be parsed.
// The function returns when a non-attribute is encountered. Should an attribute be parsed
// that is the wrong type of forwarding, it will return an error.
func (p *Parser) parseAttributes(wantForward bool) (AttributeMap, error) {
	result := NewAttributeMap()

	for {

		tok, err := p.peek()
		if err != nil {
			break
		}

		if attr, ok := tok.(*DefineAttribute); ok {
			if wantForward && !attr.Forward {
				// TODO More user friendly error message
				return nil, fmt.Errorf("expected a forwarding attribute")
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
			break
		}

		attrKey := ""
		attrValue := ""

		// Read attribute key
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
			attrValue = cd.Value
		} else {
			return nil, NewUnexpectedTokenError(tok, TokenCharData)
		}

		result.Set(attrKey, attrValue)

		tok, _ = p.next()
		if tok.tokenType() != TokenBlockEnd {
			return nil, NewUnexpectedTokenError(tok, TokenBlockEnd)
		}

	}

	return result, nil
}
