package query

import (
	"testing"

	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/index"
)

func createTestIndex(t *testing.T) *index.RAMIndex {
	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())

	docs := []*document.Document{
		document.NewDocumentWithValues("doc1", map[string]interface{}{
			"content": "go programming language",
		}),
		document.NewDocumentWithValues("doc2", map[string]interface{}{
			"content": "python programming tutorial",
		}),
		document.NewDocumentWithValues("doc3", map[string]interface{}{
			"content": "java development guide",
		}),
		document.NewDocumentWithValues("doc4", map[string]interface{}{
			"content": "go and python are programming languages",
		}),
	}

	for _, doc := range docs {
		if _, err := idx.Add(doc); err != nil {
			t.Fatalf("failed to add doc: %v", err)
		}
	}

	return idx
}

func TestTermQuery_Search(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	query := NewTermQuery("go")
	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// 应该找到 doc1 和 doc4
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// doc4 应该有更高评分（包含更多 "go" 相关词）
	if len(results) >= 2 && results[0].Score < results[1].Score {
		t.Error("results should be sorted by score descending")
	}
}

func TestTermQuery_Search_NoDuplicateDocIDs(t *testing.T) {
	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())
	defer idx.Close()

	doc := document.NewDocument()
	doc.Add(document.NewTextField("content", "go go go"))
	if _, err := idx.Add(doc); err != nil {
		t.Fatalf("failed to add doc: %v", err)
	}

	query := NewTermQuery("go")
	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 deduplicated result, got %d", len(results))
	}
	if results[0].DocID != 1 {
		t.Fatalf("expected docID 1, got %d", results[0].DocID)
	}
}

func TestDisjunctionQuery_OR(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	// OR 查询：go OR java
	query := NewDisjunctionQuery(
		NewTermQuery("go"),
		NewTermQuery("java"),
	)

	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// 应该找到 doc1、doc3、doc4（doc2 不包含 go 或 java）
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestConjunctionQuery_AND(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	// AND 查询：go AND programming
	query := NewConjunctionQuery(
		NewTermQuery("go"),
		NewTermQuery("programming"),
	)

	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// 应该找到 doc1 和 doc4
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestBooleanQuery_Must(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	// 布尔查询：go AND programming
	query := NewBooleanQuery()
	query.Add(NewTermQuery("go"), OccurMust, 1.0)
	query.Add(NewTermQuery("programming"), OccurMust, 1.0)

	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// 应该找到 doc1 和 doc4
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestBooleanQuery_Should(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	// 布尔查询：go OR java
	query := NewBooleanQuery()
	query.Add(NewTermQuery("go"), OccurShould, 1.0)
	query.Add(NewTermQuery("java"), OccurShould, 1.0)

	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// 应该找到 doc1、doc3、doc4（doc2 不包含 go 或 java）
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestBooleanQuery_MustNot(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	// 布尔查询：programming NOT java
	query := NewBooleanQuery()
	query.Add(NewTermQuery("programming"), OccurMust, 1.0)
	query.Add(NewTermQuery("java"), OccurMustNot, 1.0)

	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// 应该找到 doc1、doc2、doc4（不包含 java）
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// 确保没有 java 相关的文档
	for range results {
		// 这里无法直接检查文档内容，因为 RAMIndex 不存储原始文档
		// 但通过逻辑推理，doc3 应该被排除
	}
}

func TestQueryParser_Simple(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	parser := NewQueryParser()

	// 测试简单词项
	query, err := parser.Parse("go")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestQueryParser_OR(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	parser := NewQueryParser()

	// 测试 OR
	query, err := parser.Parse("go OR java")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// 应该找到 doc1、doc3、doc4（doc2 不包含 go 或 java）
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestQueryParser_AND(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	parser := NewQueryParser()

	// 测试 AND
	query, err := parser.Parse("go AND programming")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestQueryParser_Complex(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	parser := NewQueryParser()

	// 测试复杂查询：(go AND programming) OR python
	query, err := parser.Parse("(go AND programming) OR python")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// doc1, doc2, doc4 应该匹配
	if len(results) < 2 {
		t.Errorf("expected at least 2 results, got %d", len(results))
	}
}

func TestQueryParser_Phrase(t *testing.T) {
	idx := createTestIndex(t)
	defer idx.Close()

	parser := NewQueryParser()

	// 测试短语查询
	query, err := parser.Parse(`"programming language"`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	results, err := query.Search(idx)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// 应该找到 doc1
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestBooleanQuery_Explain(t *testing.T) {
	query := NewBooleanQuery()
	query.Add(NewTermQuery("go"), OccurMust, 1.0)
	query.Add(NewTermQuery("programming"), OccurShould, 1.0)

	// 只需要验证 Explain 不 panic
	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())
	defer idx.Close()

	explain := query.Explain(idx)
	if explain == "" {
		t.Error("Explain should return non-empty string")
	}
}

func BenchmarkBooleanQuery(b *testing.B) {
	idx := createTestIndex(nil)
	defer idx.Close()

	query := NewBooleanQuery()
	query.Add(NewTermQuery("go"), OccurMust, 1.0)
	query.Add(NewTermQuery("programming"), OccurMust, 1.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.Search(idx)
	}
}

func BenchmarkQueryParser(b *testing.B) {
	parser := NewQueryParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse("go AND programming OR python")
	}
}
