package components

import "testing"

func TestTermComponent(t *testing.T) {
	node, ok, err := (TermComponent{}).TryParse("title", "IPHONe", "title:IPHONe")
	if err != nil || !ok || node.Kind != ExprTerm {
		t.Fatalf("unexpected term parse result: node=%#v ok=%v err=%v", node, ok, err)
	}
}
