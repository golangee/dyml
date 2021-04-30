package parser

import (
	"fmt"
	"github.com/golangee/tadl/token"
	"testing"
)

func TestParseMarkup(t *testing.T) {
	story,err :=ParseMarkup("testdata/workspace/olzerp/service/context/yasa.support/requirement/developer/DownloadWorkspace.req.xml")
	if err != nil{
		fmt.Println(token.Explain(err))
		t.Fatal(err)
	}

	fmt.Println(toString(story))
}
