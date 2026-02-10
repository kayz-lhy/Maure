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

func buildTestServer(t *testing.T) *Server {
	t.Helper()

	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())
	readerDocs := make(map[int64]*document.Document, 2)
	source := make(map[int64]int64, 2)

	doc1 := document.NewDocument()
	doc1.SetID("doc-1")
	doc1.Add(document.NewTextField("message", "alpha request failed"))
	doc1.Add(document.NewStringField("level", "error"))
	doc1.Add(document.NewStringField("service", "api"))
	id1, err := idx.Add(doc1)
	if err != nil {
		t.Fatalf("add doc1 failed: %v", err)
	}
	readerDocs[1] = doc1
	source[id1] = 1

	doc2 := document.NewDocument()
	doc2.SetID("doc-2")
	doc2.Add(document.NewTextField("message", "alpha request retry"))
	doc2.Add(document.NewStringField("level", "warn"))
	id2, err := idx.Add(doc2)
	if err != nil {
		t.Fatalf("add doc2 failed: %v", err)
	}
	readerDocs[2] = doc2
	source[id2] = 2

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

func TestHandleSearchIncludeDocReturnsSummary(t *testing.T) {
	s := buildTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/search?q=alpha&include_doc=true", nil)
	rr := httptest.NewRecorder()
	s.handleSearch(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var hits []SearchHit
	if err := json.Unmarshal(rr.Body.Bytes(), &hits); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected hits")
	}
	if hits[0].Doc == nil || hits[0].Doc.Summary == "" {
		t.Fatalf("expected doc summary, got %+v", hits[0].Doc)
	}
}

func TestHandleSearchFieldsWhitelist(t *testing.T) {
	s := buildTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/search?q=alpha&fields=message,level", nil)
	rr := httptest.NewRecorder()
	s.handleSearch(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var hits []SearchHit
	if err := json.Unmarshal(rr.Body.Bytes(), &hits); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected hits")
	}

	doc := hits[0].Doc
	if doc == nil {
		t.Fatalf("expected doc payload")
	}
	if _, ok := doc.Fields["message"]; !ok {
		t.Fatalf("expected whitelisted field message")
	}
	if _, ok := doc.Fields["level"]; !ok {
		t.Fatalf("expected whitelisted field level")
	}
	if _, ok := doc.Fields["service"]; ok {
		t.Fatalf("unexpected non-whitelisted field service")
	}
}

func TestHandleSearchInvalidFieldsReturnsBadRequest(t *testing.T) {
	s := buildTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/search?q=alpha&fields=message%3Bdrop", nil)
	rr := httptest.NewRecorder()
	s.handleSearch(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
