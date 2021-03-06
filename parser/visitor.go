package parser

import (
	"errors"
	"io"

	"github.com/golangee/dyml/token"
)

// Visitable must be implemented by all things that can handle events from the push-parser.
// All methods can return an error. Should any error be encountered, parsing will be
// stopped immediately.
type Visitable interface {
	// Open marks the beginning of a new node with a given name. The BlockType will be set later
	// by a call to SetBlockType.
	// Its end will be marked by a call to Close.
	// There are special cases for return arrows which will be handled by OpenReturnArrow.
	Open(name token.Identifier) error
	// Comment marks the occurrence of a comment.
	Comment(comment token.CharData) error
	// Text marks the occurrence of a text.
	Text(text token.CharData) error

	// OpenReturnArrow marks the occurrence of a block after a return arrow, analogous to Open().
	OpenReturnArrow(arrow token.G2Arrow, name *token.Identifier) error
	// CloseReturnArrow will be called after all elements "in this return" have been handled,
	// analogous to Close().
	CloseReturnArrow() error

	// SetBlockType sets the BlockType of the node that was most recently Open-ed.
	SetBlockType(blockType BlockType) error

	// OpenForward is the same as Open, but for forwarded nodes.
	OpenForward(name token.Identifier) error
	// TextForward is the same as Text, but for forwarded text.
	TextForward(text token.CharData) error

	// Close the currently active node. For each call to Open or OpenForward there will be a
	// call to Close.
	Close() error

	// Attribute is an attribute that should be applied to the current node.
	Attribute(key token.Identifier, value token.CharData) error
	// AttributeForward is an attribute that should be applied to the next node.
	AttributeForward(key token.Identifier, value token.CharData) error

	// Finalize will be called after the whole input has been parsed.
	// You may do additional validation here.
	Finalize() error
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
	// BlockNone represents no BlockType.
	BlockNone BlockType = ""
	// BlockNormal represents curly brackets.
	BlockNormal BlockType = "{}"
	// BlockGroup represents round brackets.
	BlockGroup BlockType = "()"
	// BlockGeneric represents pointed brackets.
	BlockGeneric BlockType = "<>"
	// blockSpecial puts closeNode() in a special mode. See that method for more details.
	blockSpecial BlockType = "*"
)

// Visitor defines a visitor traversing a Syntaxtree based on Lexer output.
// Visitor calls the Methods defined in the Visitable interface to allow the
// overlying class to work with the tree.
type Visitor struct {
	visitMe Visitable

	lexer *token.Lexer
	// tokenBuffer contains all tokens that need to be processed next.
	// These could be peeked tokens or tokens that were added in the parser.
	// When it is empty, we can call lexer.Token() to get the next token.
	tokenBuffer []tokenWithError

	// tokenTailBuffer contains all tokens that need to be processed once
	// lexer.Token() returns no more tokens. tokenTailBuffer will contain
	// tokens that were added from parser code.
	tokenTailBuffer []tokenWithError

	mode token.GrammarMode

	// openNodes is a stack of all blocktypes that are currently
	// opened. These can be used to check whether a block is closed
	// with the correct type of bracket and to keep track of open
	// nodes.
	openNodes []BlockType
}

// NewVisitor creates a new visitor that can be start with Run().
// You need to call SetVisitable before that!
func NewVisitor(filename string, reader io.Reader) *Visitor {
	return &Visitor{
		lexer: token.NewLexer(filename, reader),
	}
}

// SetVisitable sets the visitMe field to an implementation of the Visitable interface.
func (v *Visitor) SetVisitable(vis Visitable) {
	v.visitMe = vis
}

// Run runs the visitor, starting the traversion of the syntax tree.
func (v *Visitor) Run() error {
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

	err := v.g1Node()
	if err != nil {
		return err
	}

	// Close remaining nodes
	for len(v.openNodes) > 0 {
		if err := v.closeNode(); err != nil {
			return err
		}
	}

	if err := v.visitMe.Finalize(); err != nil {
		return err
	}

	return nil
}

// closeNode closes the currently processed node.
// It will remove the top element from the openNodes stack, and will call Close() on our callback.
// BUT If there is a blockSpecial under the topmost element of the stack, then the callback will
// not be called, which is useful for handling the G2Arrow.
func (v *Visitor) closeNode() error {
	v.openNodes = v.openNodes[:len(v.openNodes)-1]

	if !v.isCurrentNodeSpecial() {
		return v.visitMe.Close()
	}

	return nil
}

// openNode opens a new node for processing.
func (v *Visitor) openNode(name token.Identifier) error {
	v.openNodes = append(v.openNodes, BlockNone)

	return v.visitMe.Open(name)
}

