package query

import (
	"testing"

	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/index"
)

func createAdvancedIndex(t *testing.T) *index.RAMIndex {
	t.Helper()

	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())
	docs := []*document.Document{
		document.NewDocumentWithValues("doc1", map[string]interface{}{
			"title":     "iphone 15",
			"name":      "roam",
			"price":     int64(199),
			"timestamp": "2026-02-10T09:15:00Z",
		}),
		document.NewDocumentWithValues("doc2", map[string]interface{}{
			"title":     "ipad air",
			"name":      "room",
			"price":     int64(499),
			"timestamp": "2026-02-10T09:45:00Z",
		}),
		document.NewDocumentWithValues("doc3", map[string]interface{}{
			"title":     "android phone",
			"name":      "foam",
			"price":     int64(299),
			"timestamp": "2026-02-10T10:30:00Z",
		}),
	}

	for _, doc := range docs {
		if _, err := idx.Add(doc); err != nil {
			t.Fatalf("failed to add doc: %v", err)
		}
	}

	return idx
}

func TestParser_FieldRangeNumeric(t *testing.T) {
	parser := NewQueryParser()
	q, err := parser.Parse("price:[100 TO 300]")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, ok := q.(*RangeQuery); !ok {
		t.Fatalf("expected RangeQuery, got %T", q)
	}
}

func TestParser_FieldRangeTime(t *testing.T) {
	parser := NewQueryParser()
	q, err := parser.Parse("timestamp:[2026-02-10T09:00:00Z TO 2026-02-10T10:00:00Z]")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, ok := q.(*RangeQuery); !ok {
		t.Fatalf("expected RangeQuery, got %T", q)
	}
}

func TestParser_FieldWildcard(t *testing.T) {
	parser := NewQueryParser()
	q, err := parser.Parse("title:iph*")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, ok := q.(*WildcardQuery); !ok {
		t.Fatalf("expected WildcardQuery, got %T", q)
	}
}

func TestParser_FieldWildcard_PreserveFieldCase(t *testing.T) {
	parser := NewQueryParser()
	q, err := parser.Parse("Title:iph*")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	wq, ok := q.(*WildcardQuery)
	if !ok {
		t.Fatalf("expected WildcardQuery, got %T", q)
	}
	if wq.Field != "Title" {
		t.Fatalf("expected field case preserved, got %q", wq.Field)
	}
}

func TestParser_FieldFuzzy(t *testing.T) {
	parser := NewQueryParser()
	q, err := parser.Parse("name:roam~1")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, ok := q.(*FuzzyQuery); !ok {
		t.Fatalf("expected FuzzyQuery, got %T", q)
	}
}

func TestParser_RejectUnsupportedWildcard(t *testing.T) {
	parser := NewQueryParser()
	if _, err := parser.Parse("title:*phone"); err == nil {
		t.Fatalf("expected parse error for leading wildcard")
	}
}

func TestParser_RejectUnsupportedFuzzyDistance(t *testing.T) {
	parser := NewQueryParser()
	if _, err := parser.Parse("name:roam~2"); err == nil {
		t.Fatalf("expected parse error for fuzzy ~2")
	}
}

func TestRangeQuery_Numeric(t *testing.T) {
	idx := createAdvancedIndex(t)
	defer idx.Close()

	q := NewRangeQuery("price", "100", "300", RangeValueNumber, true)
	results, err := q.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 docs in price range, got %d", len(results))
	}
}

func TestRangeQuery_Time(t *testing.T) {
	idx := createAdvancedIndex(t)
	defer idx.Close()

	q := NewRangeQuery("timestamp", "2026-02-10T09:00:00Z", "2026-02-10T10:00:00Z", RangeValueTime, true)
	results, err := q.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 docs in time range, got %d", len(results))
	}
}

func TestWildcardQuery_FieldPrefix(t *testing.T) {
	idx := createAdvancedIndex(t)
	defer idx.Close()

	q := NewWildcardQuery("title", "iph")
	results, err := q.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 wildcard hit, got %d", len(results))
	}
}

func TestFuzzyQuery_DistanceOne(t *testing.T) {
	idx := createAdvancedIndex(t)
	defer idx.Close()

	q := NewFuzzyQuery("name", "roam", 1)
	results, err := q.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 fuzzy hits, got %d", len(results))
	}
}

func TestAdvancedQuery_Composition(t *testing.T) {
	idx := createAdvancedIndex(t)
	defer idx.Close()

	parser := NewQueryParser()
	q, err := parser.Parse("price:[100 TO 300] AND title:iph*")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	results, err := q.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 composed hit, got %d", len(results))
	}
}

func TestAdvancedQuery_NotFilter(t *testing.T) {
	idx := createAdvancedIndex(t)
	defer idx.Close()

	parser := NewQueryParser()
	q, err := parser.Parse("price:[100 TO 500] NOT title:iph*")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	results, err := q.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 hits after NOT filter, got %d", len(results))
	}
}
