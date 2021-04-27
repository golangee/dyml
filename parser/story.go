package parser

import (
	"fmt"
	"github.com/golangee/tadl/ast"
	"github.com/golangee/tadl/token"
	"io/ioutil"
	"strings"
)

// ParseStory is a custom parser, because using any other parser creates more headaches than it supports. This
// can be just solved with some easy string search operations.
func ParseStory(filename string) (*ast.Story, error) {
	return parseStoryEN(filename)
}

func parseStoryEN(filename string) (*ast.Story, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open story file: %w", err)
	}

	text := string(buf)

	asA, err := substringPos(filename, text, "As an", "As a")
	if err != nil {
		return nil, err
	}

	iWant, err := substringPos(filename, text, "I want to", "I want")
	if err != nil {
		return nil, err
	}

	soThat, err := substringPos(filename, text, "so that")
	if err != nil {
		return nil, err
	}

	dot, err := substringPos(filename, text, ".")
	if err != nil {
		return nil, err
	}

	if !soThat.After(iWant.Position) {
		return nil, token.NewPosError(soThat, "invalid position of 'so that', expected after 'I want'")
	}

	if !iWant.After(asA.Position) {
		return nil, token.NewPosError(iWant, "invalid position of 'I want', expected after 'As a'")
	}

	if !dot.After(soThat.Position) {
		return nil, token.NewPosError(iWant, "invalid position of '.', expected after 'so that'")
	}

	story := &ast.Story{}
	story.Src = text
	story.BeginPos = asA.BeginPos
	story.EndPos = soThat.EndPos
	story.Role = trimSubstring(textBetween(filename, text, asA.End(), iWant.Begin()))
	story.Goal = trimSubstring(textBetween(filename, text, iWant.End(), soThat.Begin()))
	story.Reason = trimSubstring(textBetween(filename, text, soThat.End(), dot.Begin()))
	story.Other = strings.TrimSpace(text[indexOfPos(text, dot.End()):])

	return story, nil
}

func substringPos(fname, text string, matchFirst ...string) (ast.Substring, error) {
	var str ast.Substring
	eofLine := 0
	eofCol := 0
	for i, line := range strings.Split(text, "\n") {
		for _, match := range matchFirst {
			idx := strings.Index(line, match)
			if idx > -1 {
				str.SetBegin(fname, i+1, idx+1)
				str.SetEnd(fname, i+1, idx+len(match))
				str.Value = match

				return str, nil
			}
		}

		eofLine++
		eofCol = len(line)
	}

	str.SetBegin(fname, eofLine, eofCol)
	return str, token.NewPosError(str, "token not found: expected ('"+strings.Join(matchFirst, "' | '")+"')")
}

func indexOfPos(text string, pos token.Pos) int {
	idx := 0
	for i, line := range strings.Split(text, "\n") {
		lineNo := i + 1
		if lineNo == pos.Line {
			return idx + pos.Col
		}

		idx += len(line) + 1
	}

	return -1
}

func textBetween(fname, text string, a, b token.Pos) ast.Substring {
	if a.After(b) {
		a, b = b, a
	}

	sb := &strings.Builder{}
	for i, line := range strings.Split(text, "\n") {
		lineNo := i + 1
		if a.Line >= lineNo && lineNo <= b.Line {
			if a.Line == lineNo && b.Line == lineNo {
				sb.WriteString(line[a.Col:b.Col-1])
			} else if a.Line == lineNo {
				sb.WriteString(line[a.Col:])
			} else if b.Line == lineNo {
				sb.WriteString(line[:b.Col-1])
			}
		}
	}

	str := ast.Substring{}
	str.SetBegin(fname, a.Line, a.Col)
	str.SetEnd(fname, b.Line, b.Col)
	str.Value = sb.String()

	return str
}

func trimSubstring(str ast.Substring) ast.Substring {
	for len(str.Value) > 0 && isTrimChar(str.Value[0]) {
		str.BeginPos.Col++
		str.Value = str.Value[1:]
	}

	for len(str.Value) > 0 && isTrimChar(str.Value[len(str.Value)-1]) {
		str.EndPos.Col--
		str.Value = str.Value[:len(str.Value)-1]
	}

	return str
}

// TODO not utf-8 compatible
func isTrimChar(b byte) bool {
	return b == ' ' || b == ',' || b == '.'
}
