package aggregate

import (
	"testing"

	"maure/pkg/document"
)

func TestBuildCountOnly(t *testing.T) {
	docs := []*document.Document{
		document.NewDocumentWithValues("1", map[string]interface{}{"message": "a"}),
		document.NewDocumentWithValues("2", map[string]interface{}{"message": "b"}),
	}
	result, err := Build(docs, "count", "")
	if err != nil {
		t.Fatalf("build count failed: %v", err)
	}
	if result.Count != 2 {
		t.Fatalf("expected count 2, got %d", result.Count)
	}
}

func TestBuildGroupByField(t *testing.T) {
	doc1 := document.NewDocument()
	doc1.Add(document.NewStringField("level", "INFO"))
	doc2 := document.NewDocument()
	doc2.Add(document.NewStringField("level", "ERROR"))
	doc3 := document.NewDocument()
	doc3.Add(document.NewStringField("level", "ERROR"))

	result, err := Build([]*document.Document{doc1, doc2, doc3}, "", "level")
	if err != nil {
		t.Fatalf("build group by field failed: %v", err)
	}
	if len(result.Buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(result.Buckets))
	}
	if result.Buckets[0].Key != "ERROR" || result.Buckets[0].Count != 2 {
		t.Fatalf("unexpected first bucket: %+v", result.Buckets[0])
	}
}

func TestBuildGroupByTime(t *testing.T) {
	doc1 := document.NewDocument()
	doc1.Add(document.NewStringField("timestamp", "2026-02-10T10:01:00Z"))
	doc2 := document.NewDocument()
	doc2.Add(document.NewStringField("timestamp", "2026-02-10T10:04:59Z"))
	doc3 := document.NewDocument()
	doc3.Add(document.NewStringField("timestamp", "2026-02-10T10:08:00Z"))

	result, err := Build([]*document.Document{doc1, doc2, doc3}, "", "time(5m)")
	if err != nil {
		t.Fatalf("build group by time failed: %v", err)
	}
	if len(result.Buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(result.Buckets))
	}
}

func TestBuildInvalidGroupWindow(t *testing.T) {
	_, err := Build([]*document.Document{}, "", "time(bad)")
	if err == nil {
		t.Fatalf("expected invalid group window error")
	}
}
