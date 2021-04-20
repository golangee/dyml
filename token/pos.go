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

package token

import "strconv"

// Node contains access to the start and end positions of a token.
type Node interface {
	Begin() Pos
	End() Pos
}

// A Pos describes a resolved position within a file.
type Pos struct {
	// File contains the absolute file path.
	File string
	// Line denotes the one-based line number in the denoted File.
	Line int
	// Col denotes the one-based column number in the denoted Line.
	Col int
}

// String returns the content in the "file:line:col" format.
func (p Pos) String() string {
	return p.File + ":" + strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Col)
}

type defaultNode struct {
	begin, end Pos
}

func (d defaultNode) Begin() Pos {
	return d.begin
}

func (d defaultNode) End() Pos {
	return d.end
}

func NewNode(begin, end Pos) Node {
	return defaultNode{begin, end}
}
