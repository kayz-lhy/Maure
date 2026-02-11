package components

import "testing"

func TestRangeComponent(t *testing.T) {
	comp := RangeComponent{}
	if _, ok, err := comp.TryParse("price", "[100 TO 300]", "price:[100 TO 300]"); err != nil || !ok {
		t.Fatalf("expected range success, err=%v", err)
	}
	if _, ok, err := comp.TryParse("price", "[100 300]", "price:[100 300]"); err == nil || ok {
		t.Fatalf("expected range error")
	}
}
