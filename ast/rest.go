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

// Presentation contains the driver adapters.
type Presentation struct {
	Rest *Rest `("rest" "{" @@ "}")?`
}

type Rest struct {
	Version RestMajorVersion `@@*`
}

type RestMajorVersion struct {
	Version SemVer `@@ "{" `

	Types []JsonObject `@@*`

	Endpoints []RestEndpoint `@@* "}"`
}

type RestEndpoint struct {
	Usecases []DocSee  `@@*`
	Path     URLPath   `@@ "{" `
	Head     *HttpVerb `("HEAD" @@)? `
	Options  *HttpVerb `("OPTIONS" @@)? `
	Get      *HttpVerb `("GET" @@)? `
	Post     *HttpVerb `("POST" @@)? `
	Put      *HttpVerb `("PUT" @@)? `
	Patch    *HttpVerb `("PATCH" @@)? `
	Delete   *HttpVerb `("DELETE" @@)? "}"`
}

type URLPath struct {
	Elements []IdentOrVar ` @@ ("/" @@)* `
}

type IdentOrVar struct {
	Ident *Ident `parser:"@@" json:",omitempty"`
	Var   *Ident `parser:"|\":\" @@" json:",omitempty"`
}

type HttpVerb struct {
	// ContentType indicates the content type which the
	// client accepts and the server has to produce to
	// fulfill the request.
	ContentType String         `@@ "{" `
	In          []HttpInParam  `"in" "{" @@* "}" `
	Out         []HttpOutParam `"out" "{" @@*  `
	Errors      []HttpOutError `"errors" "{" @@* "}" "}" "}"`
}

type JsonObject struct {
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc    DocTypeBlock `parser:"@@"`
	Name   Ident        `"json" @@`
	Fields []*JsonField `"{" @@* "}"`
}

// A JsonField does not have a documentation, because it must "borrow"
// that from existing
type JsonField struct {
	Name String        `@@`
	Type TypeWithField `@@`
}

type HttpInParam struct {
	CopyDoc DocSee    `@@`
	Name    Ident     `@@`
	Type    Type      `@@ "="`
	From    HttpParam `@@`
}

type HttpParam struct {
	Pos, EndPos lexer.Position
	Header      *String `("HEADER" "[" @@ "]")`
	Path        *String `|("PATH" "[" @@ "]")`
	Query       *String `|("QUERY" "[" @@ "]")`
	IsBody      bool    `parser:"|@\"BODY\"" json:",omitempty"`
	IsRequest   bool    `parser:"|@\"REQUEST\"" json:",omitempty"`
}

func (n *HttpParam) Begin() token.Pos {
	return wrapPos(n.Pos)
}

func (n *HttpParam) End() token.Pos {
	return wrapPos(n.EndPos)
}

type HttpOutParam struct {
	CopyDoc  DocSee            `@@`
	Header   *HttpHeaderAssign `parser:"( (\"HEADER\" @@ )" json:",omitempty"`
	Body     *HttpAssign       `parser:"|(\"BODY\" \"=\" @@)" json:",omitempty"`
	Response *HttpAssign       `parser:"|(\"RESPONSE\" \"=\" @@))" json:",omitempty"`
}

type HttpHeaderAssign struct {
	Key   String `"[" @@ "]" "="`
	Ident Ident  `@@`
	Type  Type   `@@`
}

type HttpAssign struct {
	Ident Ident `@@`
	Type  Type  `@@`
}

type HttpOutError struct {
	Status Int                    `@@ "for"`
	Match  PathWithMemberAndParam `@@`
}
