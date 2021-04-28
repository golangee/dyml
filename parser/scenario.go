package parser

import (
	"fmt"
	"github.com/golangee/tadl/ast"
	"github.com/golangee/tadl/token"
	"io/ioutil"
	"strings"
)

// ParseScenarios parses scenario sections from an arbitrary file.
func ParseScenarios(filename string) ([]ast.Scenario, error) {
	scenarios, err := parseScenariosEN(filename)
	if err != nil {
		return nil, err
	}

	tmp := map[string]ast.Substring{}

	for _, scenario := range scenarios {
		existing, ok := tmp[scenario.ID.Value]
		if ok && existing != scenario.ID {
			return nil, token.NewPosError(existing, "duplicate scenario definition", token.NewErrDetail(scenario.ID, "also defined here"))
		}

		tmp[scenario.ID.Value] = scenario.ID
	}

	return scenarios, nil
}

func parseScenariosEN(filename string) ([]ast.Scenario, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open scenario file: %w", err)
	}

	const sectionPrefix = "== Scenario:"
	text := string(buf)

	var scenarios []ast.Scenario
	for i, line := range strings.Split(text, "\n") {
		lineNo := i + 1
		trimLine := strings.TrimSpace(line)
		scenarioStart := strings.HasPrefix(trimLine, sectionPrefix)
		if scenarioStart {
			id := ast.Substring{Value: strings.TrimSpace(trimLine[len(sectionPrefix):])}
			id.SetBegin(filename, lineNo, len(sectionPrefix))
			id.SetEnd(filename, lineNo, len(line)-len(sectionPrefix))

			scenarios = append(scenarios, ast.Scenario{ID: id})
			continue
		}

		if len(scenarios) > 0 {
			scenarios[len(scenarios)-1].Src += line + "\n"
		}
	}

	for i := range scenarios {
		if err := parseScenario(filename, &scenarios[i]); err != nil {
			return nil, fmt.Errorf("unable to parse scenario '%s': %w", scenarios[i].ID.Value, err)
		}
	}

	return scenarios, nil
}

func parseScenario(fname string, sce *ast.Scenario) error {
	src := sce.Src
	for i := 1; i < sce.ID.EndPos.Line; i++ {
		src = "\n" + src
	}

	given, err := substringPos(fname, src, "Given")
	if err != nil {
		return err
	}

	when, err := substringPos(fname, src, "when")
	if err != nil {
		return err
	}

	then, err := substringPos(fname, src, "then")
	if err != nil {
		return err
	}

	dot, err := substringPos(fname, src, ".")
	if err != nil {
		return err
	}

	if when.EndPos.After(then.EndPos) {
		return token.NewPosError(when, "expected 'when' before 'then' but is after")
	}

	if given.EndPos.After(when.EndPos) {
		return token.NewPosError(when, "expected 'given' before 'when' but is after")
	}

	if !dot.After(then.Position) {
		return token.NewPosError(dot, "invalid position of '.', expected after 'then'")
	}

	sce.Given = trimSubstring(textBetween(fname, src, given.End(), when.Begin()))
	sce.When = trimSubstring(textBetween(fname, src, when.End(), then.Begin()))
	sce.Then = trimSubstring(textBetween(fname, src, then.End(), dot.Begin()))
	sce.Roles = rolesOf(sce.Given)

	return nil
}

// rolesOf inspects the text of the form "I'm a/an" <role>, <role> or <role>
func rolesOf(text ast.Substring) []ast.Substring {
	const (
		prefixIma  = "I'm a "
		prefixIman = "I'm an "
	)

	tmp := text.Value
	if strings.HasPrefix(tmp, prefixIman) {
		tmp = tmp[len(prefixIman):]
	}

	if strings.HasPrefix(tmp, prefixIma) {
		tmp = tmp[len(prefixIma):]
	}

	var res []ast.Substring
	roles := strings.Split(tmp, ",")
	for _, role := range roles {
		orRoles := strings.Split(role, " or ")
		for _, orRole := range orRoles {
			trimRole := strings.TrimSpace(orRole)
			idx := strings.Index(text.Value, trimRole)
			substr := ast.Substring{Value: trimRole}
			substr.SetBegin(text.BeginPos.File, text.BeginPos.Line, idx)
			substr.SetEnd(text.BeginPos.File, text.BeginPos.Line, idx+len(trimRole))

			res = append(res, trimSubstring(substr))
		}
	}

	return res
}
