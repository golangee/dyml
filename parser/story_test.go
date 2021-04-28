package parser

import (
	"fmt"
	"github.com/golangee/tadl/token"
	"testing"
)

func TestParseStory(t *testing.T) {
	story,err :=ParseStory("testdata/workspace/olzerp/service/requirement/yasa/DeleteTickets.story")
	if err != nil{
		fmt.Println(token.Explain(err))
		t.Fatal(err)
	}

	fmt.Println(toString(story))
}
