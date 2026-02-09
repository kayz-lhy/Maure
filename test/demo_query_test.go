package maure

import (
	"fmt"
	"testing"

	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/index"
	"maure/pkg/query"
)

func TestDemo_Query(t *testing.T) {
	// 创建索引
	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())
	defer idx.Close()

	// 添加测试文档
	docs := []*document.Document{
		document.NewDocumentWithValues("doc1", map[string]interface{}{
			"content": "Go is a programming language designed for concurrency",
		}),
		document.NewDocumentWithValues("doc2", map[string]interface{}{
			"content": "Python is great for data science and machine learning",
		}),
		document.NewDocumentWithValues("doc3", map[string]interface{}{
			"content": "Java is widely used in enterprise applications",
		}),
		document.NewDocumentWithValues("doc4", map[string]interface{}{
			"content": "Go and Python are both popular programming languages",
		}),
	}

	fmt.Println("=== 添加文档 ===")
	for _, doc := range docs {
		docID, err := idx.Add(doc)
		if err != nil {
			t.Fatalf("添加文档失败: %v", err)
		}
		fmt.Printf("添加文档: %s (ID: %d)\n", doc.ID(), docID)
	}

	fmt.Println("\n=== 文档列表 ===")
	for i := 1; i <= 4; i++ {
		field := docs[i-1].Get("content")
		fmt.Printf("doc%d: %s\n", i, field.StringValue())
	}

	fmt.Println("\n=== 查询演示 ===")

	// 1. 简单词项查询
	fmt.Println("\n--- 查询: 'programming' ---")
	q1 := query.NewTermQuery("programming")
	results, err := q1.Search(idx)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	fmt.Printf("找到 %d 个结果:\n", len(results))
	for _, r := range results {
		fmt.Printf("  DocID: %d, Score: %.4f\n", r.DocID, r.Score)
	}

	// 2. OR 查询
	fmt.Println("\n--- 查询: 'Go OR Java' ---")
	parser := query.NewQueryParser()
	q2, err := parser.Parse("Go OR Java")
	if err != nil {
		t.Fatalf("解析查询失败: %v", err)
	}
	results, err = q2.Search(idx)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	fmt.Printf("找到 %d 个结果:\n", len(results))
	for _, r := range results {
		fmt.Printf("  DocID: %d, Score: %.4f\n", r.DocID, r.Score)
	}

	// 3. AND 查询
	fmt.Println("\n--- 查询: 'Go AND programming' ---")
	q3, err := parser.Parse("Go AND programming")
	if err != nil {
		t.Fatalf("解析查询失败: %v", err)
	}
	results, err = q3.Search(idx)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	fmt.Printf("找到 %d 个结果:\n", len(results))
	for _, r := range results {
		fmt.Printf("  DocID: %d, Score: %.4f\n", r.DocID, r.Score)
	}

	// 4. NOT 查询
	fmt.Println("\n--- 查询: 'programming NOT Java' ---")
	q4, err := parser.Parse("programming NOT Java")
	if err != nil {
		t.Fatalf("解析查询失败: %v", err)
	}
	results, err = q4.Search(idx)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	fmt.Printf("找到 %d 个结果:\n", len(results))
	for _, r := range results {
		fmt.Printf("  DocID: %d, Score: %.4f\n", r.DocID, r.Score)
	}

	// 5. 组合查询
	fmt.Println("\n--- 查询: '(Go OR Python) AND programming' ---")
	q5, err := parser.Parse("(Go OR Python) AND programming")
	if err != nil {
		t.Fatalf("解析查询失败: %v", err)
	}
	results, err = q5.Search(idx)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	fmt.Printf("找到 %d 个结果:\n", len(results))
	for _, r := range results {
		fmt.Printf("  DocID: %d, Score: %.4f\n", r.DocID, r.Score)
	}

	// 6. 短语查询
	fmt.Println("\n--- 查询: '\"programming language\"' ---")
	q6, err := parser.Parse("\"programming language\"")
	if err != nil {
		t.Fatalf("解析查询失败: %v", err)
	}
	results, err = q6.Search(idx)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	fmt.Printf("找到 %d 个结果:\n", len(results))
	for _, r := range results {
		fmt.Printf("  DocID: %d, Score: %.4f\n", r.DocID, r.Score)
	}

	// 7. BooleanQuery API
	fmt.Println("\n--- 使用 BooleanQuery API: Go AND (Python OR Java) ---")
	boolQuery := query.NewBooleanQuery()
	boolQuery.Add(query.NewTermQuery("Go"), query.OccurMust, 1.0)
	boolQuery.Add(query.NewTermQuery("Python"), query.OccurShould, 1.0)
	boolQuery.Add(query.NewTermQuery("Java"), query.OccurShould, 1.0)
	results, err = boolQuery.Search(idx)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	fmt.Printf("找到 %d 个结果:\n", len(results))
	for _, r := range results {
		fmt.Printf("  DocID: %d, Score: %.4f\n", r.DocID, r.Score)
	}

	fmt.Println("\n=== 查询演示完成 ===")
}
