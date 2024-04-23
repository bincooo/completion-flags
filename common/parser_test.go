package common

import (
	"testing"
)

func TestParse(t *testing.T) {
	//content := "111<!-- hello --><@-1>2<debug />22<regex> <![CDATA[<@-1>]]> </regex>"
	content := "111<!-- hello --><@-1>2<debug />22<@-1>333</@-1>"
	parser := XmlParser{
		[]string{
			"regex",
			`r:@-*\d+`,
			"histories",
		},
	}

	nodes := parser.parse(content)
	t.Log(nodes)
}
