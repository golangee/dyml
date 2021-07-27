package parser

import (
	"errors"
	"io"

	"github.com/golangee/tadl/token"
)

// Visitable defines the method signature of Objects
// that want to utilize the visitor
type Visitable interface {
	Close()

	NewNode(name string)
	NewTextNode(cd *token.CharData)
	NewCommentNode(cd *token.CharData)

	SetBlockType(t BlockType)
	SetStartPos(pos token.Pos)
	SetEndPos(pos token.Pos)

	GetRootBlockType() BlockType
	GetRange() token.Position
	GetForwardingLength() int
	GetForwardingAttributesLength() int
	GetForwardingPosition(i int) token.Node
	NodeIsClosedBy(tok token.Token) bool

	AddAttribute(key, value string)
	AddForwardAttribute(m AttributeMap)
	AddForwardNode(name string)
	MergeAttributes()
	MergeAttributesForwarded()
	AppendForwardingNodes()
	AppendSubTree()

	G2AddComments(cd *token.CharData)
	G2AppendComments()

	SwitchActiveTree()
}

// Visitor defines a visitor traversing a Syntaxtree based on Lexer output.
// Visitor calls the Methods defined in the Visitable interface to allow the
// overlying class to work with the tree.
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

	newNode  bool
	nestedG1 bool
	closed   bool
}

