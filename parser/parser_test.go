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
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/golangee/tadl/token"
	"regexp"
	"strings"
	"testing"
)

//go:embed test.tadl
var tadl string

func TestParse(t *testing.T) {
	mustParse := []string{
		tadl,
	}

	for _, s := range mustParse {
		ast, err := ParseFile("test.tadl", strings.NewReader(s))
		if err != nil {
			fmt.Println(err)
			t.Fatal(err)
		}

		fmt.Println(toString(ast))
		fmt.Println(token.Explain(token.NewMsgErr(&ast.Modules[0].Name, "looks nice", "make this better here")))
	}
}

func toString(i interface{}) string {
	buf, err := json.MarshalIndent(i, " ", " ")
	if err != nil {
		panic(err)
	}
	return string(buf)
}

func TestLexerRegex(t *testing.T) {
	type reg struct {
		name  string
		regex string
		ok    []string
		fail  []string
	}

	table := []reg{
		{
			name:  "local selector",
			regex: sLocalSelector,
			ok:    []string{".field", ".a.b", ".a.b.c", ".ID"},
			fail:  []string{"field", "ident.", "a.b"}, // requires unsupported lookahead ".a."
		},
		{
			name:  "qualifier",
			regex: sQualifier,
			ok:    []string{"identifier", "package.Type", "github.com/my/path", "github.com/my/path.Type", "core::rust.Type"},
			fail:  []string{".field", `"asdb"`},
		},

		{
			name:  "local slice",
			regex: sLocalSlice,
			ok:    []string{"[i]"},
			fail:  []string{"[a]"},
		},

		{
			name:  "identifier",
			regex: sIdentifier,
			ok:    []string{"hey", "hello_world", "abc", "ABC", "aBc"},
			fail:  []string{"", "_", "_ho", ".a"},
		},
	}

	for _, r := range table {
		regex := regexp.MustCompile(r.regex)
		for _, s := range r.ok {
			if regex.FindString(s) != s {
				t.Fatal(r.name + ":" + r.regex + " does not match " + s + " fully")
			}
		}

		for _, s := range r.fail {
			match := regex.FindString(s)
			if match == "" {
				continue
			}

			if strings.HasPrefix(s, match) {
				t.Fatal(r.name + ":" + r.regex + " matches with prefix '" + s + "'")
			}
		}

	}
}

const requirementsDSL = `
:requirements worldiety.com/supportiety

:epic ManageTickets
As a SupportietyAdmin or Application 
I want to manage tickets
so that I can submit or delete new incidents.

:story AdminDeletesTicket
As a SupportietyAdmin 
I want to delete tickets from a user identified by his SecId, 
so that I can comply to the DSGVO/GDPR.

:scenario DoubleDelete
Given I'm a SupportietyAdmin
when I delete the same ticket twice
then I want a message telling me that its not possible.

:epic AnotherEpic 
blablabla

:story EpicStory1
blablub

:story EpicStory2
## Should markdown be allowed?
What about 
 * pointless
 * points

And tables

| value | size |
|-------|------|
| c     | d    |
|       |      |
|       |      |


:scenario OfEpic2-Scenario1
scenario 1 of epic 2 

:scenario OfEpic2-Scenario2
scenario 2 of epic 2 


`
