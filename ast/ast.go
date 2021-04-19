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
type File struct {
	Modules []*Module `@@*`
}

// A Module is a distinct unit of generation. Usually it maps only to a single
// service like a Go microservice but there may be also library modules which may
// get emitted into different targets.
type Module struct {
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc DocTypeBlock `parser:"@@"`
	// Name of the module.
	Name     Ident      `"module" @@ "{" `
	Generate *Generate  `"generate" "{" @@ "}" `
	Contexts []*Context `@@* "}"`
}

// A Context describes the top-level grouping structure in DDD.
type Context struct {
	Pos lexer.Position
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc DocTypeBlock `parser:"@@"`
	// Name of the context package
	Name Ident `"context" @@ "{" `
	// Domain with core and application layer
	Domain         *Domain         `"domain" "{" @@ "}" `
	Infrastructure *Infrastructure `"infrastructure" "{" @@ "}" `
	Presentation   *Presentation   `"presentation" "{" @@ "}" "}"`
}

// Domain contains the application (use case) and the core layer (packages).
type Domain struct {
	Pos        lexer.Position
	Core       *Core        `"core" "{" @@ "}"`
	UseCase    *Usecase     `"usecase" "{" @@ "}"`
	Subdomains []*Subdomain `@@*`
	//Types   []*TypeDef ` @@* "}"`
}

// Core is together with any subdomain packages self-containing and creates the
// base for any another layer.
type Core struct {
	Pos   lexer.Position
	Types []*TypeDef `@@*`
}

type Usecase struct {
	Pos   lexer.Position
	Types []*TypeDef `@@*`
}

// Subdomain contains the application (use case) and the core layer (packages).
type Subdomain struct {
	Pos lexer.Position
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc DocTypeBlock `parser:"@@"`

	Name    Ident    `"subdomain" @@ "{"`
	Core    *Core    `"core" "{" @@ "}"`
	UseCase *Usecase `"usecase" "{" @@ "}" "}"`
}

// Infrastructure helps with additional hints to generate stuff like SQL or Event adapter.
type Infrastructure struct {
	MySQL *SQL `("mysql" "{" @@ "}")?`
}

type TypeDef struct {
	Pos        lexer.Position
	Struct     *Struct     ` @@`
	Repository *Repository `| @@`
	Services   []Service   `| @@`
}

type Repository struct {
	Pos lexer.Position
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc     DocTypeBlock `parser:"@@"`
	Name    Ident        `"repository" @@`
	Methods []*Method    `"{" @@* "}"`
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

type Struct struct {
	Pos, EndPos lexer.Position
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc    DocTypeBlock `parser:"@@"`
	Name   Ident        `"struct" @@`
	Fields []*Field     `"{" @@* "}"`
}

func (n *Struct) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *Struct) End() token.Pos {
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
