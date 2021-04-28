package ast

import (
	"github.com/golangee/tadl/token"
)

// Story declares a user story and the according method and potential extra types.
// A story represents a scenario in a use case or epic and has the format
//  As a < role >, I want to < a goal > so that < a reason >.
// Make a good story and avoid the following pitfalls:
//  * unclear or subjective formulation, which refers to non-measurable values
//  * avoid eventualities and define the context of things (e.g. as a user I want to see >all or relevant< stuff...)
//  * do not use multiple main verbs for the subject (e.g. as a platform user I want to import and archive...)
//  * explain the reason why an actor wants/needs/must do something.
//
// See also Mike Cohns suggestions at https://www.mountaingoatsoftware.com/agile/user-stories and
// https://www.mountaingoatsoftware.com/blog/advantages-of-the-as-a-user-i-want-user-story-template
type Story struct {
	token.Position

	ID     Substring // ID is usually the filename
	Src    string    // contains the uninterpreted original text in its entirety, the story and any other text.
	Role   Substring
	Goal   Substring
	Reason Substring
	Other  string // contains uninterpreted text of everything after the first story sentence.

	Scenarios []Scenario // scenarios are of the form == Scenario: <ID>
}

// A Substring contains an arbitrary text value.
type Substring struct {
	token.Position
	Value string
}

// Scenario is of the form Given ... when ... then.
type Scenario struct {
	token.Position
	ID    Substring
	Src   string      // contains the uninterpreted original text in its entirety.
	Roles []Substring // a given role is extracted automatically, if possible
	Given Substring
	When  Substring
	Then  Substring
}
