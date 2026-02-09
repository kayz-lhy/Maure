package maure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"maure/pkg/document"
	"maure/pkg/store"
)

func TestDemo_Persistence(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "maure-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Println("=== 文件持久化演示 ===")
	fmt.Printf("临时目录: %s\n\n", tmpDir)

	// 第一阶段：创建并保存索引
	fmt.Println("--- 第一阶段：创建索引并保存 ---")

	// 创建基于文件的目录
	dir, err := store.NewFSDirectory(tmpDir)
	if err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	// 创建写入器
	writer, err := dir.CreateIndexWriter()
	if err != nil {
		t.Fatalf("创建写入器失败: %v", err)
	}

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
	}

	for _, doc := range docs {
		docID, err := writer.AddDocument(doc)
		if err != nil {
			t.Fatalf("添加文档失败: %v", err)
		}
		fmt.Printf("添加文档: %s (ID: %d)\n", doc.ID(), docID)
	}

	fmt.Println("\n提交快照到磁盘...")
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭写入器失败: %v", err)
	}

	// 列出保存的文件
	files, _ := dir.ListFiles()
	fmt.Printf("保存的文件: %v\n", files)

	// 第二阶段：加载索引
	fmt.Println("\n--- 第二阶段：从磁盘加载索引 ---")

	// 打开读取器
	reader, err := dir.OpenIndexReader()
	if err != nil {
		t.Fatalf("打开读取器失败: %v", err)
	}

	fmt.Printf("加载文档数: %d\n", reader.DocCount())

	// 验证文档
	for i := 1; i <= 3; i++ {
		doc, err := reader.GetDocument(int64(i))
		if err != nil {
			fmt.Printf("文档 %d: 未找到\n", i)
		} else {
			fmt.Printf("文档 %d: %s\n", i, doc.Get("content").StringValue())
		}
	}

	// 验证词项
	terms := reader.GetTerms()
	fmt.Printf("\n索引词项数量: %d\n", len(terms))
	fmt.Printf("部分词项: %v\n", terms[:min(5, len(terms))])

	reader.Close()
	dir.Close()

	// 第三阶段：测试增量添加
	fmt.Println("\n--- 第三阶段：增量添加文档 ---")

	dir2, err := store.NewFSDirectory(tmpDir)
	if err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	writer2, err := dir2.OpenIndexWriter()
	if err != nil {
		t.Fatalf("打开写入器失败: %v", err)
	}

	newDoc := document.NewDocumentWithValues("doc4", map[string]interface{}{
		"content": "Rust is a systems programming language",
	})
	docID, err := writer2.AddDocument(newDoc)
	if err != nil {
		t.Fatalf("添加文档失败: %v", err)
	}
	fmt.Printf("添加文档: %s (ID: %d)\n", newDoc.ID(), docID)

	if err := writer2.Close(); err != nil {
		t.Fatalf("关闭写入器失败: %v", err)
	}

	// 验证增量添加
	reader2, err := dir2.OpenIndexReader()
	if err != nil {
		t.Fatalf("打开读取器失败: %v", err)
	}
	fmt.Printf("增量后文档数: %d\n", reader2.DocCount())

	doc4, _ := reader2.GetDocument(4)
	if doc4 != nil {
		fmt.Printf("文档 4: %s\n", doc4.Get("content").StringValue())
	}

	reader2.Close()
	dir2.Close()

	// 第四阶段：检查快照
	fmt.Println("\n--- 第四阶段：检查快照 ---")

	files, _ = dir.ListFiles()
	fmt.Printf("所有文件: %v\n", files)

	fmt.Println("\n=== 持久化演示完成 ===")
}

