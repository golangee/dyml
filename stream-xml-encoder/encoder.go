package streamxmlencoder

import (
	"bufio"
	"io"

	"github.com/golangee/tadl/parser"
	"github.com/golangee/tadl/token"
)

const (
	lt         = "\x3C"
	equals     = "\x3D"
	gt         = "\x3E"
	dquotes    = "\x22"
	slash      = "\x2F"
	whitespace = "\x20"
)

/*type TreeNodeXML struct {
	prefix 		*string
	postfix 	*string
	name 		*string
	text 		*string
	Comment 	*string
	Attributes 	AttributeMap

	parent 		*TreeNodeXML
	Children 	[]*TreeNodeXML
}

func NewNodeXML(text string) *TreeNodeXML {
	return &TreeNodeXML{
		name :	&text,
	}
}

func NewTextNodeXML(cd *token.CharData) *TreeNodeXML {
	return &TreeNodeXML{
		text: &cd.Value,
	}
}

func NewCommentNodeXML(cd *token.CharData) *TreeNodeXML {
	return &TreeNodeXML{
		text: &cd.Value,
	}
}

func NewStringNodeXML(text string) *TreeNodeXML {
	return &TreeNodeXML{
		text: &text,
	}
}

func NewStringCommentNodeXML(text string) *TreeNodeXML {
	return &TreeNodeXML{
		text:
	}
}*/

// Encoder translates tadl-input to corresponding XML
type Encoder struct {
	visitor    parser.Visitor
	writer     io.Writer
	buffWriter *bufio.Writer

	//forwardingAttributes contains all Attributes that have been forwarded to be added to the next viable node.
	forwardingAttributes parser.AttributeMap

	// root and parent are pointers to work with the successively built Tree.
	// root holds the root Node, parent holds the currently to modify Node
	root   *parser.TreeNode
	parent *parser.TreeNode

	// root- and parentForward have the same functionality as root and parent.
	// they are used to create full trees being forwarded, added later to the main tree
	rootForward   *parser.TreeNode
	parentForward *parser.TreeNode

	// g2Comments contains all comments in G2 that were eaten from the input,
	// but are not yet placed in a sensible position.
	g2Comments []*parser.TreeNode

	firstNode     bool
	globalForward bool
}

func (x *Encoder) write(in ...string) {
	for _, text := range in {
		x.buffWriter.Write([]byte(text))
	}
}

// NewEncoder creades a new XMLEncoder
// tadl-input is given as an io.Reader instance
func NewEncoder(filename string, r io.Reader, w io.Writer, buffsize int) Encoder {
	encoder := Encoder{
		visitor: *parser.NewVisitor(nil, token.NewLexer(filename, r)),
	}
	encoder.visitor.SetVisitable(encoder)
	return encoder
}

func (e *Encoder) Encode() error {
	err := e.visitor.Run()
	if err != nil {
		return err
	}

	return nil
}

// open sets the parent pointer to the latest Child of it's current Node
func (e *Encoder) open() {
	e.parent = e.parent.Children[len(e.parent.Children)-1]
}

// Close moves the parent pointer to its current parent Node
func (e *Encoder) Close() {
	if e.parent.Parent != nil {
		e.parent = e.parent.Parent
	}
}

// NewNode creates a named Node and adds it as a child to the current parent Node
// Opens the new Node
func (e *Encoder) NewNode(name string) {
	if e.root == nil || e.firstNode {
		e.root = parser.NewNode(name)
		e.parent = e.root

		if e.firstNode {
			e.firstNode = false
		}
		return
	}

	e.parent.AddChildren(parser.NewNode(name))
	e.parent.Children[len(e.parent.Children)-1].Parent = e.parent
	e.open()
}

// NewTextNode creates a new Node with Text based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (e *Encoder) NewTextNode(cd *token.CharData) {
	e.parent.AddChildren(parser.NewTextNode(cd))
	e.parent.Children[len(e.parent.Children)-1].Parent = e.parent
}

// NewCommentNode creates a new Node with Text as Comment, based on CharData and adds it as a child to the current parent Node
// Opens the new Node
func (e *Encoder) NewCommentNode(cd *token.CharData) {
	e.parent.AddChildren(parser.NewCommentNode(cd))
	e.parent.Children[len(e.parent.Children)-1].Parent = e.parent
}

// SetBlockType sets the current parent Nodes BlockType
func (e *Encoder) SetBlockType(b parser.BlockType) {
	e.parent.Block(b)
}

// SetStartPos sets the current parent Nodes Start Position
func (e *Encoder) SetStartPos(pos token.Pos) {
	if e.parent != nil {
		e.parent.Range.BeginPos = pos
	}
}

// SetEndPos sets the current parent Nodes End Position
func (e *Encoder) SetEndPos(pos token.Pos) {
	if e.parent != nil {
		e.parent.Range.EndPos = pos
	}
}

