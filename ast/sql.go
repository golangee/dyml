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

// SQL contains sql independent declaration, but the generators are dialect specific.
type SQL struct {
	Database   Ident                `@@ "{"`
	Implements []*SQLImplementation `@@+ "}"`
}

type SQLImplementation struct {
	Type Path `"impl" @@ "{"`

	Configure []*FieldWithDefault `("configure" "{" @@* "}")?`
	Inject    []*Field            `("inject" "{" @@* "}")?`
	Private   []*Field            `("private" "{" @@* "}")?`

	SQLFunc []*SQLFunc ` @@* "}"`
}

type SQLFunc struct {
	Name Ident           `@@`
	SQL  String          `@@`
	In   []SQLFuncInVar  `( "(" @@ ("," @@)* ")" )?`
	Out  []SQLFuncOutVar `( "=>" "(" @@ ("," @@)* ")" )?`
}

type SQLFuncInVar struct {
	Selector []LooperIdent `( @@ ("." @@)* )?`
	IsLooper bool          `(@SliceLooper)?`
}

type LooperIdent struct {
	Ident    Ident `@@`
	IsLooper bool  `(@SliceLooper)?`
}

type SQLFuncOutVar struct {
	Ident Ident `"." @@?`
}