// openForwardNode opens a new forwarding node for processing.
func (v *Visitor) openForwardNode(name token.Identifier) error {
	v.openNodes = append(v.openNodes, BlockNone)

	return v.visitMe.OpenForward(name)
}

// setBlockType set the BlockType of the currently processed node.
func (v *Visitor) setBlockType(blockType BlockType) error {
	if v.openNodes[len(v.openNodes)-1] != blockSpecial {
		v.openNodes[len(v.openNodes)-1] = blockType
	}

	return v.visitMe.SetBlockType(blockType)
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
	var isForwardingNode bool

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
		// Correctly set the forwarding mode.
		if v.mode == token.G1LineForward || v.mode == token.G1Line {
			if t.Forward {
				return token.NewPosError(t.Pos(), "cannot forward nodes in G1 lines")
			}
		}

		if v.mode == token.G1LineForward {
			isForwardingNode = true
		} else {
			isForwardingNode = t.Forward
		}
	case *token.CharData:
		if v.mode == token.G1LineForward {
			if err := v.visitMe.TextForward(*t); err != nil {
				return err
			}
		} else {
			if err := v.visitMe.Text(*t); err != nil {
				return err
			}
		}

		return nil
	case *token.G1Comment:
		// Expect CharData as comment
		tok, err = v.next()
		if err != nil {
			return err
		}

		if cd, ok := tok.(*token.CharData); ok {
			err = v.visitMe.Comment(*cd)
			if err != nil {
				return err
			}

			return nil
		}

		return token.NewPosError(
			tok.Pos(),
			"expected a comment",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
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
		if isForwardingNode {
			if err := v.openForwardNode(*id); err != nil {
				return err
			}
		} else {
			if err := v.openNode(*id); err != nil {
				return err
			}
		}
	} else {
		return token.NewPosError(
			tok.Pos(),
			"this token is not valid here",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenIdentifier))
	}

	// Process non-forwarding attributes.
	err = v.parseAttributes(false)
	if err != nil {
		return err
	}

	// Optional children enclosed in brackets
	tok, err = v.peek()
	if err != nil {
		return err
	}

	switch t := tok.(type) {
	case *token.BlockStart:
		_, err = v.next() // Pop the token, we know it's a BlockStart
		if err != nil {
			return err
		}

		if err := v.setBlockType(BlockNormal); err != nil {
			return err
		}

		// Append children until we encounter a TokenBlockEnd
	collect:
		for {
			tok, err = v.peek()
			if err != nil {
				return err
			}

			switch tok.(type) {
			case *token.BlockEnd:
				// The block was closed
				break collect
			case *token.G2Preamble:
				if v.mode == token.G1 {
					// Parse a single G2 node
					_, _ = v.next() // pop preamble
					v.mode = token.G2

					if err := v.g2Node(); err != nil {
						return err
					}

					v.mode = token.G1
				} else {
					return token.NewPosError(tok.Pos(), "G2 node not allowed here")
				}
			default:
				// Anything else must be another g1Node
				if err := v.g1Node(); err != nil {
					return err
				}
			}
		}

		// Expect a BlockEnd
		tok, err = v.next()
		if err != nil {
			return err
		}

		if tok.Type() != token.TokenBlockEnd {
			return token.NewPosError(
				tok.Pos(),
				"use a '}' here to close the element",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenBlockEnd))
		}
	case *token.CharData:
		_, err = v.next()
		if err != nil {
			return err
		}

		err = v.visitMe.Text(*t)
		if err != nil {
			return err
		}
	}

	if err := v.closeNode(); err != nil {
		return err
	}

	return nil
}

// g1LineNodes processes all nodes that were encountered in a G1 line.
// This function will eat the beginning DefineElement and the ending G1LineEnd token.
func (v *Visitor) g1LineNodes() error {
	// Expect beginning '#'
	tok, err := v.next()
	if err != nil {
		return err
	}

	// Set mode to G1Line or G1LineForward depending on the token.
	if de, ok := tok.(*token.DefineElement); ok {
		if de.Forward {
			v.mode = token.G1LineForward
		} else {
			v.mode = token.G1Line
		}
	} else {
		return token.NewPosError(
			tok.Pos(),
			"start of G1 line expected",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenDefineElement))
	}

	for {
		tok, _ := v.peek()
		if tok != nil && tok.Type() == token.TokenG1LineEnd {
			_, err = v.next()
			if err != nil {
				return err
			}

			break
		}

		// Read g1Nodes until we encounter G1LineEnd
		err := v.g1Node()
		if err != nil {
			return err
		}
	}

	// Restore mode
	v.mode = token.G2

	return nil
}

