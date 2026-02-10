package command

import (
	"strings"
	"testing"

	"maure/pkg/document"
)

func TestSummarizeDocPrefersMessage(t *testing.T) {
	doc := document.NewDocument()
	doc.Add(document.NewTextField("message", "request failed"))
	doc.Add(document.NewTextField("raw", `{"message":"request failed"}`))

	got := summarizeDoc(doc)
	if got != "request failed" {
		t.Fatalf("expected message summary, got %q", got)
	}
}

func TestSummarizeDocFallsBackToRaw(t *testing.T) {
	doc := document.NewDocument()
	doc.Add(document.NewTextField("raw", strings.Repeat("a", 120)))

	got := summarizeDoc(doc)
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated raw summary, got %q", got)
	}
	if len([]rune(got)) > 103 {
		t.Fatalf("unexpected summary length: %d", len([]rune(got)))
	}
}
