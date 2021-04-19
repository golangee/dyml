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

// Generate describes multiple emitter configurations.
type Generate struct {
	Go *GoGenerate `("go" "{" @@ "}")?`
}

// GoGenerate describes all go specific generator settings.
type GoGenerate struct {
	// Module is the go.mod name and prefix for all packages.
	Module String `("module" "=" @@)`
	// Output is either an absolute or relative local path to emit the source.
	Output String `("output" "=" @@)`

	Imports *GoStandardImport `("import" "{" @@)? "}"`

	Require *GoRequire `("require" "{" @@)? "}"`
}

type GoRequire struct {
	Modules []GoMod `@@*`
}

type GoMod struct {
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc DocTypeBlock `parser:"@@"`

	// Name is the module import path
	Name String `@@ "@"`

	// Version of this module, like v1.2.3 or v2.0.0-alpha or even v0.0.0-20200828070359-e00d658fcc60
	Version String `@@ `

	Imports []GoGlobalImport `"import" "{" @@* "}"`
}

type GoGlobalImport struct {
	Ident Ident  `@@`
	Path  String `@@`
}

type GoStandardImport struct {
	// Doc contains a summary, arbitrary text lines, captions, sections and more.
	Doc   DocTypeBlock `parser:"@@"`
	Ident Ident        `@@`
	Path  String       `@@`
}
