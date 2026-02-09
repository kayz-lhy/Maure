// Package store 提供了索引存储的抽象接口和实现。
//
// 存储层负责索引数据的持久化和读取，支持多种存储后端。
// Directory 接口定义了基本的存储操作，RAMDirectory 提供内存存储实现。
package store

import (
	"io"
	"maure/pkg/document"
)

// Directory 定义了索引存储的接口。
//
// Directory 是索引存储的抽象，支持以下操作：
//   - 创建索引写入器（IndexWriter）
//   - 打开索引读取器（IndexReader）
//   - 管理索引文件
//
// 实现者需要保证线程安全性。
type Directory interface {
	// CreateIndexWriter 创建新的索引写入器。
	CreateIndexWriter() (IndexWriter, error)

	// OpenIndexWriter 打开已存在的索引写入器。
	OpenIndexWriter() (IndexWriter, error)

	// OpenIndexReader 打开索引读取器。
	OpenIndexReader() (IndexReader, error)

	// Exists 检查文件是否存在。
	Exists(name string) bool

	// DeleteFile 删除文件。
	DeleteFile(name string) error

	// ListFiles 列出所有文件。
	ListFiles() ([]string, error)

	// Close 释放资源。
	Close() error
}

// IndexWriter 负责写入索引数据。
//
// IndexWriter 提供文档添加、删除和索引优化操作。
// 使用流程：
//  1. 调用 AddDocument 添加文档
//  2. 可选：调用 Delete 删除文档
//  3. 调用 Close 提交更改
type IndexWriter interface {
	// AddDocument 添加文档到索引。
	//
	// doc: 要添加的文档
	// 返回分配的文档 ID
	AddDocument(doc *document.Document) (int64, error)

	// Delete 删除文档。
	//
	// docID: 要删除的文档 ID
	Delete(docID int64) error

	// UpdateDocument 更新文档（先删除后添加）。
	UpdateDocument(docID int64, doc *document.Document) error

	// Commit 提交更改并刷新到存储。
	Commit() error

	// PendingOps 返回待提交的操作数。
	PendingOps() int

	// Close 关闭写入器并提交更改。
	Close() error
}

// IndexReader 负责读取索引数据。
type IndexReader interface {
	// DocCount 返回索引中的文档数量。
	DocCount() int64

	// GetDocument 获取指定文档 ID 的文档。
	GetDocument(docID int64) (*document.Document, error)

	// Exists 检查文档是否存在。
	Exists(docID int64) bool

	// GetTerms 获取所有词项。
	GetTerms() []string

	// Close 释放资源。
	Close() error
}

// IndexOutput 用于写入索引数据。
type IndexOutput interface {
	io.Writer
	// WriteVInt 写入可变长度整数。
	WriteVInt(v int64) error
	// WriteString 写入字符串。
	WriteString(s string) error
	// Flush 刷新到存储。
	Flush() error
	// Close 关闭输出流。
	Close() error
}

// IndexInput 用于读取索引数据。
type IndexInput interface {
	io.Reader
	// ReadVInt 读取可变长度整数。
	ReadVInt() (int64, error)
	// ReadString 读取字符串。
	ReadString() (string, error)
	// Seek 移动读取位置。
	Seek(offset int64, whence int) (int64, error)
	// Close 关闭输入流。
	Close() error
}

// PostingsWriter 写入倒排表数据。
type PostingsWriter interface {
	// AddPostings 添加倒排表条目。
	AddPostings(docID int64, freq int, positions []int) error
	// Finish 完成写入。
	Finish() error
	// Close 释放资源。
	Close() error
}

// PostingsReader 读取倒排表数据。
type PostingsReader interface {
	// GetPostings 获取词项的倒排表。
	GetPostings(term string) (*Postings, error)
	// Close 释放资源。
	Close() error
}

// Postings 倒排表数据。
type Postings struct {
	DocIDs    []int64 // 文档 ID 列表
	Freqs     []int   // 词频列表
	Positions [][]int // 位置列表
}

// NewPostings 创建新的倒排表。
func NewPostings() *Postings {
	return &Postings{
		DocIDs:    make([]int64, 0, 4),
		Freqs:     make([]int, 0, 4),
		Positions: make([][]int, 0, 4),
	}
}