// GetRootBlockType returns the root Nodes BlockType
func (e *Encoder) GetRootBlockType() parser.BlockType {
	return e.root.BlockType
}

// GetRange returns the current parent Nodes Range
func (e *Encoder) GetRange() token.Position {
	return e.parent.Range
}

// GetForwardingLenght returns the lenght of the List of forwaring Nodes
func (e *Encoder) GetForwardingLength() int {
	if e.rootForward != nil && e.rootForward.Children != nil {
		return len(e.rootForward.Children)
	}
	return 0
}

// GetForwardingAttributesLength returns the length of the forwarding AttributeMap
func (e *Encoder) GetForwardingAttributesLength() int {
	return len(e.forwardingAttributes)
}

// GetForwardingPosition retrieves a forwarded Node based on given Index and
// returns the Rangespan the Token corresponding to said Node had in the input tadl text
func (e *Encoder) GetForwardingPosition(i int) token.Node {
	return e.rootForward.Children[i].Range
}

// NodeIsClosedBy checks if the current Node is being closed by the given token.
func (e *Encoder) NodeIsClosedBy(tok token.Token) bool {
	return e.parent.IsClosedBy(tok)
}

// AddAttribute adds a given Attribute to the current parent Node
func (e *Encoder) AddAttribute(key, value string) {
	e.parent.Attributes.Set(key, value)
}

// AddForwardAttribute adds a given AttributeMap to the forwaring Attributes
func (e *Encoder) AddForwardAttribute(m parser.AttributeMap) {
	e.forwardingAttributes = e.forwardingAttributes.Merge(m)
}

// AddForwardNode appends a given Node to the list of forwarding Nodes
func (e *Encoder) AddForwardNode(name string) {
	e.SwitchActiveTree()
	e.parent = e.root
	e.NewNode(name)
	e.SwitchActiveTree()
}

// MergeAttributes merges the list of forwarded Attributes to the current parent Nodes Attributes
func (e *Encoder) MergeAttributes() {
	e.parent.Attributes = e.parent.Attributes.Merge(e.forwardingAttributes)
	e.forwardingAttributes = nil
}

// MergeAttributesForwarded adds the buffered forwarding AttributeMap to the latest forwarded Node
func (e *Encoder) MergeAttributesForwarded() {
	e.SwitchActiveTree()
	e.parent.Attributes = e.parent.Attributes.Merge(e.forwardingAttributes)
	e.forwardingAttributes = nil
	e.SwitchActiveTree()
}

// AppendForwardingNodes appends the current list of forwarding Nodes
// as Children to the current parent Node
func (e *Encoder) AppendForwardingNodes() {
	if e.rootForward != nil && e.rootForward.Children != nil && len(e.rootForward.Children) != 0 {
		e.parent.Children = append(e.parent.Children, e.rootForward.Children...)
		e.rootForward.Children = nil
		e.parentForward = e.rootForward
	}
}

// AppendSubTree appends the rootForward Tree to the current parent Nodes Children
func (e *Encoder) AppendSubTree() {
	if len(e.rootForward.Children) != 0 {
		e.parent.Children = append(e.parent.Children, e.rootForward.Children...)
		e.rootForward.Children = nil
	}
}

// g2AppendComments will append all comments that were parsed with g2EatComments as children
// into the given node.
func (e *Encoder) G2AppendComments() {
	if e.parent != nil {
		e.parent.Children = append(e.parent.Children, e.g2Comments...)
		e.g2Comments = nil
	}
}

// G2AddComments adds a new Comment Node based on given CharData to the g2Comments List,
// to be added to the tree later
func (e *Encoder) G2AddComments(cd *token.CharData) {
	e.g2Comments = append(e.g2Comments, parser.NewCommentNode(cd))
}

// SwitchActiveTree switches the active Tree between the main syntax tree and the forwarding tree
// To modify the forwarding tree, call SwitchActiveTree, call treeCreation functions, call SwitchActiveTree
func (e *Encoder) SwitchActiveTree() {
	var cache *parser.TreeNode = e.parent
	e.parent = e.parentForward
	e.parentForward = cache

	cache = e.root
	e.root = e.rootForward
	e.rootForward = cache
}

// NewStringNode creates a Node with Text and adds it as a child to the current parent Node
// Opens the new Node, used for testing purposes only
func (e *Encoder) NewStringNode(name string) {
	e.parent.AddChildren(parser.NewStringNode(name))
	e.parent.Children[len(e.parent.Children)-1].Parent = e.parent
	e.open()
}

// NewStringCommentNode creates a new Node with Text as Comment, based on string and adds it as a child to the current parent Node
// Opens the new Node, used for testing purposes only
func (e *Encoder) NewStringCommentNode(text string) {
	e.parent.AddChildren(parser.NewStringCommentNode(text))
	e.parent.Children[len(e.parent.Children)-1].Parent = e.parent
	e.open()
}
