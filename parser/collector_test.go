package parser

import (
	"fmt"
	"github.com/golangee/tadl/token"
	"testing"
)

func Test_collect(t *testing.T) {
	ws,err :=collect("testdata")
	if err != nil{
		fmt.Println(token.Explain(err))
		t.Fatal(err)
	}

	fmt.Printf("%+v\n",ws)
}
