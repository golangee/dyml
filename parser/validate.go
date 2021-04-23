package parser

import (
	"github.com/golangee/tadl/ast"
	"github.com/golangee/tadl/token"
	"github.com/golangee/tadl/types"
)

// validateContextPath checks if the given path is defined in the workspace.
func validateContextPath(ws *types.Workspace, p *ast.Path) error {
	remainingPath := p.Elements

	// fake artifical ctxdef root
	rootCtx := &ast.CtxDef{
		Pos:      ws.File.Domain.Pos,
		EndPos:   ws.File.Domain.EndPos,
		Name:     ast.Ident{
			Pos:    ws.File.Pos,
			EndPos: ws.File.EndPos,
		},
		Children: []*ast.CtxDef{&ws.File.Domain},
	}

	for len(remainingPath) > 0 {
		next := remainingPath[0]
		remainingPath = remainingPath[1:]
		nextRoot := rootCtx.ChildByName(next.String())
		if nextRoot == nil {
			return token.NewPosError(&next, "path segment '"+next.String()+"' is undefined", token.NewErrDetail(&rootCtx.Name, "has no child '"+next.String()+"'"))
		}

		rootCtx = nextRoot
	}

	if rootCtx.Stereotype.Value != ast.StereotypeContext {
		return token.NewPosError(&p.Elements[len(p.Elements)-1], "last segment must reference a context", token.NewErrDetail(rootCtx, "expected a context here"))
	}
	return nil
}

