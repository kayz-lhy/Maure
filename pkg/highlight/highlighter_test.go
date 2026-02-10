package highlight

import "testing"

func TestHighlighterFindTerm(t *testing.T) {
	h := NewHighlighter()
	start, end, ok := h.FindTerm("Go language is great", "language")
	if !ok {
		t.Fatalf("expected term found")
	}
	if start != 3 || end != 11 {
		t.Fatalf("unexpected range: [%d,%d)", start, end)
	}
}

func TestHighlighterFindTermCaseInsensitive(t *testing.T) {
	h := NewHighlighter()
	start, end, ok := h.FindTerm("Error happened", "error")
	if !ok {
		t.Fatalf("expected case-insensitive match")
	}
	if start != 0 || end != 5 {
		t.Fatalf("unexpected range: [%d,%d)", start, end)
	}
}

func TestHighlighterExtract(t *testing.T) {
	h := NewHighlighter()
	fragment, ok := h.Extract("abc def ghi jkl", "ghi")
	if !ok {
		t.Fatalf("expected fragment")
	}
	if fragment.Start != 8 || fragment.End != 11 {
		t.Fatalf("unexpected highlight range: %v", fragment)
	}
	if fragment.Text == "" {
		t.Fatalf("fragment text should not be empty")
	}
}

func TestHighlighterNoMatch(t *testing.T) {
	h := NewHighlighter()
	if _, _, ok := h.FindTerm("hello", "world"); ok {
		t.Fatalf("unexpected match")
	}
	if _, ok := h.Extract("hello", "world"); ok {
		t.Fatalf("unexpected fragment")
	}
}