func NewVisitor(visit Visitable, lexer *token.Lexer) *Visitor {
	return &Visitor{
		visitMe:  visit,
		lexer:    lexer,
		newNode:  true,
		nestedG1: false,
		closed:   false,
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
	if v.visitMe.GetRootBlockType() != BlockNormal {
		return token.NewPosError(v.visitMe.GetRange(), "root element must have curly brackets")
	}

	return nil
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
	v.visitMe.SetStartPos(v.lexer.Pos())

	// Parse forwarding attributes
	err := v.parseAttributes(true)
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
		v.visitMe.NewTextNode(t)
		return nil
	case *token.G1Comment:
		// Expect CharData as comment
		tok, err = v.next()
		if err != nil {
			return err
		}

		if cd, ok := tok.(*token.CharData); ok {
			v.visitMe.NewCommentNode(cd)
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
		if forwardingNode {
			v.visitMe.AddForwardNode(id.Value)
		} else {
			v.visitMe.NewNode(id.Value)
		}
	} else {
		return token.NewPosError(
			tok.Pos(),
			"this token is not valid here",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenIdentifier))
	}

	// We now have a valid node.
	// Place our forwardingNodes inside it, if it is not one itself.
	if !forwardingNode && !v.nestedG1 {
		v.visitMe.AppendForwardingNodes()
	}

	// Process non-forwarding attributes.
	err = v.parseAttributes(false)
	if err != nil {
		return err
	}

	if forwardingNode {
		v.visitMe.MergeAttributesForwarded()
	} else {
		v.visitMe.MergeAttributes()
	}

	// Optional children enclosed in brackets
	tok, _ = v.peek()
	if tok.TokenType() == token.TokenBlockStart {
		v.next() // Pop the token, we know it's a BlockStart

		v.visitMe.SetBlockType(BlockNormal)

		// Append children until we encounter a TokenBlockEnd
		for {

			tok, _ = v.peek()
			if tok == nil {
				return errors.New("token not identified, is nil")
			}
			if tok.TokenType() == token.TokenBlockEnd {
				v.visitMe.Close()
				break
			}

			err := v.g1Node()
			if err != nil {
				return err
			}

			v.visitMe.Close()

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
	} else if tok.TokenType() == token.TokenCharData {
		v.next()

		v.visitMe.NewTextNode(tok.(*token.CharData))
	}

	if forwardingNode {
		// We just parsed a forwarding node. We need to save it, but cannot return it,
		// as it needs to be placed inside the next non-forwarding node.
		// We will parse another node to make it opaque to our caller that this happened.

		//v.visitMe.AppendSubTreeForward()
		v.g1Node()
		return nil
	}

	v.visitMe.SetEndPos(v.lexer.Pos())
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

	v.visitMe.SwitchActiveTree()
	v.nestedG1 = true
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
	v.visitMe.SwitchActiveTree()
	v.nestedG1 = false

	v.mode = token.G2

	// Should this be a forwarding G1 line, we will store the children for later
	// and return an empty array here.
	if !forward {
		v.visitMe.AppendSubTree()
	}
	return nil
}

// g2Node recursively parses a G2 node and all its children from tokens.
func (v *Visitor) g2Node() error {
	v.visitMe.SetStartPos(v.lexer.Pos())
	// Read forward attributes
	err := v.parseAttributes(true)
	if err != nil {
		return err
	}

	if err := v.g2EatComments(); err != nil {
		return err
	}

	// Expect identifier or text
	tok, err := v.next()
	if err != nil {
		return err
	}

	switch t := tok.(type) {
	case *token.Identifier:
		v.visitMe.NewNode(t.Value)
		// Insert forwarded nodes
		v.visitMe.AppendForwardingNodes()
	case *token.CharData:
		if v.visitMe.GetForwardingAttributesLength() > 0 {
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

	v.visitMe.AppendForwardingNodes()

	// Read attributes
	err = v.parseAttributes(false)
	if err != nil {
		return err
	}

	v.visitMe.MergeAttributes()

	if err := v.g2EatComments(); err != nil {
		return err
	}

	// Process children
	tok, err = v.peek()
	if err != nil {
		return err
	}

	switch t := tok.(type) {
	case *token.CharData:
		v.next()

		v.visitMe.NewTextNode(t)
	case *token.DefineElement:
		err := v.g1LineNodes()
		if err != nil {
			return err
		}

	case *token.BlockStart, *token.GenericStart, *token.GroupStart:
		v.next()

		// Set BlockType
		switch t.(type) {
		case *token.BlockStart:
			v.visitMe.SetBlockType(BlockNormal)
		case *token.GroupStart:
			v.visitMe.SetBlockType(BlockGroup)
		case *token.GenericStart:
			v.visitMe.SetBlockType(BlockGeneric)
		}

		// Parse children
		for {
			tok, err = v.peek()
			if err != nil {
				return err
			}

			if v.visitMe.NodeIsClosedBy(tok) {
				v.next() // pop closing token

				break
			} else if tok.TokenType() == token.TokenDefineElement {
				err := v.g1LineNodes()
				if err != nil {
					return err
				}
			} else {
				err := v.g2Node()
				if err != nil {
					return err
				}
			}
		}
	case *token.BlockEnd, *token.GroupEnd, *token.GenericEnd:
		// Any closing token ends this node and will be handled by the parent.
	case *token.Comma:
		// Comma ends a node definition

		v.visitMe.Close()
		v.closed = true
		v.next()
	case *token.G2Arrow:
		if err := v.g2ParseArrow(); err != nil {
			return err
		}
	default:
		err := v.g2Node()
		if err != nil {
			return err
		}

		v.visitMe.AppendSubTree()
	}

	tok, err = v.peek()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	if tok.TokenType() == token.TokenG2Arrow {
		if err := v.g2ParseArrow(); err != nil {
			return err
		}
	}

	v.visitMe.SetEndPos(v.lexer.Pos())
	if !v.closed {
		v.visitMe.Close()
	}
	v.closed = false

	return nil
}

// g2EatComments will read all G2 comments from the current lexer position and store them in
// p.g2Comments so that the can be placed in a sensible node with g2AppendComments.
func (v *Visitor) g2EatComments() error {
	for {
		tok, err := v.peek()
		if err != nil {
			// Do not report an error at this point, as some other function will handle it.
			break
		}

		if tok.TokenType() != token.TokenG2Comment {
			// The next thing is not a comment, which means that we are done.
			break
		}

		v.next() // Pop G2Comment

		tok, err = v.next()
		if err != nil {
			return err
		}

		// Expect CharData as comment
		if cd, ok := tok.(*token.CharData); ok {
			v.visitMe.G2AddComments(cd)
		} else {
			return token.NewPosError(
				tok.Pos(),
				"empty comment is not valid",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
		}
	}
	v.visitMe.G2AppendComments()

	return nil
}

// g2ParseBlock parses a block and its children into the given node.
// The blockType of the node will be set to the type of the block.
func (v *Visitor) g2ParseBlock() error {
	tok, err := v.next()
	if err != nil {
		return err
	}

	// Set BlockType
	switch tok.(type) {
	case *token.BlockStart:
		v.visitMe.SetBlockType(BlockNormal)
	case *token.GroupStart:
		v.visitMe.SetBlockType(BlockGroup)
	case *token.GenericStart:
		v.visitMe.SetBlockType(BlockGeneric)
	default:
		return token.NewPosError(tok.Pos(), "expected a BlockStart")
	}

	// Parse children
	for {
		if err := v.g2EatComments(); err != nil {
			return err
		}

		v.visitMe.G2AppendComments()

		tok, err = v.peek()
		if err != nil {
			return err
		}

		if v.visitMe.NodeIsClosedBy(tok) {
			v.next() // pop closing token

			break
		} else if tok.TokenType() == token.TokenDefineElement {
			err := v.g1LineNodes()
			if err != nil {
				return err
			}
		} else {
			err := v.g2Node()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// g2ParseArrow is used to parse the return arrow, which has special semantics.
// It is used to append a "ret" element containing function return values to a
// function definition. For this to work, the function must be defined as:
//     name(...) -> (...)
// or
//     name -> (...)
// The "name" element will get a new child named "ret" appended that contains
// all children in the block after "->".
// The block "(...)" is required after the arrow, but can be any valid block.
func (v *Visitor) g2ParseArrow() error {
	// Expect arrow
	tok, err := v.next()
	if err != nil {
		return err
	}

	if tok.TokenType() != token.TokenG2Arrow {
		return token.NewPosError(tok.Pos(), "'->' expected")
	}

	v.visitMe.NewNode("ret")

	if err := v.g2ParseBlock(); err != nil {
		return err
	}

	v.visitMe.Close()
	return nil
}

// parseAttributes eats consecutive attributes from the lexer and returns them in an AttributeMap.
// forwarding specifies if regular or forwarding attributes should be parsed.
// The function returns when a non-attribute is encountered. Should an attribute be parsed
// that is the wrong type of forwarding, it will return an error.
// This function can read attributes in modes G1, G2.
func (v *Visitor) parseAttributes(wantForward bool) error {
	result := NewAttributeMap()

	isG1 := v.mode == token.G1 || v.mode == token.G1Line

	for {
		tok, err := v.peek()
		if err != nil {
			break
		}

		if attr, ok := tok.(*token.DefineAttribute); ok {
			if wantForward && !attr.Forward {
				return token.NewPosError(
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
			return err
		}

		if ident, ok := tok.(*token.Identifier); ok {
			attrKey = ident.Value
		} else {
			return token.NewPosError(
				tok.Pos(),
				"an identifier is required as an attribute key",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenIdentifier))
		}

		if result.Has(attrKey) {
			return token.NewPosError(
				tok.Pos(),
				"cannot define same attribute twice",
			)
		}

		// Read CharData enclosed in brackets as attribute value in G1.
		// Read CharData after Assign in G2.

		tok, _ = v.next()
		if isG1 {
			if tok.TokenType() != token.TokenBlockStart {
				return token.NewPosError(
					tok.Pos(),
					"attribute value must be enclosed in '{}'",
				).SetCause(NewUnexpectedTokenError(tok, token.TokenBlockStart))
			}
		} else {
			if tok.TokenType() != token.TokenAssign {
				return token.NewPosError(
					tok.Pos(),
					"'=' is expected here",
				).SetCause(NewUnexpectedTokenError(tok, token.TokenAssign))
			}
		}

		tok, err = v.next()
		if err != nil {
			return err
		}

		if cd, ok := tok.(*token.CharData); ok {
			attrValue = cd.Value
		} else {
			return token.NewPosError(
				tok.Pos(),
				"attribute value is required",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
		}

		result.Set(attrKey, attrValue)

		if isG1 {
			tok, _ = v.next()
			if tok.TokenType() != token.TokenBlockEnd {
				return token.NewPosError(
					tok.Pos(),
					"attribute value needs to be closed with '}'",
				).SetCause(NewUnexpectedTokenError(tok, token.TokenBlockEnd))
			}
		}
	}

	if wantForward {
		v.visitMe.AddForwardAttribute(result)
	} else {
		for key, val := range result {
			v.visitMe.AddAttribute(key, val)
		}
	}

	return nil
}
