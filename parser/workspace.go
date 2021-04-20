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

package parser

import (
	"fmt"
	"github.com/alecthomas/participle/v2"
	"github.com/golangee/tadl/ast"
	"github.com/golangee/tadl/types"
	"os"
)


// ParseModuleFile tries to parse a tadl.mod file.
func ParseModuleFile(filename string) (*ast.ModFile, error) {
	parser := participle.MustBuild(&ast.ModFile{},
		participle.Lexer(lexer),
		participle.Unquote("String"),
		participle.UseLookahead(1),
	)

	f := &ast.ModFile{}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open: %w", err)
	}

	defer file.Close()

	return f, parser.Parse(filename, file, f)
}

// ParseWorkspaceFile tries to parse a tadl.ws file.
func ParseWorkspaceFile(filename string) (*ast.WorkspaceFile, error) {
	parser := participle.MustBuild(&ast.WorkspaceFile{},
		participle.Lexer(lexer),
		participle.Unquote("String"),
		participle.UseLookahead(1),
	)

	f := &ast.WorkspaceFile{}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open: %w", err)
	}

	defer file.Close()

	return f, parser.Parse(filename, file, f)
}

// Parse takes the directory to parse an entire workspace. Inspect the returned error using token.Explain for a nice
// rendering of helpful positional errors.
func Parse(dir string) (*types.Workspace, error) {
	cws, err := collect(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to collect workspace files: %w", err)
	}

	ws := &types.Workspace{}
	wsfile, err := ParseWorkspaceFile(cws.workspaceFile)
	if err != nil {
		return nil, fmt.Errorf("unable to parse workspace file: %w", err)
	}

	ws.File = wsfile

	for _, module := range cws.modules {
		modFile, err := ParseModuleFile(module.modFile)
		if err != nil {
			return nil, fmt.Errorf("unable to parse module file: %w", err)
		}

		mod := &types.Module{}
		mod.File = modFile

		ws.Mods = append(ws.Mods, mod)

	}

	return ws, nil
}
