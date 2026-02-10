package index

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"testing"

	"maure/pkg/analyzer"
	"maure/pkg/document"
)

func setupStableBench(b *testing.B) {
	runtime.LockOSThread()
	oldProcs := runtime.GOMAXPROCS(1)
	oldGCPercent := debug.SetGCPercent(-1)
	runtime.GC()
	b.Cleanup(func() {
		runtime.GOMAXPROCS(oldProcs)
		debug.SetGCPercent(oldGCPercent)
		runtime.UnlockOSThread()
	})
}

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

func TestRAMIndex_PostingsAggregatedPerDoc(t *testing.T) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())
	defer idx.Close()

	doc := document.NewDocument()
	doc.Add(document.NewTextField("content", "go go gopher"))
	if _, err := idx.Add(doc); err != nil {
		t.Fatalf("failed to add doc: %v", err)
	}

	postings, err := idx.inverted.GetPostings("go")
	if err != nil {
		t.Fatalf("expected postings for go: %v", err)
	}
	if len(postings.DocIDs) != 1 {
		t.Fatalf("expected one posting entry for single doc, got %d", len(postings.DocIDs))
	}
	if postings.Freqs[0] != 2 {
		t.Fatalf("expected term freq 2, got %d", postings.Freqs[0])
	}
	if len(postings.Positions[0]) != 2 || postings.Positions[0][0] != 0 || postings.Positions[0][1] != 1 {
		t.Fatalf("unexpected positions: %v", postings.Positions[0])
	}

	results, err := idx.Search(NewTermQuery("go"), 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected deduplicated one result, got %d", len(results))
	}
}

func TestRAMIndex_FieldLengthUsesTokenCountOnly(t *testing.T) {
	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())
	defer idx.Close()

	doc := document.NewDocument()
	doc.Add(document.NewTextField("content", "go gopher"))
	doc.Add(document.NewStringField("id", "DOC-001"))

	docID, err := idx.Add(doc)
	if err != nil {
		t.Fatalf("failed to add doc: %v", err)
	}

	if got := idx.inverted.FieldLength(docID); got != 2 {
		t.Fatalf("expected tokenized field length 2, got %d", got)
	}
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
	setupStableBench(b)

	idx := NewRAMIndex(analyzer.NewStandardAnalyzer())

	// 添加测试文档
	for i := 0; i < 1000; i++ {
		doc := document.NewDocument()
		doc.Add(document.NewTextField("content", "test document number"))
		idx.Add(doc)
	}

	query := NewTermQuery("test")

	// 预热，降低首次执行抖动对统计的影响。
	for i := 0; i < 200; i++ {
		_, _ = idx.Search(query, 10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Search(query, 10)
	}
	idx.Close()
}

func BenchmarkRAMIndex_SearchLargeDataset(b *testing.B) {
	cases := []struct {
		name     string
		docCount int
	}{
		{name: "10k", docCount: 10000},
		{name: "50k", docCount: 50000},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			setupStableBench(b)

			idx := NewRAMIndex(analyzer.NewStandardAnalyzer())
			defer idx.Close()

			// 构造大规模数据集：
			// - "error" 为高频词（每个文档都有）
			// - "criticalspike" 为低频词（约 1% 文档）
			// - "traceid%d" 为唯一词（每个文档一个）
			for i := 0; i < c.docCount; i++ {
				doc := document.NewDocument()
				content := fmt.Sprintf(
					"error request timeout service api traceid%d",
					i,
				)
				if i%100 == 0 {
					content += " criticalspike"
				}
				doc.Add(document.NewTextField("content", content))
				if _, err := idx.Add(doc); err != nil {
					b.Fatalf("add doc failed: %v", err)
				}
			}

			hotQuery := NewTermQuery("error")
			rareQuery := NewTermQuery("criticalspike")
			uniqueQuery := NewTermQuery(fmt.Sprintf("traceid%d", c.docCount/2))

			b.Run("hot-top10", func(b *testing.B) {
				setupStableBench(b)
				b.ReportAllocs()
				for i := 0; i < 200; i++ {
					_, _ = idx.Search(hotQuery, 10)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := idx.Search(hotQuery, 10); err != nil {
						b.Fatalf("search failed: %v", err)
					}
				}
			})

			b.Run("hot-top100", func(b *testing.B) {
				setupStableBench(b)
				b.ReportAllocs()
				for i := 0; i < 200; i++ {
					_, _ = idx.Search(hotQuery, 100)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := idx.Search(hotQuery, 100); err != nil {
						b.Fatalf("search failed: %v", err)
					}
				}
			})

			b.Run("hot-top1000", func(b *testing.B) {
				setupStableBench(b)
				b.ReportAllocs()
				for i := 0; i < 200; i++ {
					_, _ = idx.Search(hotQuery, 1000)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := idx.Search(hotQuery, 1000); err != nil {
						b.Fatalf("search failed: %v", err)
					}
				}
			})

			b.Run("rare-top10", func(b *testing.B) {
				setupStableBench(b)
				b.ReportAllocs()
				for i := 0; i < 200; i++ {
					_, _ = idx.Search(rareQuery, 10)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := idx.Search(rareQuery, 10); err != nil {
						b.Fatalf("search failed: %v", err)
					}
				}
			})

			b.Run("unique-top10", func(b *testing.B) {
				setupStableBench(b)
				b.ReportAllocs()
				for i := 0; i < 200; i++ {
					_, _ = idx.Search(uniqueQuery, 10)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := idx.Search(uniqueQuery, 10); err != nil {
						b.Fatalf("search failed: %v", err)
					}
				}
			})
		})
	}
}