func TestDemo_Persistence_CrashRecovery(t *testing.T) {
	fmt.Println("\n=== 崩溃恢复演示 ===")

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "maure-crash-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 场景：模拟程序崩溃后恢复

	// 第一次运行：添加文档但未提交快照
	fmt.Println("\n--- 第一次运行：添加文档（未提交） ---")

	dir1, _ := store.NewFSDirectory(tmpDir)
	writer1, _ := dir1.CreateIndexWriter()

	doc1 := document.NewDocumentWithValues("doc1", map[string]interface{}{
		"content": "First document",
	})
	writer1.AddDocument(doc1)
	writer1.AddDocument(document.NewDocumentWithValues("doc2", map[string]interface{}{
		"content": "Second document",
	}))

	// 模拟崩溃：只关闭 writer1（不调用 Commit）
	writer1.Close()

	// 检查文件
	files, _ := dir1.ListFiles()
	fmt.Printf("崩溃前的文件: %v\n", files)

	// 第二次运行：从崩溃恢复
	fmt.Println("\n--- 第二次运行：崩溃恢复 ---")

	dir2, _ := store.NewFSDirectory(tmpDir)
	writer2, _ := dir2.OpenIndexWriter()

	// 检查待提交操作
	fmt.Printf("待提交操作数: %d\n", writer2.PendingOps())

	// 模拟恢复：提交之前的更改
	if err := writer2.Commit(); err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	fmt.Println("恢复成功，提交完成")

	writer2.Close()
	dir2.Close()

	// 第三次运行：验证恢复的数据
	fmt.Println("\n--- 第三次运行：验证数据 ---")

	dir3, _ := store.NewFSDirectory(tmpDir)
	reader, _ := dir3.OpenIndexReader()

	fmt.Printf("恢复后的文档数: %d\n", reader.DocCount())
	for i := 1; i <= 2; i++ {
		doc, _ := reader.GetDocument(int64(i))
		if doc != nil {
			fmt.Printf("文档 %d: %s\n", i, doc.Get("content").StringValue())
		}
	}

	reader.Close()
	dir3.Close()

	fmt.Println("\n=== 持久化逻辑分析测试 ===")

	// 创建临时目录
	analysisDir, err := os.MkdirTemp("", "maure-analysis-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(analysisDir)

	// 测试 1：WAL 截断验证
	fmt.Println("\n--- 测试 1：WAL 截断验证 ---")

	testDir1, _ := store.NewFSDirectory(analysisDir)
	testWriter1, _ := testDir1.CreateIndexWriter()

	// 添加多个文档
	for i := 1; i <= 5; i++ {
		testWriter1.AddDocument(document.NewDocumentWithValues(fmt.Sprintf("doc%d", i), map[string]interface{}{
			"content": fmt.Sprintf("Content for document %d", i),
		}))
	}

	// 提交
	testWriter1.Close()
	testDir1.Close()

	// 检查提交后的 WAL 文件大小
	checkDir, _ := store.NewFSDirectory(analysisDir)
	walFiles, _ := checkDir.ListWALFiles()
	fmt.Printf("WAL 文件: %v\n", walFiles)

	// 验证 WAL 已被截断（应该很小或为空）
	walPath := ""
	for _, f := range walFiles {
		if strings.HasPrefix(f, "wal_") {
			walPath = filepath.Join(analysisDir, f)
			break
		}
	}
	if walPath != "" {
		info, _ := os.Stat(walPath)
		fmt.Printf("WAL 文件大小: %d bytes\n", info.Size())
		if info.Size() == 0 {
			fmt.Println("✓ WAL 截断正确（空文件）")
		} else if info.Size() < 100 {
			fmt.Println("✓ WAL 截断正确（很小）")
		}
	}
	checkDir.Close()

	// 测试 2：DocCount 验证
	fmt.Println("\n--- 测试 2：DocCount 验证 ---")

	testDir2, _ := store.NewFSDirectory(analysisDir)
	testReader2, _ := testDir2.OpenIndexReader()

	docCount := testReader2.DocCount()
	fmt.Printf("文档计数: %d\n", docCount)
	if docCount == 5 {
		fmt.Println("✓ DocCount 正确")
	} else {
		t.Errorf("DocCount 错误: expected 5, got %d", docCount)
	}

	testReader2.Close()
	testDir2.Close()

	// 测试 3：恢复后 DocCount 验证
	fmt.Println("\n--- 测试 3：恢复后 DocCount 验证 ---")

	testDir3, _ := store.NewFSDirectory(analysisDir)
	testWriter3, _ := testDir3.OpenIndexWriter()

	// 添加更多文档但不提交
	testWriter3.AddDocument(document.NewDocumentWithValues("doc6", map[string]interface{}{
		"content": "Document 6 before crash",
	}))
	testWriter3.AddDocument(document.NewDocumentWithValues("doc7", map[string]interface{}{
		"content": "Document 7 before crash",
	}))

	// 获取添加后的待提交操作数
	pendingBeforeClose := testWriter3.PendingOps()
	fmt.Printf("关闭前的待提交操作: %d\n", pendingBeforeClose)

	// 模拟崩溃（只关闭 writer，不提交）
	testWriter3.Close()

	// 恢复
	testDir4, _ := store.NewFSDirectory(analysisDir)
	testWriter4, _ := testDir4.OpenIndexWriter()

	// WAL replay 应该恢复之前未提交的操作
	// 注意：WAL 在第一次打开时被 replay，所以 pendingOps 可能已经应用
	// 但文档数据应该已经恢复
	fmt.Printf("恢复后的待提交操作: %d\n", testWriter4.PendingOps())

	// 提交
	testWriter4.Close()

	// 验证 DocCount
	testReader4, _ := testDir4.OpenIndexReader()
	totalCount := testReader4.DocCount()
	fmt.Printf("恢复后的文档总数: %d\n", totalCount)
	if totalCount == 7 {
		fmt.Println("✓ 恢复后 DocCount 正确")
	} else {
		t.Errorf("DocCount 错误: expected 7, got %d", totalCount)
	}
	testReader4.Close()
	testDir4.Close()

	fmt.Println("\n=== 持久化逻辑分析测试完成 ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