// g2Node recursively parses a G2 node and all its children from tokens.
func (v *Visitor) g2Node() error {
	if err := v.g2EatComments(); err != nil {
		return err
	}

	// true if an arrow following this node is allowed.
	arrowAllowed := true //nolint:ifshort

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
		if err := v.openNode(*t); err != nil {
			return err
		}
	case *token.CharData:
		err := v.visitMe.Text(*t)
		if err != nil {
			return err
		}

		v.maybeEatComma()

		return nil
	default:
		return token.NewPosError(
			tok.Pos(),
			"this token is not valid here",
		).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData, token.TokenIdentifier))
	}

	// Read attributes
	err = v.parseAttributes(false)
	if err != nil {
		return err
	}

	if err := v.g2EatComments(); err != nil {
		return err
	}

	// Process children
	tok, err = v.peek()
	if err != nil {
		return err
	}

	switch tok.(type) {
	case *token.DefineElement:
		err := v.g1LineNodes()
		if err != nil {
			return err
		}

	case *token.BlockStart, *token.GenericStart, *token.GroupStart:
		err = v.g2ParseBlock()
		if err != nil {
			return err
		}

	case *token.BlockEnd, *token.GroupEnd, *token.GenericEnd:
		// Close the current node but leave the token so that the parent of this node
		// can be closed too.
	case *token.Comma, *token.Semicolon:
		// Comma/Semicolon ends a node definition
		_, err = v.next() // Pop the Comma/Semicolon
		if err != nil {
			return err
		}

		arrowAllowed = false
	case *token.G2Arrow:
		// This is a G2Arrow after an identifier
		// It ends the current element, but will not pop the token so that it can
		// be parsed correctly later.
	default:
		err := v.g2Node()
		if err != nil {
			return err
		}
	}

	if err := v.g2EatComments(); err != nil {
		return err
	}

	tok, err = v.peek()
	if err != nil {
		if errors.Is(err, io.EOF) {
			// There is no more input, close this node.
			return v.closeNode()
		}

		return err
	}

	// We have to handle the arrow before closing the node.
	if arrowAllowed && tok.Type() == token.TokenG2Arrow {
		if err := v.g2ParseArrow(); err != nil {
			return err
		}
	}

	return v.closeNode()
}

