package parser

import (
	"errors"
	"io"

	"github.com/golangee/tadl/token"
)

type Visitable interface {
	AddForwardNode(name string)
	AppendForwardingNodes()
	NewNode(name string)
	NewStringNode(name string)
	NewTextNode(cd *token.CharData)
	NewCommentNode(cd *token.CharData)
	NewStringCommentNode(name string)
	MergeAttributes(m AttributeMap)
	AddAttribute(key, value string)
	AddForwardAttribute(m AttributeMap)
	GetForwardingLength() int
	GetForwardingPosition(i int) token.Node
	SetNodeName(name string)
	SetNodeText(text string)
	SetBlockType(t BlockType)
	GetBlockType() BlockType
	Open()
	Close()
	AppendSubTree()
	AppendSubTreeForward()
	SetEndPos(pos token.Pos)
	GetPointerPosition() token.Position
}

type Visitor struct {
	visitMe Visitable

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

	// Holds the current nodes Position
	position token.Position

	forwardMode bool
	newNode     bool
}

func NewVisitor(visit Visitable, lexer *token.Lexer) *Visitor {
	return &Visitor{
		visitMe: visit,
		lexer:   lexer,
		newNode: true,
	}
}

func (v *Visitor) SetVisitable(vis Visitable) {
	v.visitMe = vis
}

func (v *Visitor) Run() error {
	v.newNode = true
	// Peek the first token to check if we should set G2 mode.
	tok, err := v.peek()

	// Edge case: When the input is empty we do not want the EOF in our buffer, as we will append tailTokens later.
	if errors.Is(err, io.EOF) {
		v.next()
	}

	v.visitMe.NewNode("root")

	if tok != nil && tok.TokenType() == token.TokenG2Preamble {
		// Prepare G2 by switching out the preamble for a root identifier.
		v.mode = token.G2
		v.next()
		v.tokenBuffer = append(v.tokenBuffer,
			tokenWithError{tok: &token.Identifier{Value: "root"}},
		)

		err = v.g2Node()
		if err != nil {
			return err
		}
	} else {
		// Prepare G1.
		// Prepend and append tokens for the root element.
		// This makes the root just another element, which simplifies parsing a lot.
		v.tokenBuffer = append([]tokenWithError{
			{tok: &token.DefineElement{}},
			{tok: &token.Identifier{Value: "root"}},
			{tok: &token.BlockStart{}},
		},
			v.tokenBuffer...,
		)

		v.tokenTailBuffer = append(v.tokenTailBuffer,
			tokenWithError{tok: &token.BlockEnd{}},
		)

		err = v.g1Node()
		if err != nil {
			return err
		}
	}

	// All forwarding nodes should have been processed earlier.
	if v.visitMe.GetForwardingLength() > 0 {
		return token.NewPosError(v.visitMe.GetForwardingPosition(0), "there is no node to forward this node into")
	}

	// The root element should always have curly brackets.
	if v.visitMe.GetBlockType() != BlockNormal {
		return token.NewPosError(v.visitMe.GetPointerPosition(), "root element must have curly brackets")
	}

	return nil
}

func (v *Visitor) prep() {

}

