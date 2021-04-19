package parser

import (
	"fmt"
	"github.com/golangee/tadl/token"
	"testing"
)

func TestParseProject(t *testing.T) {
	prj, err := ParseProject(".")
	if err != nil {
		fmt.Println(token.Explain(err))
		t.Fatal(err)
	}

	fmt.Println(toString(prj))
}
