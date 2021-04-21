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
	"github.com/golangee/tadl/token"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
)

const (
	filenameModule     = "tadl.mod"
	filenameWorkspace  = "tadl.ws"
	filenameTadlSuffix = ".tadl"
)

type collectedWorkspace struct {
	workspaceFile string
	modules       []*collectedModule
}

type collectedModule struct {
	modFile      string
	partialFiles []string
}

func collect(rootDir string) (*collectedWorkspace, error) {
	var firstWorkspaceDir string

	err := filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path != "." && info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		if !info.IsDir() && info.Name() == filenameWorkspace {
			firstWorkspaceDir = filepath.Dir(path)
			return nil
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("unable to search for workspace: %w", err)
	}

	if firstWorkspaceDir == "" {
		return nil, fmt.Errorf("no workspace found")
	}

	ws := &collectedWorkspace{}
	return ws, searchWorkspaceFiles(firstWorkspaceDir, true, ws)
}

func searchWorkspaceFiles(dir string, isRootDir bool, dst *collectedWorkspace) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("unable to read %s: %w", dir, err)
	}


	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}

		path := filepath.Join(dir, file.Name())
		if file.Name() == filenameWorkspace {
			if !isRootDir {
				return newWrongFileLocationError(path, "nested workspaces are not allowed")
			}

			dst.workspaceFile = path
		}

		if file.Name() == filenameModule {
			if isRootDir {
				return newWrongFileLocationError(path, "invalid module file location, expected sub directory")
			}

			mod := &collectedModule{}
			dst.modules = append(dst.modules, mod)
			if err := searchModFiles(dir, true, mod); err != nil {
				return fmt.Errorf("unable to scan module directory: %w", err)
			}

		}


		if file.IsDir() {
			if err := searchWorkspaceFiles(path, false, dst); err != nil {
				return err
			}
		}

	}

	return nil
}

// searchModFiles expects a module directory.
func searchModFiles(dir string, isRootDir bool, dst *collectedModule) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("unable to read %s: %w", dir, err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}

		path := filepath.Join(dir, file.Name())
		if !isRootDir && file.Name() == filenameModule {
			return newWrongFileLocationError(path, "nested modules are not allowed")
		}

		if file.Name() == filenameWorkspace {
			return newWrongFileLocationError(path, "nested workspaces are not allowed")
		}

		if file.Name() == filenameModule {
			dst.modFile = path
			continue
		}

		if file.Mode().IsRegular() && strings.HasSuffix(file.Name(), filenameTadlSuffix) {
			dst.partialFiles = append(dst.partialFiles, path)
		}

		if file.IsDir() {
			if err := searchModFiles(path, false, dst); err != nil {
				return err
			}
		}
	}

	return nil
}

func tadlModFile(files []fs.FileInfo) fs.FileInfo {
	for _, file := range files {
		if file.Name() == filenameModule {
			return file
		}
	}

	return nil
}

func newWrongFileLocationError(fname, msg string) error {
	dummyNode := token.NewNode(token.Pos{
		File: fname,
		Line: 1,
		Col:  1,
	}, token.Pos{})

	return token.NewPosError(dummyNode, msg)
}
