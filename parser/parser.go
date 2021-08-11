// SPDX-FileCopyrightText: © 2021 The tadl authors <https://github.com/golangee/tadl/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"io"

	"github.com/golangee/tadl/token"
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
	Parent     *TreeNode
	Children   []*TreeNode
	// BlockType describes the type of brackets the children were surrounded with.
	// This may be BlockNone in which case this node either has no or one children.
	BlockType BlockType
	// Range will span all tokens that were processed to build this node.
	Range token.Position
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

// isClosedBy returns true if tok is a BlockEnd/GroupEnd/GenericEnd that is the correct
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

func (t *TreeNode) Print() string {
	text := t.Name
	for _, child := range t.Children {
		text += child.Print()
	}

	return text
}

// unbindParents recursively sets all parent Pointers of a tree to nil
func unbindParents(t *TreeNode) {
	t.Parent = nil
	for _, child := range t.Children {
		unbindParents(child)
	}
}

/*
// AttributeMap is a custom map[string]string to make the
// handling of attributes easier.
type AttributeMap map[string]string

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
	result := NewAttributeList()

	for k, v := range a {
		result[k] = v
	}

	for k, v := range other {
		result[k] = v
	}

	return result
}*/

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
	//forwardingAttributes contains all Attributes that have been forwarded to be added to the next viable node.
	forwardingAttributes *AttributeList

	// root and parent are pointers to work with the successively built Tree.
	// root holds the root Node, parent holds the currently to modify Node
	root   *TreeNode
	parent *TreeNode

	// root- and parentForward have the same functionality as root and parent.
	// they are used to create full trees being forwarded, added later to the main tree
	rootForward   *TreeNode
	parentForward *TreeNode

	// g2Comments contains all comments in G2 that were eaten from the input,
	// but are not yet placed in a sensible position.
	g2Comments []*TreeNode

	visitor Visitor

	firstNode     bool
	globalForward bool
}

// NewParser creates and returns a new Parser with corresponding Visitor
func NewParser(filename string, r io.Reader) *Parser {
	parser := &Parser{
		visitor:       *NewVisitor(nil, token.NewLexer(filename, r)),
		globalForward: false,
		rootForward:   NewNode("root").Block(BlockNormal),
	}
	parser.parentForward = parser.rootForward
	parser.visitor.SetVisitable(parser)
	parser.firstNode = true
	return parser
}

// Parse returns a parsed tree.
func (p *Parser) Parse() (*TreeNode, error) {
	err := p.visitor.Run()
	if err != nil {
		return nil, err
	}
	unbindParents(p.root)

	return p.root, nil
}

// open sets the parent pointer to the latest Child of it's current Node
func (p *Parser) open() {
	p.parent = p.parent.Children[len(p.parent.Children)-1]
}

// Close moves the parent pointer to its current parent Node
func (p *Parser) Close() error {
	if p.parent.Parent != nil {
		p.parent = p.parent.Parent
	}
	return nil
}

// NewNode creates a named Node and adds it as a child to the current parent Node
// Opens the new Node
func (p *Parser) NewNode(name string) {
	if p.root == nil || p.firstNode {
		p.root = NewNode(name)
		p.parent = p.root

		if p.firstNode {
			p.firstNode = false
		}
		return
	}

	p.parent.AddChildren(NewNode(name))
	p.parent.Children[len(p.parent.Children)-1].Parent = p.parent
	p.open()
}

// NewTextNode creates a new Node with Text based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (p *Parser) NewTextNode(cd *token.CharData) {
	p.parent.AddChildren(NewTextNode(cd))
	p.parent.Children[len(p.parent.Children)-1].Parent = p.parent
}

// NewCommentNode creates a new Node with Text as Comment, based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (p *Parser) NewCommentNode(cd *token.CharData) {
	p.parent.AddChildren(NewCommentNode(cd))
	p.parent.Children[len(p.parent.Children)-1].Parent = p.parent
}

// SetBlockType sets the current parent Nodes BlockType
func (p *Parser) SetBlockType(b BlockType) {
	p.parent.Block(b)
}

// SetStartPos sets the current parent Nodes Start Position
func (p *Parser) SetStartPos(pos token.Pos) {
	if p.parent != nil {
		p.parent.Range.BeginPos = pos
	}
}

// SetEndPos sets the current parent Nodes End Position
func (p *Parser) SetEndPos(pos token.Pos) {
	if p.parent != nil {
		p.parent.Range.EndPos = pos
	}
}

// GetRootBlockType returns the root Nodes BlockType
func (p *Parser) GetRootBlockType() BlockType {
	return p.root.BlockType
}

// GetRange returns the current parent Nodes Range
func (p *Parser) GetRange() token.Position {
	return p.parent.Range
}