// next returns the next token or (nil, io.EOF) if there are no more tokens.
// Repeatedly calling this can be used to get all tokens by advancing the lexer.
func (v *Visitor) next() (token.Token, error) {
	// Check the buffer for tokens
	if len(v.tokenBuffer) > 0 {
		twe := v.tokenBuffer[0]
		v.tokenBuffer = v.tokenBuffer[1:] // pop token

		return twe.tok, twe.err
	}

	tok, err := v.lexer.Token()

	if errors.Is(err, io.EOF) {
		// Check tail buffer for tokens that need to be appended
		if len(v.tokenTailBuffer) > 0 {
			twe := v.tokenTailBuffer[0]
			v.tokenTailBuffer = v.tokenTailBuffer[1:] // pop token

			// Tail tokens are generated and have no positional information associated.
			// We fix that here, so that potential errors point to the right place.
			if twe.tok != nil {
				lexPos := v.lexer.Pos()
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
func (v *Visitor) peek() (token.Token, error) {
	// Check the buffer for tokens
	if len(v.tokenBuffer) > 0 {
		twe := v.tokenBuffer[0]
		return twe.tok, twe.err
	}

	tok, err := v.next()

	// Store token+error for use in next()
	v.tokenBuffer = append(v.tokenBuffer, tokenWithError{
		tok: tok,
		err: err,
	})

	return tok, err
}

// g1Node recursively parses a G1 node and all its children from tokens.
func (v *Visitor) g1Node() error {
	forwardingNode := false
	v.visitMe.NewNode("invalid name") // name will be set later
	v.position.BeginPos = v.lexer.Pos()

	// Parse forwarding attributes
	forwardedAttributes, err := v.parseAttributes(true)
	if err != nil {
		return err
	}

	// Expect ElementDefinition or CharData
	tok, err := v.next()
	if err != nil {
		return err
	}

	switch t := tok.(type) {
	case *token.DefineElement:
		forwardingNode = t.Forward
	case *token.CharData:
		v.visitMe.SetNodeText(t.Value)
		v.newNode = false

		return nil
	case *token.G1Comment:
		// Expect CharData as comment
		tok, err = v.next()
		if err != nil {
			return err
		}

		if cd, ok := tok.(*token.CharData); ok {
			v.visitMe.NewCommentNode(cd)
			v.visitMe.Close()
			v.newNode = false
			v.visitMe.Close()
			return nil
		} else {
			return token.NewPosError(
				tok.Pos(),
				"expected a comment",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
		}
	default:
		return token.NewPosError(
			tok.Pos(),
			"this token is not valid here",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenDefineElement, token.TokenCharData))
	}

	// Expect identifier for new element
	tok, err = v.next()
	if err != nil {
		return err
	}

	if id, ok := tok.(*token.Identifier); ok {
		if v.newNode {
			v.visitMe.NewNode(id.Value)
			v.newNode = false
		} else {
			v.visitMe.SetNodeName(id.Value)
		}
	} else {
		return token.NewPosError(
			tok.Pos(),
			"this token is not valid here",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenIdentifier))
	}

	// We now have a valid node.
	// Place our forwardingNodes inside it, if it is not one itself.
	if !forwardingNode {
		v.visitMe.AppendForwardingNodes()
	}

	// Process non-forwarding attributes.
	attributes, err := v.parseAttributes(false)
	if err != nil {
		return err
	}

	v.visitMe.MergeAttributes(attributes.Merge(forwardedAttributes))

	// Optional children enclosed in brackets
	tok, _ = v.peek()
	if tok.TokenType() == token.TokenBlockStart {
		v.next() // Pop the token, we know it's a BlockStart

		v.visitMe.SetBlockType(BlockNormal)

		// Append children until we encounter a TokenBlockEnd
		for {
			tok, _ = v.peek()
			if tok.TokenType() == token.TokenBlockEnd {
				v.visitMe.Close()
				break
			}

			err := v.g1Node()
			if err != nil {
				return err
			}
		}

		// Expect a BlockEnd
		tok, err = v.next()
		if err != nil {
			return err
		}

		if tok.TokenType() != token.TokenBlockEnd {
			return token.NewPosError(
				tok.Pos(),
				"use a '}' here to close the element",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenBlockEnd))
		}
	}

	if forwardingNode {
		// We just parsed a forwarding node. We need to save it, but cannot return it,
		// as it needs to be placed inside the next non-forwarding node.
		// We will parse another node to make it opaque to our caller that this happened.

		//v.visitMe.CurrentToGlobalTree
		v.visitMe.AppendSubTreeForward()
		v.g1Node()
		return nil
	}

	v.visitMe.SetEndPos(v.lexer.Pos())
	v.visitMe.Close()
	return nil
}

// g1LineNodes returns all nodes that were encountered in a G1 line.
// This function will eat the beginning DefineElement and the ending G1LineEnd token.
func (v *Visitor) g1LineNodes() error {
	// Expect beginning '#'
	tok, err := v.next()
	if err != nil {
		return err
	}

	var forward bool

	if de, ok := tok.(*token.DefineElement); ok {
		forward = de.Forward
	} else {
		return token.NewPosError(
			tok.Pos(),
			"start of G1 line expected",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenDefineElement))
	}

	v.mode = token.G1Line

	for {
		tok, _ := v.peek()
		if tok != nil && tok.TokenType() == token.TokenG1LineEnd {
			v.next()
			break
		}

		// Read g1Nodes until we encounter G1LineEnd
		err := v.g1Node()
		if err != nil {
			return err
		}
	}

	v.mode = token.G2

	// Should this be a forwarding G1 line, we will store the children for later
	// and return an empty array here.
	if forward {
		v.visitMe.AppendSubTreeForward()
		return nil
	} else {
		v.visitMe.AppendSubTree()
		return nil
	}
}

// g2Node recursively parses a G2 node and all its children from tokens.
func (v *Visitor) g2Node() error {
	node := NewNode("invalid name") // name will be set later
	node.Range.BeginPos = v.lexer.Pos()

	// Read forward attributes
	forwardedAttributes, err := v.parseAttributes(true)
	if err != nil {
		return err
	}

	// Expect identifier or text
	tok, err := v.next()
	if err != nil {
		return err
	}

	switch t := tok.(type) {
	case *token.Identifier:
		node.Name = t.Value
		// Insert forwarded nodes
		v.visitMe.AppendForwardingNodes()
	case *token.CharData:
		if len(forwardedAttributes) > 0 {
			// We have forwarded attributes for a text, where an identifier would be appropriate.
			return token.NewPosError(
				tok.Pos(),
				"attributes cannot be forwarded into this text",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
		}
		v.visitMe.NewTextNode(t)
		return nil
	default:
		return token.NewPosError(
			tok.Pos(),
			"this token is not valid here",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData, token.TokenIdentifier))
	}

	// Read attributes
	attributes, err := v.parseAttributes(false)
	if err != nil {
		return err
	}

	node.Attributes = forwardedAttributes.Merge(attributes)

	// Process children
	tok, err = v.peek()
	if err != nil {
		return err
	}

	switch t := tok.(type) {
	case *token.CharData:
		v.next()

		node.AddChildren(NewTextNode(t))
	case *token.DefineElement:
		err := v.g1LineNodes()
		if err != nil {
			return err
		}

		v.visitMe.AppendSubTree()
	case *token.BlockStart, *token.GenericStart, *token.GroupStart:
		v.next()

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
			tok, err = v.peek()
			if err != nil {
				return err
			}

			if node.isClosedBy(tok) {
				v.next() // pop closing token

				break
			} else if tok.TokenType() == token.TokenDefineElement {
				err := v.g1LineNodes()
				if err != nil {
					return err
				}
				v.visitMe.AppendSubTree()
			} else {
				err := v.g2Node()
				if err != nil {
					return err
				}

				v.visitMe.AppendSubTree()
			}
		}
	case *token.BlockEnd, *token.GroupEnd, *token.GenericEnd:
		// Any closing token ends this node and will be handled by the parent.
	case *token.Comma:
		// Comma ends a node definition
		v.next()
	default:
		err := v.g2Node()
		if err != nil {
			return err
		}

		v.visitMe.AppendSubTree()
	}

	node.Range.EndPos = v.lexer.Pos()

	return nil
}

// parseAttributes eats consecutive attributes from the lexer and returns them in an AttributeMap.
// forwarding specifies if regular or forwarding attributes should be parsed.
// The function returns when a non-attribute is encountered. Should an attribute be parsed
// that is the wrong type of forwarding, it will return an error.
// This function can read attributes in modes G1, G2.
func (v *Visitor) parseAttributes(wantForward bool) (AttributeMap, error) {
	result := NewAttributeMap()

	isG1 := v.mode == token.G1 || v.mode == token.G1Line

	for {
		tok, err := v.peek()
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

			v.next() // pop DefineAttribute
		} else {
			// The next token is not a DefineAttribute
			break
		}

		var attrKey, attrValue string

		// Read attribute key
		tok, err = v.next()
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

		tok, _ = v.next()
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

		tok, err = v.next()
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
			tok, _ = v.next()
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
