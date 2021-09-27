// SPDX-FileCopyrightText: © 2021 The dyml authors <https://github.com/golangee/dyml/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"os"
	"path/filepath"
	"strconv"
)

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
	// Offset in bytes
	Offset int
}

func (p Pos) After(other Pos) bool {
	if p.Line < other.Line {
		return false
	}

	if p.Line > other.Line {
		return true
	}

	return p.Col > other.Col
}

// String returns the content in the "file:line:col" format.
func (p Pos) String() string {
	return p.File + ":" + strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Col)
}

type Position struct {
	BeginPos, EndPos Pos
}

// After returns true, if this position end is beyond the other position begin.
func (d Position) After(other Position) bool {
	if d.EndPos.Line < other.BeginPos.Line {
		return false
	}

	if d.EndPos.Line > other.BeginPos.Line {
		return true
	}

	return d.EndPos.Col > other.BeginPos.Col
}

func (d *Position) SetBegin(filename string, line, col int) {
	d.BeginPos.File = filename
	d.BeginPos.Line = line
	d.BeginPos.Col = col
}

func (d Position) Begin() Pos {
	return d.BeginPos
}

func (d *Position) SetEnd(filename string, line, col int) {
	d.EndPos.File = filename
	d.EndPos.Line = line
	d.EndPos.Col = col
}

func (d Position) End() Pos {
	return d.EndPos
}

func NewNode(begin, end Pos) Node {
	return Position{begin, end}
}

// NewFileNode returns a fake node which just points to 1:1 of the file, whatever that is.
func NewFileNode(filename string) Node {
	cwd, _ := os.Getwd()
	fname := filepath.Join(cwd, filename)

	return Position{
		BeginPos: Pos{
			File: fname,
			Line: 1,
			Col:  1,
		},
		EndPos: Pos{
			File: fname,
			Line: 1,
			Col:  1,
		},
	}
}
