package parser

import (
	"testing"
)

const testMarkdown = `
<story id="DownloadWorkspace">
As a <role>developer</role>, I want to <goal>be able to download a workspace</goal> so that <reason>I can fix it</reason>
</story>

<story id="DownloadWorkspace">
Irgendein Text &lt; oops >
</story>

`

func TestParseMarkdownText(t *testing.T) {

}
