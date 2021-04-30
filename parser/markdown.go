package parser

import (
	"fmt"
	"github.com/golangee/tadl/ast"
	"io/ioutil"
)

func ParseMarkdown(filename string) (*ast.MLNode, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open story file: %w", err)
	}

	return ParseMarkdownText(filename, string(buf))
}

func ParseMarkdownText(filename, text string) (*ast.MLNode, error) {

	return nil,nil
}

