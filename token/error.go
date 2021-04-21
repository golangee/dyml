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

import (
	"errors"
	"fmt"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ErrDetail struct {
	Node    Node
	Message string
}

func NewErrDetail(node Node, msg string) ErrDetail {
	return ErrDetail{
		Node:    node,
		Message: msg,
	}
}

// PosError represents a very specific positional error with a lot of explaining noise. Use Explain.
type PosError struct {
	Details []ErrDetail
	Cause   error
	Hint    string
}

// NewPosError creates a new PosError with the given root cause and optional details.
func NewPosError(node Node, msg string, details ...ErrDetail) *PosError {
	tmp := append([]ErrDetail{}, ErrDetail{
		Node:    node,
		Message: msg,
	})
	tmp = append(tmp, details...)

	return &PosError{
		Details: tmp,
	}
}

func (p *PosError) SetCause(err error) *PosError {
	p.Cause = err
	return p
}

func (p *PosError) SetHint(str string) *PosError {
	p.Hint = str
	return p
}

func (p *PosError) Unwrap() error {
	return p.Cause
}

func (p *PosError) firstDetail() ErrDetail {
	if len(p.Details) > 0 {
		return p.Details[0]
	}

	return ErrDetail{}
}

func (p *PosError) Error() string {
	if p.Cause == nil {
		return p.firstDetail().Message
	}

	return p.firstDetail().Message + ": " + p.Cause.Error()
}

// src tries to load the source code based on the given file name. If it fails, the empty string is returned.
func src(fname string) string {
	buf, err := ioutil.ReadFile(fname)
	if err != nil {
		wd, err := os.Getwd()
		if err != nil {
			return ""
		}

		buf, err = ioutil.ReadFile(filepath.Join(wd, fname))
		if err != nil {
			return ""
		}
	}

	return string(buf)
}

// docLines returns associated source lines to the given node. It evaluate the magic attribute "src" from Obj
// which has the Stereotype Document.
func docLines(n Node) []string {
	if n == nil {
		return nil
	}

	src := src(n.Begin().File)
	return strings.Split(src, "\n")
}

// posLine returns the line from lines which fits to the given pos.
func posLine(lines []string, pos Pos) string {
	no := pos.Line - 1

	if no > len(lines) {
		no = len(lines) - 1
	}

	ltext := ""
	if no < len(lines) && no >= 0 {
		ltext = lines[no]
	}

	return ltext
}

// Explain returns a multi-line text suited to be printed into the console.
func (p PosError) Explain() string {
	// grab the required indent for the line numbers
	indent := 0
	for _, detail := range p.Details {
		l := len(strconv.Itoa(detail.Node.Begin().Line))
		if l > indent {
			indent = l
		}
	}

	sb := &strings.Builder{}

	/*for i := 0; i < indent; i++ {
		sb.WriteByte(' ')
	}
	sb.WriteString("--> ")
	if p.Node == nil {
		sb.WriteString("node is nil")
		return sb.String()
	}

	sb.WriteString(p.Node.Begin().String())
	sb.WriteString("\n")*/

	for i, detail := range p.Details {
		source := docLines(detail.Node)
		line := posLine(source, detail.Node.Begin())

		if i == 0 || (i > 0 && detail.Node.Begin().File != p.Details[i-1].Node.Begin().File) {
			sb.WriteString(detail.Node.Begin().String())
			sb.WriteString("\n")
		}

		sb.WriteString(fmt.Sprintf("%"+strconv.Itoa(indent)+"s |\n", ""))
		sb.WriteString(fmt.Sprintf("%"+strconv.Itoa(indent)+"d |", detail.Node.Begin().Line))
		sb.WriteString(line)
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("%"+strconv.Itoa(indent)+"s |", ""))

		if detail.Node.End().Col-detail.Node.Begin().Col <= 1 {
			sb.WriteString(fmt.Sprintf("%"+strconv.Itoa(detail.Node.Begin().Col-1)+"s", ""))
			sb.WriteString("^~~~ ")
		} else {
			sb.WriteString(fmt.Sprintf("%"+strconv.Itoa(detail.Node.Begin().Col-1)+"s", ""))
			for i := 0; i < detail.Node.End().Col-detail.Node.Begin().Col; i++ {
				sb.WriteRune('^')
			}
			sb.WriteRune(' ')
		}

		sb.WriteString(detail.Message)
		sb.WriteString("\n")

		if i < len(p.Details)-1 {
			for i := 0; i < indent; i++ {
				sb.WriteByte(' ')
			}
			sb.WriteString("...")
			sb.WriteByte('\n')
		}
	}

	if p.Hint != "" {
		sb.WriteString(fmt.Sprintf("%"+strconv.Itoa(indent)+"s |\n", ""))
		sb.WriteString(fmt.Sprintf("%"+strconv.Itoa(indent)+"s = hint: %s\n", "", p.Hint))
	}

	return sb.String()
}

// Explain takes the given wrapped error chain and explains it, if it can.
func Explain(err error) string {
	var posErr *PosError
	if errors.As(err, &posErr) {
		sb := &strings.Builder{}
		sb.WriteString("error: ")
		sb.WriteString(err.Error())
		sb.WriteString("\n")

		sb.WriteString(posErr.Explain())

		return sb.String()
	}

	var particplePos participle.Error
	if errors.As(err, &particplePos) {
		return Explain(NewPosError(adapterNode{particplePos.Position()}, particplePos.Message()))
	}

	return err.Error()
}

type adapterNode struct {
	pos lexer.Position
}

func (a adapterNode) Begin() Pos {
	return Pos{
		File: a.pos.Filename,
		Line: a.pos.Line,
		Col:  a.pos.Column,
	}
}

func (a adapterNode) End() Pos {
	return Pos{}
}
