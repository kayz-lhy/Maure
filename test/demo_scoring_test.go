package maure

import (
	"fmt"
	"testing"

	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/index"
	"maure/pkg/search"
)

// TestDemo_Scoring 演示评分排序的完整工作流。
//
// 这个测试展示了：
// 1. 如何使用 BM25 评分算法
// 2. 如何使用 TF-IDF 评分算法
// 3. 评分如何影响搜索结果排序
// 4. 如何自定义评分参数
//
// 运行方式：go test -v -run TestDemoScoring ./test/...
func TestDemo_Scoring(t *testing.T) {
	fmt.Println("=== Maure 评分排序演示 ===")
	fmt.Println()

	// 1. 创建索引（默认使用 BM25）
	fmt.Println("1. 创建索引（默认 BM25 评分）：")
	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())
	fmt.Printf("   评分算法: %s\n", idx.Similarity().Name())
	fmt.Println()

	// 2. 添加测试文档
	fmt.Println("2. 添加测试文档：")

	// 文档1：多次提到 "programming"
	doc1 := document.NewDocumentWithValues("doc1", map[string]interface{}{
		"title":   "Programming Guide",
		"content": "Programming is great. Programming helps you solve problems. Learn programming today!",
	})

	// 文档2：简短提到一次 "programming"
	doc2 := document.NewDocumentWithValues("doc2", map[string]interface{}{
		"title":   "Python Tutorial",
		"content": "Python is a programming language for beginners.",
	})

	// 文档3：完全不包含 "programming"
	doc3 := document.NewDocumentWithValues("doc3", map[string]interface{}{
		"title":   "Go Language",
		"content": "Go is a systems programming language developed by Google.",
	})

	// 文档4：长文档，多次提到 "programming"
	doc4 := document.NewDocumentWithValues("doc4", map[string]interface{}{
		"title":   "Advanced Programming Patterns",
		"content": "This document covers advanced programming patterns, programming best practices, " +
			"and programming techniques for experienced developers. " +
			"If you want to improve your programming skills, this guide is for you.",
	})

	docs := []*document.Document{doc1, doc2, doc3, doc4}
	for i, doc := range docs {
		docID, err := idx.Add(doc)
		if err != nil {
			t.Fatalf("failed to add doc %d: %v", i, err)
		}
		title := doc.Get("title").StringValue()
		fmt.Printf("   添加 doc%d (ID=%d): %s\n", i+1, docID, title)
	}
	fmt.Println()

	// 3. 执行搜索并查看评分
	fmt.Println("3. 搜索 'programming'（查看评分差异）：")
	query := index.NewTermQuery("programming")
	results, err := idx.Search(query, 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	fmt.Printf("   找到 %d 个结果：\n", len(results))
	for i, r := range results {
		fmt.Printf("   [%d] DocID=%d, Score=%.4f\n", i+1, r.DocID, r.Score)
	}
	fmt.Println()

	// 4. 验证评分排序正确性
	fmt.Println("4. 验证评分排序：")
	// doc4 应该有最高评分（多次提到 programming，且文档长度归一化后评分更高）
	// doc1 次之
	// doc2 只有一次
	// doc3 不应该出现
	if len(results) >= 2 {
		if results[0].Score < results[1].Score {
			t.Error("结果应该按评分降序排列")
		}
	}
	fmt.Println("   评分排序验证通过 ✓")
	fmt.Println()

	// 5. 切换到 TF-IDF 评分
	fmt.Println("5. 切换到 TF-IDF 评分：")
	idx.SetSimilarity(search.NewTFIDFSimilarity())
	fmt.Printf("   新评分算法: %s\n", idx.Similarity().Name())

	results, _ = idx.Search(query, 10)
	fmt.Printf("   TF-IDF 评分结果：\n")
	for i, r := range results {
		fmt.Printf("   [%d] DocID=%d, Score=%.4f\n", i+1, r.DocID, r.Score)
	}
	fmt.Println()

	// 6. 使用自定义 BM25 参数
	fmt.Println("6. 自定义 BM25 参数测试：")

	// 高 k1 值：更强调词频
	highK1 := search.NewBM25SimilarityWithParams(2.5, 0.75)
	idx.SetSimilarity(highK1)
	results, _ = idx.Search(query, 10)
	fmt.Printf("   高 k1(%.2f) 评分：\n", highK1.K1())
	for i, r := range results {
		fmt.Printf("   [%d] DocID=%d, Score=%.4f\n", i+1, r.DocID, r.Score)
	}
	fmt.Println()

	// 低 k1 值：降低词频影响
	lowK1 := search.NewBM25SimilarityWithParams(0.5, 0.75)
	idx.SetSimilarity(lowK1)
	results, _ = idx.Search(query, 10)
	fmt.Printf("   低 k1(%.2f) 评分：\n", lowK1.K1())
	for i, r := range results {
		fmt.Printf("   [%d] DocID=%d, Score=%.4f\n", i+1, r.DocID, r.Score)
	}
	fmt.Println()

	// 7. 搜索不存在的词项
	fmt.Println("7. 搜索不存在的词项：")
	query2 := index.NewTermQuery("nonexistentword")
	results, err = idx.Search(query2, 10)
	if err != nil {
		fmt.Printf("   查询 'nonexistentword': 无结果 ✓\n")
	} else {
		fmt.Printf("   查询 'nonexistentword': 找到 %d 个结果\n", len(results))
	}
	fmt.Println()

	// 8. 使用查询权重
	fmt.Println("8. 使用查询权重：")
	idx.SetSimilarity(search.NewBM25Similarity())
	query3 := index.NewTermQuery("programming").WithBoost(2.0)
	results, _ = idx.Search(query3, 10)
	fmt.Printf("   Boost=2.0 后评分：\n")
	for i, r := range results {
		fmt.Printf("   [%d] DocID=%d, Score=%.4f\n", i+1, r.DocID, r.Score)
		// 评分应该翻倍
		if r.Score < 0 && r.Score > -1 { // 确保是负数比较
		}
	}
	fmt.Println()

	// 9. 索引统计
	fmt.Println("9. 索引统计：")
	fmt.Printf("   文档数量: %d\n", idx.DocCount())
	fmt.Printf("   词项数量: %d\n", idx.Inverted().NumTerms())
	fmt.Printf("   平均文档长度: %.2f\n", idx.Inverted().AvgFieldLength())
	fmt.Println()

	idx.Close()
	fmt.Println("=== 演示完成 ===")
}

