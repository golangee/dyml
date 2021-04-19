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
	"bytes"
	"fmt"
	"github.com/golangee/tadl/ast"
	"github.com/golangee/tadl/token"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// ParseProject loads all *.tadl files recursively and assembles them into a merged module tree.
func ParseProject(dir string) (*ast.Project, error) {
	prj := &ast.Project{Directory: dir}

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path != "." && info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tadl") {
			buf, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("unable to read tadl file: %w", err)
			}

			file, err := ParseFile(path, bytes.NewReader(buf))
			if err != nil {
				return fmt.Errorf("unable to parse %s: %w", path, err)
			}

			prj.Files = append(prj.Files, ast.ProjectFile{
				Filename: path,
				File:     file,
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("unable to walk dir: %w", err)
	}

	if err := mergeProjectModules(prj); err != nil {
		return nil, fmt.Errorf("unable to merge files: %w", err)
	}
	return prj, nil
}

func mergeProjectModules(prj *ast.Project) error {
	for _, file := range prj.Files {
		for _, module := range file.File.Modules {
			existingModule := prj.Module(module.Name.Value)
			if existingModule != nil {
				return token.NewPosError(&existingModule.Name, "has been already declared",
					token.NewErrDetail(&existingModule.Name, "alread declared here"),
					token.NewErrDetail(&module.Name, "the name must be unique"))
			}

			prj.Modules = append(prj.Modules, module)
		}
	}

	return nil
}