// GetForwardingLenght returns the lenght of the List of forwaring Nodes
func (p *Parser) GetForwardingLength() int {
	if p.rootForward != nil && p.rootForward.Children != nil {
		return len(p.rootForward.Children)
	}
	return 0
}

// GetForwardingAttributesLength returns the length of the forwarding AttributeMap
func (p *Parser) GetForwardingAttributesLength() int {
	return p.forwardingAttributes.Len()
}

// GetForwardingPosition retrieves a forwarded Node based on given Index and
// returns the Rangespan the Token corresponding to said Node had in the input tadl text
func (p *Parser) GetForwardingPosition(i int) token.Node {
	return p.rootForward.Children[i].Range
}

// NodeIsClosedBy checks if the current Node is being closed by the given token.
func (p *Parser) NodeIsClosedBy(tok token.Token) bool {
	return p.parent.IsClosedBy(tok)
}

// AddAttribute adds a given Attribute to the current parent Node
func (p *Parser) AddAttribute(key, value string) {
	p.parent.Attributes.Set(key, value)
}

// AddForwardAttribute adds a given AttributeMap to the forwaring Attributes
func (p *Parser) AddForwardAttribute(key, value string) {
	if p.forwardingAttributes == nil {
		*p.forwardingAttributes = NewAttributeList()
	}
	p.forwardingAttributes.Push(key, value)
}

// AddForwardNode appends a given Node to the list of forwarding Nodes
func (p *Parser) AddForwardNode(name string) {
	p.SwitchActiveTree()
	p.parent = p.root
	p.NewNode(name)
	p.SwitchActiveTree()
}

// MergeAttributes merges the list of forwarded Attributes to the current parent Nodes Attributes
func (p *Parser) MergeAttributes() {
	if p.forwardingAttributes != nil && p.forwardingAttributes.Len() > 0 {
		p.parent.Attributes = p.parent.Attributes.Merge(*p.forwardingAttributes)
		p.forwardingAttributes = nil
	}
}

// MergeAttributesForwarded adds the buffered forwarding AttributeMap to the latest forwarded Node
func (p *Parser) MergeAttributesForwarded() {
	if p.forwardingAttributes != nil && p.forwardingAttributes.Len() > 0 {
		p.SwitchActiveTree()
		p.parent.Attributes = p.parent.Attributes.Merge(*p.forwardingAttributes)
		p.forwardingAttributes = nil
		p.SwitchActiveTree()
	}
}

// AppendForwardingNodes appends the current list of forwarding Nodes
// as Children to the current parent Node
func (p *Parser) AppendForwardingNodes() {
	if p.rootForward != nil && p.rootForward.Children != nil && len(p.rootForward.Children) != 0 {
		p.parent.Children = append(p.parent.Children, p.rootForward.Children...)
		p.rootForward.Children = nil
		p.parentForward = p.rootForward
	}
}

// AppendSubTree appends the rootForward Tree to the current parent Nodes Children
func (p *Parser) AppendSubTree() {
	if len(p.rootForward.Children) != 0 {
		p.parent.Children = append(p.parent.Children, p.rootForward.Children...)
		p.rootForward.Children = nil
	}
}

// g2AppendComments will append all comments that were parsed with g2EatComments as children
// into the given node.
func (p *Parser) G2AppendComments() {
	if p.parent != nil {
		p.parent.Children = append(p.parent.Children, p.g2Comments...)
		p.g2Comments = nil
	}
}

// G2AddComments adds a new Comment Node based on given CharData to the g2Comments List,
// to be added to the tree later
func (p *Parser) G2AddComments(cd *token.CharData) {
	p.g2Comments = append(p.g2Comments, NewCommentNode(cd))
}

// SwitchActiveTree switches the active Tree between the main syntax tree and the forwarding tree
// To modify the forwarding tree, call SwitchActiveTree, call treeCreation functions, call SwitchActiveTree
func (p *Parser) SwitchActiveTree() {
	var cache *TreeNode = p.parent
	p.parent = p.parentForward
	p.parentForward = cache

	cache = p.root
	p.root = p.rootForward
	p.rootForward = cache
}

// NewStringNode creates a Node with Text and adds it as a child to the current parent Node
// Opens the new Node, used for testing purposes only
func (p *Parser) NewStringNode(name string) {
	p.parent.AddChildren(NewStringNode(name))
	p.parent.Children[len(p.parent.Children)-1].Parent = p.parent
	p.open()
}

// NewStringCommentNode creates a new Node with Text as Comment, based on string and adds it as a child to the current parent Node
// Opens the new Node, used for testing purposes only
func (p *Parser) NewStringCommentNode(text string) {
	p.parent.AddChildren(NewStringCommentNode(text))
	p.parent.Children[len(p.parent.Children)-1].Parent = p.parent
	p.open()
}
