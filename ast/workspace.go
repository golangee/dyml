// Copyright 2021 Torben Schinke
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ast

import (
	"fmt"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/golangee/tadl/token"
)

const (
	// A StereotypeDomain represents the core of a workspace. It defines the companies domain and splits it into multiple subdomains.
	// A Domain has no parent.
	StereotypeDomain = "domain"

	// StereotypeSubdomain indicates that it has either another subdomain or domain as its parent.
	StereotypeSubdomain = "subdomain"

	// A StereotypeContext describes the top-level grouping structure in DDD and must have a ubiquitous language. The literature
	// is not that clear, because some define it as a cross-concerning element which may intersect with multiple subdomains.
	// For us, a context must be a subset of a subdomain. They must be treated different, however requirements and
	// glossaries may be shared between everything.
	//
	// A context creates the root of existence for any implementation by different modules.
	StereotypeContext = "context"
)

type CtxStereotype struct {
	Pos    lexer.Position
	EndPos lexer.Position
	Value  string `@Ident`
}

func (n *CtxStereotype) Capture(values []string) error {
	switch values[0] {
	case StereotypeDomain:
		fallthrough
	case StereotypeContext:
		fallthrough
	case StereotypeSubdomain:
		n.Value = values[0]
		return nil
	default:
		return fmt.Errorf("invalid stereotype as identifier: %s", values[0])
	}
}

func (n *CtxStereotype) String() string {
	return n.Value
}

func (n *CtxStereotype) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *CtxStereotype) End() token.Pos {
	return wrapPos(n.EndPos)
}

// A WorkspaceFile represents the top level declaration type. For example, this may represent a company as a whole.
// It is always named "tadl.ws".
type WorkspaceFile struct {
	Pos    lexer.Position
	EndPos lexer.Position
	Domain CtxDef `@@`
}

type CtxDef struct {
	Pos    lexer.Position
	EndPos lexer.Position
	// DocTypeBlock provides at least a summary line.
	Doc        DocTypeBlock  `@@`
	Stereotype CtxStereotype `@@`
	Name       Ident         `@@`
	Children   []*CtxDef     `("{" @@* "}")?`
}

// ChildByName returns the named direct child or nil.
func (n *CtxDef) ChildByName(name string) *CtxDef {
	for _, child := range n.Children {
		if child.Name.String() == name {
			return child
		}
	}

	return nil
}

func (n *CtxDef) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *CtxDef) End() token.Pos {
	return wrapPos(n.EndPos)
}

func (n *CtxDef) String() string {
	return n.Stereotype.String() + " " + n.Name.String()
}
