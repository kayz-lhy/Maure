// Package index 提供了索引的核心实现。
//
// RAMIndex 是基于内存的索引实现，适合中小规模数据集。
// 它封装了倒排索引、文档存储和并发控制。
package index

import (
	"errors"
	"sort"
	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/search"
	"sync"
)

var (
	// ErrClosed 索引已关闭。
	ErrClosed = errors.New("index closed")
	// ErrDocNotFound 文档未找到。
	ErrDocNotFound = errors.New("document not found")
)

// Index 定义了索引的接口。
type Index interface {
	// Add 添加文档到索引。
	Add(doc *document.Document) (int64, error)

	// Delete 从索引中删除文档。
	Delete(docID int64) error

	// Update 更新文档（先删除后添加）。
	Update(docID int64, doc *document.Document) error

	// Search 搜索查询。
	Search(query Query, n int) ([]ScoreDoc, error)

	// DocCount 返回文档数量。
	DocCount() int64

	// GetDocument 获取文档。
	GetDocument(docID int64) (*document.Document, error)

	// Close 关闭索引。
	Close() error
}

// RAMIndex 是基于内存的索引实现。
type RAMIndex struct {
	mu         sync.RWMutex
	inverted   *InvertedIndex       // 倒排索引
	analyzer   analyzer.Analyzer    // 分析器
	similarity search.Similarity   // 评分算法
	closed     bool
}

// NewRAMIndex 创建新的 RAMIndex。
func NewRAMIndex(analyzerInstance analyzer.Analyzer) *RAMIndex {
	return &RAMIndex{
		inverted:   NewInvertedIndex(),
		analyzer:   analyzerInstance,
		similarity: search.NewBM25Similarity(),
	}
}

// NewRAMIndexWithConfig 创建带有配置的 RAMIndex。
func NewRAMIndexWithConfig(opts ...RAMIndexOption) *RAMIndex {
	idx := &RAMIndex{
		inverted:   NewInvertedIndex(),
		analyzer:   analyzer.NewStandardAnalyzer(),
		similarity: search.NewBM25Similarity(),
	}
	for _, opt := range opts {
		opt(idx)
	}
	return idx
}

// RAMIndexOption 配置 RAMIndex。
type RAMIndexOption func(*RAMIndex)

// WithAnalyzer 设置分析器。
func WithAnalyzer(a analyzer.Analyzer) RAMIndexOption {
	return func(idx *RAMIndex) {
		idx.analyzer = a
	}
}

// WithSimilarity 设置评分算法。
func WithSimilarity(s search.Similarity) RAMIndexOption {
	return func(idx *RAMIndex) {
		idx.similarity = s
	}
}

// Add 实现了 Index 接口。
func (idx *RAMIndex) Add(doc *document.Document) (int64, error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.closed {
		return 0, ErrClosed
	}

	return idx.inverted.AddDocument(doc)
}

// Delete 实现了 Index 接口。
func (idx *RAMIndex) Delete(docID int64) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.closed {
		return ErrClosed
	}

	return idx.inverted.Delete(docID)
}

// Update 实现了 Index 接口。
func (idx *RAMIndex) Update(docID int64, doc *document.Document) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.closed {
		return ErrClosed
	}

	if err := idx.inverted.Delete(docID); err != nil {
		return err
	}
	_, err := idx.inverted.AddDocument(doc)
	return err
}

// Search 实现了 Index 接口。
func (idx *RAMIndex) Search(query Query, n int) ([]ScoreDoc, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.closed {
		return nil, ErrClosed
	}

	return query.Search(idx)
}

// DocCount 实现了 Index 接口。
func (idx *RAMIndex) DocCount() int64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.inverted.DocCount()
}

// GetDocument 获取文档。
func (idx *RAMIndex) GetDocument(docID int64) (*document.Document, error) {
	// RAMIndex 目前不存储原始文档
	return nil, ErrDocNotFound
}

// Close 实现了 Index 接口。
func (idx *RAMIndex) Close() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.closed {
		return nil
	}
	idx.closed = true
	return nil
}

// Inverted 返回内部的倒排索引。
func (idx *RAMIndex) Inverted() *InvertedIndex {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.inverted
}

// Analyzer 返回分析器。
func (idx *RAMIndex) Analyzer() analyzer.Analyzer {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.analyzer
}

// Similarity 返回评分算法。
func (idx *RAMIndex) Similarity() search.Similarity {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.similarity
}

// SetSimilarity 设置评分算法。
func (idx *RAMIndex) SetSimilarity(s search.Similarity) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.similarity = s
}

// ScoreDoc 表示搜索结果中的一个文档。
type ScoreDoc struct {
	DocID int64   // 文档 ID
	Score float32 // 相关性评分
}

// ScoreDocs 是 ScoreDoc 的切片，实现了排序接口。
type ScoreDocs []ScoreDoc

// Len 实现了 sort.Interface。
func (s ScoreDocs) Len() int {
	return len(s)
}

// Less 实现了 sort.Interface（按评分降序排序）。
func (s ScoreDocs) Less(i, j int) bool {
	return s[i].Score > s[j].Score
}

// Swap 实现了 sort.Interface。
func (s ScoreDocs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// TopN 返回评分最高的 n 个结果。
func (s ScoreDocs) TopN(n int) []ScoreDoc {
	if n >= len(s) {
		return s
	}
	result := make([]ScoreDoc, n)
	copy(result, s[:n])
	return result
}

// Query 定义了查询的接口。
type Query interface {
	// Search 执行查询并返回匹配的文档。
	Search(idx *RAMIndex) ([]ScoreDoc, error)
}

// TermQuery 是词项查询。
type TermQuery struct {
	Term     string
	Field    string
	Boost    float32
}

// NewTermQuery 创建新的 TermQuery。
func NewTermQuery(term string) *TermQuery {
	return &TermQuery{
		Term:  term,
		Boost: 1.0,
	}
}

// WithField 设置查询字段。
func (q *TermQuery) WithField(field string) *TermQuery {
	q.Field = field
	return q
}

// WithBoost 设置权重。
func (q *TermQuery) WithBoost(boost float32) *TermQuery {
	q.Boost = boost
	return q
}

// Search 实现了 Query 接口。
func (q *TermQuery) Search(idx *RAMIndex) ([]ScoreDoc, error) {
	postings, err := idx.inverted.GetPostings(q.Term)
	if err != nil {
		return nil, err
	}

	// 获取索引统计信息
	numDocs := idx.inverted.DocCount()
	avgLength := idx.inverted.AvgFieldLength()

	// 计算每个文档的评分
	results := make([]ScoreDoc, 0, len(postings.DocIDs))
	for i, docID := range postings.DocIDs {
		termFreq := postings.Freqs[i]
		docLength := idx.inverted.FieldLength(docID)

		// 使用评分算法计算
		score := idx.similarity.Score(
			termFreq,
			len(postings.DocIDs),
			docLength,
			avgLength,
			numDocs,
		) * q.Boost

		results = append(results, ScoreDoc{
			DocID: docID,
			Score: score,
		})
	}

	// 按评分降序排序
	sort.Sort(sort.Reverse(ScoreDocs(results)))

	return results, nil
}
