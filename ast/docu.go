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
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/golangee/tadl/token"
)

// DocText is just anything preceeded by #.
type DocText struct {
	Pos, EndPos lexer.Position
	Value       string `@DocText`
}

func (n *DocText) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocText) End() token.Pos {
	return wrapPos(n.EndPos)
}

type DocTypeBlock struct {
	Pos, EndPos lexer.Position
	Summary     string        `@DocSummary`
	Elems       []DocTypeElem `parser:"@@*" json:",omitempty"`
}

func (n *DocTypeBlock) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocTypeBlock) End() token.Pos {
	return wrapPos(n.EndPos)
}

type DocTypeElem struct {
	Title     *DocTitle `parser:"@@" json:",omitempty"`
	Text      *DocText  `parser:"|@@" json:",omitempty"`
	Reference *DocSee   `parser:"|@@" json:",omitempty"`
}

type DocMethodBlock struct {
	Pos, EndPos lexer.Position
	Summary     string          `@DocSummary`
	Elems       []DocMethodElem `parser:"@@*" json:",omitempty"`
}

func (n *DocMethodBlock) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocMethodBlock) End() token.Pos {
	return wrapPos(n.EndPos)
}

type DocMethodElem struct {
	Title         *DocTitle      `parser:"@@" json:",omitempty"`
	DocParameters *DocParameters `parser:"|@@" json:",omitempty"`
	DocReturns    *DocReturns    `parser:"|@@" json:",omitempty"`
	DocErrors     *DocErrors     `parser:"|@@" json:",omitempty"`
	Text          *DocText       `parser:"|@@" json:",omitempty"`
	Reference     *DocSee        `parser:"|@@" json:",omitempty"`
}

// DocTitle is of the form # = <Title>
type DocTitle struct {
	Pos, EndPos lexer.Position
	Value       string `@DocTitle`
}

func (n *DocTitle) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocTitle) End() token.Pos {
	return wrapPos(n.EndPos)
}

// DocSee is of the form # see <Path>
type DocSee struct {
	Pos, EndPos lexer.Position
	Value       string                 `@DocSeePrefix`
	Path        PathWithMemberAndParam `@@`
}

func (n *DocSee) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocSee) End() token.Pos {
	return wrapPos(n.EndPos)
}

type DocParameters struct {
	Pos, EndPos lexer.Position
	Value       string     `@DocSubTitleParameters`
	Params      []DocParam `(@@)*`
}

func (n *DocParameters) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocParameters) End() token.Pos {
	return wrapPos(n.EndPos)
}

type DocReturns struct {
	Pos, EndPos lexer.Position
	Value       string    `@DocSubTitleReturns`
	Text        []DocText `(@@)*`
}

func (n *DocReturns) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocReturns) End() token.Pos {
	return wrapPos(n.EndPos)
}

type DocErrors struct {
	Pos, EndPos lexer.Position
	Value       string     `@DocSubTitleErrors`
	Params      []DocParam `(@@)*`
}

func (n *DocErrors) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocErrors) End() token.Pos {
	return wrapPos(n.EndPos)
}

// TODO we need more validation stuff
type DocParam struct {
	Pos, EndPos lexer.Position
	Summary     string            `@DocListLevel0`
	More        []DocIndentLevel0 `parser:"(@@)*" json:",omitempty"`
}

func (n *DocParam) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocParam) End() token.Pos {
	return wrapPos(n.EndPos)
}

type DocListItem0 struct {
	Pos, EndPos lexer.Position
	Value       string            `@DocListLevel0`
	More        []DocIndentLevel0 `(@@)*`
}

func (n *DocListItem0) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocListItem0) End() token.Pos {
	return wrapPos(n.EndPos)
}

type DocIndentLevel0 struct {
	Pos, EndPos lexer.Position
	Value       string `@DocIndentLevel0`
}

func (n *DocIndentLevel0) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *DocIndentLevel0) End() token.Pos {
	return wrapPos(n.EndPos)
}
