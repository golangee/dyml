package parser2

import (
	"io"

	"github.com/golangee/tadl/token"
)

func (d *Decoder) g2Root() (*Element, error) {
	// Eat '#!{' from input
	r, _ := d.nextR()
	if r != '#' {
		return nil, token.NewPosError(d.node(), "expected '#' in g2 mode")
	}
	r, _ = d.nextR()
	if r != '!' {
		return nil, token.NewPosError(d.node(), "expected '!' in g2 mode")
	}
	r, _ = d.nextR()
	if r != '{' {
		return nil, token.NewPosError(d.node(), "expected '{' in g2 mode")
	}

	return nil, io.EOF
}
