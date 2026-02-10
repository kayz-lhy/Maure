package command

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/highlight"
	"maure/pkg/index"
	"maure/pkg/query"
)

type mockIndexReader struct {
	docs map[int64]*document.Document
}

func (m *mockIndexReader) DocCount() int64 {
	return int64(len(m.docs))
}

func (m *mockIndexReader) GetDocument(docID int64) (*document.Document, error) {
	doc, ok := m.docs[docID]
	if !ok {
		return nil, errors.New("not found")
	}
	return doc, nil
}

func (m *mockIndexReader) Exists(docID int64) bool {
	_, ok := m.docs[docID]
	return ok
}

func (m *mockIndexReader) GetTerms() []string {
	return nil
}

func (m *mockIndexReader) Close() error {
	return nil
}

func buildTestServerWithDocs(t *testing.T, n int) *Server {
	t.Helper()

	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())
	source := make(map[int64]int64, n)
	readerDocs := make(map[int64]*document.Document, n)

	for i := 1; i <= n; i++ {
		doc := document.NewDocument()
		doc.Add(document.NewTextField("message", "alpha"))
		doc.Add(document.NewStringField("id", string(rune('a'+i-1))))

		docID, err := idx.Add(doc)
		if err != nil {
			t.Fatalf("add doc failed: %v", err)
		}
		source[docID] = int64(i)
		readerDocs[int64(i)] = doc
	}

	return &Server{
		idx:         idx,
		parser:      query.NewQueryParser(),
		highlighter: highlight.NewHighlighter(),
		sourceDocID: source,
		ctx: &IndexContext{
			Reader: &mockIndexReader{docs: readerDocs},
		},
	}
}

func TestHandleSearchPaginationPagesWithoutOverlap(t *testing.T) {
	s := buildTestServerWithDocs(t, 5)

	collect := func(rawQuery string) []int64 {
		req := httptest.NewRequest(http.MethodGet, "/search?"+rawQuery, nil)
		rr := httptest.NewRecorder()
		s.handleSearch(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var payload struct {
			Results []struct {
				DocID int64 `json:"doc_id"`
			} `json:"results"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response failed: %v", err)
		}

		docIDs := make([]int64, 0, len(payload.Results))
		for _, r := range payload.Results {
			docIDs = append(docIDs, r.DocID)
		}
		return docIDs
	}

	page1 := collect("q=alpha&from=0&size=2")
	page2 := collect("q=alpha&from=2&size=2")
	page3 := collect("q=alpha&from=4&size=2")

	if len(page1) != 2 || len(page2) != 2 || len(page3) != 1 {
		t.Fatalf("unexpected page size: p1=%d p2=%d p3=%d", len(page1), len(page2), len(page3))
	}

	seen := make(map[int64]bool)
	for _, id := range append(append(page1, page2...), page3...) {
		if seen[id] {
			t.Fatalf("duplicated doc id across pages: %d", id)
		}
		seen[id] = true
	}
	if len(seen) != 5 {
		t.Fatalf("expected 5 unique docs across pages, got %d", len(seen))
	}
}

func TestHandleSearchPaginationParamValidation(t *testing.T) {
	s := buildTestServerWithDocs(t, 3)

	cases := []string{
		"q=alpha&from=-1&size=2",
		"q=alpha&from=0&size=0",
		"q=alpha&from=0&size=201",
		"q=alpha&from=abc&size=2",
	}

	for _, rawQuery := range cases {
		req := httptest.NewRequest(http.MethodGet, "/search?"+rawQuery, nil)
		rr := httptest.NewRecorder()
		s.handleSearch(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %q, got %d", rawQuery, rr.Code)
		}
	}
}
