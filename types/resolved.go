package types

import "github.com/golangee/tadl/ast"

// A Workspace contains all resolved files and types.
type Workspace struct {
	// File contains the actual workspace file.
	File *ast.WorkspaceFile

	// Mods contains all resolved modules.
	Mods []*Module
}


// Module contains all resolved module files and their unified declarations.
type Module struct {
	File     *ast.ModFile
}

type Requirements struct{
	epics map[string]*Epic

}

type Epic struct {
	file *ast.Story
	stories map[string]*Story
}

type Story struct {
	scenarios map[string]*ast.Scenario
}