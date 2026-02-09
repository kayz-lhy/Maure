package maure

import (
	"fmt"
	"testing"

	"maure/pkg/analyzer"
	"maure/pkg/document"
)

// TestDemo_DocumentAndAnalyzer 演示文档和分词器的完整工作流。
//
// 这个测试展示了：
// 1. 如何创建文档并添加字段
// 2. StandardAnalyzer 如何对文本进行分析
// 3. TokenStream 如何迭代访问分词结果
//
// 运行方式：go test -v -run TestDemo
func TestDemo_DocumentAndAnalyzer(t *testing.T) {
	fmt.Println("=== Maure 文档与分词器演示 ===")
	fmt.Println()

	// 1. 创建文档
	fmt.Println("1. 创建文档并添加字段：")
	doc := document.NewDocument()
	doc.Add(document.NewTextField("title", "The Quick Brown Fox"))
	doc.Add(document.NewTextField("content", "A lazy dog jumps over the fast cat"))
	doc.Add(document.NewStringField("status", "published"))
	doc.Add(document.NewInt64Field("views", 12345))
	fmt.Printf("   文档字段：%s\n", doc.String())
	fmt.Println()

	// 2. 使用 StandardAnalyzer 分析文本
	fmt.Println("2. 使用 StandardAnalyzer 分析文本：")
	analyzerInstance := analyzer.NewStandardAnalyzer()

	// 分析 title 字段
	fmt.Println("   输入（title）: \"The Quick Brown Fox\"")
	stream := analyzerInstance.Analyze("title", "The Quick Brown Fox")
	fmt.Print("   输出（词项）: [")
	first := true
	for stream.Next() {
		token := stream.Current()
		if !first {
			fmt.Print(", ")
		}
		fmt.Printf("%q(%s)", token.Text, token.Type)
		first = false
	}
	fmt.Println("]")
	stream.Close()
	fmt.Println()

	// 分析 content 字段
	fmt.Println("   输入（content）: \"A lazy dog jumps over the fast cat\"")
	stream = analyzerInstance.Analyze("content", "A lazy dog jumps over the fast cat")
	fmt.Print("   输出（词项）: [")
	first = true
	for stream.Next() {
		token := stream.Current()
		if !first {
			fmt.Print(", ")
		}
		fmt.Printf("%q(%s)", token.Text, token.Type)
		first = false
	}
	fmt.Println("]")
	stream.Close()
	fmt.Println()

	// 3. 演示不同的 Analyzer 配置
	fmt.Println("3. 不同 Analyzer 配置对比：")

	// 不带停用词过滤
	analyzerNoStop := analyzer.NewStandardAnalyzerWithOpts(
		analyzer.WithoutLowerCase(),
	)
	fmt.Println("   输入: \"This is a test document\"")

	stream = analyzerNoStop.Analyze("test", "This is a test document")
	fmt.Print("   不带停用词过滤: [")
	first = true
	for stream.Next() {
		token := stream.Current()
		if !first {
			fmt.Print(", ")
		}
		fmt.Printf("%q", token.Text)
		first = false
	}
	fmt.Println("]")
	stream.Close()

	// 带停用词过滤（默认）
	stream = analyzerInstance.Analyze("test", "This is a test document")
	fmt.Print("   带停用词过滤:   [")
	first = true
	for stream.Next() {
		token := stream.Current()
		if !first {
			fmt.Print(", ")
		}
		fmt.Printf("%q", token.Text)
		first = false
	}
	fmt.Println("]")
	stream.Close()
	fmt.Println()

	// 4. 演示字段类型
	fmt.Println("4. 字段类型说明：")
	fields := []*document.Field{
		document.NewTextField("content", "Full text for searching"),
		document.NewStringField("category", "technology"),
		document.NewInt64Field("year", 2024),
		document.NewFloat64Field("price", 29.99),
		document.NewStoredField("metadata", "some data"),
	}
	for _, f := range fields {
		fmt.Printf("   %-12s: 类型=%s, 索引=%v, 分词=%v, 存储=%v\n",
			f.Name, f.FieldType, f.Indexed, f.Tokenized, f.Stored)
	}
	fmt.Println()

	fmt.Println("=== 演示完成 ===")
}