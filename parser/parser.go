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
	Attributes AttributeMap
	parent     *TreeNode
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
		Attributes: NewAttributeMap(),
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

func (t *TreeNode) Print() string {
	text := t.Name
	for _, child := range t.Children {
		text += child.Print()
	}

	return text
}

// unbindParents recursively sets all parent Pointers of a tree to nil
func unbindParents(t *TreeNode) {
	t.parent = nil
	for _, child := range t.Children {
		unbindParents(child)
	}
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
	// forwardingNodes is a list of all nodes that were defined as forwarded.
	// They will be inserted into the next node.
	forwardingNodes      []*TreeNode
	forwardingAttributes AttributeMap

	// root and parent are pointers to work with the successively built Tree.
	// root holds the root Node, parent holds the currently to modify Node
	root   *TreeNode
	parent *TreeNode

	// root- and parentForward have the same functionality as root and parent.
	// they are used to create full trees being forwarded, added later to the main tree
	rootForward   *TreeNode
	parentForward *TreeNode

	visitor Visitor

	firstNode     bool
	globalForward bool
}

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

/*func NewParserEncoder(filename string, r io.Reader) *Parser {
	return &Parser{
		lexer:   NewLexer(filename, r),
		mode:    G1,
		visitor: NewVisitorEncoder(),
	}
}*/

// Parse returns a parsed tree.
func (p *Parser) Parse() (*TreeNode, error) {
	err := p.visitor.Run()
	if err != nil {
		return nil, err
	}
	unbindParents(p.root)

	return p.root, nil
}

// Open sets the parent pointer to the latest Child of it's current Node
func (p *Parser) Open() {
	p.parent = p.parent.Children[len(p.parent.Children)-1]
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
	p.parent.Children[len(p.parent.Children)-1].parent = p.parent
	p.Open()
}

// NewStringNode creates a Node with Text and adds it as a child to the current parent Node
// Opens the new Node
func (p *Parser) NewStringNode(name string) {
	p.parent.AddChildren(NewStringNode(name))
	p.parent.Children[len(p.parent.Children)-1].parent = p.parent
	p.Open()
}

// NewTextNode creates a new Node with Text based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (p *Parser) NewTextNode(cd *token.CharData) {
	p.parent.AddChildren(NewTextNode(cd))
	p.parent.Children[len(p.parent.Children)-1].parent = p.parent
}

// NewCommentNode creates a new Node with Text as Comment, based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (p *Parser) NewCommentNode(cd *token.CharData) {
	p.parent.AddChildren(NewCommentNode(cd))
	p.parent.Children[len(p.parent.Children)-1].parent = p.parent
}

// NewStringCommentNode creates a new Node with Text as Comment, based on string and adds it as a child to the current parent Node
// Opens the new Node
func (p *Parser) NewStringCommentNode(text string) {
	p.parent.AddChildren(NewStringCommentNode(text))
	p.parent.Children[len(p.parent.Children)-1].parent = p.parent
	p.Open()
}

// AddAttribute adds a given Attribute to the current parent Node
func (p *Parser) AddAttribute(key, value string) {
	p.parent.Attributes.Set(key, value)
}

// AddForwardAttribute adds a given AttributeMap to the forwaring Attributes
func (p *Parser) AddForwardAttribute(m AttributeMap) {
	p.forwardingAttributes = p.forwardingAttributes.Merge(m)
}

// Block sets the current parent Nodes BlockType from given parameter
func (p *Parser) Block(blockType BlockType) {
	p.parent.Block(blockType)
}

// Close moves the parent pointer to its current parent Node
func (p *Parser) Close() {
	if p.parent.parent != nil {
		p.parent = p.parent.parent
	}
}

// AddForwardNode appends a given Node to the list of forwarding Nodes
func (p *Parser) AddForwardNode(name string) {
	p.SwitchActiveTree()
	p.parent = p.root
	p.NewNode(name)
	p.SwitchActiveTree()
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

// MergeAttributes merges the list of forwarded Attributes to the current parent Nodes Attributes
func (p *Parser) MergeAttributes() {
	p.parent.Attributes = p.parent.Attributes.Merge(p.forwardingAttributes)
	p.forwardingAttributes = nil
}

// GetForwardingLenght returns the lenght of the List of forwaring Nodes
func (p *Parser) GetForwardingLength() int {
	return len(p.forwardingNodes)
}

// GetForwardingPosition retrieves a forwarded Node based on given Index and
// returns the Rangespan the Token corresponding to said Node had in the input tadl text
func (p *Parser) GetForwardingPosition(i int) token.Node {
	return p.forwardingNodes[i].Range
}

// SetNodeName sets the current parent Nodes name
func (p *Parser) SetNodeName(name string) {
	p.parent.Name = name
}

// SetBlockType sets the current parent Nodes BlockType
func (p *Parser) SetBlockType(b BlockType) {
	p.parent.BlockType = b
}

// GetBlockType returns the current parent Nodes BlockType
func (p *Parser) GetBlockType() BlockType {
	return p.parent.BlockType
}

// GetRootBlockType returns the root Nodes BlockType
func (p *Parser) GetRootBlockType() BlockType {
	return p.root.BlockType
}

// SetStartPos sets the current parent Nodes Start Position
func (p *Parser) SetStartPos(pos token.Pos) {
	p.parent.Range.BeginPos = pos
}

// SetEndPos sets the current parent Nodes End Position
func (p *Parser) SetEndPos(pos token.Pos) {
	p.parent.Range.EndPos = pos
}

/*
func (p *Parser) InsertForwardNodes(nodes []*TreeNode) {
	p.parent.Children = append(p.parent.Children, nodes...)
}*/

// SetNodeText sets the current parent Nodes text
func (p *Parser) SetNodeText(text string) {
	p.parent.Text = &text
}

// GetRange returns the current parent Nodes Range
func (p *Parser) GetRange() token.Position {
	return p.parent.Range
}

// MergeAttributesForwarded adds the buffered forwarding AttributeMap to the latest forwarded Node
func (p *Parser) MergeAttributesForwarded() {
	p.SwitchActiveTree()
	p.parent.Attributes = p.parent.Attributes.Merge(p.forwardingAttributes)
	p.forwardingAttributes = nil
	p.SwitchActiveTree()
}

// SetGlobalForwarding sets the globalForward Field, allowing to build the root- and parentForward tree.
// This enables determining wether newly created nodes and attributes are supposed to be
// added to the main Tree, or to the forwarding Tree, to be added later
func (p *Parser) SetGlobalForwarding(f bool) {
	p.globalForward = f
}

// AppendSubTreeForward appends the rootForward Tree to the forwarding Nodes
func (p *Parser) AppendSubTreeForward() {
	p.forwardingNodes = append(p.forwardingNodes, p.rootForward)
}

// AppendSubTree appends the rootForward Tree to the current parent Nodes Children
func (p *Parser) AppendSubTree() {
	p.parent.Children = append(p.parent.Children, p.rootForward)
}

// GetForwardingAttributesLength returns the length of the forwarding AttributeMap
func (p *Parser) GetForwardingAttributesLength() int {
	return len(p.forwardingAttributes)
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
