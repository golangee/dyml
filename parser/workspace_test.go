package parser

import (
	"fmt"
	"github.com/golangee/tadl/token"
	"testing"
)

func TestParseDir(t *testing.T) {
	ws,err :=Parse("testdata")
	if err != nil{
		fmt.Println(token.Explain(err))
		t.Fatal(err)
	}

	fmt.Println(toString(ws))
}
