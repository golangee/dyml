package ast

// Project is an artificial holder.
type Project struct {
	// the root of this project.
	Directory string

	// all single files from this project.
	Files []ProjectFile

	// all files merged into a single module tree.
	Modules []*Module `@@*`
}

// Module returns nil or the module by name.
func (p *Project) Module(name string) *Module {
	for _, module := range p.Modules {
		if module.Name.Value == name {
			return module
		}
	}

	return nil
}

type ProjectFile struct {
	Filename string
	File     *File
}
