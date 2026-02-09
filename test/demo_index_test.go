package maure

import (
	"fmt"
	"testing"

	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/index"
)

// TestDemo_IndexAndInvertedIndex 演示倒排索引的完整工作流。
//
// 这个测试展示了：
// 1. 如何创建 RAMIndex
// 2. 如何添加文档到索引
// 3. 倒排索引如何存储词项到文档的映射
// 4. 如何执行词项查询
//
// 运行方式：go test -v -run TestDemoIndex ./test/...
func TestDemo_IndexAndInvertedIndex(t *testing.T) {
	fmt.Println("=== Maure 倒排索引演示 ===")
	fmt.Println()

	// 1. 创建索引
	fmt.Println("1. 创建 RAMIndex：")
	idx := index.NewRAMIndex(analyzer.NewStandardAnalyzer())
	fmt.Println("   使用 StandardAnalyzer（英文分词、小写转换、停用词过滤）")
	fmt.Println()

	// 2. 添加文档
	fmt.Println("2. 添加文档到索引：")
	docs := []*document.Document{
		document.NewDocumentWithValues("doc1", map[string]interface{}{
			"title":   "The Quick Brown Fox",
			"content": "A fast brown fox jumps over the lazy dog",
		}),
		document.NewDocumentWithValues("doc2", map[string]interface{}{
			"title":   "Python Programming",
			"content": "Python is a popular programming language",
		}),
		document.NewDocumentWithValues("doc3", map[string]interface{}{
			"title":   "Go Language",
			"content": "Go is a systems programming language",
		}),
	}

	for i, doc := range docs {
		docID, err := idx.Add(doc)
		if err != nil {
			t.Fatalf("failed to add doc %d: %v", i, err)
		}
		fmt.Printf("   添加 doc%d (ID=%d): %s\n", i+1, docID, doc.Get("title").StringValue())
	}
	fmt.Println()

	// 3. 查看索引统计
	fmt.Println("3. 索引统计：")
	fmt.Printf("   文档数量: %d\n", idx.DocCount())
	fmt.Printf("   词项数量: %d\n", idx.Inverted().NumTerms())
	fmt.Println()

	// 4. 查看倒排索引内容
	fmt.Println("4. 倒排索引内容（部分词项）：")
	terms := idx.Inverted().GetTerms()
	shown := 0
	for _, term := range terms {
		if shown >= 10 {
			fmt.Println("   ...")
			break
		}
		postings, err := idx.Inverted().GetPostings(term)
		if err != nil {
			continue
		}
		fmt.Printf("   %q -> [%v] (freq:%v)\n", term, postings.DocIDs, postings.Freqs)
		shown++
	}
	fmt.Println()

	// 5. 执行查询
	fmt.Println("5. 词项查询示例：")

	queries := []struct {
		term    string
		desc    string
	}{
		{"fox", "查询 'fox'"},
		{"programming", "查询 'programming'"},
		{"python", "查询 'python'（小写转换）"},
		{"language", "查询 'language'"},
		{"nonexistent", "查询不存在的词项"},
	}

	for _, q := range queries {
		query := index.NewTermQuery(q.term)
		results, err := idx.Search(query, 10)
		if err != nil {
			fmt.Printf("   %s: 无结果\n", q.desc)
		} else {
			fmt.Printf("   %s: 找到 %d 个文档 %v\n", q.desc, len(results), results)
		}
	}
	fmt.Println()

	// 6. 演示文档删除
	fmt.Println("6. 演示文档删除：")
	fmt.Printf("   删除前文档数量: %d\n", idx.DocCount())
	if err := idx.Delete(1); err != nil {
		t.Fatalf("failed to delete doc: %v", err)
	}
	fmt.Printf("   删除后文档数量: %d\n", idx.DocCount())
	fmt.Printf("   'fox' 词项还存在: %v\n", idx.Inverted().ContainsTerm("fox"))
	fmt.Println()

	// 7. 演示文档更新
	fmt.Println("7. 演示文档更新：")
	newDoc := document.NewDocumentWithValues("doc2-updated", map[string]interface{}{
		"title":   "Python Updated",
		"content": "Python has been updated to a new version",
	})
	if err := idx.Update(2, newDoc); err != nil {
		t.Fatalf("failed to update doc: %v", err)
	}
	fmt.Printf("   更新后 'python' 词项存在: %v\n", idx.Inverted().ContainsTerm("python"))
	fmt.Printf("   更新后 'updated' 词项存在: %v\n", idx.Inverted().ContainsTerm("updated"))
	fmt.Println()

	idx.Close()
	fmt.Println("=== 演示完成 ===")
}
