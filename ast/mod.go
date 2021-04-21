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

// A ModFile is a distinct unit of generation. Usually it maps only to a single
// service like a Go microservice but there may be also library modules which may
// get emitted into different targets.
type ModFile struct {
	// DocTypeBlock provides at least a summary line.
	DocTypeBlock DocTypeBlock `@@`
	// Name of the module must be unique across the entire workspace.
	Name     Ident      `"module" @@ "{" `
	Generate *Generate  `"generate" "{" @@ "}" "}"`
}



