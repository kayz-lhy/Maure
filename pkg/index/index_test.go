package index

import (
	"testing"

	"maure/pkg/analyzer"
	"maure/pkg/document"
)

func TestRAMIndex_Add(t *testing.T) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())

	// 添加文档
	doc := document.NewDocument()
	doc.Add(document.NewTextField("title", "Hello World"))
	doc.Add(document.NewTextField("content", "Test document"))

	docID, err := idx.Add(doc)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	if docID != 1 {
		t.Errorf("expected docID 1, got %d", docID)
	}

	if idx.DocCount() != 1 {
		t.Errorf("expected docCount 1, got %d", idx.DocCount())
	}

	idx.Close()
}

func TestRAMIndex_MultipleDocs(t *testing.T) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())

	// 添加多个文档
	docs := []*document.Document{
		document.NewDocumentWithValues("doc1", map[string]interface{}{
			"title":   "Go Programming Language",
			"content": "Go is a programming language designed at Google",
		}),
		document.NewDocumentWithValues("doc2", map[string]interface{}{
			"title":   "Python Programming",
			"content": "Python is a popular programming language",
		}),
		document.NewDocumentWithValues("doc3", map[string]interface{}{
			"title":   "Rust Programming",
			"content": "Rust is a systems programming language",
		}),
	}

	for i, doc := range docs {
		docID, err := idx.Add(doc)
		if err != nil {
			t.Fatalf("failed to add doc %d: %v", i, err)
		}
		if docID != int64(i+1) {
			t.Errorf("expected docID %d, got %d", i+1, docID)
		}
	}

	if idx.DocCount() != 3 {
		t.Errorf("expected docCount 3, got %d", idx.DocCount())
	}

	// 验证词项
	if !idx.inverted.ContainsTerm("go") {
		t.Error("expected term 'go' to exist")
	}
	if !idx.inverted.ContainsTerm("programming") {
		t.Error("expected term 'programming' to exist")
	}
	// Python 应该被转换为小写
	if !idx.inverted.ContainsTerm("python") {
		t.Error("expected term 'python' to exist (lowercase)")
	}
	// Rust 应该被转换为小写
	if !idx.inverted.ContainsTerm("rust") {
		t.Error("expected term 'rust' to exist (lowercase)")
	}

	idx.Close()
}

func TestRAMIndex_Delete(t *testing.T) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())

	doc := document.NewDocument()
	doc.Add(document.NewTextField("title", "Test"))

	docID, _ := idx.Add(doc)

	if idx.DocCount() != 1 {
		t.Errorf("expected docCount 1, got %d", idx.DocCount())
	}

	if err := idx.Delete(docID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	if idx.DocCount() != 0 {
		t.Errorf("expected docCount 0, got %d", idx.DocCount())
	}

	idx.Close()
}

func TestRAMIndex_Update(t *testing.T) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())

	// 添加文档
	doc1 := document.NewDocument()
	doc1.Add(document.NewTextField("title", "Original"))

	docID, _ := idx.Add(doc1)

	// 更新文档
	doc2 := document.NewDocument()
	doc2.Add(document.NewTextField("title", "Updated"))

	if err := idx.Update(docID, doc2); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// 验证词项
	if !idx.inverted.ContainsTerm("updated") {
		t.Error("expected term 'updated' to exist")
	}
	if idx.inverted.ContainsTerm("original") {
		t.Error("expected term 'original' to be removed")
	}

	idx.Close()
}

func TestRAMIndex_TermQuery(t *testing.T) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())

	// 添加文档
	docs := []*document.Document{
		document.NewDocumentWithValues("doc1", map[string]interface{}{
			"content": "apple banana cherry",
		}),
		document.NewDocumentWithValues("doc2", map[string]interface{}{
			"content": "banana date elderberry",
		}),
		document.NewDocumentWithValues("doc3", map[string]interface{}{
			"content": "cherry fig grape",
		}),
	}

	for _, doc := range docs {
		idx.Add(doc)
	}

	// 查询 "banana" - 在 doc1 和 doc2 中都存在
	query := NewTermQuery("banana")
	results, err := idx.Search(query, 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// 验证返回的文档 ID（应该包含 doc1 和 doc2）
	docIDs := make(map[int64]bool)
	for _, r := range results {
		docIDs[r.DocID] = true
	}
	if !docIDs[1] || !docIDs[2] {
		t.Errorf("expected docIDs 1 and 2, got %v", docIDs)
	}

	idx.Close()
}

func TestRAMIndex_DocCount(t *testing.T) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())

	if idx.DocCount() != 0 {
		t.Errorf("expected 0, got %d", idx.DocCount())
	}

	doc := document.NewDocument()
	idx.Add(doc)

	if idx.DocCount() != 1 {
		t.Errorf("expected 1, got %d", idx.DocCount())
	}

	idx.Close()
}

func TestRAMIndex_Close(t *testing.T) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())

	doc := document.NewDocument()
	idx.Add(doc)

	if err := idx.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// 关闭后应该返回错误
	_, err := idx.Add(document.NewDocument())
	if err == nil {
		t.Error("expected error after close")
	}
}

func TestRAMIndex_NumTerms(t *testing.T) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())

	if idx.inverted.NumTerms() != 0 {
		t.Errorf("expected 0 terms, got %d", idx.inverted.NumTerms())
	}

	doc := document.NewDocument()
	doc.Add(document.NewTextField("content", "hello world test"))
	idx.Add(doc)

	if idx.inverted.NumTerms() != 3 {
		t.Errorf("expected 3 terms, got %d", idx.inverted.NumTerms())
	}

	idx.Close()
}

func BenchmarkRAMIndex_Add(b *testing.B) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())
	doc := document.NewDocument()
	doc.Add(document.NewTextField("content", "benchmark test document"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Add(doc)
	}
	idx.Close()
}

func BenchmarkRAMIndex_Search(b *testing.B) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())

	// 添加测试文档
	for i := 0; i < 1000; i++ {
		doc := document.NewDocument()
		doc.Add(document.NewTextField("content", "test document number"))
		idx.Add(doc)
	}

	query := NewTermQuery("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Search(query, 10)
	}
	idx.Close()
}