// g2EatComments will read all G2 comments from the lexer.
func (v *Visitor) g2EatComments() error {
	for {
		tok, err := v.peek()
		if err != nil {
			// Do not report an error at this point, as some other function will handle it.
			break
		}

		if tok.Type() != token.TokenG2Comment {
			// The next thing is not a comment, which means that we are done.
			break
		}

		_, err = v.next() // Pop G2Comment
		if err != nil {
			return err
		}

		tok, err = v.next()
		if err != nil {
			return err
		}

		// Expect CharData as comment
		if cd, ok := tok.(*token.CharData); ok {
			err = v.visitMe.Comment(*cd)
			if err != nil {
				return err
			}
		} else {
			return token.NewPosError(
				tok.Pos(),
				"empty comment is not valid",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
		}
	}

	return nil
}

// g2ParseBlock parses a block and its children into the given node.
// The blockType of the node will be set to the type of the block.
func (v *Visitor) g2ParseBlock() error {
	tok, err := v.next()
	if err != nil {
		return err
	}

	var blockType BlockType

	// Set BlockType
	switch tok.(type) {
	case *token.BlockStart:
		blockType = BlockNormal

		err = v.setBlockType(BlockNormal)
		if err != nil {
			return err
		}

	case *token.GroupStart:
		blockType = BlockGroup

		err = v.setBlockType(BlockGroup)
		if err != nil {
			return err
		}

	case *token.GenericStart:
		blockType = BlockGeneric

		err = v.setBlockType(BlockGeneric)
		if err != nil {
			return err
		}
	default:
		return token.NewPosError(tok.Pos(), "expected a BlockStart")
	}

	// Parse children
	for {
		if err := v.g2EatComments(); err != nil {
			return err
		}

		if err != nil {
			return err
		}

		tok, err = v.peek()
		if err != nil {
			return err
		}

		if correctClosingToken(blockType, tok) {
			_, err = v.next() // pop closing token
			if err != nil {
				return err
			}

			break // Stop parsing the block, closing the current node will be handled by the caller
		} else if tok.Type() == token.TokenDefineElement {
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

// correctClosingToken returns true if the token is a closing token for the given BlockType.
func correctClosingToken(blockType BlockType, tok token.Token) bool {
	switch tok.(type) {
	case *token.BlockEnd:
		return blockType == BlockNormal
	case *token.GenericEnd:
		return blockType == BlockGeneric
	case *token.GroupEnd:
		return blockType == BlockGroup
	default:
		return false
	}
}

// g2ParseArrow is used to parse the return arrow, which has special semantics.
// It is used to append a "ret" element containing function return values to a
// function definition. For this to work, the function must be defined as:
//     name(...) -> [opt](...)
// The "name" element will get a new child named "ret" appended that contains
// all children in the block after "->". The block after name is optional.
// The block "(...)" is required after the arrow, but can be any valid block with
// or without a name.
// After this method has been called the topmost element in openNodes will be a blockSpecial,
// which you need to handle.
func (v *Visitor) g2ParseArrow() error {
	// Expect arrow
	tok, err := v.next()
	if err != nil {
		return err
	}

	switch t := tok.(type) {
	case *token.G2Arrow:
		// There can be an optional identifier as a name for the block.
		var name *token.Identifier

		tok, err := v.peek()
		if err != nil {
			return err
		}

		switch tokName := tok.(type) {
		case *token.Identifier:
			// Use the identifier as a name, pop the token
			if _, err := v.next(); err != nil {
				return err
			}

			name = tokName
		}

		// closeNode has a special mode, when blockSpecial is on the stack, see that method
		// for more details.
		v.openNodes = append(v.openNodes, blockSpecial, BlockNone)

		err = v.visitMe.OpenReturnArrow(*t, name)
		if err != nil {
			return err
		}

		// Try parsing a block if there is one
		tok, err = v.peek()
		if err == nil {
			switch tok.(type) {
			case *token.BlockStart, *token.GroupStart, *token.GenericStart:
				if err := v.g2ParseBlock(); err != nil {
					return err
				}
			}
		}

		v.openNodes = v.openNodes[:len(v.openNodes)-2]

		err = v.visitMe.CloseReturnArrow()
		if err != nil {
			return err
		}

		return nil
	default:
		return token.NewPosError(tok.Pos(), "'->' expected")
	}
}

// parseAttributes eats consecutive attributes from the lexer.
// wantForward specifies if regular or forwarding attributes should be parsed.
// The function returns when a non-attribute is encountered. Should an attribute be parsed
// that is the wrong type of forwarding, it will return an error.
// This function can read attributes in modes G1, G2.
func (v *Visitor) parseAttributes(wantForward bool) error {
	isG1 := v.mode == token.G1 || v.mode == token.G1Line || v.mode == token.G1LineForward

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

			_, err = v.next() // pop DefineAttribute
			if err != nil {
				return err
			}
		} else {
			// The next token is not a DefineAttribute
			break
		}

		var attrKey token.Identifier

		var attrValue token.CharData

		// Read attribute key
		tok, err = v.next()
		if err != nil {
			return err
		}

		if ident, ok := tok.(*token.Identifier); ok {
			attrKey = *ident
		} else {
			return token.NewPosError(
				tok.Pos(),
				"an identifier is required as an attribute key",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenIdentifier))
		}

		// Read CharData enclosed in brackets as attribute value in G1.
		// Read CharData after Assign in G2.

		tok, err = v.next()
		if err != nil {
			return err
		}

		if isG1 {
			if tok.Type() != token.TokenBlockStart {
				return token.NewPosError(
					tok.Pos(),
					"attribute value must be enclosed in '{}'",
				).SetCause(NewUnexpectedTokenError(tok, token.TokenBlockStart))
			}
		} else {
			if tok.Type() != token.TokenAssign {
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
			attrValue = *cd
		} else {
			return token.NewPosError(
				tok.Pos(),
				"attribute value is required",
			).SetCause(NewUnexpectedTokenError(tok, token.TokenCharData))
		}

		if wantForward {
			if err := v.visitMe.AttributeForward(attrKey, attrValue); err != nil {
				return err
			}
		} else {
			if err := v.visitMe.Attribute(attrKey, attrValue); err != nil {
				return err
			}
		}

		if isG1 {
			tok, err = v.next()
			if err == nil && tok.Type() != token.TokenBlockEnd {
				return token.NewPosError(
					tok.Pos(),
					"attribute value needs to be closed with '}'",
				).SetCause(NewUnexpectedTokenError(tok, token.TokenBlockEnd))
			}
		}
	}

	return nil
}

func (v *Visitor) isCurrentNodeSpecial() bool {
	return len(v.openNodes) > 0 && v.openNodes[len(v.openNodes)-1] == blockSpecial
}

// maybeEatComma will pop the next token from the lexer, if it is a token.Comma (or token.Semicolon).
// This is useful for allowing trailing commas.
func (v *Visitor) maybeEatComma() {
	tok, err := v.peek()
	if err != nil {
		return
	}

	if tok.Type() == token.TokenComma {
		_, _ = v.next()
	}
}
