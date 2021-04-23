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
	"golang.org/x/mod/semver"
	"strconv"
	"strings"
)

// String is just anything like sString accepts.
type String struct {
	Pos    lexer.Position
	EndPos lexer.Position
	Value  string `@String`
}

func (n *String) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *String) End() token.Pos {
	return wrapPos(n.EndPos)
}

// Int is just anything an int may be
type Int struct {
	Pos, EndPos lexer.Position
	Value       int64 `@IntLiteral`
}

func (n *Int) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *Int) End() token.Pos {
	return wrapPos(n.EndPos)
}

// Bool is either true|false
type Bool struct {
	Pos, EndPos lexer.Position
	Value       bool `@BoolLit`
}

func (n *Bool) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *Bool) End() token.Pos {
	return wrapPos(n.EndPos)
}

func (n *Bool) Capture(values []string) error {
	v, err := strconv.ParseBool(values[0])
	if err != nil {
		return fmt.Errorf("unable to parse bool literal: %w", err)
	}

	n.Value = v
	return nil
}

type SemVer struct {
	Pos, EndPos lexer.Position
	Value       string `@Ident`
}

func (n *SemVer) Capture(values []string) error {
	n.Value = values[0]
	if !semver.IsValid(n.Value) {
		return fmt.Errorf("invalid semantic version number")
	}

	return nil
}

func (n *SemVer) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *SemVer) End() token.Pos {
	return wrapPos(n.EndPos)
}

type Path struct {
	Pos, EndPos lexer.Position
	Elements    []Ident ` @@ ("::" @@)* `
}

func (n *Path) String() string {
	sb := &strings.Builder{}
	for i, element := range n.Elements {
		sb.WriteString(element.String())
		if i < len(n.Elements)-1 {
			sb.WriteString("::")
		}
	}

	return sb.String()
}

func (n *Path) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *Path) End() token.Pos {
	return wrapPos(n.EndPos)
}

type PathWithMemberAndParam struct {
	Path   `@@`
	Member *Ident `("." @@)?`
	Param  *Ident `("$" @@)?`
}

func (n *PathWithMemberAndParam) String() string {
	tmp := n.Path.String()
	if n.Member != nil {
		tmp += "." + n.Member.String()
	}

	if n.Param != nil {
		tmp += "$" + n.Param.String()
	}

	return tmp
}

type TypeWithField struct {
	Type  Type  `@@`
	Field Ident `"." @@`
}

type Type struct {
	Pos, EndPos lexer.Position
	Pointer     bool `(@Pointer)?`
	Qualifier   Path `@@`
	// Transpile flag indicates that the given name should be transformed by the rules of the architecture
	// standard library. E.g. a string! becomes a java.lang.String in Java but just a string in Go.
	Transpile bool   `parser:"@MacroSep?" json:",omitempty"`
	Optional  bool   `parser:"@Optional?" json:",omitempty"`
	Params    []Type `("<" @@ ("," @@)* ">")?`
}

func (n *Type) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *Type) End() token.Pos {
	return wrapPos(n.EndPos)
}

type Ident struct {
	Pos    lexer.Position
	EndPos lexer.Position
	Value  string `@Ident`
}

func (n *Ident) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *Ident) End() token.Pos {
	return wrapPos(n.EndPos)
}

func (n *Ident) String() string {
	return n.Value
}

// Qualifier is just anything like sQualifier accepts.
type Qualifier struct {
	Pos, EndPos lexer.Position
	IsSliceType bool   `@SliceType?`
	Value       string `@Qualifier`
	IsStdType   bool   `@StdlibType?`
}

func (n *Qualifier) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *Qualifier) End() token.Pos {
	return wrapPos(n.EndPos)
}

// File represents a source code file.
type TadlFile struct {
	Context Context `"context" @@`
}

// A Context describes the top-level grouping structure in DDD.
type Context struct {
	Pos lexer.Position
	// Path of the context, must be fully qualified
	Path Path `@@ "{" `
	// Domain with core and application layer
	Core           *Core           `("core" "{" @@ "}")?`
	Usecase        *Usecase        `("usecase" "{" @@ "}")?`
	Infrastructure *Infrastructure `("infrastructure" "{" @@ "}")?`
	Presentation   *Presentation   `("presentation" "{" @@ "}" )? "}"`
}

// CorePackage is a language module like grouping feature.
type CorePackage struct {
	Pos   lexer.Position
	Name  Ident          `"package" @@ "{"`
	Types []*TypeDefCore `@@* "}"`
}

// Core is together with any subdomain packages self-containing and creates the
// base for any another layer.
type Core struct {
	Pos   lexer.Position
	Types []*TypeDefCore `@@*`
}

type Usecase struct {
	Pos   lexer.Position
	Types []*TypeDefUsecase `@@*`
}

// Infrastructure helps with additional hints to generate stuff like SQL or Event adapter.
type Infrastructure struct {
	MySQL *SQL `("mysql" @@)?`
}

// TypeDefCore declares the allowed types for domain core declarations.
type TypeDefCore struct {
	Pos lexer.Position
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc        DocTypeBlock `parser:"@@"`
	Dto        *DTO         `( @@`
	Repository *Repository  `| @@`
	Package    *CorePackage `| @@`
	Services   *Service     `| @@)`
}

// TypeDefUsecase declares the allowed types for domain application/usecase declarations.
// Similar to TypeDefCore but Repositories are not allowed.
type TypeDefUsecase struct {
	Pos lexer.Position
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc      DocTypeBlock    `parser:"@@"`
	Dto      *DTO            `( @@`
	Package  *UsecasePackage `| @@`
	Services *Service        `| @@)`
}

// UsecasePackage is a language module like grouping feature.
type UsecasePackage struct {
	Pos   lexer.Position
	Name  Ident             `"package" @@ "{"`
	Types []*TypeDefUsecase `@@* "}"`
}


type Repository struct {
	Pos     lexer.Position
	Name    Ident     `"repository" @@`
	Methods []*Method `"{" @@* "}"`
}

type Method struct {
	Pos lexer.Position
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc    DocMethodBlock `parser:"@@"`
	Name   Ident          `@@`
	Params []*Param       `"(" @@? | ("," @@)* ")"`
	Return *Type          `"->" "(" @@? `
	Error  *Error         ` ("," @@)?  ")"`
}

type Error struct {
	Kinds []Ident `"error" "<" @@ ("|" @@)* ">"`
}

type Param struct {
	Name Ident `@@`
	Type Type  `@@`
}

// DTO is a struct without behavior, even if it looks like an entity.
// In the anemic world, the Entity part is
type DTO struct {
	Pos, EndPos lexer.Position

	Name   Ident    `"dto" @@`
	Fields []*Field `"{" @@* "}"`
}

func (n *DTO) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DTO) End() token.Pos {
	return wrapPos(n.EndPos)
}

type Field struct {
	Pos, EndPos lexer.Position
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc  DocTypeBlock `parser:"@@"`
	Name Ident        `@@`
	Type Type         `@@`
}

func (n *Field) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *Field) End() token.Pos {
	return wrapPos(n.EndPos)
}

type FieldWithDefault struct {
	Pos, EndPos lexer.Position
	Field       Field   `@@ "="`
	StrLit      *String `(@@`
	IntLit      *Int    `|@@`
	BoolLit     *Bool   `|@@)`
}

func (n *FieldWithDefault) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *FieldWithDefault) End() token.Pos {
	return wrapPos(n.EndPos)
}