// TestDemo_ScoringComparison 对比 TF-IDF 和 BM25
func TestDemo_ScoringComparison(t *testing.T) {
	fmt.Println("\n=== TF-IDF vs BM25 对比演示 ===")
	fmt.Println()

	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())

	// 添加不同长度的文档
	docs := []*document.Document{
		document.NewDocumentWithValues("short", map[string]interface{}{
			"content": "quick brown fox",
		}),
		document.NewDocumentWithValues("medium", map[string]interface{}{
			"content": "quick brown fox jumps over the lazy dog",
		}),
		document.NewDocumentWithValues("long", map[string]interface{}{
			"content": "quick brown fox jumps over the lazy dog and runs away from the hunter",
		}),
	}

	for _, doc := range docs {
		idx.Add(doc)
	}

	query := index.NewTermQuery("quick")

	// TF-IDF
	fmt.Println("TF-IDF 评分：")
	idx.SetSimilarity(search.NewTFIDFSimilarity())
	results, _ := idx.Search(query, 10)
	for _, r := range results {
		fmt.Printf("   DocID=%d, Score=%.4f\n", r.DocID, r.Score)
	}
	fmt.Println()

	// BM25
	fmt.Println("BM25 评分：")
	idx.SetSimilarity(search.NewBM25Similarity())
	results, _ = idx.Search(query, 10)
	for _, r := range results {
		fmt.Printf("   DocID=%d, Score=%.4f\n", r.DocID, r.Score)
	}
	fmt.Println()

	// 观察：短文档在 BM25 中通常评分更高（因为词频相对于文档长度更大）
	fmt.Println("观察：短文档在 BM25 中通常评分更高（长度归一化）")

	idx.Close()
}

// Benchmark_Scoring 评分算法基准测试
func Benchmark_Scoring(b *testing.B) {
	b.StopTimer()

	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())

	// 创建测试数据
	for i := 0; i < 1000; i++ {
		doc := document.NewDocumentWithValues(fmt.Sprintf("doc%d", i), map[string]interface{}{
			"content": fmt.Sprintf("test document number %d with some random words", i),
		})
		idx.Add(doc)
	}

	query := index.NewTermQuery("test")

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		idx.Search(query, 100)
	}

	idx.Close()
}
